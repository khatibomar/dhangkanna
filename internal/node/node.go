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
		n.logger.Printf("Error upgrading WebSocket connection: %v", err)
		return
	}

	n.mutex.Lock()
	n.activeConnections[conn] = struct{}{}
	n.mutex.Unlock()

	n.logger.Printf("WebSocket connection established with %v", conn.RemoteAddr())

	go func(client *websocket.Conn) {
		defer func() {
			if err := conn.Close(); err != nil {
				n.logger.Printf("Error while closing connection with %v: %v", conn.RemoteAddr(), err)
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go n.receiveMessages(ctx, client)

		n.sendGameState()

		select {
		case <-ctx.Done():
			n.logger.Printf("Connection with %v closed.", conn.RemoteAddr())
			return
		}
	}(conn)

	n.logger.Printf("WebSocket connection handler started for %v", conn.RemoteAddr())
}

func (n *Node) receiveMessages(ctx context.Context, client *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			n.logger.Printf("Stopped receiving messages from %v", client.RemoteAddr())
			return
		default:
			var msg struct {
				Letter  string `json:"letter"`
				Restart bool   `json:"restart"`
			}

			err := client.ReadJSON(&msg)
			n.logger.Printf("received message %v from socket", msg)
			if err != nil {
				n.logger.Printf("Error reading JSON message from %v: %v", client.RemoteAddr(), err)
				return
			}

			n.logger.Printf("Received message %v from %v", msg, client.RemoteAddr())

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
			n.logger.Printf("Stopped sending messages")
			return

		case e := <-n.sendChannel:
			n.logger.Printf("Sending message to socket %+v\n", e)

			n.mutex.Lock()
			connectionsToDelete := make([]*websocket.Conn, 0)
			for client := range n.activeConnections {
				err := client.WriteJSON(e)
				if err != nil {
					n.logger.Printf("Error sending message to %v: %v", client.RemoteAddr(), err)
					connectionsToDelete = append(connectionsToDelete, client)
				}
			}
			for _, client := range connectionsToDelete {
				delete(n.activeConnections, client)
				n.logger.Printf("Connection with %v removed from active connections", client.RemoteAddr())
			}
			n.mutex.Unlock()
		}
	}
}

func (n *Node) handleNewLetter(letter string) {
	if !isValidLetter(letter) {
		n.sendSocketEvent(SocketEvent{Name: "invalid_character"})
		n.logger.Printf("Invalid letter received: %v", letter)
		return
	}
	n.logger.Printf("Handling letter %v", letter)
	n.state.HandleNewLetter(letter)
	n.logger.Printf("Letter %v handled successfully", letter)
	n.sendGameState()
}

func (n *Node) sendGameState() {
	n.sendSocketEvent(SocketEvent{Name: "state", Content: n.state})
	n.logger.Println("Sending game state to all connected clients")
}

func (n *Node) sendNotification(message string) {
	n.sendSocketEvent(SocketEvent{Name: "notification", Content: message})
	n.logger.Printf("Sending notification: %v", message)
}

func (n *Node) sendSocketEvent(event SocketEvent) {
	n.logger.Printf("Sending message to channel %+v", event)
	n.sendChannel <- event
}

func (n *Node) resetGame() {
	n.state.Reset()
	n.sendGameState()
	n.logger.Println("Game has been reset.")
}

func isValidLetter(letter string) bool {
	return len(letter) == 1 && regexp.MustCompile("^[a-z]$").MatchString(letter)
}
