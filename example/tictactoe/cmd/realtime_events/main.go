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
		"Draw": 0,
		"X":    0,
		"O":    0,
	}
	es := memory.Create()
	aggregate.Register(&tictactoe.Game{})
	aggregate.SaveHook(func(events []eventsourcing.Event) {
		lastEvent := events[len(events)-1]
		switch lastEvent.Data().(type) {
		case *tictactoe.Draw:
			m["Draw"]++
		case *tictactoe.XWon:
			m["X"]++
		case *tictactoe.OWon:
			m["O"]++
		}
	}, &tictactoe.Game{})

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
