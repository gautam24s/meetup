package rtppool

import (
	"errors"
	"io"
	"sync"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

var (
	errPacketReleased          = errors.New("packet has been released")
	errFailedToCastPacketPool  = errors.New("failed to cast packet pool")
	errFailedToCastHeaderPool  = errors.New("failed to cast header pool")
	errFailedToCastPayloadPool = errors.New("failed to cast payload pool")
)

const maxPayloadLen = 1460

type PacketManager struct {
	PacketPool  *sync.Pool
	HeaderPool  *sync.Pool
	PayloadPool *sync.Pool
	AttrPool    *sync.Pool
}


func NewPacketManager() *PacketManager {
	return &PacketManager{
		PacketPool: &sync.Pool{
			New: func() any {
				return &RetainablePacket{}
			},
		},
		HeaderPool: &sync.Pool{
			New: func() any {
				return &rtp.Header{}
			},
		},
		PayloadPool: &sync.Pool{
			New: func() any {
				buf := make([]byte, maxPayloadLen)
				return &buf
			},
		},
		AttrPool: &sync.Pool{
			New: func() any {
				return interceptor.Attributes{}
			},
		},
	}
}

func (m *PacketManager) NewPacket(header *rtp.Header, payload []byte, attr interceptor.Attributes) (*RetainablePacket, error) {
	if len(payload) > maxPayloadLen {
		return nil, io.ErrShortBuffer
	}

	rp, ok := m.PacketPool.Get().(*RetainablePacket)
	if !ok {
		return nil, errFailedToCastPacketPool
	}
	rp.onRelease = m.releasePacket
	rp.count = 1

	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.header, ok = m.HeaderPool.Get().(*rtp.Header)
	if !ok {
		return nil, errFailedToCastHeaderPool
	}

	*rp.header = header.Clone()

	if payload != nil {
		rp.buffer, ok = m.PayloadPool.Get().(*[]byte)
		if !ok {
			return nil, errFailedToCastPayloadPool
		}
		size := copy(*rp.buffer, payload)
		rp.payload = (*rp.buffer)[:size]
	}

	if attr != nil {
		rp.attr, ok = m.AttrPool.Get().(interceptor.Attributes)
		if !ok {
			return nil, errFailedToCastPayloadPool
		}

		for k, v := range attr {
			rp.attr[k] = v
		}
	}
	return rp, nil
}

func (m *PacketManager) releasePacket(header *rtp.Header, payload *[]byte, rp *RetainablePacket) {
	m.HeaderPool.Put(header)
	if payload != nil {
		copy(*payload, blankPayload)
		m.PayloadPool.Put(payload)
	}
	if rp.attr != nil {
		for k := range rp.attr {
			delete(rp.attr, k)
		}
		m.AttrPool.Put(rp.attr)
	}
	m.PacketPool.Put(rp)
}

type RetainablePacket struct {
	onRelease func(*rtp.Header, *[]byte, *RetainablePacket)
	mu        sync.RWMutex
	count     int

	header  *rtp.Header
	buffer  *[]byte
	payload []byte
	attr    interceptor.Attributes
}

func (p *RetainablePacket) Header() *rtp.Header {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.header
}

func (p *RetainablePacket) Payload() []byte {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.payload
}

func (p *RetainablePacket) Attributes() interceptor.Attributes {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.attr
}

func (p *RetainablePacket) Retain() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.count == 0 {
		return errPacketReleased
	}
	p.count++
	return nil
}

func (p *RetainablePacket) Release() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	p.count--

	if p.count == 0 {
		p.onRelease(p.header, p.buffer, p)
	}

}
