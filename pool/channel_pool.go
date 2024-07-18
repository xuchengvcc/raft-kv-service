package pool

import "sync"

type ChannelPool[T any] struct {
	pool chan chan T
	mu   sync.Mutex
}

func NewChannelPool[T any](size int) *ChannelPool[T] {
	pool := make(chan chan T, size)
	for i := 0; i < size; i++ {
		pool <- make(chan T, 1)
	}
	return &ChannelPool[T]{pool: pool}
}

func (cp *ChannelPool[T]) Get() chan T {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return <-cp.pool
}

func (cp *ChannelPool[T]) Put(ch chan T) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.pool <- ch
}

func (cp *ChannelPool[T]) Close() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	close(cp.pool)
	for ch := range cp.pool {
		close(ch)
	}
}
