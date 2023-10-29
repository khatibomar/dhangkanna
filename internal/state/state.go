package state

import (
	"fmt"
	"github.com/khatibomar/dhangkanna/internal"
	"regexp"
	"strings"
	"sync"
)

const characterName = "kanna kamui"
const initialChances = 6

const (
	GameStart = iota
	GameGoing
	GameWon
	GameLost
)

type State struct {
	GuessedCharacter []string `json:"guessedCharacter"`
	IncorrectGuesses []string `json:"incorrectGuesses"`
	ChancesLeft      int      `json:"chancesLeft"`
	GameState        int8     `json:"gameState"`
	Message          string   `json:"message"`

	mutex sync.Mutex
}

func New() *State {
	return &State{
		GuessedCharacter: initializeGuessedCharacter(characterName),
		IncorrectGuesses: make([]string, 0),
		ChancesLeft:      initialChances,
		GameState:        GameStart,
	}
}

func (s *State) Update(
	guessedCharacter []string,
	incorrectGuesses []string,
	chancesLeft int,
	gameState int8,
	message string,
) {
	s.GuessedCharacter = guessedCharacter
	s.IncorrectGuesses = incorrectGuesses
	s.ChancesLeft = chancesLeft
	s.GameState = gameState
	s.Message = message
}

func (s *State) HandleNewLetter(letter string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.GameState = GameGoing
	s.Message = ""
	if !isValidLetter(letter) {
		s.handleInvalidCharacter()
	} else if !internal.Contains(s.GuessedCharacter, letter) && !internal.Contains(s.IncorrectGuesses, letter) {
		if strings.Contains(characterName, letter) {
			s.handleCorrectGuess(letter)
		} else {
			s.handleIncorrectGuess(letter)
		}
	} else {
		s.handleRepeatedGuess(letter)
	}
}

func (s *State) Reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.GuessedCharacter = initializeGuessedCharacter(characterName)
	s.IncorrectGuesses = make([]string, 0)
	s.ChancesLeft = initialChances
	s.GameState = GameStart
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

func (s *State) handleCorrectGuess(letter string) {
	for i, char := range characterName {
		if string(char) == letter {
			s.GuessedCharacter[i] = letter
		}
	}
	if !internal.Contains(s.GuessedCharacter, "_") {
		s.GameState = GameWon
		s.Message = "Congratulations! You win!"
	}
}

func (s *State) handleIncorrectGuess(letter string) {
	s.IncorrectGuesses = append(s.IncorrectGuesses, letter)
	s.ChancesLeft--
	if s.ChancesLeft == 0 {
		s.GameState = GameLost
		s.Message = fmt.Sprintf("You lose! The character was: %s", characterName)
	}
}

func (s *State) handleRepeatedGuess(letter string) {
	s.Message = fmt.Sprintf("You already picked %s", letter)
}

func (s *State) handleInvalidCharacter() {
	s.Message = "Please enter a valid single letter."
}

func isValidLetter(letter string) bool {
	return len(letter) == 1 && regexp.MustCompile("^[a-z]$").MatchString(letter)
}
