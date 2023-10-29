package state

import (
	"context"
	api "github.com/khatibomar/dhangkanna/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"os"
	"sync"
)

type Replicator struct {
	DialOptions []grpc.DialOption
	LocalServer api.StateServiceClient
	logger      *log.Logger
	mu          sync.Mutex
	servers     map[string]chan struct{}
	closed      bool
	close       chan struct{}
}

func (r *Replicator) Join(name, addr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()

	if r.closed {
		return nil
	}

	if _, ok := r.servers[name]; ok {
		return nil
	}

	r.servers[name] = make(chan struct{})

	r.replicate(addr, r.servers[name])

	return nil
}

func (r *Replicator) Leave(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.init()
	if _, ok := r.servers[name]; !ok {
		return nil
	}
	close(r.servers[name])
	delete(r.servers, name)
	return nil
}

func (r *Replicator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()

	if r.closed {
		return nil
	}

	r.closed = true
	close(r.close)

	return nil
}

func (r *Replicator) replicate(addr string, _ chan struct{}) {
	cc, err := grpc.Dial(addr, r.DialOptions...)
	if err != nil {
		r.logError(err, "failed to dial", addr)
		return
	}
	defer func() {
		if err := cc.Close(); err != nil {
			r.logger.Printf("error while closing rpc connection: %v\n", err)
		}
	}()

	client := api.NewStateServiceClient(cc)
	ctx := context.Background()
	s, _ := client.Receive(ctx, &emptypb.Empty{})
	if err != nil {
		r.logError(err, "failed to receive", addr)
		return
	}

	_, err = r.LocalServer.Send(ctx, s)
	if err != nil {
		r.logger.Printf("failed to send state to local server: %v\n", err)
	}
}

func (r *Replicator) init() {
	if r.logger == nil {
		r.logger = log.New(os.Stdout, "Replicator", log.LstdFlags)
	}
	if r.servers == nil {
		r.servers = make(map[string]chan struct{})
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
