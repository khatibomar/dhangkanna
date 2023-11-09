package client

import (
	"fmt"
	api "github.com/khatibomar/dhangkanna/cmd/api/v1"
	"github.com/khatibomar/dhangkanna/internal/loadbalance"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func New(rpcAddr string) (api.GameServiceClient, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.Dial(fmt.Sprintf(
		"%s:///%s",
		loadbalance.Name,
		rpcAddr,
	), opts...)
	if err != nil {
		return nil, err
	}
	client := api.NewGameServiceClient(conn)
	return client, nil
}
