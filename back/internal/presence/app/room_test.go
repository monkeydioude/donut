package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/presence/app"
	"donut/internal/presence/domain"
	"donut/pkg/logs"
)

// fakeClient records what it is sent. A full one refuses everything, like a player reading too slowly.
type fakeClient struct {
	events []domain.Event
	full   bool
	closed bool
}

func (c *fakeClient) Send(event domain.Event) bool {
	if c.full {
		return false
	}
	c.events = append(c.events, event)
	return true
}

func (c *fakeClient) Close() { c.closed = true }

// take returns the events received since the last take.
func (c *fakeClient) take() []domain.Event {
	events := c.events
	c.events = nil
	return events
}

// fakeWorld is an open map with rocks that tests may add: 5 x 5 by default. Its paths go straight toward the target,
// diagonally first, across the seams of a wrapping map, and there is none when a rock is in the way, nor from a rock:
// like the real terrain, a path needs a walkable start.
type fakeWorld struct {
	grid  core.Grid
	rocks map[core.Cell]bool
	spawn core.Cell
	info  core.MapInfo
	domes []core.Dome
	edge  []core.Cell
	walls map[core.Cell]bool
}

func (w *fakeWorld) Grid() core.Grid { return w.grid }

func (w *fakeWorld) IsWalkable(c core.Cell) bool {
	if !w.grid.Contains(c) || w.rocks[c] || w.walls[c] {
		return false
	}
	_, onDome := w.DomeAt(c)
	return !onDome
}

func (w *fakeWorld) SpawnPoint() (core.Cell, error) { return w.spawn, nil }

func (w *fakeWorld) Path(from, to core.Cell) []core.Cell {
	if !w.IsWalkable(from) {
		return nil
	}
	var path []core.Cell
	for at := from; at != to; {
		d := w.grid.Offset(at, to)
		next, ok := w.grid.Normalize(core.Cell{Col: at.Col + sign(d.Col), Row: at.Row + sign(d.Row)})
		if !ok || !w.IsWalkable(next) {
			return nil
		}
		at = next
		path = append(path, at)
	}
	return path
}

// Info, the domes, the edge and the outside: an open map has no door, unless a test adds one (ADR 0012).
func (w *fakeWorld) Info() core.MapInfo { return w.info }

func (w *fakeWorld) DomeAt(c core.Cell) (core.Dome, bool) {
	for _, dome := range w.domes {
		if dome.Covers(w.grid, c) {
			return dome, true
		}
	}
	return core.Dome{}, false
}

func (w *fakeWorld) DomeLeadingTo(slug string) (core.Dome, bool) {
	for _, dome := range w.domes {
		if dome.LeadsTo == slug {
			return dome, true
		}
	}
	return core.Dome{}, false
}

func (w *fakeWorld) Edge() []core.Cell { return w.edge }

func (w *fakeWorld) IsOutside(c core.Cell) bool {
	return !w.grid.Wraps && (!w.grid.Contains(c) || w.walls[c])
}

// Chunk gives contents holding only their key: tests check which chunks are sent, not what is in them.
func (w *fakeWorld) Chunk(key core.ChunkKey) (*core.ChunkContents, bool) {
	if !w.grid.HasChunk(key) {
		return nil, false
	}
	return &core.ChunkContents{Key: key}, true
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}

func cell(col, row int) core.Cell { return core.Cell{Col: col, Row: row} }

var (
	center = cell(2, 2)
	rock   = cell(4, 4)
)

// fakeClock moves forward only when told.
type fakeClock struct{ at time.Time }

func (c *fakeClock) now() time.Time           { return c.at }
func (c *fakeClock) advance(by time.Duration) { c.at = c.at.Add(by) }

type world struct {
	room  *app.Room
	clock *fakeClock
	world *fakeWorld
}

// newWorld is a room on a 5 x 5 map: one chunk, with a rock in a corner.
func newWorld() world {
	return newWorldOn(core.Grid{Width: 5, Height: 5}, rock)
}

func newWorldOn(grid core.Grid, rocks ...core.Cell) world {
	clock := &fakeClock{at: time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)}
	fake := &fakeWorld{grid: grid, rocks: map[core.Cell]bool{}, spawn: center}
	for _, rock := range rocks {
		fake.rocks[rock] = true
	}
	return world{room: app.NewRoom("world", fake, clock.now, logs.Discard(), nil), clock: clock, world: fake}
}

// tickAfter moves the clock forward, then ticks the room.
func (w world) tickAfter(by time.Duration) {
	w.clock.advance(by)
	w.room.Tick()
}

const (
	ann = "a-public-id"
	bob = "b-public-id"
)

// joinBoth puts ann then bob on the room and forgets what they were told.
func joinBoth(t *testing.T, room *app.Room) (*fakeClient, *fakeClient) {
	t.Helper()
	annClient, bobClient := &fakeClient{}, &fakeClient{}
	require.NoError(t, room.Join(domain.Profile{ID: ann, Name: "Ann"}, annClient))
	require.NoError(t, room.Join(domain.Profile{ID: bob, Name: "Bob"}, bobClient))
	annClient.take()
	bobClient.take()
	return annClient, bobClient
}

func moved(id string, c core.Cell) domain.Event { return domain.Moved{ID: id, Cell: c} }

// ADR 0001 test 11
func TestJoinWelcomesAndTellsTheOthers(t *testing.T) {
	w := newWorld()
	annClient, bobClient := &fakeClient{}, &fakeClient{}
	require.NoError(t, w.room.Join(domain.Profile{ID: ann, Name: "Ann"}, annClient))
	annClient.take()

	require.NoError(t, w.room.Join(domain.Profile{ID: bob, Name: "Bob"}, bobClient))

	assert.Equal(t, []domain.Event{
		domain.Welcome{You: bob, Step: domain.StepInterval, Players: []domain.Player{
			{ID: ann, Name: "Ann", Cell: center},
			{ID: bob, Name: "Bob", Cell: center},
		}},
		domain.ChunkShown{Chunk: &core.ChunkContents{}},
	}, bobClient.take(), "the welcome, then the map's only chunk")
	assert.Equal(t, []domain.Event{domain.Joined{Player: domain.Player{ID: bob, Name: "Bob", Cell: center}}}, annClient.take())
}

// ADR 0002 test 7
func TestPlanTellsThePathWithoutMoving(t *testing.T) {
	w := newWorld()
	annClient, bobClient := joinBoth(t, w.room)

	w.room.Plan(ann, annClient, cell(4, 2))
	w.tickAfter(time.Second)

	assert.Equal(t, []domain.Event{domain.Planned{To: cell(4, 2), Path: []core.Cell{cell(3, 2), cell(4, 2)}, Action: domain.ActionMove}}, annClient.take())
	assert.Empty(t, bobClient.take(), "the others learn nothing, nobody moved")

	w.room.Plan(ann, annClient, rock)
	assert.Equal(t, []domain.Event{domain.Planned{To: rock, Action: domain.ActionMove}}, annClient.take(), "no path to a rock")
}

// ADR 0002 test 8
func TestWalkStepsAtTheServersPace(t *testing.T) {
	w := newWorld()
	annClient, bobClient := joinBoth(t, w.room)

	w.room.Walk(ann, annClient, cell(4, 2))
	assert.Equal(t, []domain.Event{domain.Walking{Path: []core.Cell{cell(3, 2), cell(4, 2)}}}, annClient.take())
	assert.Empty(t, bobClient.take())

	w.tickAfter(0)
	first := []domain.Event{moved(ann, cell(3, 2))}
	assert.Equal(t, first, annClient.take(), "a walker standing still steps at the next tick")
	assert.Equal(t, first, bobClient.take(), "everyone sees the step")

	w.tickAfter(domain.StepInterval - time.Millisecond)
	assert.Empty(t, bobClient.take(), "not before a step's time")

	w.tickAfter(time.Millisecond)
	assert.Equal(t, []domain.Event{moved(ann, cell(4, 2))}, bobClient.take())

	w.tickAfter(time.Second)
	assert.Empty(t, bobClient.take(), "arrived: no more steps")
}

// ADR 0002 test 9
func TestANewWalkKeepsThePace(t *testing.T) {
	w := newWorld()
	annClient, bobClient := joinBoth(t, w.room)
	w.room.Walk(ann, annClient, cell(4, 2))
	w.tickAfter(0)
	annClient.take()
	bobClient.take()

	w.clock.advance(domain.StepInterval / 2)
	w.room.Walk(ann, annClient, cell(3, 0))
	assert.Equal(t, []domain.Event{domain.Walking{Path: []core.Cell{cell(3, 1), cell(3, 0)}}}, annClient.take(),
		"the new path starts from where ann stands")

	w.tickAfter(0)
	assert.Empty(t, bobClient.take(), "walking again doesn't make ann faster")
	w.tickAfter(domain.StepInterval / 2)
	assert.Equal(t, []domain.Event{moved(ann, cell(3, 1))}, bobClient.take())
}

// ADR 0002 test 10
func TestAWalkToNowhereStopsTheWalk(t *testing.T) {
	w := newWorld()
	annClient, bobClient := joinBoth(t, w.room)
	w.room.Walk(ann, annClient, cell(0, 2))
	annClient.take()

	w.room.Walk(ann, annClient, rock)
	assert.Equal(t, []domain.Event{domain.Walking{}}, annClient.take())

	w.tickAfter(time.Second)
	assert.Empty(t, bobClient.take(), "ann stands still")
}

// ADR 0002 test 11, and ADR 0001 tests 12 and 16
func TestLeavingStopsTheWalkAndComingBackStandsWhereItGot(t *testing.T) {
	w := newWorld()
	annClient, bobClient := joinBoth(t, w.room)
	w.room.Walk(ann, annClient, cell(0, 0))
	w.tickAfter(0)

	w.room.Leave(ann, annClient)
	assert.Equal(t, []domain.Event{moved(ann, cell(1, 1)), domain.Left{ID: ann}}, bobClient.take())
	w.tickAfter(time.Second)
	assert.Empty(t, bobClient.take(), "a player who left doesn't walk")

	again := &fakeClient{}
	require.True(t, w.room.Return(domain.Profile{ID: ann, Name: "Ann"}, again))
	welcome := again.take()[0].(domain.Welcome)
	assert.Contains(t, welcome.Players, domain.Player{ID: ann, Name: "Ann", Cell: cell(1, 1)})
	w.tickAfter(time.Second)
	assert.Empty(t, bobClient.take()[1:], "the walk doesn't come back with her") // [0] is her joined
}

// ADR 0012 test 14: a room puts a player back only on a walkable cell they left; the hub sends the others home.
func TestReturning(t *testing.T) {
	t.Run("never here: not placed", func(t *testing.T) {
		w := newWorld()
		client := &fakeClient{}
		assert.False(t, w.room.Return(domain.Profile{ID: ann, Name: "Ann"}, client))
		assert.Empty(t, client.take())
	})

	t.Run("left from a cell that isn't walkable anymore: not placed, and nobody told", func(t *testing.T) {
		w := newWorld()
		annClient, bobClient := joinBoth(t, w.room)
		w.room.Walk(ann, annClient, cell(3, 3))
		w.tickAfter(0)
		w.room.Leave(ann, annClient)
		bobClient.take()

		w.world.rocks[cell(3, 3)] = true
		again := &fakeClient{}
		assert.False(t, w.room.Return(domain.Profile{ID: ann, Name: "Ann"}, again))
		assert.Empty(t, again.take())
		assert.Empty(t, bobClient.take())
	})

	t.Run("still here on another connection, even on a cell that isn't walkable: kept there", func(t *testing.T) {
		w := newWorld()
		annClient, _ := joinBoth(t, w.room)
		w.room.Teleport(ann, annClient, rock)

		newTab := &fakeClient{}
		require.True(t, w.room.Return(domain.Profile{ID: ann, Name: "Ann"}, newTab))
		assert.True(t, annClient.closed)
		assert.Contains(t, newTab.take()[0].(domain.Welcome).Players, domain.Player{ID: ann, Name: "Ann", Cell: rock})
	})

	t.Run("joining after leaving: at the spawn, not where they left", func(t *testing.T) {
		w := newWorld()
		annClient, _ := joinBoth(t, w.room)
		w.room.Teleport(ann, annClient, cell(0, 0))
		w.room.Leave(ann, annClient)

		again := &fakeClient{}
		require.NoError(t, w.room.Join(domain.Profile{ID: ann, Name: "Ann"}, again))
		assert.Contains(t, again.take()[0].(domain.Welcome).Players, domain.Player{ID: ann, Name: "Ann", Cell: center})
	})
}

// ADR 0002 test 13
func TestAStepNoLongerWalkableStopsTheWalk(t *testing.T) {
	w := newWorld()
	annClient, bobClient := joinBoth(t, w.room)
	w.room.Walk(ann, annClient, cell(4, 2))
	annClient.take()

	w.world.rocks[cell(3, 2)] = true // a server event drops a rock on the way
	w.tickAfter(0)

	assert.Equal(t, []domain.Event{domain.Stopped{Cell: center}}, annClient.take())
	assert.Empty(t, bobClient.take())
	delete(w.world.rocks, cell(3, 2))
	w.tickAfter(time.Second)
	assert.Empty(t, bobClient.take(), "the walk ended")
}

// ADR 0002 test 14, and ADR 0001 test 17
func TestJoiningAgainReplacesTheOldConnection(t *testing.T) {
	w := newWorld()
	annClient, bobClient := joinBoth(t, w.room)

	newTab := &fakeClient{}
	require.NoError(t, w.room.Join(domain.Profile{ID: ann, Name: "Ann"}, newTab))
	assert.True(t, annClient.closed, "the old connection is closed")
	assert.Len(t, newTab.take()[0].(domain.Welcome).Players, 2, "still one ann")
	assert.Empty(t, bobClient.take(), "bob is not told ann joined twice")

	w.room.Plan(ann, annClient, cell(4, 2))
	w.room.Walk(ann, annClient, cell(4, 2))
	w.room.Leave(ann, annClient)
	w.tickAfter(time.Second)
	assert.Empty(t, annClient.take(), "the old connection's requests are ignored")
	assert.Empty(t, bobClient.take(), "and its leaving changes nothing")

	w.room.Walk(ann, newTab, cell(3, 2))
	w.tickAfter(0)
	assert.Equal(t, []domain.Event{moved(ann, cell(3, 2))}, bobClient.take())
}

// ADR 0006 test 7
func TestAJoinerIsToldWithTheirRobot(t *testing.T) {
	w := newWorld()
	annClient, bobClient := &fakeClient{}, &fakeClient{}
	require.NoError(t, w.room.Join(domain.Profile{ID: ann, Name: "Ann", Robot: "dome-head"}, annClient))
	annClient.take()

	require.NoError(t, w.room.Join(domain.Profile{ID: bob, Name: "Bob", Robot: "rusty-toy"}, bobClient))

	assert.Equal(t, []domain.Player{
		{ID: ann, Name: "Ann", Robot: "dome-head", Cell: center},
		{ID: bob, Name: "Bob", Robot: "rusty-toy", Cell: center},
	}, bobClient.take()[0].(domain.Welcome).Players)
	assert.Equal(t, []domain.Event{domain.Joined{Player: domain.Player{ID: bob, Name: "Bob", Robot: "rusty-toy", Cell: center}}},
		annClient.take())
}

// ADR 0006 test 8
func TestJoiningAgainKeepsTheFirstRobot(t *testing.T) {
	w := newWorld()
	require.NoError(t, w.room.Join(domain.Profile{ID: ann, Name: "Ann", Robot: "dome-head"}, &fakeClient{}))

	newTab := &fakeClient{}
	require.NoError(t, w.room.Join(domain.Profile{ID: ann, Name: "Ann", Robot: "rusty-toy"}, newTab))

	players := newTab.take()[0].(domain.Welcome).Players
	assert.Equal(t, []domain.Player{{ID: ann, Name: "Ann", Robot: "dome-head", Cell: center}}, players)
}

// ADR 0001 test 18
func TestASlowClientIsDisconnected(t *testing.T) {
	w := newWorld()
	annClient, bobClient := joinBoth(t, w.room)
	w.room.Walk(ann, annClient, cell(3, 2))
	annClient.take()
	bobClient.full = true

	w.tickAfter(0)

	assert.True(t, bobClient.closed)
	assert.Equal(t, []domain.Event{moved(ann, cell(3, 2)), domain.Left{ID: bob}}, annClient.take())
}
