package meetup

import (
	"errors"
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type IRemoteTrack interface {
	ID() string
	RID() string
	PayloadType() webrtc.PayloadType
	Kind() webrtc.RTPCodecType
	StreamID() string
	SSRC() webrtc.SSRC
	Msid() string
	Codec() webrtc.RTPCodecParameters
	Read(b []byte) (n int, attributes interceptor.Attributes, err error)
	ReadRTP() (*rtp.Packet, interceptor.Attributes, error)
	SetReadDeadline(deadline time.Time) error
}

type RelayTrack struct {
	mu sync.RWMutex

	id          string
	streamID    string
	payloadType webrtc.PayloadType
	kind        webrtc.RTPCodecType
	ssrc        webrtc.SSRC
	mimeType    string
	rid         string
	rtpChan     chan *rtp.Packet
}

func NewTrackRelay(id, streamid, rid string, kind webrtc.RTPCodecType, ssrc webrtc.SSRC, mimeType string, rtpChan chan *rtp.Packet) IRemoteTrack {
	return &RelayTrack{}
}

func (t *RelayTrack) ID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.id
}

func (t *RelayTrack) RID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.rid
}

func (t *RelayTrack) PayloadType() webrtc.PayloadType {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.payloadType
}

func (t *RelayTrack) Kind() webrtc.RTPCodecType {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.kind
}

func (t *RelayTrack) StreamID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.streamID
}

func (t *RelayTrack) SSRC() webrtc.SSRC {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.ssrc
}

func (t *RelayTrack) Msid() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.StreamID() + " " + t.ID()
}

func (t *RelayTrack) Codec() webrtc.RTPCodecParameters {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return getRTPParameters(t.mimeType)
}

func (t *RelayTrack) Read(b []byte) (n int, attributes interceptor.Attributes, err error) {
	return 0, nil, errors.New("relaytrack: not implemented, use ReadRTP instead")
}

func (t *RelayTrack) ReadRTP() (*rtp.Packet, interceptor.Attributes, error) {
	p := <-t.rtpChan
	return p, nil, nil
}

func (t *RelayTrack) SetReadDeadline(deadline time.Time) error {
	return errors.New("relaytrack: not implemented")
}

func (t *RelayTrack) IsRelay() bool {
	return true
}
