package main

import (
	"fmt"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	"github.com/hallgren/eventsourcing/example/tictactoe"
)

func main() {
	es := memory.Create()
	aggregate.Register(&tictactoe.Game{})
	aggregate.SaveHook(func(events []eventsourcing.Event) {
		fmt.Println(events, len(events))
	},
		&tictactoe.Game{},
	)

	game := tictactoe.NewGame()
	game.PlayMove(1, 2)
	aggregate.Save(es, game)
	fmt.Println("hej")
}
