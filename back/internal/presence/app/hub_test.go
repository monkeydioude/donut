package app_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/presence/app"
	"donut/internal/presence/domain"
	"donut/pkg/errd"
	"donut/pkg/logs"
)

// The hub's maps (ADR 0012): "world", 64 x 64 and wrapping, with a dome on (10, 10) to (11, 11) leading to "city";
// "city", 16 x 16, walled on its border, whose edge is the ring of cells one in, standing on that dome.
var domeBack = cell(10, 10)

func worldWithDome(spawn core.Cell) *fakeWorld {
	return &fakeWorld{
		grid: core.Grid{Width: 64, Height: 64, Wraps: true}, rocks: map[core.Cell]bool{}, spawn: spawn,
		info:  core.MapInfo{Slug: "world", Name: "World", Width: 64, Height: 64, Wraps: true},
		domes: []core.Dome{{Back: domeBack, LeadsTo: "city"}},
	}
}

func walledCity() *fakeWorld {
	city := &fakeWorld{
		grid: core.Grid{Width: 16, Height: 16}, rocks: map[core.Cell]bool{}, spawn: cell(8, 8), walls: map[core.Cell]bool{},
		info: core.MapInfo{Slug: "city", Name: "City", Width: 16, Height: 16,
			StandsOn: &core.Place{Map: "world", Col: domeBack.Col, Row: domeBack.Row, Width: 64, Height: 64}},
	}
	for row := range 16 {
		for col := range 16 {
			c := cell(col, row)
			switch {
			case col == 0 || col == 15 || row == 0 || row == 15:
				city.walls[c] = true
			case col == 1 || col == 14 || row == 1 || row == 14:
				city.edge = append(city.edge, c)
			}
		}
	}
	return city
}

type hubFixture struct {
	hub   *app.Hub
	clock *fakeClock
	maps  map[string]*fakeWorld
}

// newHub's rooms don't tick on their own: the tests tick them, with a clock that moves only when told.
func newHub(t *testing.T, home string, maps map[string]*fakeWorld) hubFixture {
	t.Helper()
	clock := &fakeClock{at: time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)}
	stopped, stop := context.WithCancel(context.Background())
	stop()
	hub := app.NewHub(stopped, func(_ context.Context, slug string) (app.World, error) {
		if world, ok := maps[slug]; ok {
			return world, nil
		}
		return nil, fmt.Errorf("no map %q: %w", slug, errd.ErrNotFound)
	}, clock.now, home, logs.Discard())
	return hubFixture{hub: hub, clock: clock, maps: maps}
}

func newDomeHub(t *testing.T, home string, worldSpawn core.Cell) hubFixture {
	return newHub(t, home, map[string]*fakeWorld{"world": worldWithDome(worldSpawn), "city": walledCity()})
}

func (f hubFixture) connect(t *testing.T, id, name string) (*app.Session, *fakeClient) {
	t.Helper()
	client := &fakeClient{}
	session, err := f.hub.Connect(context.Background(), domain.Profile{ID: id, Name: name}, client)
	require.NoError(t, err)
	return session, client
}

func (f hubFixture) room(t *testing.T, slug string) *app.Room {
	t.Helper()
	room, err := f.hub.Room(context.Background(), slug)
	require.NoError(t, err)
	return room
}

// walk ticks both rooms `steps` times, a step apart.
func (f hubFixture) walk(t *testing.T, steps int) {
	t.Helper()
	for range steps {
		f.clock.advance(domain.StepInterval)
		f.room(t, "world").Tick()
		f.room(t, "city").Tick()
	}
}

// welcomeOf is the first welcome among the events, failing without one.
func welcomeOf(t *testing.T, events []domain.Event) domain.Welcome {
	t.Helper()
	for _, event := range events {
		if welcome, ok := event.(domain.Welcome); ok {
			return welcome
		}
	}
	require.Fail(t, "no welcome", "%v", events)
	return domain.Welcome{}
}

// youIn is where the welcome puts the player `id`.
func youIn(t *testing.T, welcome domain.Welcome, id string) core.Cell {
	t.Helper()
	for _, p := range welcome.Players {
		if p.ID == id {
			return p.Cell
		}
	}
	require.Fail(t, "not in the welcome", id)
	return core.Cell{}
}

// ADR 0012 test 14, with the city as the home map, as Doma-1 is
func TestConnecting(t *testing.T) {
	t.Run("a new guest wakes up at the home map's spawn, and is told the map", func(t *testing.T) {
		f := newDomeHub(t, "city", cell(5, 5))
		_, client := f.connect(t, ann, "Ann")
		welcome := welcomeOf(t, client.take())
		assert.Equal(t, "city", welcome.Map.Slug)
		assert.Equal(t, cell(8, 8), youIn(t, welcome, ann))
	})

	t.Run("a guest who left the world at a cell comes back there", func(t *testing.T) {
		f := newDomeHub(t, "city", cell(5, 5))
		session, _ := f.connect(t, ann, "Ann")
		session.Teleport(context.Background(), "world", cell(20, 20))
		session.Leave()

		_, client := f.connect(t, ann, "Ann")
		welcome := welcomeOf(t, client.take())
		assert.Equal(t, "world", welcome.Map.Slug)
		assert.Equal(t, cell(20, 20), youIn(t, welcome, ann))
	})

	t.Run("one who left the world from a cell that isn't walkable wakes up at the home map's spawn", func(t *testing.T) {
		f := newDomeHub(t, "city", cell(5, 5))
		f.maps["world"].rocks[cell(20, 20)] = true
		session, _ := f.connect(t, ann, "Ann")
		session.Teleport(context.Background(), "world", cell(20, 20))
		session.Leave()

		session, client := f.connect(t, ann, "Ann")
		welcome := welcomeOf(t, client.take())
		assert.Equal(t, "city", welcome.Map.Slug)
		assert.Equal(t, cell(8, 8), youIn(t, welcome, ann))

		session.Teleport(context.Background(), "city", cell(3, 3))
		assert.IsType(t, domain.Teleported{}, client.take()[0], "their requests go to the home map's room")
	})

	t.Run("so does one who left the home map from a wall", func(t *testing.T) {
		f := newDomeHub(t, "city", cell(5, 5))
		session, _ := f.connect(t, ann, "Ann")
		session.Teleport(context.Background(), "city", cell(0, 0))
		session.Leave()

		_, client := f.connect(t, ann, "Ann")
		assert.Equal(t, cell(8, 8), youIn(t, welcomeOf(t, client.take()), ann))
	})

	t.Run("one still on the world on another tab keeps their place there, walkable or not", func(t *testing.T) {
		f := newDomeHub(t, "city", cell(5, 5))
		f.maps["world"].rocks[cell(20, 20)] = true
		session, oldTab := f.connect(t, ann, "Ann")
		session.Teleport(context.Background(), "world", cell(20, 20))

		_, newTab := f.connect(t, ann, "Ann")
		welcome := welcomeOf(t, newTab.take())
		assert.True(t, oldTab.closed)
		assert.Equal(t, "world", welcome.Map.Slug)
		assert.Equal(t, cell(20, 20), youIn(t, welcome, ann))
	})
}

// ADR 0012 test 15
func TestEnter(t *testing.T) {
	t.Run("from afar: walk beside the dome, then into the city on the side one came from", func(t *testing.T) {
		f := newDomeHub(t, "world", cell(5, 5))
		ann, annClient := f.connect(t, ann, "Ann")
		_, bobClient := f.connect(t, bob, "Bob")
		annClient.take()
		bobClient.take()

		ann.Enter(context.Background(), cell(11, 11))
		walking := annClient.take()
		require.Equal(t, []domain.Event{domain.Walking{Path: []core.Cell{cell(6, 6), cell(7, 7), cell(8, 8), cell(9, 9)}}}, walking,
			"to (9, 9), the side closest by straight distance")
		f.walk(t, 4)

		events := annClient.take()
		welcome := welcomeOf(t, events)
		assert.Equal(t, "city", welcome.Map.Slug)
		assert.Equal(t, cell(1, 1), youIn(t, welcome, "a-public-id"), "from the dome's north-west, the city's north-west corner")
		assert.Contains(t, bobClient.take(), domain.Event(domain.Left{ID: "a-public-id"}), "bob saw her go")
	})

	t.Run("from beside the dome: in at once", func(t *testing.T) {
		f := newDomeHub(t, "world", cell(12, 11))
		session, client := f.connect(t, ann, "Ann")
		client.take()
		session.Enter(context.Background(), domeBack)
		welcome := welcomeOf(t, client.take())
		assert.Equal(t, "city", welcome.Map.Slug)
		assert.Equal(t, 14, youIn(t, welcome, ann).Col, "from the east, the east edge")
	})

	t.Run("not a dome's cell: refused, nobody moves", func(t *testing.T) {
		f := newDomeHub(t, "world", cell(5, 5))
		session, client := f.connect(t, ann, "Ann")
		client.take()
		session.Enter(context.Background(), cell(3, 3))
		assert.Equal(t, []domain.Event{domain.Refused{Request: "enter", Reason: "There's no dome at (3, 3)"}}, client.take())
	})

	t.Run("every side blocked: an empty path", func(t *testing.T) {
		f := newDomeHub(t, "world", cell(5, 5))
		for _, c := range (core.Dome{Back: domeBack}).Around(f.maps["world"].grid) {
			f.maps["world"].rocks[c] = true
		}
		session, client := f.connect(t, ann, "Ann")
		client.take()
		session.Enter(context.Background(), domeBack)
		assert.Equal(t, []domain.Event{domain.Walking{}}, client.take())
	})

	t.Run("a plan on a dome's cell names the action, and the way beside it", func(t *testing.T) {
		f := newDomeHub(t, "world", cell(5, 5))
		session, client := f.connect(t, ann, "Ann")
		client.take()
		session.Plan(cell(11, 10))
		assert.Equal(t, []domain.Event{domain.Planned{To: cell(11, 10), Action: domain.ActionEnter,
			Path: []core.Cell{cell(6, 6), cell(7, 7), cell(8, 8), cell(9, 9)}}}, client.take())
	})
}

// ADR 0012 test 16
func TestExit(t *testing.T) {
	t.Run("walk to the edge closest to the click, then out beside the dome on that side", func(t *testing.T) {
		f := newDomeHub(t, "city", cell(5, 5))
		session, client := f.connect(t, ann, "Ann")
		client.take()

		session.Exit(context.Background(), cell(-2, 8))
		assert.Len(t, client.take()[0].(domain.Walking).Path, 7, "from (8, 8) to (1, 8)")
		f.walk(t, 7)

		welcome := welcomeOf(t, client.take())
		assert.Equal(t, "world", welcome.Map.Slug)
		assert.Equal(t, 9, youIn(t, welcome, ann).Col, "the west edge gives the dome's west side")
	})

	t.Run("a plan outside the city names the action", func(t *testing.T) {
		f := newDomeHub(t, "city", cell(5, 5))
		session, client := f.connect(t, ann, "Ann")
		client.take()
		session.Plan(cell(8, 15))
		planned := client.take()[0].(domain.Planned)
		assert.Equal(t, domain.ActionExit, planned.Action)
		assert.Equal(t, cell(8, 14), planned.Path[len(planned.Path)-1], "to the south edge, below the wall clicked")
	})

	t.Run("that side's cell not walkable: out on the next closest", func(t *testing.T) {
		f := newDomeHub(t, "city", cell(5, 5))
		f.maps["world"].rocks[cell(9, 11)] = true
		session, client := f.connect(t, ann, "Ann")
		f.room(t, "world") // opened, so its rocks are the ones above
		client.take()
		session.Exit(context.Background(), cell(-2, 8))
		f.walk(t, 7)
		arrival := youIn(t, welcomeOf(t, client.take()), ann)
		assert.NotEqual(t, cell(9, 11), arrival)
		assert.True(t, f.maps["world"].IsWalkable(arrival))
	})

	t.Run("on the world: not in a city", func(t *testing.T) {
		f := newDomeHub(t, "world", cell(5, 5))
		session, client := f.connect(t, ann, "Ann")
		client.take()
		session.Exit(context.Background(), cell(0, 0))
		assert.Equal(t, []domain.Event{domain.Refused{Request: "exit", Reason: "You're not in a city"}}, client.take())
	})

	t.Run("no dome leads to the city: it stands nowhere", func(t *testing.T) {
		f := newDomeHub(t, "city", cell(5, 5))
		f.maps["world"].domes = nil
		session, client := f.connect(t, ann, "Ann")
		client.take()
		session.Exit(context.Background(), cell(-2, 8))
		assert.Equal(t, []domain.Event{domain.Refused{Request: "exit", Reason: "City stands nowhere on the world"}}, client.take())
	})
}

// ADR 0012 test 17
func TestAMoveBetweenMaps(t *testing.T) {
	t.Run("the players of the city see the traveler come, who is welcomed with the map then its chunks", func(t *testing.T) {
		f := newDomeHub(t, "world", cell(12, 11))
		carl, carlClient := f.connect(t, "c-public-id", "Carl")
		carl.Teleport(context.Background(), "city", cell(13, 9))
		carlClient.take()

		session, client := f.connect(t, ann, "Ann")
		client.take()
		session.Enter(context.Background(), domeBack)

		events := client.take()
		require.GreaterOrEqual(t, len(events), 2)
		assert.IsType(t, domain.Welcome{}, events[0], "the welcome first")
		assert.IsType(t, domain.ChunkShown{}, events[1], "then the chunks")
		arrival := youIn(t, events[0].(domain.Welcome), ann)
		assert.Equal(t, []domain.Event{domain.Joined{Player: domain.Player{ID: ann, Name: "Ann", Cell: arrival}}}, carlClient.take(),
			"carl, in the city, sees ann come")
	})

	t.Run("the request limit follows the guest across maps", func(t *testing.T) {
		f := newDomeHub(t, "world", cell(12, 11))
		session, client := f.connect(t, ann, "Ann")
		client.take()
		for range domain.MaxRequestsPerSecond - 1 {
			session.Plan(cell(3, 3))
		}
		session.Enter(context.Background(), domeBack)
		assert.Equal(t, "city", welcomeOf(t, client.take()).Map.Slug, "the 10th request goes through")

		session.Plan(cell(5, 5))
		assert.Empty(t, client.take(), "the 11th, in the city, is ignored")
		f.clock.advance(time.Second / domain.MaxRequestsPerSecond)
		session.Plan(cell(5, 5))
		assert.Len(t, client.take(), 1, "time gives requests back")
	})

	t.Run("another walk on the way cancels going through", func(t *testing.T) {
		f := newDomeHub(t, "world", cell(5, 5))
		session, client := f.connect(t, ann, "Ann")
		client.take()
		session.Enter(context.Background(), domeBack)
		f.walk(t, 1)
		session.Walk(cell(6, 4))
		f.walk(t, 6)
		for _, event := range client.take() {
			_, welcomed := event.(domain.Welcome)
			assert.False(t, welcomed, "ann never went in")
		}
	})
}

// ADR 0012 test 18
func TestTeleportAcrossMaps(t *testing.T) {
	t.Run("lands on the cell: the welcome, the chunks, then teleported", func(t *testing.T) {
		f := newDomeHub(t, "world", cell(5, 5))
		session, client := f.connect(t, ann, "Ann")
		client.take()
		session.Teleport(context.Background(), "city", cell(0, 0))
		events := client.take()
		require.Len(t, events, 3)
		assert.Equal(t, "city", events[0].(domain.Welcome).Map.Slug)
		assert.IsType(t, domain.ChunkShown{}, events[1])
		assert.Equal(t, domain.Teleported{Cell: cell(0, 0), Walkable: false}, events[2], "a wall: landed anyway")
	})

	for name, c := range map[string]struct {
		toMap, reason string
		to            core.Cell
	}{
		"an unknown map":                   {"mars", "unknown map: mars", cell(1, 1)},
		"a cell outside the map named":     {"city", "(20, 20) is outside the map", cell(20, 20)},
		"a cell outside another map's box": {"city", "(-1, 3) is outside the map", cell(-1, 3)},
	} {
		t.Run("refused before anything moves: "+name, func(t *testing.T) {
			f := newDomeHub(t, "world", cell(5, 5))
			session, client := f.connect(t, ann, "Ann")
			client.take()
			session.Teleport(context.Background(), c.toMap, c.to)
			assert.Equal(t, []domain.Event{domain.Refused{Request: "teleport", Reason: c.reason}}, client.take())
			session.Plan(cell(6, 5))
			assert.Equal(t, cell(6, 5), client.take()[0].(domain.Planned).Path[0], "still on the world, at the spawn")
		})
	}
}

// ADR 0012 test 19: two rooms trading players both ways at once never deadlock (run with -race).
func TestRoomsTradingPlayersNeverDeadlock(t *testing.T) {
	maps := map[string]*fakeWorld{"world": worldWithDome(cell(5, 5)), "city": walledCity()}
	hub := app.NewHub(t.Context(), func(_ context.Context, slug string) (app.World, error) {
		return maps[slug], nil
	}, time.Now, "world", logs.Discard())
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session, err := hub.Connect(context.Background(), domain.Profile{ID: fmt.Sprintf("p%d", i), Name: "P"}, &lockedClient{})
			if !assert.NoError(t, err) {
				return
			}
			defer session.Leave()
			for j := range 10 {
				target := []string{"city", "world"}[j%2]
				session.Teleport(context.Background(), target, cell(3, 3))
				session.Enter(context.Background(), domeBack)
				session.Exit(context.Background(), cell(-1, 3))
			}
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock: the players never finished travelling")
	}
}

// lockedClient takes events from several goroutines: the rooms' loops and the requests.
type lockedClient struct {
	mu     sync.Mutex
	events int
}

func (c *lockedClient) Send(domain.Event) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events++
	return true
}

func (c *lockedClient) Close() {}
