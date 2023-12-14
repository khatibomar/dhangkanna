package server

import (
	"context"
	api "github.com/khatibomar/dhangkanna/cmd/api/v1"
	"github.com/khatibomar/dhangkanna/internal/game"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"os"
	"time"
)

var _ api.GameServiceServer = (*grpcServer)(nil)

type Config struct {
	Game        *game.DistributedGame
	GetServerer GetServerer
}

type grpcServer struct {
	api.UnimplementedGameServiceServer
	*Config
	logger *log.Logger
}

func newGrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
		logger: log.New(os.Stdout, "grpc server: ", log.LstdFlags),
	}
	return srv, nil
}

func NewGRPCServer(config *Config, grpcOpts ...grpc.ServerOption) (
	*grpc.Server,
	error,
) {
	gsrv := grpc.NewServer(grpcOpts...)
	srv, err := newGrpcServer(config)
	if err != nil {
		return nil, err
	}
	api.RegisterGameServiceServer(gsrv, srv)
	return gsrv, nil
}

func (s *grpcServer) Send(_ context.Context, letter *api.Letter) (*emptypb.Empty, error) {
	s.logger.Printf("Received new letter %s", letter)
	s.Game.HandleNewLetter(letter.Letter)

	g := game.Game{
		GuessedCharacter: s.Game.GuessedCharacter,
		IncorrectGuesses: s.Game.IncorrectGuesses,
		ChancesLeft:      s.Game.ChancesLeft,
		GameState:        s.Game.GameState,
		Message:          s.Game.Message,
		Version:          s.Game.Version,
	}

	b, err := proto.Marshal(game.ConvertGameToGameApi(g))
	if err != nil {
		return &emptypb.Empty{}, err
	}
	s.Game.Raft.Apply(b, 5*time.Second)
	return &emptypb.Empty{}, nil
}

func (s *grpcServer) Receive(_ context.Context, _ *emptypb.Empty) (*api.Game, error) {
	s.logger.Println("this server handling reading game state")
	st := &api.Game{
		GuessedCharacter: s.Game.GuessedCharacter,
		IncorrectGuesses: s.Game.IncorrectGuesses,
		ChancesLeft:      int32(s.Game.ChancesLeft),
		GameState:        int32(s.Game.GameState),
		Message:          s.Game.Message,
		Version:          int32(s.Game.Version),
	}

	return st, nil
}

func (s *grpcServer) Reset(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	s.logger.Println("Reset received")
	s.Game.Reset()
	s.logger.Println("Reset completed")
	return &emptypb.Empty{}, nil
}

func (s *grpcServer) GetServers(_ context.Context, _ *emptypb.Empty) (*api.GetServersResponse, error) {
	servers, err := s.GetServerer.GetServers()
	if err != nil {
		return nil, err
	}
	return &api.GetServersResponse{Servers: servers}, nil
}

type GetServerer interface {
	GetServers() ([]*api.Server, error)
}
