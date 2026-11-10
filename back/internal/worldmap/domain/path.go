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
// A straight step costs 1, a diagonal one √2, and a diagonal step needs both cells beside it walkable:
// no cutting a rock's corner. On a wrapping map, paths go across the seams. Nil when `to` is `from`, isn't
// walkable, can't be reached, or needs exploring more cells than the search limit.
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
		next, ok := t.neighbor(c, d)
		if !ok || !t.IsWalkable(next) {
			continue
		}
		diagonal := d.Col != 0 && d.Row != 0
		if diagonal && (!t.walkableStep(c, core.Cell{Col: d.Col}) || !t.walkableStep(c, core.Cell{Row: d.Row})) {
			continue
		}
		steps = append(steps, next)
	}
	return steps
}

// walkableStep: the cell one step `d` from c is walkable.
func (t Terrain) walkableStep(c, d core.Cell) bool {
	next, ok := t.neighbor(c, d)
	return ok && t.IsWalkable(next)
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
