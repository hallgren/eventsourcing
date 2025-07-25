package tictactoe_test

import (
	"testing"

	"github.com/hallgren/eventsourcing/example/tictactoe"
)

func TestValidMove(t *testing.T) {
	game := tictactoe.NewGame()
	err := game.PlayMove(0, 0)
	if err != nil {
		t.Errorf("Expected valid move, got error: %v", err)
	}
	// verify the events
	if len(game.Events()) != 2 {
		t.Fatalf("expected two events got %d", len(game.Events()))
	}

	if game.Events()[0].Reason() != "Started" {
		t.Fatalf("expected first event to be started was %v", game.Events()[0].Reason())
	}

	switch e := game.Events()[1].Data().(type) {
	case *tictactoe.XMoved:
		if e.X != 0 && e.Y != 0 {
			t.Fatalf("Expected 'X' at 0,0, got %d,%d", e.X, e.Y)
		}
	default:
		t.Fatal("expeted XMoved event")
	}
}

func TestInvalidMoveAlreadyTaken(t *testing.T) {
	game := tictactoe.NewGame()
	_ = game.PlayMove(0, 0)
	err := game.PlayMove(0, 0)
	if err == nil {
		t.Errorf("Expected error for move on occupied square, got nil")
	}
}

func TestTurnSwitching(t *testing.T) {
	game := tictactoe.NewGame()
	_ = game.PlayMove(0, 0)
	if game.Turn() != "O" {
		t.Errorf("Expected turn to switch to O, got %s", game.Turn())
	}
}

func TestWinDetection(t *testing.T) {
	game := tictactoe.NewGame()
	game.PlayMove(0, 0)
	game.PlayMove(1, 0)
	game.PlayMove(0, 1)
	game.PlayMove(1, 1)
	game.PlayMove(0, 2) // X wins
	if !game.GameOver() || game.Winner() != "X" {
		t.Errorf("Expected X to win, got GameOver=%v, Winner=%s", game.GameOver(), game.Winner())
	}
}

func TestDrawDetection(t *testing.T) {
	game := tictactoe.NewGame()
	// first row
	game.PlayMove(0, 0) // X
	game.PlayMove(0, 1) // O
	game.PlayMove(0, 2) // X
	// second row
	game.PlayMove(1, 1) // O
	game.PlayMove(1, 0) // X
	game.PlayMove(1, 2) // O
	// third row
	game.PlayMove(2, 1) // X
	game.PlayMove(2, 0) // X
	_ = game.PlayMove(2, 2)
	if !game.GameOver() || game.Winner() != "" {
		t.Errorf("Expected draw, got GameOver=%v, Winner=%s", game.GameOver(), game.Winner())
	}
}
