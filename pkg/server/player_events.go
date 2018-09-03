package server

import (
	"encoding/json"

	"github.com/GregoryDosh/game-server/pkg/gsinterfaces"
	log "github.com/Sirupsen/logrus"
)

func (s *server) playerEventHandler(u gsinterfaces.User, j json.RawMessage) {
	log.Debugf("Received player event from '%s' '%s': '%s'", u.ID(), u.Name(), j)
	e := &generalEventFromPlayer{}
	err := json.Unmarshal(j, &e)
	if err != nil {
		log.Error(err)
		return
	}

	switch e.EventType {
	case "CHANGE_USERNAME":
		name := &deferredPayload{}
		err = json.Unmarshal(j, name)
		if err != nil {
			log.Error(err)
			return
		}
		log.Debugf("Trying to change username of '%s' from '%s' to '%s'", u.ID(), u.Name(), name.Payload)
		err := u.SetName(name.Payload)
		if err != nil {
			errMsg, _ := json.Marshal(&generalEventToPlayer{
				EventType: "ERROR",
				Payload:   err.Error(),
			})
			u.SendEvent(errMsg)
			return
		}
		succMsg, _ := json.Marshal(&generalEventToPlayer{
			EventType: "USERNAME_CHANGED",
			Payload:   u.Name(),
		})
		u.SendEvent(succMsg)
	}
}
