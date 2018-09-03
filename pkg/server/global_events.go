package server

import (
	"encoding/json"

	"github.com/GregoryDosh/game-server/pkg/gsinterfaces"
	log "github.com/Sirupsen/logrus"
)

func (s *server) globalEventHandler(u gsinterfaces.User, j json.RawMessage) {
	log.Debugf("Received global event from '%s' '%s': '%s'", u.ID(), u.Name(), j)
	e := &generalEventFromPlayer{}
	err := json.Unmarshal(j, &e)
	if err != nil {
		log.Error(err)
	}

	switch e.EventType {
	case "BROADCAST":
		log.Debugf("Sending broadcast to all players %s", e.Payload)
		s.umtx.RLock()
		for _, u := range s.users {
			u.SendEvent(e.Payload)
		}
		s.umtx.RUnlock()
	}
}
