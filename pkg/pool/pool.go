package pool

import (
	"reflect"
	"sync"
)

type Resettable interface {
	Reset()
}

type Pool[T Resettable] struct {
	pool sync.Pool
}

func New[T Resettable](factory ...func() T) *Pool[T] {
	p := &Pool[T]{}

	if len(factory) > 0 && factory[0] != nil {
		p.pool.New = func() any {
			return factory[0]()
		}
	}

	return p
}

func (p *Pool[T]) Get() T {
	item := p.pool.Get()
	if item == nil {
		var zero T
		return zero
	}

	value, ok := item.(T)
	if !ok {
		var zero T
		return zero
	}

	return value
}

func (p *Pool[T]) Put(value T) {
	if isNil(value) {
		return
	}

	value.Reset()
	p.pool.Put(value)
}

func isNil[T Resettable](value T) bool {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
