package quantest

import (
	"sort"
)

/*
 * Notes from paper:
 *  - rank refers to the position of an item in the sorted order of items that
 *    has been observed
 *  - let r_i be the sum of the previous item's g values, the true rank of a value
 *    is bounded by r_i + g, and r_i + g + delta. These values are maintained to
 *    keep the error at an appropriate level.
 */

const bufSize = 1024

type CKMS struct {
	count       int
	compressIdx int
	nodeList
	buf       [bufSize]int
	bufCount  int
	quantiles []Quantile
}

func NewCKMS(quantiles []Quantile) *CKMS {
	return &CKMS{quantiles: quantiles}
}

func (c *CKMS) allowableError(rank int) float64 {
	// NOTE: according to CKMS, this should be count, not size, but this leads
	// to error larger than the error bounds. Leaving it like this is
	// essentially a HACK, and blows up memory, but does "work".
	//size := c.count;

	size := c.len()
	minError := float64(size + 1)
	for _, q := range c.quantiles {
		var epsilon float64
		if float64(rank) <= q.q*float64(size) {
			epsilon = q.u * float64(size-rank)
		} else {
			epsilon = q.v * float64(rank)
		}

		if epsilon < minError {
			minError = epsilon
		}
	}

	return minError
}

func (c *CKMS) Insert(v int) {
	c.buf[c.bufCount] = v
	c.bufCount++

	if c.bufCount == bufSize {
		c.insertBatch()
		c.compress()
	}
}

func (c *CKMS) insertBatch() {
	if c.bufCount == 0 {
		return
	}
	tmpArr := c.buf[0:c.bufCount]
	sort.Sort(sort.IntSlice(tmpArr))

	var start int
	if c.len() == 0 {
		item := NewItem(c.buf[0], 1, 0)
		c.append(item)
		start++
		c.count++
	}

	c.resetIterator()

	curr := c.next()
	for i := start; i < c.bufCount; i++ {
		v := c.buf[i]
		for c.nextIndex() < c.len() && curr.value < v {
			curr = c.next()
		}

		// If we found the bigger item, back up so we insert just before
		if curr.value > v {
			c.prev()
		}

		var delta int
		if !(c.prevIndex() == 0 || c.nextIndex() == c.len()) {
			delta = int(c.allowableError(c.nextIndex()))
		}

		newItem := NewItem(v, 1, delta)
		c.add(newItem)
		c.count++
		curr = newItem
	}

	c.bufCount = 0
}

func (c *CKMS) compress() {
	if c.len() < 2 {
		return
	}

	c.resetIterator()
	var prev item
	curr := c.next()
	removed := 0

	for c.hasNext() {
		prev = curr
		curr = c.next()

		if float64(prev.g+curr.g+curr.delta) <= c.allowableError(c.prevIndex()) {
			// The following is does not actually update the value, doh
			c.merge()
			removed++
		}
	}

}

func (c *CKMS) Query(quantile float64) int {
	// clear the buffer
	c.insertBatch()
	c.compress()

	if c.len() == 0 {
		panic("no samples present")
	}

	rankMin := 0
	desired := float64(c.count) * quantile

	c.resetIterator()

	curr := c.next()
	var prev item
	for c.hasNext() {
		prev = curr
		curr = c.next()

		rankMin += prev.g
		a := float64(rankMin+curr.g+curr.delta)
		d := desired+(c.allowableError(int(desired))/2)

		if a > d {
			return prev.value
		}
	}

	// Edge case of wanting max value
	return c.get(c.len() - 1).value
}
