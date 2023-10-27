package node

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/khatibomar/dhangkanna/internal"
	"github.com/khatibomar/dhangkanna/internal/state"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type Node struct {
	state *state.State

	upgrader       websocket.Upgrader
	mutex          *sync.Mutex
	clientConn     *websocket.Conn
	logger         *log.Logger
	receiveChannel chan SocketEvent
	sendChannel    chan SocketEvent
	clients        map[*websocket.Conn]struct{}
}

type SocketEvent struct {
	Name    string `json:"name"`
	Content any    `json:"content,omitempty"`
}

func New() *Node {
	return &Node{
		state: state.New(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		mutex:          &sync.Mutex{},
		logger:         log.New(log.Writer(), "Node: ", log.LstdFlags),
		receiveChannel: make(chan SocketEvent, 2),
		sendChannel:    make(chan SocketEvent, 2),
		clients:        make(map[*websocket.Conn]struct{}),
	}
}

func (n *Node) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := n.upgrader.Upgrade(w, r, nil)
	if err != nil {
		n.logger.Println(err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			n.logger.Fatalf("error while closing connection: %v\n", err)
		}
	}()

	n.mutex.Lock()
	n.clients[conn] = struct{}{}
	n.clientConn = conn
	n.mutex.Unlock()
	n.sendGameState()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go n.receiveMessages(ctx)
	go n.sendMessages(ctx)

	select {
	case <-ctx.Done():
		n.mutex.Lock()
		delete(n.clients, conn)
		n.mutex.Unlock()
		return
	}
}

func (n *Node) receiveMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			n.mutex.Lock()
			delete(n.clients, n.clientConn)
			n.mutex.Unlock()
			return

		default:
			var msg struct {
				Letter  string `json:"letter"`
				Restart bool   `json:"restart"`
			}

			err := n.clientConn.ReadJSON(&msg)
			if err != nil {
				delete(n.clients, n.clientConn)
				n.logger.Println(err)
				return
			}

			if msg.Restart {
				n.resetGame()
				n.sendGameState()
			} else if !n.state.GameWon {
				letter := strings.ToLower(msg.Letter)
				n.handleNewLetter(letter)
				n.sendGameState()
			}
		}
	}
}

func (n *Node) sendMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			n.mutex.Lock()
			delete(n.clients, n.clientConn)
			n.mutex.Unlock()
			return

		case e := <-n.sendChannel:
			for client := range n.clients {
				n.mutex.Lock()
				err := client.WriteJSON(e)
				n.mutex.Unlock()
				if err != nil {
					delete(n.clients, client)
					n.logger.Println(err)
					return
				}
			}
		}
	}
}

func (n *Node) handleNewLetter(letter string) {
	n.state.NewGame = false
	if !isValidLetter(letter) {
		n.sendSocketEvent(SocketEvent{Name: "invalid_character"})
		return
	}
	if !internal.Contains(n.state.GuessedCharacter, letter) && !internal.Contains(n.state.IncorrectGuesses, letter) {
		if strings.Contains(n.state.CharacterName, letter) {
			n.handleCorrectGuess(letter)
		} else {
			n.handleIncorrectGuess(letter)
		}
	} else {
		n.handleRepeatedGuess(letter)
	}
}

func (n *Node) handleCorrectGuess(letter string) {
	for i, char := range n.state.CharacterName {
		if string(char) == letter {
			n.state.GuessedCharacter[i] = letter
		}
	}
	if !internal.Contains(n.state.GuessedCharacter, "_") {
		n.state.GameWon = true
	}

	n.state.RepeatedGuess = ""
}

func (n *Node) handleIncorrectGuess(letter string) {
	n.state.IncorrectGuesses = append(n.state.IncorrectGuesses, letter)
	n.state.ChancesLeft--
	if n.state.ChancesLeft == 0 {
		n.state.GameWon = false
	}

	n.state.RepeatedGuess = ""
}

func (n *Node) handleRepeatedGuess(letter string) {
	n.state.RepeatedGuess = letter
}

func (n *Node) sendGameState() {
	n.sendSocketEvent(SocketEvent{Name: "state", Content: n.state})
}

func (n *Node) sendNotification(message string) {
	n.sendSocketEvent(SocketEvent{Name: "notification", Content: message})
}

func (n *Node) sendSocketEvent(event SocketEvent) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.sendChannel <- event
}

func (n *Node) resetGame() {
	n.state.CharacterName = state.CharacterName
	n.state.GuessedCharacter = state.InitializeGuessedCharacter(state.CharacterName)
	n.state.IncorrectGuesses = make([]string, 0)
	n.state.ChancesLeft = 6
	n.state.GameWon = false
	n.state.NewGame = true
}

func isValidLetter(letter string) bool {
	return len(letter) == 1 && regexp.MustCompile("^[a-z]$").MatchString(letter)
}
