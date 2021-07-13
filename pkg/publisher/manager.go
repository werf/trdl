package publisher

import "sync"

type Publisher struct {
	mu sync.Mutex
}

func NewPublisher() Interface {
	return &Publisher{}
}
