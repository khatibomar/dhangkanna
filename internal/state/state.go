package state

import (
	"github.com/khatibomar/dhangkanna/internal"
	"strings"
	"sync"
)

const characterName = "kanna kamui"

type State struct {
	CharacterName    string   `json:"characterName"`
	GuessedCharacter []string `json:"guessedCharacter"`
	IncorrectGuesses []string `json:"incorrectGuesses"`
	RepeatedGuess    string   `json:"repeatedGuess"`
	ChancesLeft      int      `json:"chancesLeft"`
	GameWon          bool     `json:"gameWon"`
	NewGame          bool     `json:"newGame"`

	mutex sync.Mutex
}

func New() *State {
	return &State{
		CharacterName:    characterName,
		GuessedCharacter: initializeGuessedCharacter(characterName),
		IncorrectGuesses: make([]string, 0),
		ChancesLeft:      6,
	}
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

func (s *State) HandleNewLetter(letter string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.NewGame = false

	if !internal.Contains(s.GuessedCharacter, letter) && !internal.Contains(s.IncorrectGuesses, letter) {
		if strings.Contains(s.CharacterName, letter) {
			s.handleCorrectGuess(letter)
		} else {
			s.handleIncorrectGuess(letter)
		}
	} else {
		s.handleRepeatedGuess(letter)
	}
}

func (s *State) handleCorrectGuess(letter string) {
	for i, char := range s.CharacterName {
		if string(char) == letter {
			s.GuessedCharacter[i] = letter
		}
	}
	if !internal.Contains(s.GuessedCharacter, "_") {
		s.GameWon = true
	}

	s.RepeatedGuess = ""
}

func (s *State) handleIncorrectGuess(letter string) {
	s.IncorrectGuesses = append(s.IncorrectGuesses, letter)
	s.ChancesLeft--
	if s.ChancesLeft == 0 {
		s.GameWon = false
	}

	s.RepeatedGuess = ""
}

func (s *State) handleRepeatedGuess(letter string) {
	s.RepeatedGuess = letter
}

func (s *State) Reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.CharacterName = characterName
	s.GuessedCharacter = initializeGuessedCharacter(characterName)
	s.IncorrectGuesses = make([]string, 0)
	s.ChancesLeft = 6
	s.GameWon = false
	s.NewGame = true
}
