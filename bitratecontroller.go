package meetup

import (
	"context"
	"sync"
	"time"

	"github.com/pion/logging"
)

type bitrateClaim struct {
	mu        sync.RWMutex
	track     iClientTrack
	quality   QualityLevel
	simulcase bool
}

type bitrateController struct {
	client               *Client
	claims               sync.Map
	enabledQualityLevels []QualityLevel
	log                  logging.LeveledLogger
}

func newbitrateController(client *Client, qualityLevels []QualityLevel) *bitrateController {
	bc := &bitrateController{
		client:               client,
		claims:               sync.Map{},
		enabledQualityLevels: qualityLevels,
		log:                  logging.NewDefaultLoggerFactory().NewLogger("bitratecontroller"),
	}

	go bc.loopMonitor()

	return bc
}

func (bc *bitrateController) loopMonitor() {
	ctx, cancel := context.WithCancel(bc.client.Context())
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var needAdjustment bool

			totalSendBitrates := bc.totalSentBitrates()
			bw := bc.client.GetEstimatedBandwith()
		}
	}
}

func (bc *bitrateController) totalSentBitrates() uint32 {
	total := uint32(0)

	for _, claim := range bc.Claims() {
		total += claim.track.SendBitrate()
	}

	return total
}

func (bc *bitrateController) Claims() map[string]*bitrateClaim {
	claims := make(map[string]*bitrateClaim, 0)
	bc.claims.Range(func(key, value any) bool {
		claims[key.(string)] = value.(*bitrateClaim)
		return true
	})

	return claims
}
