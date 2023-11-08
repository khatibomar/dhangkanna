package discovery

import (
	"github.com/hashicorp/serf/serf"
	"log"
	"net"
	"os"
)

type Discovery struct {
	Config
	handler Handler
	serf    *serf.Serf
	events  chan serf.Event
	logger  *log.Logger
}

type Config struct {
	NodeName            string
	BindAddr            string
	Tags                map[string]string
	StartJoinsAddresses []string
}

type Handler interface {
	Join(name, addr string) error
	Leave(name string) error
}

func New(handler Handler, config Config) (*Discovery, error) {
	d := &Discovery{
		Config:  config,
		handler: handler,
		logger:  log.New(os.Stdout, "discovery: ", log.LstdFlags|log.Lshortfile),
	}

	if err := d.setup(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Discovery) setup() error {
	addr, err := net.ResolveTCPAddr("tcp", d.BindAddr)
	if err != nil {
		return err
	}
	d.events = make(chan serf.Event)

	config := serf.DefaultConfig()
	config.Init()
	config.MemberlistConfig.BindAddr = addr.IP.String()
	config.MemberlistConfig.BindPort = addr.Port
	config.EventCh = d.events
	config.Tags = d.Config.Tags
	config.NodeName = d.Config.NodeName

	d.serf, err = serf.Create(config)
	if err != nil {
		return err
	}

	go d.handleSerfEvents()

	if d.StartJoinsAddresses != nil {
		if _, err = d.serf.Join(d.StartJoinsAddresses, true); err != nil {
			return err
		}
	}

	return nil
}

func (d *Discovery) handleSerfEvents() {
	for e := range d.events {
		d.logger.Printf("received serf event : %+v", e.EventType())
		switch e.EventType() {
		case serf.EventMemberJoin:
			for _, member := range e.(serf.MemberEvent).Members {
				if d.isLocal(member) {
					continue
				}
				d.handleJoin(member)
			}
			break
		case serf.EventMemberLeave, serf.EventMemberFailed:
			for _, member := range e.(serf.MemberEvent).Members {
				d.handleLeave(member)
			}
			break
		}
	}
}

func (d *Discovery) handleJoin(member serf.Member) {
	if err := d.handler.Join(member.Name, member.Tags["rpc_addr"]); err != nil {
		d.logError(err, "failed to join", member)
	} else {
		d.logger.Printf("Joined: Name=%s, RPC Address=%s", member.Name, member.Tags["rpc_addr"])
	}
}

func (d *Discovery) handleLeave(member serf.Member) {
	if err := d.handler.Leave(member.Name); err != nil {
		d.logError(err, "failed to leave", member)
		return
	}

	d.logger.Printf("Left: Name=%s, RPC Address=%s", member.Name, member.Tags["rpc_addr"])
}

func (d *Discovery) isLocal(member serf.Member) bool {
	return d.serf.LocalMember().Name == member.Name
}

func (d *Discovery) Members() []serf.Member {
	return d.serf.Members()
}

func (d *Discovery) Leave() error {
	return d.serf.Leave()
}

func (d *Discovery) logError(err error, msg string, member serf.Member) {
	d.logger.Printf(
		"Error: %v, Message: %s, Name: %s, RPC Address: %s",
		err,
		msg,
		member.Name,
		member.Tags["rpc_addr"],
	)
}
