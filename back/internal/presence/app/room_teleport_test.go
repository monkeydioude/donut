package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/presence/domain"
)

func teleported(c core.Cell, walkable bool) domain.Event {
	return domain.Teleported{Cell: c, Walkable: walkable}
}

// chunksAround are the 9 chunks around one, by column then row.
func chunksAround(cx, cy int) []core.ChunkKey {
	var keys []core.ChunkKey
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			keys = append(keys, chunk(cx+dx, cy+dy))
		}
	}
	return keys
}

func forgottenKeys(events []domain.Event) []core.ChunkKey {
	var keys []core.ChunkKey
	for _, event := range events {
		if e, ok := event.(domain.ChunkForgotten); ok {
			keys = append(keys, e.Key)
		}
	}
	return keys
}

// ADR 0010 tests 2 and 4
func TestTeleportFar(t *testing.T) {
	w := newDonut()
	annClient := w.joinAt(t, ann, cell(40, 40)) // chunk (1, 1)
	bobClient := w.joinAt(t, bob, cell(50, 50)) // sees ann
	w.room.Walk(ann, annClient, cell(45, 40))
	annClient.take()
	bobClient.take()

	w.room.Teleport(ann, annClient, "world", cell(600, 300)) // chunk (18, 9)

	events := annClient.take()
	require.NotEmpty(t, events)
	assert.Equal(t, teleported(cell(600, 300), true), events[0], "told first")
	assert.Equal(t, shown(chunk(18, 9)), events[1], "its own chunk first")
	assert.ElementsMatch(t, chunksAround(18, 9), shownKeys(events), "the 9 chunks around the new cell")
	assert.ElementsMatch(t, chunksAround(1, 1), forgottenKeys(events), "every chunk held is far now")
	assert.Equal(t, []domain.Event{domain.Left{ID: ann}}, bobClient.take(), "bob only saw the old cell")

	w.tickAfter(domain.StepInterval)
	assert.Empty(t, annClient.take(), "the walk was dropped")
}

// ADR 0010 test 3
func TestTeleportOntoAnUnwalkableCell(t *testing.T) {
	w := newDonut()
	annClient := w.joinAt(t, ann, cell(40, 40))
	annClient.take()
	w.world.rocks[cell(600, 300)] = true

	w.room.Teleport(ann, annClient, "world", cell(600, 300))
	events := annClient.take()
	require.NotEmpty(t, events)
	assert.Equal(t, teleported(cell(600, 300), false), events[0], "landed there anyway")

	w.room.Walk(ann, annClient, cell(601, 300))
	assert.Equal(t, []domain.Event{domain.Walking{Path: nil}}, annClient.take(), "no path from an unwalkable cell: teleport again")
}

// ADR 0010 test 4
func TestTeleportWithinTheChunkSendsNoChunk(t *testing.T) {
	w := newDonut()
	annClient := w.joinAt(t, ann, cell(40, 40))
	annClient.take()

	w.room.Teleport(ann, annClient, "world", cell(41, 41))
	assert.Equal(t, []domain.Event{teleported(cell(41, 41), true)}, annClient.take())
}

// ADR 0010 test 5
func TestTeleportAsTheOthersSeeIt(t *testing.T) {
	w := newDonut()
	annClient := w.joinAt(t, ann, cell(40, 40))     // chunk (1, 1)
	bobClient := w.joinAt(t, bob, cell(50, 50))     // sees chunk (1, 1), not (18, 9)
	carlClient := w.joinAt(t, carl, cell(610, 300)) // sees chunk (18, 9), not (1, 1)
	daveClient := w.joinAt(t, dave, cell(900, 450)) // sees neither
	for _, client := range []*fakeClient{annClient, bobClient, carlClient, daveClient} {
		client.take()
	}

	w.room.Teleport(ann, annClient, "world", cell(45, 45))
	assert.Equal(t, []domain.Event{moved(ann, cell(45, 45))}, bobClient.take(), "saw both cells: moved")

	w.room.Teleport(ann, annClient, "world", cell(600, 300))
	assert.Equal(t, []domain.Event{domain.Left{ID: ann}}, bobClient.take(), "saw the old cell only: left")
	assert.Equal(t, []domain.Event{domain.Joined{Player: player(ann, cell(600, 300))}}, carlClient.take(), "sees the new cell only: joined")
	assert.Empty(t, daveClient.take(), "sees neither: nothing")
}

// ADR 0010 test 6
func TestTeleportRefused(t *testing.T) {
	cases := map[string]struct {
		toMap  string
		to     core.Cell
		reason string
	}{
		"an unknown map":         {"mars", cell(1, 1), "unknown map: mars"},
		"a cell outside the map": {"world", cell(5000, 2), "(5000, 2) is outside the map"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			w := newDonut()
			annClient := w.joinAt(t, ann, cell(40, 40))
			bobClient := w.joinAt(t, bob, cell(50, 50))
			annClient.take()
			bobClient.take()

			w.room.Teleport(ann, annClient, c.toMap, c.to)
			assert.Equal(t, []domain.Event{domain.Refused{Request: "teleport", Reason: c.reason}}, annClient.take())
			assert.Empty(t, bobClient.take(), "nobody moved")

			w.room.Teleport(ann, annClient, "world", cell(41, 41))
			assert.Equal(t, []domain.Event{moved(ann, cell(41, 41))}, bobClient.take(), "ann is still on her cell: bob sees the next teleport")
		})
	}
}

// ADR 0010 test 7
func TestTeleportCountsInTheLimiter(t *testing.T) {
	t.Run("the 11th request within a second is ignored", func(t *testing.T) {
		w := newDonut()
		annClient := w.joinAt(t, ann, cell(40, 40))
		annClient.take()
		for i := range domain.MaxRequestsPerSecond + 1 {
			w.room.Teleport(ann, annClient, "world", cell(41+i%2, 41))
		}
		assert.Len(t, annClient.take(), domain.MaxRequestsPerSecond)
	})

	t.Run("a replaced connection's teleport is ignored", func(t *testing.T) {
		w := newDonut()
		oldClient := w.joinAt(t, ann, cell(40, 40))
		newClient := w.joinAt(t, ann, cell(40, 40))
		oldClient.take()
		newClient.take()
		w.room.Teleport(ann, oldClient, "world", cell(41, 41))
		assert.Empty(t, oldClient.take())
		assert.Empty(t, newClient.take())
	})
}
