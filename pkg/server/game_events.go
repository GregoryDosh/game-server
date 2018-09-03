package server

import (
	"encoding/json"

	"github.com/GregoryDosh/game-server/pkg/gsinterfaces"
	log "github.com/Sirupsen/logrus"
)

func (s *server) gameEventHandler(u gsinterfaces.User, j json.RawMessage) {
	log.Debugf("Received game event from '%s' '%s': '%s'", u.ID(), u.Name(), j)
	e := &generalEventFromPlayer{}
	err := json.Unmarshal(j, &e)
	if err != nil {
		log.Error(err)
	}
}
