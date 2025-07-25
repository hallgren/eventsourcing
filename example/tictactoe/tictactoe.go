package tictactoe

import (
	"fmt"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
)

type Game struct {
	aggregate.Root
	board    [3][3]string // "X", "O", or ""
	turn     string       // "X" or "O"
	gameOver bool
	winner   string // "X", "O", or ""
}

// Transition is an method to transform events to game state. The state is then used
// to uphold the rules of the game. This method is needed for the Game to be an aggregate
// and be used by the functions in the aggregate package.
func (g *Game) Transition(event eventsourcing.Event) {
	switch e := event.Data().(type) {
	case *Started:
		g.turn = "X"
	case *XMoved:
		g.board[e.X][e.Y] = "X"
		g.turn = "O"
	case *OMoved:
		g.board[e.X][e.Y] = "O"
		g.turn = "X"
	case *GameOver:
		g.gameOver = true
		g.winner = e.Winner
	}
}

// Register is used to register the events and the aggregate to the internal register.
func (g *Game) Register(f aggregate.RegisterFunc) {
	f(&Started{}, &XMoved{}, &OMoved{}, &GameOver{})
}

// Events

// Started indicates that the game has started
type Started struct{}

// XMoved then the X player moved togheter with the cordinates.
type XMoved struct {
	X int
	Y int
}

// OMoved then the O player moved togheter with the cordinates.
type OMoved struct {
	X int
	Y int
}

// GameOver is the last event containing the winner and enpty string if it's a draw.
type GameOver struct {
	Winner string
}

// Constructor
func NewGame() *Game {
	g := Game{}
	aggregate.TrackChange(&g, &Started{})
	return &g
}

// Query methods
func (g *Game) Turn() string {
	return g.turn
}

func (g *Game) GameOver() bool {
	return g.gameOver
}

func (g *Game) Winner() string {
	return g.winner
}

// Render prints the board
func (g *Game) Render() {
	for i, row := range g.board {
		for j, cell := range row {
			if cell == "" {
				fmt.Print("   ")
			} else {
				fmt.Printf(" %s ", cell)
			}
			if j < len(row)-1 {
				fmt.Print("|")
			}
		}
		fmt.Println()
		if i < len(g.board)-1 {
			fmt.Println("---+---+---")
		}
	}
}

// Commands
func (g *Game) PlayMove(x, y int) error {
	// verify that the move can be made
	if g.gameOver {
		return fmt.Errorf("game over")
	}
	if g.board[x][y] != "" {
		return fmt.Errorf("position already taken")
	}

	if g.turn == "X" {
		aggregate.TrackChange(g, &XMoved{x, y})
	} else {
		aggregate.TrackChange(g, &OMoved{x, y})
	}

	if winner := checkWinner(g.board); winner != "" {
		aggregate.TrackChange(g, &GameOver{Winner: winner})
		return nil
	}

	if isDraw(g.board) {
		aggregate.TrackChange(g, &GameOver{})
	}

	return nil
}

// helpers
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
