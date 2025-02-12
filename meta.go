package meetup

import "sync"

type Metadata struct {
	mu                 sync.RWMutex
	m                  map[string]any
	onChangedCallbacks map[string]func(key string, value any)
}
