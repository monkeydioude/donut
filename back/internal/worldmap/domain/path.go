package domain

import (
	"container/heap"
	"math"

	core "donut/internal/core/domain"
)

// stepDirections are the 8 neighbors, in the order they are tried: straight ones first.
var stepDirections = [8]core.Cell{
	{Col: 1}, {Col: -1}, {Row: 1}, {Row: -1},
	{Col: 1, Row: 1}, {Col: 1, Row: -1}, {Col: -1, Row: 1}, {Col: -1, Row: -1},
}

// Path is the shortest walk from `from` to `to` (A*), the start excluded and the target included.
// A straight step costs 1, a diagonal one √2, climbing or not. Each step goes up or down MaxClimb levels at most,
// and a diagonal step needs both straight ways round to be steps: no cutting the corner of a rock, a hole, the river
// or a cliff. On a wrapping map, paths go across the seams. Nil when `to` is `from`, isn't walkable, can't be
// reached, or needs exploring more cells than the search limit.
func (t Terrain) Path(from, to core.Cell) []core.Cell {
	if from == to || !t.sameArea(from, to) {
		return nil
	}
	search := newSearch(t.grid, from, to)
	for explored := 0; search.open.Len() > 0 && explored < t.searchLimit; {
		current := heap.Pop(&search.open).(*searchNode)
		if current.cell == to {
			return search.pathTo(to)
		}
		if search.closed[current.cell] {
			continue
		}
		search.closed[current.cell] = true
		explored++
		for _, next := range t.steps(current.cell) {
			search.reach(current, next)
		}
	}
	return nil
}

// steps are the cells one step away from c that a walker may step on.
func (t Terrain) steps(c core.Cell) []core.Cell {
	steps := make([]core.Cell, 0, len(stepDirections))
	for _, d := range stepDirections {
		if next, ok := t.step(c, d); ok {
			steps = append(steps, next)
		}
	}
	return steps
}

// step is the cell one step `d` from c, when a walker may step on it. A diagonal step also needs both straight ways
// round to be steps.
func (t Terrain) step(c, d core.Cell) (core.Cell, bool) {
	next, ok := t.neighbor(c, d)
	if !ok || !t.canStep(c, next) {
		return core.Cell{}, false
	}
	if d.Col == 0 || d.Row == 0 {
		return next, true
	}
	return next, t.goesRound(c, next, core.Cell{Col: d.Col}) && t.goesRound(c, next, core.Cell{Row: d.Row})
}

// canStep: b, a neighbor of a, is walkable and at most MaxClimb levels up or down from a.
func (t Terrain) canStep(a, b core.Cell) bool {
	climb := int(t.HeightAt(b)) - int(t.HeightAt(a))
	return t.IsWalkable(b) && climb <= MaxClimb && climb >= -MaxClimb
}

// goesRound: from c, the straight step `side`, then on to `to`, are both steps.
func (t Terrain) goesRound(c, to, side core.Cell) bool {
	beside, ok := t.neighbor(c, side)
	return ok && t.canStep(c, beside) && t.canStep(beside, to)
}

// StepCost is 1 for a straight step, √2 for a diagonal one. Across a seam too: a diagonal step changes both the
// column and the row.
func StepCost(from, to core.Cell) float64 {
	if from.Col != to.Col && from.Row != to.Row {
		return math.Sqrt2
	}
	return 1
}

// octile is the cost of the shortest path between a and b on an empty map, across the seams on a wrapping one:
// the A* estimate.
func octile(grid core.Grid, a, b core.Cell) float64 {
	d := grid.Offset(a, b)
	dc, dr := math.Abs(float64(d.Col)), math.Abs(float64(d.Row))
	return dc + dr + (math.Sqrt2-2)*math.Min(dc, dr)
}

type search struct {
	grid     core.Grid
	to       core.Cell
	open     openSet
	cost     map[core.Cell]float64
	cameFrom map[core.Cell]core.Cell
	closed   map[core.Cell]bool
	found    int // how many nodes were pushed: ties go to the one found first
}

func newSearch(grid core.Grid, from, to core.Cell) *search {
	s := &search{grid: grid, to: to, cost: map[core.Cell]float64{from: 0}, cameFrom: map[core.Cell]core.Cell{}, closed: map[core.Cell]bool{}}
	s.push(from, 0)
	return s
}

// reach records `next` when going through `current` is cheaper than any way found so far.
func (s *search) reach(current *searchNode, next core.Cell) {
	cost := s.cost[current.cell] + StepCost(current.cell, next)
	if known, ok := s.cost[next]; ok && cost >= known-epsilon {
		return
	}
	s.cost[next] = cost
	s.cameFrom[next] = current.cell
	s.push(next, cost)
}

func (s *search) push(cell core.Cell, cost float64) {
	estimate := octile(s.grid, cell, s.to)
	heap.Push(&s.open, &searchNode{cell: cell, total: cost + estimate, estimate: estimate, order: s.found})
	s.found++
}

func (s *search) pathTo(to core.Cell) []core.Cell {
	path := []core.Cell{}
	for cell := to; ; {
		previous, ok := s.cameFrom[cell]
		if !ok {
			break
		}
		path = append(path, cell)
		cell = previous
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// epsilon absorbs float rounding when comparing path costs made of 1s and √2s.
const epsilon = 1e-9

type searchNode struct {
	cell     core.Cell
	total    float64 // cost so far + estimate
	estimate float64
	order    int
}

// openSet is a min-heap: lowest total first, then lowest estimate, then found first.
type openSet []*searchNode

func (o openSet) Len() int { return len(o) }

func (o openSet) Less(i, j int) bool {
	a, b := o[i], o[j]
	if math.Abs(a.total-b.total) > epsilon {
		return a.total < b.total
	}
	if math.Abs(a.estimate-b.estimate) > epsilon {
		return a.estimate < b.estimate
	}
	return a.order < b.order
}

func (o openSet) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

func (o *openSet) Push(x any) { *o = append(*o, x.(*searchNode)) }

func (o *openSet) Pop() any {
	old := *o
	last := old[len(old)-1]
	*o = old[:len(old)-1]
	return last
}
