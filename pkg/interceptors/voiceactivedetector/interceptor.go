package voiceactivedetector

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/logging"
	"github.com/pion/rtp"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v4"
)

const ATTRIBUTE_KEY = "audioLevel"

type InterceptorFactory struct {
	onNew   func(u *Interceptor)
	context context.Context
	log     logging.LeveledLogger
}

type Config struct {
	Interval   time.Duration
	HeadMargin time.Duration
	TailMargin time.Duration
	Threshold  uint8
}

func DefaultConfig() Config {
	return Config{
		Interval:   100 * time.Millisecond,
		HeadMargin: 200 * time.Millisecond,
		TailMargin: 300 * time.Millisecond,
		Threshold:  40,
	}
}

type Interceptor struct {
	context context.Context
	mu      sync.RWMutex
	vads    map[uint32]*VoiceDetector
	config  Config
	onNew   func(vad *VoiceDetector)
	log     logging.LeveledLogger
}

func NewInterceptor(ctx context.Context, log logging.LeveledLogger) *InterceptorFactory {
	return &InterceptorFactory{
		context: ctx,
		log:     log,
	}
}

func (g *InterceptorFactory) NewInterceptor(_ string) (interceptor.Interceptor, error) {
	i := new(g.context, g.log)

	if g.onNew != nil {
		g.onNew(i)
	}

	return i, nil
}

func (g *InterceptorFactory) OnNew(callback func(i *Interceptor)) {
	g.onNew = callback
}

func new(ctx context.Context, log logging.LeveledLogger) *Interceptor {
	return &Interceptor{
		context: ctx,
		mu:      sync.RWMutex{},
		vads:    make(map[uint32]*VoiceDetector),
		config:  DefaultConfig(),
		log:     log,
	}
}

func (v *Interceptor) SetConfig(config Config) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.config = config
}

func (v *Interceptor) OnNewVAD(callback func(vad *VoiceDetector)) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.onNew = callback
}

func (v *Interceptor) BindLocalStream(info *interceptor.StreamInfo, writer interceptor.RTPWriter) interceptor.RTPWriter {
	return writer
}

func (v *Interceptor) UnbindLocalStream(info *interceptor.StreamInfo) {
}

func (v *Interceptor) BindRemoteStream(info *interceptor.StreamInfo, reader interceptor.RTPReader) interceptor.RTPReader {
	log.Println("stream received: ", info.SSRC)
	if info.MimeType != webrtc.MimeTypeOpus && info.MimeType != "audio/red" {
		return reader
	}

	vad := v.getVadBySSRC(info.SSRC)
	if vad != nil {
		vad.updateStreamInfo(info)
	}

	if vad == nil {
		v.vads[info.SSRC] = newVAD(v.context, v.config, info)
		vad = v.vads[info.SSRC]
	}

	if v.onNew != nil {
		v.onNew(vad)
	}
	return interceptor.RTPReaderFunc(func(b []byte, a interceptor.Attributes) (int, interceptor.Attributes, error) {
		i, attr, err := reader.Read(b, a)
		if err != nil {
			return 0, nil, err
		}

		if attr == nil {
			attr = make(interceptor.Attributes)
		}

		header, err := attr.GetRTPHeader(b[:i])
		if err != nil {
			return 0, nil, err
		}

		audioAttribute := v.processPacket(info.SSRC, header)

		attr.Set(ATTRIBUTE_KEY, audioAttribute.Level)
		attr.Set("isVoice", audioAttribute.Voice)

		return i, attr, nil
	})
}

func (v *Interceptor) UnbindRemoteStream(info *interceptor.StreamInfo) {
	vad := v.getVadBySSRC(info.SSRC)
	if vad != nil {
		vad.Stop()
	}
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.vads, info.SSRC)
}

func (v *Interceptor) Close() error {
	return nil
}

func (v *Interceptor) BindRTCPReader(reader interceptor.RTCPReader) interceptor.RTCPReader {
	return reader
}

func (v *Interceptor) BindRTCPWriter(writer interceptor.RTCPWriter) interceptor.RTCPWriter {
	return writer
}

func (v *Interceptor) processPacket(ssrc uint32, header *rtp.Header) rtp.AudioLevelExtension {
	audioData := v.getAudioLevel(ssrc, header)
	log.Println(audioData)

	if audioData.Level == 0 {
		return rtp.AudioLevelExtension{}
	}

	vad := v.getVadBySSRC(ssrc)
	if vad == nil {
		log.Println("vad: not found for track ssrc", ssrc)
		return rtp.AudioLevelExtension{}
	}

	if audioData.Voice {
		vad.addPacket(header, audioData.Level, audioData.Voice)
	}
	return audioData
}

func (v *Interceptor) getAudioLevel(ssrc uint32, header *rtp.Header) rtp.AudioLevelExtension {
	audioLevel := rtp.AudioLevelExtension{}
	headerID := v.getAudioLevelExtensionID(ssrc)

	if headerID == 0 {
		return audioLevel
	}

	ext := header.GetExtension(headerID)
	if ext == nil {
		return audioLevel
	}

	_ = audioLevel.Unmarshal(ext)
	log.Printf("audio level: %v", audioLevel)
	return audioLevel
}

func (v *Interceptor) getVadBySSRC(ssrc uint32) *VoiceDetector {
	v.mu.RLock()
	defer v.mu.RUnlock()

	vad, ok := v.vads[ssrc]
	if ok {
		return vad
	}
	return nil
}

func (v *Interceptor) getAudioLevelExtensionID(ssrc uint32) uint8 {
	vad := v.getVadBySSRC(ssrc)
	if vad != nil {
		for _, extension := range vad.streamInfo.RTPHeaderExtensions {
			if extension.URI == sdp.AudioLevelURI {
				return uint8(extension.ID)
			}
		}
	}
	return 0
}

func (v *Interceptor) MapAudioTrack(ssrc uint32, t *webrtc.TrackRemote) *VoiceDetector {
	if t.Kind() != webrtc.RTPCodecTypeAudio {
		log.Println("vad: track is not audio track")
		return nil
	}

	vad := v.getVadBySSRC(ssrc)
	if vad == nil {
		vad = newVAD(v.context, v.config, nil)
		v.mu.Lock()
		v.vads[ssrc] = vad
		v.mu.Unlock()
	}

	vad.UpdateTrack(t.ID(), t.StreamID())
	return vad
}

func RegisterAudioLevelHeaderExtension(m *webrtc.MediaEngine) {
	if err := m.RegisterHeaderExtension(webrtc.RTPHeaderExtensionCapability{URI: sdp.AudioLevelURI}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}
}
