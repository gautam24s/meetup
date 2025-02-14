package meetup

import (
	"context"
	"sync"

	"github.com/pion/logging"
	"github.com/pion/webrtc/v4"
)

type Manager struct {
	rooms      map[string]*Room
	context    context.Context
	cancel     context.CancelFunc
	iceServers []webrtc.ICEServer
	name       string
	mutext     sync.RWMutex
	options    Options
	log        logging.LeveledLogger
}

func NewManager(ctx context.Context, name string, opts Options) *Manager {
	localCtx, cancel := context.WithCancel(ctx)

	logger := logging.NewDefaultLoggerFactory().NewLogger("sfu")

	m := &Manager{
		rooms:      make(map[string]*Room),
		context:    localCtx,
		cancel:     cancel,
		iceServers: opts.IceServers,
		name:       name,
		mutext:     sync.RWMutex{},
		options:    opts,
		log:        logger,
	}

	return m
}

func (m *Manager) CreateRoomID() string {
	return GenerateID(16)
}

// func (m *Manager) NewRoom(id, name, roomType string, opts RoomOptions) (*Room, error) {
// }
