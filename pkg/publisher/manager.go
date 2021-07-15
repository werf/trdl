package publisher

import "sync"

type Publisher struct {
	mu sync.Mutex
}

func NewPublisher() *Publisher {
	return &Publisher{}
}
