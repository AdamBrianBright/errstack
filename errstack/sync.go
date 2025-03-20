package errstack

import (
	"slices"
	"sync"
)

type Locked[T any] struct {
	l sync.Mutex
	v T
}

func NewLocked[T any](v T) *Locked[T] {
	return &Locked[T]{
		v: v,
	}
}

func (l *Locked[T]) Get() T {
	l.l.Lock()
	defer l.l.Unlock()
	return l.v
}

func (l *Locked[T]) Set(v T) {
	l.l.Lock()
	defer l.l.Unlock()
	l.v = v
}

type Map[K comparable, V any] struct {
	m map[K]V
	l sync.Mutex
}

func NewMap[K comparable, V any](capacity int) *Map[K, V] {
	return &Map[K, V]{
		m: make(map[K]V, capacity),
	}
}

func (m *Map[K, V]) Len() int {
	m.l.Lock()
	defer m.l.Unlock()
	return len(m.m)
}

func (m *Map[K, V]) Get(key K) (value V, ok bool) {
	m.l.Lock()
	defer m.l.Unlock()
	value, ok = m.m[key]
	return
}

func (m *Map[K, V]) Set(key K, value V) {
	m.l.Lock()
	defer m.l.Unlock()
	m.m[key] = value
}

func (m *Map[K, V]) Delete(key K) {
	m.l.Lock()
	defer m.l.Unlock()
	delete(m.m, key)
}

func (m *Map[K, V]) Clone() map[K]V {
	m.l.Lock()
	defer m.l.Unlock()
	clone := make(map[K]V, len(m.m))
	for key, value := range m.m {
		clone[key] = value
	}
	return clone
}

type List[T comparable] struct {
	list []T
	l    sync.Mutex
}

func NewList[T comparable](length, capacity int) *List[T] {
	return &List[T]{
		list: make([]T, length, capacity),
	}
}

func (l *List[T]) Len() int {
	l.l.Lock()
	defer l.l.Unlock()
	return len(l.list)
}

func (l *List[T]) Push(value T) {
	l.l.Lock()
	defer l.l.Unlock()
	l.list = append(l.list, value)
}

func (l *List[T]) Pop() (value T) {
	l.l.Lock()
	defer l.l.Unlock()
	if len(l.list) == 0 {
		return
	}
	value, l.list = l.list[len(l.list)-1], l.list[:len(l.list)-1]
	return
}

func (l *List[T]) Clone() []T {
	l.l.Lock()
	defer l.l.Unlock()
	var clone = make([]T, len(l.list))
	copy(clone, l.list)
	return clone
}

func (l *List[T]) AddUnique(value T) {
	l.l.Lock()
	defer l.l.Unlock()
	if !slices.Contains(l.list, value) {
		l.list = append(l.list, value)
	}
}
