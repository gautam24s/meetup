package meetup

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
