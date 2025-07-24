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
	if game.Board[0][0] != "X" {
		t.Errorf("Expected 'X' at 0,0, got %s", game.Board[0][0])
	}

	// verify the events
	if len(game.Events()) != 2 {
		t.Fatalf("expected two events got %d", len(game.Events()))
	}
	if game.Events()[0].Reason() != "Started" {
		t.Fatalf("expected first event to be started was %v", game.Events()[0].Reason())
	}
	if game.Events()[1].Reason() != "XMoved" {
		t.Fatalf("expected second event to be x moved was %v", game.Events()[1].Reason())
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
	if game.Turn != "O" {
		t.Errorf("Expected turn to switch to O, got %s", game.Turn)
	}
}

func TestWinDetection(t *testing.T) {
	game := tictactoe.NewGame()
	game.PlayMove(0, 0)
	game.PlayMove(1, 0)
	game.PlayMove(0, 1)
	game.PlayMove(1, 1)
	game.PlayMove(0, 2) // X wins
	if !game.GameOver || game.Winner != "X" {
		t.Errorf("Expected X to win, got GameOver=%v, Winner=%s", game.GameOver, game.Winner)
	}
}

func TestDrawDetection(t *testing.T) {
	game := tictactoe.NewGame()
	game.Board = [3][3]string{
		{"X", "O", "X"},
		{"X", "O", "O"},
		{"O", "X", ""},
	}
	game.Turn = "X"
	_ = game.PlayMove(2, 2)
	if !game.GameOver || game.Winner != "" {
		t.Errorf("Expected draw, got GameOver=%v, Winner=%s", game.GameOver, game.Winner)
	}
}
