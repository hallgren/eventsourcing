package main

import (
	"fmt"
	"math/rand"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	"github.com/hallgren/eventsourcing/example/tictactoe"
)

func main() {
	m := map[string]int{
		"Draw":   0,
		"XWon":   0,
		"OWon":   0,
		"XMoves": 0,
		"OMoves": 0,
	}
	es := memory.Create()
	aggregate.Register(&tictactoe.Game{})

	// Summaries results
	err := aggregate.SetSaveHook(func(events []eventsourcing.Event) {
		lastEvent := events[len(events)-1]
		switch lastEvent.Data().(type) {
		case *tictactoe.Draw:
			m["Draw"]++
		case *tictactoe.XWon:
			m["XWon"]++
		case *tictactoe.OWon:
			m["OWon"]++
		}
	}, &tictactoe.Game{})
	if err != nil {
		panic(err)
	}

	// Calculate moves
	err = aggregate.SetSaveHook(func(events []eventsourcing.Event) {
		for _, event := range events {
			switch event.Data().(type) {
			case *tictactoe.XMoved:
				m["XMoves"]++
			case *tictactoe.OMoved:
				m["OMoves"]++
			}
		}
	}, &tictactoe.Game{})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		game := PlayGame()
		aggregate.Save(es, game)
	}
	fmt.Println(m)
}

func PlayGame() *tictactoe.Game {
	game := tictactoe.NewGame()
	for !game.Done() {
		x := rand.Intn(3)
		y := rand.Intn(3)
		game.PlayMove(x, y)
	}
	return game
}
