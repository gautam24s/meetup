package meetup

import "sync"

type ClientStats struct {
	mu         sync.Mutex
	senderMu   sync.Mutex
	receiverMu sync.Mutex
	Client     *Client
}
