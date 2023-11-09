package game

import (
	"github.com/hashicorp/raft"
	api "github.com/khatibomar/dhangkanna/cmd/api/v1"
	"google.golang.org/protobuf/proto"
	"io"
	"sync"
)

var _ raft.FSM = (*fsm)(nil)

type fsm struct {
	game *Game
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	g := &api.Game{
		GuessedCharacter: f.game.GuessedCharacter,
		IncorrectGuesses: f.game.IncorrectGuesses,
		ChancesLeft:      int32(f.game.ChancesLeft),
		GameState:        int32(f.game.GameState),
		Message:          f.game.Message,
		Version:          int32(f.game.Version),
	}
	snapshotData, err := proto.Marshal(g)
	if err != nil {
		return nil, err
	}

	return &fsmSnapshot{data: snapshotData}, nil
}

func (f *fsm) Restore(snapshot io.ReadCloser) error {
	data, err := io.ReadAll(snapshot)
	if err != nil {
		return err
	}

	var gameSnapshot api.Game
	err = proto.Unmarshal(data, &gameSnapshot)
	if err != nil {
		return err
	}

	f.game = &Game{
		GuessedCharacter: gameSnapshot.GuessedCharacter,
		IncorrectGuesses: gameSnapshot.IncorrectGuesses,
		ChancesLeft:      int(gameSnapshot.ChancesLeft),
		GameState:        int8(gameSnapshot.GameState),
		Message:          gameSnapshot.Message,
		Version:          int(gameSnapshot.Version),
		mu:               &sync.Mutex{},
	}

	return nil
}

func (f *fsm) Apply(record *raft.Log) interface{} {
	var req api.Game
	err := proto.Unmarshal(record.Data, &req)
	if err != nil {
		return err
	}
	if req.IncorrectGuesses == nil {
		req.IncorrectGuesses = make([]string, 0)
	}

	f.game.Update(
		req.GuessedCharacter,
		req.IncorrectGuesses,
		int(req.ChancesLeft),
		int8(req.GameState),
		req.Message,
		int(req.Version),
	)
	return nil
}
