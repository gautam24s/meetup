package meetup

import (
	"context"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

type Options struct {
	EnableBridging          bool
	EnableBandwithEstimator bool
	IceServers              []webrtc.ICEServer
	MinPlayoutDelay         uint16
	MaxPlayoutDelay         uint16
	SettingEngine           *webrtc.SettingEngine
}

func DefaultOptions() Options {
	settingEngine := &webrtc.SettingEngine{}
	_ = settingEngine.SetEphemeralUDPPortRange(49152, 65535)
	settingEngine.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})

	return Options{
		EnableBandwithEstimator: true,
		IceServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
		MinPlayoutDelay: 100,
		MaxPlayoutDelay: 100,
		SettingEngine:   settingEngine,
	}
}

type RoomOptions struct {
	Bitrates         BitrateConfigs `json:"bitrates,omitempty"`
	Codecs           *[]string      `json:"codecs,omitempty"`
	PLIInterval      *time.Duration `json:"pli_interval_ns,omitempty"`
	QualityLevels    []QualityLevel `json:"quality_levels,omitempty"`
	EmptyRoomTimeout *time.Duration `json:"empty_room_timeout_ns,omitempty"`
}

func DefaultRoomOptions() RoomOptions {
	pli := time.Duration(0)
	emptyDuration := time.Duration(3) * time.Minute
	return RoomOptions{
		Bitrates:      DefaultBitrates(),
		QualityLevels: DefaultQualityLevels(),
		Codecs: &[]string{
			webrtc.MimeTypeVP9,
			webrtc.MimeTypeH264,
			webrtc.MimeTypeVP8,
			"audio/red",
			webrtc.MimeTypeOpus,
		},
		PLIInterval:      &pli,
		EmptyRoomTimeout: &emptyDuration,
	}
}

type Event struct {
	Type string
	Time time.Time
	Data map[string]any
}

type Room struct {
	onRoomClosedCallbacks   []func(id string)
	onClientJoinedCallbacks []func(*Client)
	onClientLeftCallbacks   []func(*Client)
	context                 context.Context
	cancel                  context.CancelFunc
	id                      string
	token                   string
	RenegotiationChan       map[string]chan bool
	name                    string
	mu                      *sync.RWMutex
	meta                    *Metadata
	state                   string
	kind                    string
	OnEvent                 func(event Event)
	options                 RoomOptions
}
