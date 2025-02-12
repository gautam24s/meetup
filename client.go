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
	id              string
	name            string
	context         context.Context
	cancel          context.CancelFunc
	canAddCandidate *atomic.Bool
	muTracks        sync.Mutex

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
