package loadbalance

import (
	"context"
	"fmt"
	api "github.com/khatibomar/dhangkanna/cmd/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
	"google.golang.org/protobuf/types/known/emptypb"
	"os"

	"log"
	"sync"
)

const Name = "dhangkanna"

type Resolver struct {
	mu            sync.Mutex
	clientConn    resolver.ClientConn
	resolverConn  *grpc.ClientConn
	serviceConfig *serviceconfig.ParseResult
	logger        *log.Logger
}

var _ resolver.Builder = (*Resolver)(nil)

func (r *Resolver) Build(
	target resolver.Target,
	cc resolver.ClientConn,
	opts resolver.BuildOptions,
) (resolver.Resolver, error) {
	r.logger = log.New(os.Stdout, "resolver: ", log.LstdFlags|log.Lshortfile)
	r.clientConn = cc
	var dialOpts []grpc.DialOption
	dialOpts = append(
		dialOpts,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if opts.DialCreds != nil {
		dialOpts = append(
			dialOpts,
			grpc.WithTransportCredentials(opts.DialCreds),
		)
	}
	r.serviceConfig = r.clientConn.ParseServiceConfig(
		fmt.Sprintf(`{"loadBalancingConfig":[{"%s":{}}]}`, Name),
	)
	var err error
	r.resolverConn, err = grpc.Dial(target.Endpoint(), dialOpts...)
	if err != nil {
		return nil, err
	}
	r.ResolveNow(resolver.ResolveNowOptions{})
	return r, nil
}

func (r *Resolver) Scheme() string {
	return Name
}

func init() {
	resolver.Register(&Resolver{})
}

var _ resolver.Resolver = (*Resolver)(nil)

func (r *Resolver) ResolveNow(resolver.ResolveNowOptions) {
	r.mu.Lock()
	defer r.mu.Unlock()
	client := api.NewGameServiceClient(r.resolverConn)
	// get cluster and then set on cc attributes
	ctx := context.Background()
	res, err := client.GetServers(ctx, &emptypb.Empty{})
	if err != nil {
		r.logger.Printf("failed to resolve server: %v\n", err)
		return
	}
	var addrs []resolver.Address
	for _, server := range res.Servers {
		addrs = append(addrs, resolver.Address{
			Addr: server.RpcAddr,
			Attributes: attributes.New(
				"is_leader",
				server.IsLeader,
			),
		})
	}
	r.clientConn.UpdateState(resolver.State{
		Addresses:     addrs,
		ServiceConfig: r.serviceConfig,
	})
}

func (r *Resolver) Close() {
	if err := r.resolverConn.Close(); err != nil {
		r.logger.Printf("failed to close conn: %v\n", err)
	}
}
