package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"
	api "github.com/khatibomar/dhangkanna/cmd/api/v1"
	"github.com/khatibomar/dhangkanna/internal/client"
	"github.com/khatibomar/dhangkanna/internal/game"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Socket struct {
	backendAddrs      []string
	upgrader          websocket.Upgrader
	logger            *log.Logger
	sendChannel       chan Event
	activeConnections map[*websocket.Conn]struct{}
	mutex             sync.Mutex
}

type Event struct {
	Name    string `json:"name"`
	Content any    `json:"content"`
}

func NewSocket(ctx context.Context, backendAddrs []string) (*Socket, error) {
	n := &Socket{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		logger:            log.New(log.Writer(), "Socket: ", log.LstdFlags|log.Lshortfile),
		sendChannel:       make(chan Event, 1),
		activeConnections: make(map[*websocket.Conn]struct{}),
		backendAddrs:      backendAddrs,
	}

	go n.sendMessages(ctx)

	return n, nil
}

func (n *Socket) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
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

		err := n.sendGameState(ctx)
		if err != nil {
			n.logger.Printf("failed to send state: %v", err)
			return
		}

		<-ctx.Done()
		n.logger.Printf("Connection with %v closed.", conn.RemoteAddr())
	}(conn)

	n.logger.Printf("WebSocket connection handler started for %v", conn.RemoteAddr())
}

func (n *Socket) receiveMessages(ctx context.Context, client *websocket.Conn) {
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
				err := n.resetGame(ctx)
				if err != nil {
					n.logger.Println(err)
					return
				}
			} else {
				letter := strings.ToLower(msg.Letter)
				err := n.handleNewLetter(ctx, letter)
				if err != nil {
					n.logger.Println(err)
					return
				}
			}
		}
	}
}

func (n *Socket) sendMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			n.logger.Printf("Stopped sending messages")
			return

		case e := <-n.sendChannel:
			n.logger.Printf("Sending message to socket %+v\n", e)

			n.mutex.Lock()
			connectionsToDelete := make([]*websocket.Conn, 0)
			for c := range n.activeConnections {
				err := c.WriteJSON(e)
				if err != nil {
					n.logger.Printf("Error sending message to %v: %v", c.RemoteAddr(), err)
					connectionsToDelete = append(connectionsToDelete, c)
				}
			}
			for _, c := range connectionsToDelete {
				delete(n.activeConnections, c)
				n.logger.Printf("Connection with %v removed from active connections", c.RemoteAddr())
			}
			n.mutex.Unlock()
		}
	}
}

func (n *Socket) handleNewLetter(ctx context.Context, letter string) error {
	n.logger.Printf("Handling letter %v", letter)

	c, err := n.connectToRandomServer()
	if err != nil {
		return err
	}

	_, err = c.Send(ctx, &api.Letter{Letter: letter})
	if err != nil {
		return err
	}
	n.logger.Printf("Letter %v handled successfully", letter)
	err = n.sendGameState(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (n *Socket) sendGameState(ctx context.Context) error {
	c, err := n.connectToRandomServer()
	if err != nil {
		return err
	}

	g, err := c.Receive(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}

	n.sendSocketEvent(Event{Name: "game", Content: game.ConvertGameApiToGame(g)})
	n.logger.Println("Sending game to all connected clients")
	return nil
}

func (n *Socket) sendNotification(message string) {
	n.sendSocketEvent(Event{Name: "notification", Content: message})
	n.logger.Printf("Sending notification: %v", message)
}

func (n *Socket) sendSocketEvent(event Event) {
	n.logger.Printf("Sending message to channel %+v", event)
	n.sendChannel <- event
}

func (n *Socket) resetGame(ctx context.Context) error {
	c, err := n.connectToRandomServer()
	if err != nil {
		return err
	}

	_, err = c.Reset(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	n.logger.Println("DistributedGame has been reset.")

	err = n.sendGameState(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (n *Socket) connectToRandomServer() (api.GameServiceClient, error) {
	var servers []string
	var err error
	if len(n.backendAddrs) > 0 {
		servers = n.backendAddrs
	} else {
		servers, err = getAllServerAddresses()
		if err != nil {
			return nil, err
		}
	}

	for _, s := range servers {
		c, err := client.New(s)
		if err != nil {
			continue
		}

		return c, nil
	}

	return nil, errors.New("no backend servers found")
}

func getAllServerAddresses() ([]string, error) {
	dir := path.Join(os.TempDir(), "dhangkanna", "serverlist.db")
	db, err := bolt.Open(dir, 0600, nil)
	if err != nil {
		return make([]string, 0), err
	}
	defer func(db *bolt.DB) {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
	}(db)

	var addresses []string

	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("ServerAddresses"))
		if bucket == nil {
			return nil
		}
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			log.Printf("Found server with key/value - %s : %s\n", k, v)
			addresses = append(addresses, string(k))
		}
		return nil
	})

	return addresses, err
}
