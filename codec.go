package meetup

import (
	"fmt"
	"slices"

	"github.com/pion/webrtc/v4"
)

var (
	videoRTCPFeedback = []webrtc.RTCPFeedback{{Type: "goog-remb", Parameter: ""}, {Type: "ccm", Parameter: "fir"}, {Type: "nack", Parameter: ""}, {Type: "nack", Parameter: "pli"}}

	videoCodecs = []webrtc.RTPCodecParameters{
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        96,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeRTX, ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=96", RTCPFeedback: nil},
			PayloadType:        97,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        102,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeRTX, ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=102", RTCPFeedback: nil},
			PayloadType:        103,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        104,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeRTX, ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=104", RTCPFeedback: nil},
			PayloadType:        105,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        106,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeRTX, ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=106", RTCPFeedback: nil},
			PayloadType:        107,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42e01f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        108,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeRTX, ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=108", RTCPFeedback: nil},
			PayloadType:        109,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=4d001f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        127,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeRTX, ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=127", RTCPFeedback: nil},
			PayloadType:        125,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=4d001f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        39,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeRTX, ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=39", RTCPFeedback: nil},
			PayloadType:        40,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP9, ClockRate: 90000, Channels: 0, SDPFmtpLine: "profile-id=0", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        98,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeRTX, ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=98", RTCPFeedback: nil},
			PayloadType:        99,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP9, ClockRate: 90000, Channels: 0, SDPFmtpLine: "profile-id=2", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        100,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeRTX, ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=100", RTCPFeedback: nil},
			PayloadType:        101,
		},

		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=64001f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        112,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeRTX, ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=112", RTCPFeedback: nil},
			PayloadType:        113,
		},
	}

	audioCodecs = []webrtc.RTPCodecParameters{
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/red", ClockRate: 48000, Channels: 2, SDPFmtpLine: "111/111", RTCPFeedback: nil},
			PayloadType:        63,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2, SDPFmtpLine: "minptime=10;useinbandfec=1", RTCPFeedback: nil},
			PayloadType:        111,
		},
	}
)

func RegisterCodecs(m *webrtc.MediaEngine, codecs []string) error {
	errors := []error{}

	for _, codec := range audioCodecs {
		if slices.Contains(codecs, codec.MimeType) {
			if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeAudio); err != nil {
				errors = append(errors, err)
			}
		}
	}

	registeredVideoCodecs := make([]webrtc.RTPCodecParameters, 0)

	for _, codec := range videoCodecs {
		if slices.Contains(codecs, codec.MimeType) {
			if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeVideo); err != nil {
				errors = append(errors, err)
			}

			registeredVideoCodecs = append(registeredVideoCodecs, codec)
		}
	}

	for _, codec := range registeredVideoCodecs {
		for _, videoCodec := range videoCodecs {
			if videoCodec.RTPCodecCapability.MimeType == "video/rtx" && videoCodec.RTPCodecCapability.SDPFmtpLine == fmt.Sprintf("apt=%d", codec.PayloadType) {
				if err := m.RegisterCodec(videoCodec, webrtc.RTPCodecTypeVideo); err != nil {
					errors = append(errors, err)
				}
			}
		}
	}

	return FlattenErrors(errors)
}

func getRTPParameters(mimeType string) webrtc.RTPCodecParameters {
	for _, codec := range audioCodecs {
		if codec.RTPCodecCapability.MimeType == mimeType {
			return codec
		}
	}

	for _, codec := range videoCodecs {
		if codec.RTPCodecCapability.MimeType == mimeType {
			return codec
		}
	}

	return webrtc.RTPCodecParameters{}
}
