package server

import (
	"context"
	api "github.com/khatibomar/dhangkanna/api/v1"
	"github.com/khatibomar/dhangkanna/internal/game"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"os"
)

var _ api.GameServiceServer = (*grpcServer)(nil)

type Config struct {
	Game *game.Game
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

func (s *grpcServer) Send(_ context.Context, state *api.Game) (*emptypb.Empty, error) {
	s.logger.Println("Received new state with version %d ==? old version %d", state.Version, s.Game.Version)
	if int(state.Version) > s.Game.Version {
		if state.GuessedCharacter == nil {
			state.GuessedCharacter = make([]string, 0)
		}

		if state.IncorrectGuesses == nil {
			state.IncorrectGuesses = make([]string, 0)
		}
		s.Config.Game.Update(
			state.GuessedCharacter,
			state.IncorrectGuesses,
			int(state.ChancesLeft),
			int8(state.GameState),
			state.Message,
			int(state.Version),
		)
	}

	return &emptypb.Empty{}, nil
}

func (s *grpcServer) Receive(_ context.Context, _ *emptypb.Empty) (*api.Game, error) {
	st := &api.Game{
		GuessedCharacter: s.Game.GuessedCharacter,
		IncorrectGuesses: s.Game.IncorrectGuesses,
		ChancesLeft:      int32(s.Game.ChancesLeft),
		GameState:        int32(s.Game.GameState),
		Message:          s.Game.Message,
		Version:          int32(s.Game.Version),
	}

	if st.GuessedCharacter == nil {
		st.GuessedCharacter = make([]string, 0)
	}

	if st.IncorrectGuesses == nil {
		st.IncorrectGuesses = make([]string, 0)
	}

	return st, nil
}
