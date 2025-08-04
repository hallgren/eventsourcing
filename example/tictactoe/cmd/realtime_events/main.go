// This realtime events example demonstrates capturing and analyzing game events such as player moves and results
// by using save hooks in the event sourcing framework. The program outputs aggregated statistics
// after running a set of games.
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	"github.com/hallgren/eventsourcing/example/tictactoe"
)

func main() {
	// Buffered channels to receive move and result events
	chanMoved := make(chan string, 100)
	chanResult := make(chan string, 100)

	// Aggregated counts of moves and end results
	moveResult := map[string]int{
		"XMoved": 0,
		"OMoved": 0,
	}
	endResults := map[string]int{
		"Draw": 0,
		"XWon": 0,
		"OWon": 0,
	}
	// In-memory event store
	es := memory.Create()

	// Register the TicTacToe Game aggregate
	aggregate.Register(&tictactoe.Game{})

	// Hook to capture move events (XMoved/OMoved)
	aggregate.SetRealtimeEventsFunc(func(events []eventsourcing.Event) {
		for _, event := range events {
			switch event.Data().(type) {
			case *tictactoe.XMoved, *tictactoe.OMoved:
				chanMoved <- event.Reason()
			case *tictactoe.Draw, *tictactoe.XWon, *tictactoe.OWon:
				chanResult <- event.Reason()
			}
		}
	})
	// waitgroup to sync the finish of the async workers
	wg := sync.WaitGroup{}
	wg.Add(2)

	// Start async workers for processing move and result events
	go movesWorker(chanMoved, moveResult, &wg)
	go resultWorker(chanResult, endResults, &wg)

	// Simulate and save 10 TicTacToe games
	for i := 0; i < 10; i++ {
		game := PlayGame()
		aggregate.Save(es, game)
	}

	// Close channels to signal no more incoming data
	close(chanMoved)
	close(chanResult)
	fmt.Println("Events are saved, wait for workers to finish.")
	wg.Wait()

	// Print out aggregated move and result statistics
	fmt.Println(moveResult, endResults)
}

func resultWorker(c chan string, m map[string]int, wg *sync.WaitGroup) {
	for {
		select {
		case result, ok := <-c:
			// no more results
			if !ok {
				wg.Done()
				fmt.Println("results worker finsished")
				return
			}
			time.Sleep(50 * time.Millisecond)
			m[result]++
		}
	}
}

// do some time consuming operations in the go routine
func movesWorker(c chan string, m map[string]int, wg *sync.WaitGroup) {
	for {
		select {
		case move, ok := <-c:
			// no more moves
			if !ok {
				wg.Done()
				fmt.Println("moves worker finsished")
				return
			}
			time.Sleep(50 * time.Millisecond)
			m[move]++
		}
	}
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
