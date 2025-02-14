package meetup

import (
	"context"
	"sync"
	"time"

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
	iceServers  []webrtc.ICEServer
	mu          sync.Mutex
	onStop      func()
	pliInterval time.Duration
	onTracksAvailableCallbacks []func()
}
