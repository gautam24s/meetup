package voiceactivedetector

import (
	"context"
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/logging"
	"github.com/pion/rtp"
)

type VoicePacketData struct {
	SequenceNo uint16 `json:"sequenceNo"`
	Timestamp  uint32 `json:"timestamp"`
	AudioLevel uint8  `json:"audioLevel"`
	IsVoice    bool   `json:"isVoice"`
}

type VoiceActivity struct {
	TrackID     string            `json:"trackID"`
	StreamID    string            `json:"streamID"`
	SSRC        uint32            `json:"ssrc"`
	ClockRate   uint32            `json:"clockRate"`
	AudioLevels []VoicePacketData `json:"audioLevels"`
}

type VoiceDetector struct {
	streamInfo   *interceptor.StreamInfo
	config       Config
	streamID     string
	trackID      string
	context      context.Context
	cancel       context.CancelFunc
	channel      chan VoicePacketData
	mu           sync.RWMutex
	VoicePackets []VoicePacketData
	callback     func([]VoicePacketData)
	log          logging.LeveledLogger
}

func newVAD(ctx context.Context, config Config, streamInfo *interceptor.StreamInfo) *VoiceDetector {
	v := &VoiceDetector{
		context:      ctx,
		config:       config,
		streamInfo:   streamInfo,
		channel:      make(chan VoicePacketData, 1024),
		mu:           sync.RWMutex{},
		VoicePackets: make([]VoicePacketData, 0),
		log:          logging.NewDefaultLoggerFactory().NewLogger("vad"),
	}

	v.run()

	return v
}

func (v *VoiceDetector) SSRC() uint32 {
	return v.streamInfo.SSRC
}

func (v *VoiceDetector) run() {
	go func() {
		ticker := time.NewTicker(v.config.TailMargin)
		ctx, cancel := context.WithCancel(v.context)

		v.cancel = cancel

		bufferTicker := time.NewTicker(v.config.Interval)

		defer func() {
			bufferTicker.Stop()
			ticker.Stop()
			cancel()
		}()

		active := false
		lastSent := time.Now()

		buffer := make([]VoicePacketData, 1024)
		bufferLength := 0

		for {
			select {
			case <-ctx.Done():
				return
			case <-bufferTicker.C:
				voicePackets := make([]VoicePacketData, bufferLength)
				copy(voicePackets, buffer)

				v.onVoiceDetected(voicePackets)
				buffer = buffer[:0]
				bufferLength = 0
			case voicePacket := <-v.channel:
				if voicePacket.AudioLevel < v.config.Threshold {
					buffer = append(buffer, voicePacket)
					bufferLength++
					lastSent = time.Now()
					active = true
				}
			case <-ticker.C:
				if active && time.Since(lastSent) > v.config.TailMargin {
					v.onVoiceDetected(nil)
					active = false
				}
			}
		}
	}()
}

func (v *VoiceDetector) onVoiceDetected(pkts []VoicePacketData) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.callback != nil {
		v.callback(pkts)
	}
}

func (v *VoiceDetector) OnVoiceDetected(callback func([]VoicePacketData)) {
	if v == nil {
		return
	}
	v.mu.RLock()
	defer v.mu.RUnlock()

	v.callback = callback
}

func (v *VoiceDetector) addPacket(header *rtp.Header, audioLevel uint8, isVoice bool) {
	vp := VoicePacketData{
		SequenceNo: header.SequenceNumber,
		Timestamp:  header.Timestamp,
		AudioLevel: audioLevel,
		IsVoice:    isVoice,
	}

	v.channel <- vp
}

func (v *VoiceDetector) UpdateTrack(trackID, streamID string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	v.trackID = trackID
	v.streamID = streamID
}

func (v *VoiceDetector) Stop() {
	v.cancel()
}

func (v *VoiceDetector) updateStreamInfo(streamInfo *interceptor.StreamInfo) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	v.streamInfo = streamInfo
}
