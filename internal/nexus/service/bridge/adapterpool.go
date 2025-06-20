package bridge

import (
	"context"

	"go.uber.org/zap"
)

type AdapterFactory interface {
	New() (Adapter, error)
}

type AdapterPool struct {
	adapters chan Adapter
	factory  AdapterFactory
}

func NewAdapterPool(size int, factory AdapterFactory) *AdapterPool {
	pool := &AdapterPool{
		adapters: make(chan Adapter, size),
		factory:  factory,
	}
	for i := 0; i < size; i++ {
		adapter, err := factory.New()
		if err == nil {
			pool.adapters <- adapter
		}
	}
	return pool
}

func (p *AdapterPool) Get(_ context.Context) (Adapter, error) {
	select {
	case adapter := <-p.adapters:
		return adapter, nil
	default:
		return p.factory.New()
	}
}

func (p *AdapterPool) Put(adapter Adapter) {
	select {
	case p.adapters <- adapter:
		// returned to pool
	default:
		if err := adapter.Close(); err != nil {
			zap.L().Warn("Failed to close adapter in pool", zap.Error(err))
		}
	}
}
