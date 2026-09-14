package tictactoe_test

import (
	"context"
	"testing"

	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	"github.com/hallgren/eventsourcing/example/tictactoe"
)

// BenchmarkPlayGame measures the cost of creating a game and tracking the
// changes for a full game (TrackChange + Transition + aggregateType lookups).
func BenchmarkPlayGame(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		game := tictactoe.NewGame()
		_ = game.PlayMove(0, 0) // X
		_ = game.PlayMove(1, 0) // O
		_ = game.PlayMove(0, 1) // X
		_ = game.PlayMove(1, 1) // O
		_ = game.PlayMove(0, 2) // X wins
	}
}

// BenchmarkSaveAndLoad measures the cost of saving a game to an event store
// and loading it back, exercising the replay/Transition path (buildFromHistory).
func BenchmarkSaveAndLoad(b *testing.B) {
	aggregate.Register(&tictactoe.Game{})
	es := memory.Create()
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		game := tictactoe.NewGame()
		_ = game.PlayMove(0, 0)
		_ = game.PlayMove(1, 0)
		_ = game.PlayMove(0, 1)
		_ = game.PlayMove(1, 1)
		_ = game.PlayMove(0, 2)

		if err := aggregate.Save(es, game); err != nil {
			b.Fatal(err)
		}

		loaded := &tictactoe.Game{}
		loaded.SetID(game.ID())
		if err := aggregate.Load(ctx, es, game.ID(), loaded); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadLongHistory measures loading (replaying) an aggregate with a
// long event history, which is where per-event allocation overhead in the
// replay loop shows up most clearly.
func BenchmarkLoadLongHistory(b *testing.B) {
	aggregate.Register(&tictactoe.Game{})
	es := memory.Create()
	ctx := context.Background()

	game := tictactoe.NewGame()
	// generate a long history of alternating moves (board correctness doesn't
	// matter here, only the number of events being replayed)
	coords := [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}}
	for i := 0; i < 2000; i++ {
		c := coords[i%len(coords)]
		if game.Turn() == "X" {
			aggregate.TrackChange(game, &tictactoe.XMoved{X: c[0], Y: c[1]})
		} else {
			aggregate.TrackChange(game, &tictactoe.OMoved{X: c[0], Y: c[1]})
		}
	}
	if err := aggregate.Save(es, game); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		loaded := &tictactoe.Game{}
		loaded.SetID(game.ID())
		if err := aggregate.Load(ctx, es, game.ID(), loaded); err != nil {
			b.Fatal(err)
		}
	}
}
