package server

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/GregoryDosh/game-server/pkg/channels"
	"github.com/GregoryDosh/game-server/pkg/gsinterfaces"
	ws "github.com/GregoryDosh/game-server/pkg/websocket"
	log "github.com/Sirupsen/logrus"
)

type serverEx struct {
	sync.RWMutex
	users map[string]gsinterfaces.User
	games map[string]gsinterfaces.Game
}

func (s *serverEx) GetUser(uuid string) gsinterfaces.User {
	s.RLock()
	u, ok := s.users[uuid]
	s.RUnlock()
	if !ok {
		nu := ws.NewUser(uuid)
		nu.SetFromHandler(s.EventHandler)
		s.Lock()
		s.users[uuid] = nu
		s.Unlock()
		return nu
	}
	return u
}

type generalEvent struct {
	EventType string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

func (s *serverEx) EventHandler(playeruuid string, b []byte) {
	log.Debugf("Received from '%s' this message: %s", playeruuid, b)
	u := s.GetUser(playeruuid)
	u.SendEvent([]byte(fmt.Sprintf("You sent '%s'", b)))
	e := &generalEvent{}
	err := json.Unmarshal(b, &e)
	if err != nil {
		log.Error(err)
	}
	switch e.EventType {
	case channels.Global:
		log.Infof("Received global event from '%s': '%s'", playeruuid, e.Payload)
	case channels.Player:
		log.Infof("Received player event from '%s': '%s'", playeruuid, e.Payload)
	case channels.Game:
		log.Infof("Received game event from '%s': '%s'", playeruuid, e.Payload)
	default:
		log.Errorf("unknown event from '%s': '%s'", playeruuid, e.EventType)
	}
}

// func (s *serverEx) newGame(g gsinterfaces.Game) string {
// 	u := uuid.Must(uuid.NewV4()).String()
// 	s.Lock()
// 	s.games[u] = g
// 	s.Unlock()
// 	return u
// }

func New() gsinterfaces.Server {
	return &serverEx{
		users: make(map[string]gsinterfaces.User),
		games: make(map[string]gsinterfaces.Game),
	}
}
