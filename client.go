package meetup

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gautam24s/meetup/pkg/interceptors/voiceactivedetector"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/cc"
	"github.com/pion/interceptor/pkg/gcc"
	"github.com/pion/interceptor/pkg/nack"
	"github.com/pion/logging"
	"github.com/pion/webrtc/v4"
)

const (
	ClientStateNew     = 0
	ClientStateActive  = 1
	ClientStateRestart = 2
	ClientStateEnded   = 3

	ClientTypePeer       = "peer"
	ClientTypeUpBridge   = "upbridge"
	ClientTypeDownBridge = "downbridge"

	QualityAudioRed = 11
	QualityAudio    = 10
	QualityHigh     = 9
	QualityHighMid  = 8
	QualityHighLow  = 7
	QualityMid      = 6
	QualityMidMid   = 5
	QualityMidLow   = 4
	QualityLow      = 3
	QualityLowMid   = 2
	QualityLowLow   = 1
	QualityNone     = 0

	messageTypeVideoSize  = "video_size"
	messageTypeStats      = "stats"
	messageTypeVADStarted = "vad_started"
	messageTypeVADEnded   = "vad_ended"
)

type QualityLevel uint32

var (
	ErrNegotiationIsNotRequested = errors.New("client: error negotitation is called before requested")
	ErrRenegotiationCallback     = errors.New("client: error renegotiation callback is not set")
	ErrClientStopped             = errors.New("client: error client already stopped")
)

type ClientOptions struct {
	IceTrickle           bool          `json:"ice_trickle"`
	IdleTimeout          time.Duration `json:"idle_timeout"`
	Type                 string        `json:"type"`
	EnableVoiceDetection bool          `json:"enable_voice_detection"`
	MinPlayoutDelay      uint16        `json:"min_playout_delay"`
	MaxPlayoutDelay      uint16        `json:"max_playout_delay"`
	JitterBufferMinWait  time.Duration `json:"jitter_buffer_min_wait"`
	JitterBufferMaxWait  time.Duration `json:"jitter_buffer_max_wait"`
	ReorderPackets       bool          `json:"reorder_packets"`
	Log                  logging.LeveledLogger
	settingEngine        webrtc.SettingEngine
	qualityLevels        []QualityLevel
}

func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		IceTrickle:           true,
		IdleTimeout:          5 * time.Minute,
		Type:                 ClientTypePeer,
		EnableVoiceDetection: true,
		MinPlayoutDelay:      100,
		MaxPlayoutDelay:      200,
		JitterBufferMinWait:  20 * time.Millisecond,
		JitterBufferMaxWait:  150 * time.Millisecond,
		ReorderPackets:       false,
		Log:                  logging.NewDefaultLoggerFactory().NewLogger("sfu"),
	}
}

type Client struct {
	id                  string
	name                string
	bitrateController   *bitrateController
	context             context.Context
	cancel              context.CancelFunc
	canAddCandidate     *atomic.Bool
	clientTracks        map[string]iClientTrack
	muTracks            sync.Mutex
	internalDataChannel *webrtc.DataChannel

	estimator             cc.BandwidthEstimator
	initialReceiverCount  atomic.Int32
	initialSenderCount    atomic.Int32
	isInRenegotiation     *atomic.Bool
	isInRemoteNegotiation *atomic.Bool
	idleTimeoutContext    context.Context
	idleTimeoutCancel     context.CancelFunc

	mu             sync.Mutex
	peerConnection *PeerConnection

	pendingRemoteRenegotiation *atomic.Bool
	receiveRED                 bool
	state                      *atomic.Value

	muCallback                        sync.Mutex
	onConnectionStateChangedCallbacks []func(webrtc.PeerConnectionState)
	onJoinedCallbacks                 []func()
	onLeftCallbacks                   []func()
	onVoiceSentDetectedCallbacks      []func(voiceactivedetector.VoiceActivity)
	onVoiceReceivedDetectedCallbacks  []func(voiceactivedetector.VoiceActivity)
	onTrackRemovedCallbacks           []func(sourceType string, track *webrtc.TrackLocalStaticRTP)
	onIceCandidate                    func(context.Context, *webrtc.ICECandidate)
	onRenegotiation                   func(context.Context, webrtc.SessionDescription) (webrtc.SessionDescription, error)
	onAllowedRemoteRenegotiation      func()

	options           ClientOptions
	negotiationNeeded *atomic.Bool

	pendingRemoteCandidates        []webrtc.ICECandidateInit
	pendingLocalCandidates         []*webrtc.ICECandidate
	quality                        *atomic.Uint32
	receivingBandwith              *atomic.Uint32
	egressBandwith                 *atomic.Uint32
	ingressBandwith                *atomic.Uint32
	ingressQualityLimitationReason *atomic.Value
	isDebug                        bool
	vadInterceptor                 *voiceactivedetector.Interceptor
	vads                           map[uint32]*voiceactivedetector.VoiceDetector
	log                            logging.LeveledLogger
}

func NewClient(s *SFU, id string, name string, peerConnectionConfig webrtc.Configuration, opts ClientOptions) {
	var client *Client
	var vadInterceptor *voiceactivedetector.Interceptor

	localCtx, cancel := context.WithCancel(s.context)
	m := &webrtc.MediaEngine{}

	opts.settingEngine.EnableSCTPZeroChecksum(true)

	if err := RegisterCodecs(m, s.codecs); err != nil {
		panic(err)
	}

	RegisterSimulcastHeaderExtensions(m, webrtc.RTPCodecTypeVideo)

	if opts.EnableVoiceDetection {
		voiceactivedetector.RegisterAudioLevelHeaderExtension(m)
	}

	i := &interceptor.Registry{}

	var vads = make(map[uint32]*voiceactivedetector.VoiceDetector)

	if opts.EnableVoiceDetection {
		opts.Log.Infof("client: voice detection is enabled")
		vadInterceptorFactory := voiceactivedetector.NewInterceptor(localCtx, opts.Log)

		vadInterceptorFactory.OnNew(func(i *voiceactivedetector.Interceptor) {
			vadInterceptor = i
			i.OnNewVAD(func(vad *voiceactivedetector.VoiceDetector) {
				vads[vad.SSRC()] = vad
			})
		})

		i.Add(vadInterceptorFactory)
	}

	estimatorChan := make(chan cc.BandwidthEstimator, 1)

	congestionController, err := cc.NewInterceptor(func() (cc.BandwidthEstimator, error) {
		return gcc.NewSendSideBWE(
			gcc.SendSideBWEInitialBitrate(int(s.bitrateConfigs.InitialBandwith)),
			gcc.SendSideBWEPacer(gcc.NewNoOpPacer()),
		)
	})
	if err != nil {
		panic(err)
	}

	congestionController.OnNewPeerConnection(func(id string, estimator cc.BandwidthEstimator) {
		estimatorChan <- estimator
	})

	i.Add(congestionController)

	if err = webrtc.ConfigureTWCCHeaderExtensionSender(m, i); err != nil {
		panic(err)
	}

	if err := registerInterceptors(m, i); err != nil {
		panic(err)
	}

	peerConnection, err := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithSettingEngine(opts.settingEngine), webrtc.WithInterceptorRegistry(i)).NewPeerConnection(peerConnectionConfig)
	if err != nil {
		panic(err)
	}

	var stateNew atomic.Value
	stateNew.Store(ClientStateNew)

	var quality atomic.Uint32

	quality.Store(QualityHigh)

	client = &Client{
		id:      id,
		name:    name,
		context: localCtx,
		cancel:  cancel,
	}
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Context() context.Context {
	return c.context
}

func registerInterceptors(m *webrtc.MediaEngine, interceptorRegistry *interceptor.Registry) error {
	generator, err := nack.NewGeneratorInterceptor()
	if err != nil {
		return err
	}

	responder, err := nack.NewResponderInterceptor()
	if err != nil {
		return err
	}

	m.RegisterFeedback(webrtc.RTCPFeedback{Type: "nack"}, webrtc.RTPCodecTypeVideo)
	m.RegisterFeedback(webrtc.RTCPFeedback{Type: "nack", Parameter: "pli"}, webrtc.RTPCodecTypeVideo)
	interceptorRegistry.Add(generator)
	interceptorRegistry.Add(responder)

	if err := webrtc.ConfigureRTCPReports(interceptorRegistry); err != nil {
		return err
	}

	return webrtc.ConfigureTWCCSender(m, interceptorRegistry)
}

func (c *Client) GetEstimatedBandwith() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.estimator == nil {
		return c.sfu
	}
}
