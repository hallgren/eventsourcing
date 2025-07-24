package tictactoe

import (
	"fmt"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
)

type Game struct {
	aggregate.Root
	Board    [3][3]string // "X", "O", or ""
	Turn     string       // "X" or "O"
	GameOver bool
	Winner   string // "X", "O", or ""
}

func (g *Game) Transition(event eventsourcing.Event) {
	switch e := event.Data().(type) {
	case *Started:
		g.Turn = "X"
	case *XMoved:
		g.Board[e.X][e.Y] = "X"
		g.Turn = "O"
	case *OMoved:
		g.Board[e.X][e.Y] = "O"
		g.Turn = "X"
	case *GameOver:
		g.GameOver = true
		g.Winner = e.Winner
	}
}

func (g *Game) Register(f aggregate.RegisterFunc) {
	f(&XMoved{}, &OMoved{})
}

type Started struct{}

type XMoved struct {
	X int
	Y int
}

type OMoved struct {
	X int
	Y int
}

type GameOver struct {
	Winner string
}

func NewGame() *Game {
	g := Game{}
	aggregate.TrackChange(&g, &Started{})
	return &g
}

func (g *Game) PlayMove(x, y int) error {
	if g.GameOver {
		return fmt.Errorf("game over")
	}
	if g.Board[x][y] != "" {
		return fmt.Errorf("position already taken")
	}

	if g.Turn == "X" {
		aggregate.TrackChange(g, &XMoved{x, y})
	} else {
		aggregate.TrackChange(g, &OMoved{x, y})
	}

	if winner := checkWinner(g.Board); winner != "" {
		aggregate.TrackChange(g, &GameOver{Winner: winner})
		return nil
	}

	if isDraw(g.Board) {
		aggregate.TrackChange(g, &GameOver{})
	}

	return nil
}

// checkWinner returns "X" or "O" if there's a winner, or "" if none.
func checkWinner(board [3][3]string) string {
	// Check rows and columns
	for i := 0; i < 3; i++ {
		if board[i][0] != "" && board[i][0] == board[i][1] && board[i][1] == board[i][2] {
			return board[i][0] // row win
		}
		if board[0][i] != "" && board[0][i] == board[1][i] && board[1][i] == board[2][i] {
			return board[0][i] // column win
		}
	}

	// Check diagonals
	if board[0][0] != "" && board[0][0] == board[1][1] && board[1][1] == board[2][2] {
		return board[0][0]
	}
	if board[0][2] != "" && board[0][2] == board[1][1] && board[1][1] == board[2][0] {
		return board[0][2]
	}

	// No winner
	return ""
}

func isDraw(board [3][3]string) bool {
	if checkWinner(board) != "" {
		return false
	}
	for _, row := range board {
		for _, cell := range row {
			if cell == "" {
				return false
			}
		}
	}
	return true
}
