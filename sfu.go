package meetup

import (
	"context"
	"sync"
	"time"

	"github.com/pion/logging"
	"github.com/pion/webrtc/v4"
)

type BitrateConfigs struct {
	AudioRed        uint32 `json:"audio_red" example:"75000"`
	Audio           uint32 `json:"audio" example:"48000"`
	Video           uint32 `json:"video" example:"1200000"`
	VideoHigh       uint32 `json:"video_high" example:"1200000"`
	VideoHighPixels uint32 `json:"video_high_pixels" example:"921600"`
	VideoMid        uint32 `json:"video_mid" example:"500000"`
	VideoMidPixels  uint32 `json:"video_mid_pixels" example:"259200"`
	VideoLow        uint32 `json:"video_low" example:"150000"`
	VideoLowPixels  uint32 `json:"video_low_pixels" example:"64800"`
	InitialBandwith uint32 `json:"initial_bandwith" example:"1000000"`
}

func DefaultBitrates() BitrateConfigs {
	return BitrateConfigs{
		AudioRed:        75_000,
		Audio:           48_000,
		Video:           700_000,
		VideoHigh:       700_000,
		VideoHighPixels: 720 * 360,
		VideoMid:        300_000,
		VideoMidPixels:  360 * 180,
		VideoLow:        90_000,
		VideoLowPixels:  180 * 90,
		InitialBandwith: 1_000_000,
	}
}

type SFUClients struct {
	clients map[string]*Client
	mu      sync.Mutex
}

func (s *SFUClients) GetClients() map[string]*Client {
	s.mu.Lock()
	defer s.mu.Unlock()

	clients := make(map[string]*Client)
	for k, v := range s.clients {
		clients[k] = v
	}

	return clients
}

func (s *SFUClients) GetClient(id string) (*Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, ok := s.clients[id]; ok {
		return client, nil
	}

	return nil, ErrClientNotFound
}

func (s *SFUClients) Length() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.clients)
}

func (s *SFUClients) Add(client *Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[client.ID()]; ok {
		return ErrClientExists
	}

	s.clients[client.ID()] = client

	return nil
}

func (s *SFUClients) Remove(client *Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[client.ID()]; !ok {
		return ErrClientNotFound
	}

	delete(s.clients, client.ID())

	return nil
}

type SFU struct {
	bitrateConfigs BitrateConfigs
	clients        *SFUClients
	context        context.Context
	cancel         context.CancelFunc
	codecs         []string
	// dataChannels   *SFUDataChannelList
	iceServers                 []webrtc.ICEServer
	mu                         sync.Mutex
	onStop                     func()
	pliInterval                time.Duration
	onTracksAvailableCallbacks []func(tracks ITrack)
	onClientRemovedCallbacks   []func(*Client)
	onClientAddedCallbacks     []func(*Client)
	relayTracks                map[string]ITrack
	// clientStats                map[string]*ClientStats
	log                  logging.LeveledLogger
	defaultSettingEngine *webrtc.SettingEngine
}

type PublishedTrack struct {
	ClientID string
	Track    webrtc.TrackLocal
}

type sfuOptions struct {
	IceServers    []webrtc.ICEServer
	Bitrates      BitrateConfigs
	QualityLevel  []QualityLevel
	Codecs        []string
	PLIInterval   time.Duration
	Log           logging.LeveledLogger
	SettingEngine *webrtc.SettingEngine
}

func New(ctx context.Context, opts sfuOptions) *SFU {
	localCtx, cancel := context.WithCancel(ctx)

	sfu := &SFU{
		clients:                    &SFUClients{clients: make(map[string]*Client), mu: sync.Mutex{}},
		context:                    localCtx,
		cancel:                     cancel,
		codecs:                     opts.Codecs,
		iceServers:                 opts.IceServers,
		mu:                         sync.Mutex{},
		bitrateConfigs:             opts.Bitrates,
		pliInterval:                opts.PLIInterval,
		relayTracks:                make(map[string]ITrack),
		onTracksAvailableCallbacks: make([]func(tracks ITrack), 0),
		onClientRemovedCallbacks:   make([]func(*Client), 0),
		onClientAddedCallbacks:     make([]func(*Client), 0),
		log:                        opts.Log,
		defaultSettingEngine:       opts.SettingEngine,
	}

	return sfu
}

func (s *SFU) addClient(client *Client) {
	if err := s.clients.Add(client); err != nil {
		s.log.Errorf("sfu: failed to add client ", err)
		return
	}

	s.onClientAdded(client)
}

func (s *SFU) onClientAdded(client *Client) {
	for _, callback := range s.onClientAddedCallbacks {
		callback(client)
	}
}

func (s *SFU) onClientRemoved(client *Client) {
	for _, callback := range s.onClientRemovedCallbacks {
		callback(client)
	}
}
