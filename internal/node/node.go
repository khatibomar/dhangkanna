package node

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/khatibomar/dhangkanna/internal/state"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type Node struct {
	state *state.State

	upgrader          websocket.Upgrader
	logger            *log.Logger
	sendChannel       chan SocketEvent
	activeConnections map[*websocket.Conn]struct{}
	mutex             sync.Mutex
}

type SocketEvent struct {
	Name    string `json:"name"`
	Content any    `json:"content,omitempty"`
}

func New(ctx context.Context) *Node {
	n := &Node{
		state: state.New(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		logger:            log.New(log.Writer(), "Node: ", log.LstdFlags),
		sendChannel:       make(chan SocketEvent, 1),
		activeConnections: make(map[*websocket.Conn]struct{}),
	}

	go n.sendMessages(ctx)
	return n
}

func (n *Node) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := n.upgrader.Upgrade(w, r, nil)
	if err != nil {
		n.logger.Println(err)
		return
	}

	n.mutex.Lock()
	n.activeConnections[conn] = struct{}{}
	n.mutex.Unlock()

	go func(client *websocket.Conn) {
		defer func() {
			if err := conn.Close(); err != nil {
				n.logger.Printf("error while closing connection: %v\n", err)
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go n.receiveMessages(ctx, client)

		n.sendGameState()

		select {
		case <-ctx.Done():
			return
		}
	}(conn)
}

func (n *Node) receiveMessages(ctx context.Context, client *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msg struct {
				Letter  string `json:"letter"`
				Restart bool   `json:"restart"`
			}

			err := client.ReadJSON(&msg)
			n.logger.Printf("received message %v from socket", msg)
			if err != nil {
				n.logger.Println(err)
				return
			}

			if msg.Restart {
				n.resetGame()
			} else if !n.state.GameWon {
				letter := strings.ToLower(msg.Letter)
				n.handleNewLetter(letter)
			}
		}
	}
}

func (n *Node) sendMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case e := <-n.sendChannel:
			n.logger.Printf("sending message to socket %+v\n", e)

			n.mutex.Lock()
			connectionsToDelete := make([]*websocket.Conn, 0)
			for client := range n.activeConnections {
				err := client.WriteJSON(e)
				if err != nil {
					n.logger.Println(err)
					connectionsToDelete = append(connectionsToDelete, client)
				}
			}
			for _, client := range connectionsToDelete {
				delete(n.activeConnections, client)
			}
			n.mutex.Unlock()
		}
	}
}

func (n *Node) handleNewLetter(letter string) {
	if !isValidLetter(letter) {
		n.sendSocketEvent(SocketEvent{Name: "invalid_character"})
		return
	}
	n.logger.Printf("handling letter %v\n", letter)
	n.state.HandleNewLetter(letter)
	n.logger.Printf("handling letter %v done\n", letter)
	n.sendGameState()
}

func (n *Node) sendGameState() {
	n.sendSocketEvent(SocketEvent{Name: "state", Content: n.state})
}

func (n *Node) sendNotification(message string) {
	n.sendSocketEvent(SocketEvent{Name: "notification", Content: message})
}

func (n *Node) sendSocketEvent(event SocketEvent) {
	n.logger.Printf("sending message to channel %+v", event)
	n.sendChannel <- event
}

func (n *Node) resetGame() {
	n.state.Reset()
	n.sendGameState()
}

func isValidLetter(letter string) bool {
	return len(letter) == 1 && regexp.MustCompile("^[a-z]$").MatchString(letter)
}
