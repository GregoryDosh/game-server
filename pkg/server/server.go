package server

import (
	"encoding/json"
	"sync"

	"github.com/GregoryDosh/game-server/pkg/channels"
	"github.com/GregoryDosh/game-server/pkg/gsinterfaces"
	ws "github.com/GregoryDosh/game-server/pkg/websocket"
	log "github.com/Sirupsen/logrus"
)

type server struct {
	umtx  sync.RWMutex
	users map[string]gsinterfaces.User
	gmtx  sync.RWMutex
	games map[string]gsinterfaces.Game
}

func New() gsinterfaces.Server {
	return &server{
		users: make(map[string]gsinterfaces.User),
		games: make(map[string]gsinterfaces.Game),
	}
}

func (s *server) GetUser(uuid string) gsinterfaces.User {
	s.umtx.RLock()
	u, ok := s.users[uuid]
	s.umtx.RUnlock()
	if !ok {
		nu := ws.NewUser(uuid)
		nu.SetFromHandler(s.EventHandler)
		s.umtx.Lock()
		s.users[uuid] = nu
		s.umtx.Unlock()
		return nu
	}
	return u
}

type generalEventFromPlayer struct {
	EventType string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type generalEventToPlayer struct {
	EventType string `json:"type"`
	Payload   string `json:"payload"`
}

type deferredPayload struct {
	Payload string `json:"payload"`
}

func (s *server) EventHandler(playeruuid string, b []byte) {
	log.Debugf("Received from '%s' this message: %s", playeruuid, b)
	u := s.GetUser(playeruuid)
	e := &generalEventFromPlayer{}
	err := json.Unmarshal(b, &e)
	if err != nil {
		log.Error(err)
	}
	switch e.EventType {
	case channels.Global:
		s.globalEventHandler(u, e.Payload)
	case channels.Player:
		s.playerEventHandler(u, e.Payload)
	case channels.Game:
		s.gameEventHandler(u, e.Payload)
	default:
		log.Errorf("unknown event from '%s': '%s'", playeruuid, e.EventType)
	}
}
