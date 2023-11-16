package game

import (
	"github.com/hashicorp/raft"
	"log"
)

var _ raft.FSMSnapshot = (*fsmSnapshot)(nil)

type fsmSnapshot struct {
	data []byte
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	log.Println("Taking a snapshot...")
	_, err := sink.Write(s.data)
	if err != nil {
		if err2 := sink.Cancel(); err2 != nil {
			return err2
		}
		return err
	}

	if err := sink.Close(); err != nil {
		return err
	}
	log.Println("Taking snapshot done, with no errors")
	return nil
}

func (s *fsmSnapshot) Release() {
}
