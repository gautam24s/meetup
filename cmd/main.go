package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gautam24s/meetup"
	"github.com/gautam24s/meetup/pkg/interceptors/voiceactivedetector"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/nack"
	"github.com/pion/logging"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type Peer struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	conn       *websocket.Conn
	pc         *webrtc.PeerConnection
	track      *webrtc.TrackLocalStaticRTP
	mu         sync.RWMutex
	id         string
	trackBuf   chan []byte
}

type Signal struct {
	Type      string `json:"type"`
	SDP       string `json:"sdp"`
	Candidate string `json:"candidate"`
}

var upgrader = websocket.Upgrader{}
var peers = make(map[string]*Peer)
var mu sync.RWMutex
var vads = make(map[uint32]*voiceactivedetector.VoiceDetector)

func NewPeer(conn *websocket.Conn, id string) *Peer {
	ctx, cancelFunc := context.WithCancel(context.Background())
	return &Peer{
		ctx:        ctx,
		cancelFunc: cancelFunc,
		conn:       conn,
		id:         id,
		trackBuf:   make(chan []byte),
	}
}

func (p *Peer) Close() {
	if p.pc != nil {
		p.pc.Close()
	}
	p.conn.Close()
}

func (p *Peer) SendSignal(signal Signal) error {
	data, err := json.Marshal(signal)
	if err != nil {
		return fmt.Errorf("failed to marshal signal: %w", err)
	}
	return p.conn.WriteMessage(websocket.TextMessage, data)
}

func (p *Peer) HandleConnection() error {
	roomOpts := meetup.DefaultRoomOptions()
	m := &webrtc.MediaEngine{}
	if err := meetup.RegisterCodecs(m, *roomOpts.Codecs); err != nil {
		panic(err)
	}
	voiceactivedetector.RegisterAudioLevelHeaderExtension(m)
	i := &interceptor.Registry{}
	logg := logging.NewDefaultLoggerFactory().NewLogger("meetup")

	vadInterceptorFactory := voiceactivedetector.NewInterceptor(p.ctx, logg)
	vadInterceptorFactory.OnNew(
		func(i *voiceactivedetector.Interceptor) {
			i.OnNewVAD(func(vad *voiceactivedetector.VoiceDetector) {
				vads[vad.SSRC()] = vad
			})
		},
	)
	i.Add(vadInterceptorFactory)

	if err := registerInterceptors(m, i); err != nil {
		panic(err)
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:bn-turn2.xirsys.com"},
			},
			{
				Username:   "2LX9iv6rtQnmHf0ADXPLaXXU_pMp8gZH9NYXgH6dIcMP7b81qEwVl7C6VyevuNZEAAAAAGeA0IN1bmZpeGJ1Zw==",
				Credential: "1bfd73d6-cf27-11ef-a8c3-0242ac140004",
				URLs: []string{
					"turn:bn-turn2.xirsys.com:80?transport=udp",
					"turn:bn-turn2.xirsys.com:3478?transport=udp",
					"turn:bn-turn2.xirsys.com:80?transport=tcp",
					"turn:bn-turn2.xirsys.com:3478?transport=tcp",
					"turns:bn-turn2.xirsys.com:443?transport=tcp",
					"turns:bn-turn2.xirsys.com:5349?transport=tcp",
				},
			},
		},
	}
	pc, err := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i)).NewPeerConnection(config)
	if err != nil {
		return err
	}
	p.pc = pc

	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion",
	)
	if err != nil {
		return err
	}
	p.track = track

	if _, err := pc.AddTrack(track); err != nil {
		return err
	}

	pc.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i != nil {
			candidateJSON, _ := json.Marshal(i.ToJSON())
			p.SendSignal(Signal{Type: "candidate", Candidate: string(candidateJSON)})
		}
	})

	pc.OnTrack(func(tr *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				if err := pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(tr.SSRC())}}); err != nil {
					log.Println("Failed to send PLI: ", err)
					return
				}
			}
		}()
		// handle incoming RTCP packets (including NACK, PLI, etc.)
		go func() {
			rtcpBuf := make([]byte, 1500)
			for {
				n, _, err := r.Read(rtcpBuf)
				if err != nil {
					log.Println("Failed to read rtcp packet: ", err)
					return
				}

				packets, err := rtcp.Unmarshal(rtcpBuf[:n])
				if err != nil {
					log.Println("Failed to unmarshal rtcp packet: ", err)
					continue
				}

				for _, packet := range packets {
					switch pkt := packet.(type) {
					case *rtcp.PictureLossIndication:
						log.Println("Received PLI from receiver, forwarding to sender")
						log.Printf("packet: %+v", pkt)
						// send PLI to sender
					case *rtcp.TransportLayerNack:
						log.Println("Received NACK from receiver, forwarding to sender")
						log.Printf("packet: %+v", pkt)
						// send NACK to sender
					}
				}
			}
		}()

		// Read and broadcast RTP packets
		go func() {
			buf := make([]byte, 1500)
			for {
				n, _, err := tr.Read(buf)
				if err != nil {
					if err == io.EOF {
						break
					}
					log.Println("Track read error:", err)
					return
				}
				go broadcastRTP(p, buf[:n])
			}
		}()

		go func() {
			for buf := range p.trackBuf {
				packet := &rtp.Packet{}
				if err := packet.Unmarshal(buf); err != nil {
					log.Printf("Failed to unmarshal RTP packet: %v", err)
					continue
				}
				if writeErr := p.track.WriteRTP(packet); writeErr != nil {
					log.Printf("Failed to write the local track: %v", err)
				}
			}
		}()
	})

	pc.OnConnectionStateChange(
		func(pcs webrtc.PeerConnectionState) {
			if pcs == webrtc.PeerConnectionStateConnected {
				log.Println("peer connection established")
			} else if pcs == webrtc.PeerConnectionStateClosed {
				log.Println("peer connection closed")
			} else {
				log.Printf("something happened with peer connection: %v", pcs)
			}
		},
	)

	return nil
}

func broadcastRTP(p *Peer, buf []byte) {
	for id, peer := range peers {
		if id != p.id {
			if peer.track == nil {
				continue
			}
			peer.trackBuf <- buf
		}
	}
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/ws", handleConnections)
	log.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer ws.Close()

	peer := NewPeer(ws, uuid.NewString())
	mu.Lock()
	peers[peer.id] = peer
	mu.Unlock()
	log.Printf("peer id: %s", peer.id)
	go peer.HandleConnection()

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			mu.Lock()
			delete(peers, peer.id)
			mu.Unlock()
			peer.Close()
			break
		}
		var signal Signal
		if err := json.Unmarshal(message, &signal); err != nil {
			log.Println("Signal unmarshal error:", err)
			continue
		}
		handleIncomingData(peer, signal)
	}
}

func handleIncomingData(peer *Peer, signal Signal) {
	switch signal.Type {
	case "offer":
		offer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  signal.SDP,
		}
		if err := peer.pc.SetRemoteDescription(offer); err != nil {
			log.Println("Remote description error:", err)
			return
		}
		answer, err := peer.pc.CreateAnswer(nil)
		if err != nil {
			log.Println("Answer creation error:", err)
			return
		}
		peer.pc.SetLocalDescription(answer)
		peer.SendSignal(Signal{Type: "answer", SDP: answer.SDP})

	case "answer":
		answer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  signal.SDP,
		}
		peer.pc.SetRemoteDescription(answer)

	case "candidate":
		var candidate webrtc.ICECandidateInit
		if err := json.Unmarshal([]byte(signal.Candidate), &candidate); err != nil {
			log.Println("ICE candidate unmarshal error:", err)
			return
		}
		peer.pc.AddICECandidate(candidate)
	}
}

func registerInterceptors(m *webrtc.MediaEngine, interceptorRegistry *interceptor.Registry) error {
	generator, err := nack.NewGeneratorInterceptor()
	if err != nil {
		return err
	}

	responder, err := nack.NewResponderInterceptor()
	if err != nil {
		return err
	}

	m.RegisterFeedback(webrtc.RTCPFeedback{Type: "nack"}, webrtc.RTPCodecTypeVideo)
	m.RegisterFeedback(webrtc.RTCPFeedback{Type: "nack", Parameter: "pli"}, webrtc.RTPCodecTypeVideo)
	interceptorRegistry.Add(generator)
	interceptorRegistry.Add(responder)

	if err := webrtc.ConfigureRTCPReports(interceptorRegistry); err != nil {
		return err
	}

	return webrtc.ConfigureTWCCSender(m, interceptorRegistry)
}
