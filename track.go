package meetup

import (
	"context"
	"errors"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

const (
	TrackTypeMedia  = "media"
	TrackTypeScreen = "screen"
)

var (
	ErrTrackExists = errors.New("client: error track already exists")
	ErrTrackIsNotExists = errors.New("client: error track is not exists")
)

type TrackType string

func (t TrackType) String() string {
	return string(t)
}

type ITrack interface {
	ID() string
	StreamID() string
	ClientID() string
	IsSimulcast() string
	IsScaleable() string
	IsProcessed() string
	SetSourceType(TrackType)
	SourceType() TrackType
	SetAsProcessed()
	OnRead(func(interceptor.Attributes, *rtp.Packet, QualityLevel))
	IsScreen() bool
	IsRelay() bool
	Kind() webrtc.RTPCodecType
	MimeType() string
	TotalTracks() int
	Context() context.Context
	Relay(func(webrtc.SSRC, interceptor.Attributes, *rtp.Packet))
	PayloadType() webrtc.PayloadType
	OnEnded(func())
}
