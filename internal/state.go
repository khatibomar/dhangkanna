package internal

import (
	"context"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
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

	upgrader       websocket.Upgrader
	mutex          *sync.Mutex
	clientConn     *websocket.Conn
	logger         *log.Logger
	receiveChannel chan struct{}
	sendChannel    chan struct{}
}

func New() *State {
	return &State{
		CharacterName:    characterName,
		GuessedCharacter: initializeGuessedCharacter(characterName),
		IncorrectGuesses: make([]string, 0),
		ChancesLeft:      6,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		mutex:          &sync.Mutex{},
		logger:         log.New(log.Writer(), "State: ", log.LstdFlags),
		receiveChannel: make(chan struct{}, 2),
		sendChannel:    make(chan struct{}, 2),
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

func (s *State) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Println(err)
		return
	}
	defer conn.Close()

	s.mutex.Lock()
	s.clientConn = conn
	s.sendGameState()
	s.mutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.receiveMessages(ctx)
	go s.sendMessages(ctx)

	select {
	case <-ctx.Done():
		return
	}
}

func (s *State) receiveMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		default:
			var msg struct {
				Letter  string `json:"letter"`
				Restart bool   `json:"restart"`
			}

			err := s.clientConn.ReadJSON(&msg)
			if err != nil {
				s.logger.Println(err)
				return
			}

			if msg.Restart {
				s.resetGame()
				s.sendGameState()
			} else if !s.GameWon {
				letter := strings.ToLower(msg.Letter)
				if !contains(s.GuessedCharacter, letter) && !contains(s.IncorrectGuesses, letter) {
					if strings.Contains(s.CharacterName, letter) {
						s.handleCorrectGuess(letter)
					} else {
						s.handleIncorrectGuess(letter)
					}
				} else {
					s.handleRepeatedGuess(letter)
				}
				s.sendGameState()
			}
		}
	}
}

func (s *State) sendMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-s.sendChannel:
			err := s.clientConn.WriteJSON(s)
			if err != nil {
				s.logger.Println(err)
				return
			}
		}
	}
}

func (s *State) handleCorrectGuess(letter string) {
	for i, char := range s.CharacterName {
		if string(char) == letter {
			s.GuessedCharacter[i] = letter
		}
	}
	if !contains(s.GuessedCharacter, "_") {
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

func (s *State) sendGameState() {
	s.sendChannel <- struct{}{}
}

func (s *State) resetGame() {
	s.CharacterName = characterName
	s.GuessedCharacter = initializeGuessedCharacter(characterName)
	s.IncorrectGuesses = make([]string, 0)
	s.ChancesLeft = 6
	s.GameWon = false
}
