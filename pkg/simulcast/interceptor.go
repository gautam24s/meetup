package simulcast

import (
	"sync"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v4"
)

type InterceptorFactory struct {
	onNew func(i *Interceptor)
}

func NewInterceptor() *InterceptorFactory {
	return &InterceptorFactory{}
}

func (g *InterceptorFactory) NewInterceptor(_ string) (interceptor.Interceptor, error) {
	i := new()

	if g.onNew != nil {
		g.onNew(i)
	}
	return i, nil
}

func (g *InterceptorFactory) OnNew(callback func(i *Interceptor)) {
	g.onNew = callback
}

type Interceptor struct {
	mu         sync.Mutex
	parameters webrtc.RTPSendParameters
	streams    map[string]*interceptor.StreamInfo
	close      bool
}

func new() *Interceptor {
	return &Interceptor{
		mu:      sync.Mutex{},
		streams: make(map[string]*interceptor.StreamInfo),
	}
}

func (s *Interceptor) BindLocalStream(info *interceptor.StreamInfo, writer interceptor.RTPWriter) interceptor.RTPWriter {
	return interceptor.RTPWriterFunc(func(header *rtp.Header, payload []byte, attributes interceptor.Attributes) (int, error) {
		midID, ridID, trackRID := s.getHeaderExtension(info.SSRC)
		if midID != 0 && ridID != 0 && trackRID != "" {
			if err := header.SetExtension(ridID, []byte(trackRID)); err != nil {
				panic(err)
			}

			if err := header.SetExtension(midID, []byte("0")); err != nil {
				panic(err)
			}
		}
		return writer.Write(header, payload, attributes)
	})
}

func (s *Interceptor) getHeaderExtension(ssrc uint32) (uint8, uint8, string) {
	var midID, ridID uint8
	var trackRID string

	for _, extension := range s.parameters.HeaderExtensions {
		switch extension.URI {
		case sdp.SDESMidURI:
			midID = uint8(extension.ID)
		case sdp.SDESRTPStreamIDURI:
			ridID = uint8(extension.ID)
		}
	}

	for _, encoding := range s.parameters.Encodings {
		if uint32(encoding.SSRC) == ssrc {
			trackRID = encoding.RID
		}
	}
	return midID, ridID, trackRID
}

func (s *Interceptor) BindRTCPReader(reader interceptor.RTCPReader) interceptor.RTCPReader {
	return reader
}

func (s *Interceptor) BindRTCPWriter(writer interceptor.RTCPWriter) interceptor.RTCPWriter {
	if s.close {
		return writer
	}
	return writer
}

func (s *Interceptor) BindRemoteStream(info *interceptor.StreamInfo, reader interceptor.RTPReader) interceptor.RTPReader {
	return reader
}

func (s *Interceptor) Close() error {
	return nil
}

func (s *Interceptor) UnbindRemoteStream(info *interceptor.StreamInfo) {}

func (s *Interceptor) UnbindLocalStream(info *interceptor.StreamInfo) {}

func (s *Interceptor) SetSenderParameters(parameters webrtc.RTPSendParameters) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.parameters = parameters
}
