package quantest

import (
	"bytes"
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

type node struct {
	item
	next *node
	prev *node
}

func (n *node) insertBefore(in *node) {
	in.next = n
	if n != nil {
		in.prev = n.prev
		in.next.prev = in

		if in.prev != nil {
			in.prev.next = in
		}
	}
}

func (n *node) insertAfter(in *node) {
	in.prev = n
	if n != nil {
		in.next = n.next
		in.prev.next = in
		if in.next != nil {
			in.next.prev = in

		}
	}
}

func (n *node) remove() *node {
	// assume non nil
	if n.prev != nil {
		n.prev.next = n.next
	}

	if n.next != nil {
		n.next.prev = n.prev
	}

	return n.prev
}

type nodeList struct {
	head      *node
	curr      *node
	currIndex int
	total     int
}

func (n nodeList) String() string {
	buf := bytes.NewBufferString("[")
	for curr := n.head; curr != nil; curr = curr.next {
		buf.WriteString(curr.item.String() + ", ")
	}
	buf.WriteString("]")
	return buf.String()
}

func (s *nodeList) insert(i int, it item) {
	curr := s.head
	for j := 0; j < i; j++ {
		curr = curr.next
	}

	newNode := &node{item: it}
	curr.insertBefore(newNode)

	if i == 0 {
		s.head = newNode
	}
	s.total++
}

func (s *nodeList) remove(i int) {
	curr := s.head
	for j := 0; j < i; j++ {
		curr = curr.next
	}
	curr.remove()

	if i == 0 {
		s.head = s.head.next
	}
	s.total--
}

func (s *nodeList) get(i int) item {
	curr := s.head
	for j := 0; j < i; j++ {
		curr = curr.next
	}

	return curr.item
}

func (s *nodeList) next() item {
	if s.curr == nil {
		s.curr = s.head
	} else {
		s.curr = s.curr.next
		s.currIndex++
	}

	return s.curr.item
}

func (s *nodeList) prev() item {
	if s.curr == s.head {
		s.curr = nil
		return item{}
	}
	s.curr = s.curr.prev
	s.currIndex--
	return s.curr.item
}

func (s *nodeList) nextIndex() int {
	if s.curr == nil {
		return 0
	}
	return s.currIndex + 1
}

func (s *nodeList) prevIndex() int {
	return s.currIndex - 1
}

func (s *nodeList) len() int {
	return s.total
}

func (s *nodeList) add(it item) {
	s.total++
	newNode := &node{item: it}
	if s.curr == nil {
		s.head.insertBefore(newNode)
		s.head = newNode
		return
	}
	s.curr.insertAfter(newNode)
}

func (s *nodeList) resetIterator() {
	s.currIndex = 0
	s.curr = nil
}

// Ths is pretty poorly performing
func (s *nodeList) append(it item) {
	newNode := &node{item: it}
	s.total++
	if s.head == nil {
		s.head = newNode
		return
	}

	var last *node
	curr := s.head
	for curr != nil {
		last = curr
		curr = curr.next
	}

	last.next = newNode
	newNode.prev = last
}

func (s *nodeList) hasNext() bool {
	if s.curr == nil {
		// has not started iterating
		return s.head != nil
	}

	return s.curr.next != nil
}

func (s *nodeList) merge() {
	s.curr.g += s.curr.prev.g
	s.prev()
	s.total--
	if s.curr == s.head {
		s.head = s.curr.next
		s.head.prev = nil
	} else {
		s.curr = s.curr.remove()
	}
	s.next()
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
