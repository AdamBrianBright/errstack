package errstack

import "slices"

type Stack[T comparable] []T

func (s *Stack[T]) Push(v ...T) *Stack[T] {
	*s = append(*s, v...)
	return s
}

func (s *Stack[T]) Pop() *T {
	if len(*s) == 0 {
		return nil
	}

	v := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]

	return &v
}

func (s *Stack[T]) AddUnique(v ...T) *Stack[T] {
	for _, e := range v {
		if !slices.Contains(*s, e) {
			s.Push(e)
		}
	}

	return s
}
