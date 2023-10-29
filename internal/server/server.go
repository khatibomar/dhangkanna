package server

import (
	"context"
	api "github.com/khatibomar/dhangkanna/api/v1"
	"github.com/khatibomar/dhangkanna/internal/state"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ api.StateServiceServer = (*grpcServer)(nil)

type Config struct {
	State *state.State
}

type grpcServer struct {
	api.UnimplementedStateServiceServer
	*Config
}

func newGrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
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
	api.RegisterStateServiceServer(gsrv, srv)
	return gsrv, nil
}

func (s *grpcServer) Send(_ context.Context, state *api.State) (*emptypb.Empty, error) {
	if state.GuessedCharacter == nil {
		state.GuessedCharacter = make([]string, 0)
	}

	if state.IncorrectGuesses == nil {
		state.IncorrectGuesses = make([]string, 0)
	}

	if int(state.Version) > s.State.Version {
		s.Config.State.Update(
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

func (s *grpcServer) Receive(_ context.Context, _ *emptypb.Empty) (*api.State, error) {
	st := &api.State{
		GuessedCharacter: s.State.GuessedCharacter,
		IncorrectGuesses: s.State.IncorrectGuesses,
		ChancesLeft:      int32(s.State.ChancesLeft),
		GameState:        int32(s.State.GameState),
		Message:          s.State.Message,
		Version:          int32(s.State.Version),
	}

	if st.GuessedCharacter == nil {
		st.GuessedCharacter = make([]string, 0)
	}

	if st.IncorrectGuesses == nil {
		st.IncorrectGuesses = make([]string, 0)
	}

	return st, nil
}
