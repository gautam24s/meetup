package rtppool

import (
	"sync"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
)

type RTPPool struct {
	pool          sync.Pool
	PacketManager *PacketManager
}

var blankPayload = make([]byte, maxPayloadLen)

func New() *RTPPool {
	return &RTPPool{
		pool: sync.Pool{
			New: func() any {
				return &rtp.Packet{}
			},
		},
		PacketManager: NewPacketManager(),
	}

}

func (r *RTPPool) GetPacket() *rtp.Packet {
	return r.pool.Get().(*rtp.Packet)
}

func (r *RTPPool) PutPacket(localPacket *rtp.Packet) {
	localPacket.Header = rtp.Header{}
	copy(localPacket.Payload, blankPayload)

	r.pool.Put(localPacket)
}

func (r *RTPPool) GetPayload() *[]byte {
	return r.PacketManager.PayloadPool.Get().(*[]byte)
}

func (r *RTPPool) PutPayload(localPayload *[]byte) {
	copy(*localPayload, blankPayload)
	r.PacketManager.PayloadPool.Put(localPayload)
}

func (r *RTPPool) NewPacket(header *rtp.Header, payload []byte, attr interceptor.Attributes) *RetainablePacket{
	pkt, err := r.PacketManager.NewPacket(header, payload, attr)
	if err != nil {
		return nil
	}
	return pkt
}