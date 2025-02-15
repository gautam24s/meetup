package meetup

import (
	"context"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type iClientTrack interface {
	push(rtp *rtp.Packet, quality QualityLevel)
	ID() string
	StreamID() string
	Context() context.Context
	Kind() webrtc.RTPCodecType
	MimeType() string
	Localtrack() *webrtc.TrackLocalStaticRTP
	IsScreen() bool
	IsSimulcast() bool
	IsScaleable() bool
	SetSourceType(TrackType)
	Client() *Client
	RequestPLI()
	SetMaxQuality(quality QualityLevel)
	MaxQuality() QualityLevel
	ReceiveBitrate() uint32
	SendBitrate() uint32
	Quality() QualityLevel
	OnEnded(func())
}

type clientTrack struct {
	id         string
	streamid   string
	context    context.Context
	mu         sync.RWMutex
	client     *Client
	kind       webrtc.RTPCodecType
	mineType   string
	localTrack *webrtc.TrackLocalStaticRTP
	remoteTrack
}
