package quantest

import (
	"fmt"
)

type Quantile struct {
	q       float64
	errFrac float64
	u       float64
	v       float64
}

func NewQuantile(quantile, errFrac float64) Quantile {
	return Quantile{
		q:       quantile,
		errFrac: errFrac,
		u:       2 * errFrac / (1 - quantile),
		v:       2 * errFrac / quantile,
	}
}

func (q Quantile) String() string {
	return fmt.Sprintf("Q{q=%.3f, eps=%.3f}", q.q, q.errFrac)
}

type item struct {
	// value is sampled from the data stream
	value int

	// g is the difference between the lowest possible rank of the current
	// and previous item's values
	g int

	// delta is the difference between the greatest possible rank of
	// the current value and the lowest possible rank of the value
	delta int
}

func NewItem(value, g, delta int) item {
	return item{value: value, g: g, delta: delta}
}

func (i item) String() string {
	return fmt.Sprintf("I{%d, %d, %d}", i.value, i.g, i.delta)
}

// sample is an array of items. It is a convenient wrapper that allows
// easy insertion and removal of items, also contains an iterator
type sample struct {
	items []item
	i     int
}

func (s *sample) insert(i int, it item) {
	s.items = append(s.items, item{})
	copy(s.items[i+1:], s.items[i:])
	s.items[i] = it
	if i < s.i {
		// inserted before cursor, so need to increment
		s.i++
	}
}

func (s *sample) remove(i int) {
	copy(s.items[i-1:], s.items[i:])
	s.items = s.items[:len(s.items)-1]
	if i < s.i {
		// removed before cursor, need to decrement
		s.i--
	}
}

func (s *sample) get(i int) item {
	return s.items[i]
}

func (s *sample) next() item {
	ret := s.items[s.i]
	s.i++
	return ret
}

func (s *sample) prev() item {
	s.i--
	return s.items[s.i]
}

func (s *sample) nextIndex() int {
	return s.i
}

func (s *sample) prevIndex() int {
	return s.i - 1
}

func (s *sample) len() int {
	return len(s.items)
}

func (s *sample) add(it item) {
	s.insert(s.i, it)
}

func (s *sample) resetIterator() {
	s.i = 0
}

func (s *sample) append(it item) {
	s.items = append(s.items, it)
}

func (s *sample) hasNext() bool {
	return s.i < len(s.items)
}

func (s *sample) merge() {
	// current is kept, previous is removed, the g values are combined
	s.items[s.i-1].g += s.items[s.i-2].g
	s.remove(s.i - 1)
}
