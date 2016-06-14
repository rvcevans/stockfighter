package quantest

import (
	"log"
	"math"
)

type GK struct {
	epsilon     float64
	count       int
	compactSize int
	sample      []item
}

func (g *GK) Insert(val int) {
	idx := 0

	for _, item := range g.sample {
		if item.value > val {
			break
		}
		idx++
	}

	delta := 0
	if !(idx == 0 || idx == len(g.sample)) {
		delta = int(math.Floor(2 * g.epsilon * float64(g.count)))
	}

	newItem := item{value: val, g: 1, delta: delta}
	g.sample = append(g.sample, item{})
	copy(g.sample[idx+1:], g.sample[idx:])
	g.sample[idx] = newItem

	if len(g.sample) > g.compactSize {
		g.compress()
	}
	g.count++
}

func (g *GK) compress() {
	removed := 0
	for i := 0; i < len(g.sample)-1; i++ {
		item := g.sample[i]
		item1 := g.sample[i+1]

		if item.g+item1.g+item1.delta <= int(math.Floor(2*g.epsilon*float64(g.count))) {
			item1.g += item.g
			copy(g.sample[i:], g.sample[i+1:])
			g.sample = g.sample[:len(g.sample)-1]
		}

		removed++
	}

	log.Printf("removed %d", removed)
}

func (g *GK) Query(quantile float64) int {
	var rankMin int
	desired := int(quantile * float64(g.count))
	for i := 1; i < len(g.sample); i++ {
		prev := g.sample[i-1]
		cur := g.sample[i]

		rankMin += prev.g

		if rankMin+cur.g+cur.delta > int(desired+int(2*g.epsilon*float64(g.count))) {
			return prev.value
		}
	}

	return g.sample[len(g.sample)-1].value
}
