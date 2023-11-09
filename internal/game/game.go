package game

import (
	"fmt"
	api "github.com/khatibomar/dhangkanna/cmd/api/v1"
	"github.com/khatibomar/dhangkanna/internal"
	"regexp"
	"strings"
	"sync"
)

const characterName = "kanna kamui"
const initialChances = 6

const (
	Start = iota
	Going
	Won
	Lost
)

type Game struct {
	GuessedCharacter []string `json:"guessedCharacter"`
	IncorrectGuesses []string `json:"incorrectGuesses"`
	ChancesLeft      int      `json:"chancesLeft"`
	GameState        int8     `json:"gameState"`
	Message          string   `json:"message"`
	Version          int      `json:"version,omitempty"`

	mu *sync.Mutex
}

func New() *Game {
	return &Game{
		GuessedCharacter: initializeGuessedCharacter(characterName),
		IncorrectGuesses: make([]string, 0),
		ChancesLeft:      initialChances,
		GameState:        Start,
		mu:               &sync.Mutex{},
	}
}

func (g *Game) Update(
	guessedCharacter []string,
	incorrectGuesses []string,
	chancesLeft int,
	gameState int8,
	message string,
	version int,
) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.GuessedCharacter = guessedCharacter
	g.IncorrectGuesses = incorrectGuesses
	g.ChancesLeft = chancesLeft
	g.GameState = gameState
	g.Message = message
	g.Version = version
}

func (g *Game) HandleNewLetter(letter string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.GameState = Going
	g.Message = ""
	if !isValidLetter(letter) {
		g.handleInvalidCharacter()
	} else if !internal.Contains(g.GuessedCharacter, letter) && !internal.Contains(g.IncorrectGuesses, letter) {
		if strings.Contains(characterName, letter) {
			g.handleCorrectGuess(letter)
		} else {
			g.handleIncorrectGuess(letter)
		}
	} else {
		g.handleRepeatedGuess(letter)
	}
	g.Version++
}

func (g *Game) Reset() {
	g.Update(
		initializeGuessedCharacter(characterName),
		make([]string, 0),
		initialChances,
		Start,
		"",
		g.Version+1,
	)
}

func ConvertGameApiToGame(apiGame *api.Game) Game {
	g := Game{
		GuessedCharacter: apiGame.GuessedCharacter,
		IncorrectGuesses: apiGame.IncorrectGuesses,
		ChancesLeft:      int(apiGame.ChancesLeft),
		GameState:        int8(apiGame.GameState),
		Message:          apiGame.Message,
		Version:          int(apiGame.Version),
	}

	if g.GuessedCharacter == nil {
		g.GuessedCharacter = make([]string, 0)
	}

	if g.IncorrectGuesses == nil {
		g.IncorrectGuesses = make([]string, 0)
	}

	return g
}

func initializeGuessedCharacter(characterName string) []string {
	guessedCharacter := make([]string, len(characterName))
	for i, char := range characterName {
		if char == ' ' {
			guessedCharacter[i] = " "
		} else {
			guessedCharacter[i] = "_"
		}
	}
	return guessedCharacter
}

func (g *Game) handleCorrectGuess(letter string) {
	for i, char := range characterName {
		if string(char) == letter {
			g.GuessedCharacter[i] = letter
		}
	}
	if !internal.Contains(g.GuessedCharacter, "_") {
		g.GameState = Won
		g.Message = "Congratulations! You win!"
	}
}

func (g *Game) handleIncorrectGuess(letter string) {
	g.IncorrectGuesses = append(g.IncorrectGuesses, letter)
	g.ChancesLeft--
	if g.ChancesLeft == 0 {
		g.GameState = Lost
		g.Message = fmt.Sprintf("You lose! The character was: %s", characterName)
	}
}

func (g *Game) handleRepeatedGuess(letter string) {
	g.Message = fmt.Sprintf("You already picked %s", letter)
}

func (g *Game) handleInvalidCharacter() {
	g.Message = "Please enter a valid single letter."
}

func isValidLetter(letter string) bool {
	return len(letter) == 1 && regexp.MustCompile("^[a-z]$").MatchString(letter)
}
