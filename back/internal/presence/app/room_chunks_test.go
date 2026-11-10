package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "donut/internal/core/domain"
	"donut/internal/presence/domain"
)

const (
	carl = "c-public-id"
	dave = "d-public-id"
)

// newDonut is a room on an open 1024 x 512 world that wraps both ways: 32 x 16 chunks.
func newDonut() world {
	return newWorldOn(core.Grid{Width: 1024, Height: 512, Wraps: true})
}

// joinAt puts a player on a cell and gives their client, with what they were told so far still in it.
func (w world) joinAt(t *testing.T, id string, at core.Cell) *fakeClient {
	t.Helper()
	w.world.spawn = at
	client := &fakeClient{}
	require.NoError(t, w.room.Join(id, id, client))
	return client
}

// walkTo walks a player to a cell, a tick per step.
func (w world) walkTo(id string, client *fakeClient, to core.Cell) {
	w.room.Walk(id, client, to)
	for range 64 {
		w.tickAfter(domain.StepInterval)
	}
}

func chunk(cx, cy int) core.ChunkKey { return core.ChunkKey{CX: cx, CY: cy} }

func shown(key core.ChunkKey) domain.Event {
	return domain.ChunkShown{Chunk: &core.ChunkContents{Key: key}}
}

// shownKeys are the chunks among the events, in order.
func shownKeys(events []domain.Event) []core.ChunkKey {
	var keys []core.ChunkKey
	for _, event := range events {
		if e, ok := event.(domain.ChunkShown); ok {
			keys = append(keys, e.Chunk.Key)
		}
	}
	return keys
}

func player(id string, c core.Cell) domain.Player { return domain.Player{ID: id, Name: id, Cell: c} }

// withoutWalking drops the walking answers, to look at what the steps told.
func withoutWalking(events []domain.Event) []domain.Event {
	var kept []domain.Event
	for _, event := range events {
		if _, ok := event.(domain.Walking); !ok {
			kept = append(kept, event)
		}
	}
	return kept
}

// ADR 0003 test 14
func TestJoiningSendsTheChunksAroundAndOnlyThePlayersOnThem(t *testing.T) {
	w := newDonut()
	annClient := w.joinAt(t, ann, cell(40, 40))  // chunk (1, 1)
	bobClient := w.joinAt(t, bob, cell(300, 40)) // chunk (9, 1): far
	annClient.take()
	bobClient.take()

	carlClient := w.joinAt(t, carl, cell(70, 40)) // chunk (2, 1)

	events := carlClient.take()
	require.Len(t, events, 10)
	assert.Equal(t, domain.Welcome{You: carl, Step: domain.StepInterval, Players: []domain.Player{
		player(ann, cell(40, 40)), player(carl, cell(70, 40)),
	}}, events[0], "only the players on its 3 x 3 chunks")
	assert.Equal(t, shown(chunk(2, 1)), events[1], "its own chunk first")
	assert.ElementsMatch(t, []core.ChunkKey{
		chunk(1, 0), chunk(2, 0), chunk(3, 0), chunk(1, 1), chunk(2, 1), chunk(3, 1), chunk(1, 2), chunk(2, 2), chunk(3, 2),
	}, shownKeys(events))

	assert.Equal(t, []domain.Event{domain.Joined{Player: player(carl, cell(70, 40))}}, annClient.take(), "ann holds carl's chunk")
	assert.Empty(t, bobClient.take(), "bob doesn't")
}

// ADR 0003 test 15
func TestWalkingIntoTheNextChunk(t *testing.T) {
	t.Run("straight: 3 new chunks, the far ones forgotten only past 2 chunks", func(t *testing.T) {
		w := newDonut()
		annClient := w.joinAt(t, ann, cell(63, 40))
		annClient.take()

		w.walkTo(ann, annClient, cell(64, 40))
		assert.Equal(t, []domain.Event{moved(ann, cell(64, 40)), shown(chunk(3, 0)), shown(chunk(3, 1)), shown(chunk(3, 2))},
			withoutWalking(annClient.take()), "nothing forgotten: chunk column 0 is 2 away")

		w.walkTo(ann, annClient, cell(96, 40))
		events := annClient.take()
		assert.ElementsMatch(t, []core.ChunkKey{chunk(4, 0), chunk(4, 1), chunk(4, 2)}, shownKeys(events))
		assert.Subset(t, events, []domain.Event{
			domain.ChunkForgotten{Key: chunk(0, 0)}, domain.ChunkForgotten{Key: chunk(0, 1)}, domain.ChunkForgotten{Key: chunk(0, 2)},
		}, "chunk column 0 is 3 away")
	})

	t.Run("back and forth across a border, nothing more is sent", func(t *testing.T) {
		w := newDonut()
		annClient := w.joinAt(t, ann, cell(95, 40))
		w.walkTo(ann, annClient, cell(96, 40))
		annClient.take()

		for _, to := range []core.Cell{cell(95, 40), cell(96, 40), cell(95, 40), cell(96, 40)} {
			w.walkTo(ann, annClient, to)
		}
		for _, event := range withoutWalking(annClient.take()) {
			assert.IsType(t, domain.Moved{}, event)
		}
	})

	t.Run("diagonally: 5 new chunks", func(t *testing.T) {
		w := newDonut()
		annClient := w.joinAt(t, ann, cell(63, 63))
		annClient.take()

		w.walkTo(ann, annClient, cell(64, 64))
		assert.ElementsMatch(t, []core.ChunkKey{chunk(3, 1), chunk(3, 2), chunk(3, 3), chunk(1, 3), chunk(2, 3)}, shownKeys(annClient.take()))
	})
}

// ADR 0003 test 16
func TestOthersSeeAStepByTheChunksTheyHold(t *testing.T) {
	w := newDonut()
	annClient := w.joinAt(t, ann, cell(40, 40)) // holds chunk columns 0 to 2
	bobClient := w.joinAt(t, bob, cell(70, 40))
	annClient.take()

	w.walkTo(bob, bobClient, cell(71, 40))
	assert.Equal(t, []domain.Event{moved(bob, cell(71, 40))}, annClient.take(), "both cells seen: moved")

	w.walkTo(bob, bobClient, cell(96, 40))
	events := annClient.take()
	assert.Equal(t, domain.Left{ID: bob}, events[len(events)-1], "onto a chunk ann doesn't hold: left")

	w.walkTo(bob, bobClient, cell(97, 40))
	assert.Empty(t, annClient.take(), "neither cell seen: nothing")

	w.walkTo(bob, bobClient, cell(95, 40))
	assert.Equal(t, []domain.Event{domain.Joined{Player: player(bob, cell(95, 40))}}, annClient.take(), "back onto a chunk ann holds: joined")
}

// ADR 0003 test 17
func TestTheWalkersChunksBringAndTakeAwayPlayers(t *testing.T) {
	w := newDonut()
	annClient := w.joinAt(t, ann, cell(63, 40))
	w.joinAt(t, carl, cell(100, 40)) // chunk (3, 1): not held by ann yet
	w.joinAt(t, dave, cell(5, 40))   // chunk (0, 1): held by ann
	assert.Contains(t, annClient.take(), domain.Event(domain.Joined{Player: player(dave, cell(5, 40))}))

	w.walkTo(ann, annClient, cell(64, 40))
	assert.Equal(t, []domain.Event{
		moved(ann, cell(64, 40)), shown(chunk(3, 0)), shown(chunk(3, 1)), shown(chunk(3, 2)),
		domain.Joined{Player: player(carl, cell(100, 40))},
	}, withoutWalking(annClient.take()), "the chunks, then the players on them")

	w.walkTo(ann, annClient, cell(96, 40))
	events := annClient.take()
	left := domain.Event(domain.Left{ID: dave})
	assert.Contains(t, events, left)
	assert.Less(t, indexOf(events, left), indexOf(events, domain.ChunkForgotten{Key: chunk(0, 1)}), "the players, then the chunks")
}

func indexOf(events []domain.Event, event domain.Event) int {
	for i, e := range events {
		if e == event {
			return i
		}
	}
	return -1
}

// ADR 0003 test 18
func TestTheChunksAroundWrap(t *testing.T) {
	w := newDonut()
	w.joinAt(t, bob, cell(1020, 40)) // chunk (31, 1)
	annClient := w.joinAt(t, ann, cell(5, 5))

	events := annClient.take()
	assert.Contains(t, shownKeys(events), chunk(31, 1))
	assert.Contains(t, shownKeys(events), chunk(31, 15), "across both seams")
	assert.Contains(t, events[0].(domain.Welcome).Players, player(bob, cell(1020, 40)))

	w.walkTo(ann, annClient, cell(1023, 5))
	events = annClient.take()
	assert.Less(t, indexOf(events, moved(ann, cell(0, 5))), indexOf(events, moved(ann, cell(1023, 5))), "the walk crossed the seam westward")
	assert.ElementsMatch(t, []core.ChunkKey{chunk(30, 15), chunk(30, 0), chunk(30, 1)}, shownKeys(events),
		"into chunk column 31: column 30 comes, across the seam")
}

// ADR 0003 test 19
func TestNoPathOutsideTheHeldChunks(t *testing.T) {
	w := newDonut()
	annClient := w.joinAt(t, ann, cell(40, 40))
	w.room.Walk(ann, annClient, cell(50, 40))
	annClient.take()

	w.room.Plan(ann, annClient, cell(200, 40))
	assert.Equal(t, []domain.Event{domain.Planned{To: cell(200, 40)}}, annClient.take(), "ann doesn't hold that chunk")

	w.room.Walk(ann, annClient, cell(200, 40))
	assert.Equal(t, []domain.Event{domain.Walking{}}, annClient.take())
	w.tickAfter(time.Second)
	assert.Empty(t, annClient.take(), "the walk stopped")

	w.room.Plan(ann, annClient, cell(-1, 40))
	assert.Equal(t, []domain.Event{domain.Planned{To: cell(-1, 40)}}, annClient.take(), "cells are told within the map")
}

// ADR 0003 test 20
func TestOnlyThoseWhoSawALeaverAreTold(t *testing.T) {
	w := newDonut()
	annClient := w.joinAt(t, ann, cell(40, 40))
	bobClient := w.joinAt(t, bob, cell(70, 40))
	carlClient := w.joinAt(t, carl, cell(300, 40))
	bobClient.take()
	carlClient.take()

	w.room.Leave(ann, annClient)

	assert.Equal(t, []domain.Event{domain.Left{ID: ann}}, bobClient.take())
	assert.Empty(t, carlClient.take())
}

// ADR 0003 test 21
func TestJoiningAgainSendsEverythingAgain(t *testing.T) {
	w := newDonut()
	annClient := w.joinAt(t, ann, cell(40, 40))
	annClient.take()

	newTab := &fakeClient{}
	require.NoError(t, w.room.Join(ann, ann, newTab))

	events := newTab.take()
	assert.IsType(t, domain.Welcome{}, events[0])
	assert.Len(t, shownKeys(events), 9)
}
