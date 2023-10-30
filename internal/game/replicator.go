package game

import (
	"context"
	api "github.com/khatibomar/dhangkanna/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"

	"log"
	"os"
)

type Replicator struct {
	DialOptions      []grpc.DialOption
	LocalServer      api.GameServiceClient
	UpdateSocketChan chan struct{}
	logger           *log.Logger
	mu               sync.Mutex
	servers          map[string]struct{}
	closed           bool
	close            chan struct{}
}

func (r *Replicator) Join(name, addr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()

	if r.closed {
		r.logger.Printf("Join request ignored for %s. Replicator is closed.", name)
		return nil
	}

	if _, ok := r.servers[name]; ok {
		r.logger.Printf("Server %s is already joined. Ignoring.", name)
		return nil
	}

	r.servers[name] = struct{}{}

	r.replicate(addr)

	return nil
}

func (r *Replicator) Leave(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.init()
	if _, ok := r.servers[name]; !ok {
		r.logger.Printf("Server %s is not in the list. Ignoring leave request.", name)
		return nil
	}
	delete(r.servers, name)
	r.logger.Printf("Server %s left the cluster.", name)
	return nil
}

func (r *Replicator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()

	if r.closed {
		r.logger.Printf("Replicator is already closed. Ignoring close request.")
		return nil
	}

	r.closed = true
	close(r.close)
	r.logger.Println("Replicator is closed.")
	return nil
}

func (r *Replicator) replicate(addr string) {
	cc, err := grpc.Dial(addr, r.DialOptions...)
	if err != nil {
		r.logError(err, "failed to dial", addr)
		return
	}
	defer func() {
		if err := cc.Close(); err != nil {
			r.logger.Printf("Error while closing rpc connection: %v", err)
		}
	}()

	client := api.NewGameServiceClient(cc)
	ctx := context.Background()
	s, err := client.Receive(ctx, &emptypb.Empty{})
	if err != nil {
		r.logError(err, "failed to receive", addr)
		return
	}

	r.logger.Printf("received %+v from %s\n", s, addr)

	_, err = r.LocalServer.Send(ctx, s)
	if err != nil {
		r.logError(err, "failed to send DistributedGame to local server", addr)
	}
	r.UpdateSocketChan <- struct{}{}
}

func (r *Replicator) init() {
	if r.logger == nil {
		r.logger = log.New(os.Stdout, "replicator: ", log.LstdFlags)
	}
	if r.servers == nil {
		r.servers = make(map[string]struct{})
	}
	if r.close == nil {
		r.close = make(chan struct{})
	}
}

func (r *Replicator) logError(err error, msg, addr string) {
	r.logger.Printf(
		"Error: %v, Message: %s, RPC Address: %s",
		err,
		msg,
		addr,
	)
}
