package server

import (
	"fmt"

	"github.com/GregoryDosh/game-server/pkg/event"
	"github.com/GregoryDosh/game-server/pkg/games"
	"github.com/GregoryDosh/game-server/pkg/gsinterfaces"
	log "github.com/Sirupsen/logrus"
)

// validatePayloadKeys takes the event to check and a variable amount of keys to check for
// ex validatePayloadKeys(e, "id", "name", "gameid")
func validatePayloadKeys(e *event.General, keys ...string) error {
	for _, k := range keys {
		if _, ok := e.Payload[k]; !ok {
			return fmt.Errorf("'%s' missing from payload keys", k)
		}
	}
	return nil
}

func (s *server) broadcastHandler(u gsinterfaces.User, e *event.General) error {
	log.Debugf("Sending broadcast to all users %s", e)

	if err := validatePayloadKeys(e, "message"); err != nil {
		return err
	}
	message, _ := e.Payload["message"]
	fromUser := u.Name()
	if m, ok := message.(string); ok {
		s.umtx.RLock()
		for _, bu := range s.users {
			bu.SendData(event.WrapValues("GLOBAL_BROADCAST", map[string]interface{}{
				"from":    fromUser,
				"message": m,
			}))
		}
		s.umtx.RUnlock()
	}
	return nil
}

func (s *server) createGameHandler(u gsinterfaces.User, e *event.General) error {
	log.Debugf("'%s' - '%s' creating a new game of type '%s'", u.ID(), u.Name(), e)
	if err := validatePayloadKeys(e, "type"); err != nil {
		return err
	}
	gameType, _ := e.Payload["type"]
	if gt, ok := gameType.(string); ok {
		switch gt {
		case "MOOSE":
			ng := games.NewMoose("")
			s.gmtx.Lock()
			s.games[ng.ID()] = ng
			ng.SetFromGameHandler(s.eventFromGameHandler)
			s.gmtx.Unlock()
			u.SendData(event.WrapValues("GAME_CREATED", map[string]interface{}{
				"id":   ng.ID(),
				"name": ng.Name(),
			}))
		default:
			return fmt.Errorf("Unknown game type '%s'", e.Payload)
		}
	}
	return nil
}

func (s *server) changeUsernameHandler(u gsinterfaces.User, e *event.General) error {
	log.Debugf("Trying to change username of '%s' from '%s' to '%s'", u.ID(), u.Name(), e)
	if err := validatePayloadKeys(e, "name"); err != nil {
		return err
	}
	newName, _ := e.Payload["name"]
	if name, ok := newName.(string); ok {
		if err := u.SetName(name); err == nil {
			u.SendData(event.WrapValue("USERNAME_CHANGED", "new_username", name))
			return nil
		}
	}
	return fmt.Errorf("Invalid username '%v'", newName)
}

func (s *server) gameEventHandler(u gsinterfaces.User, e *event.General) error {
	log.Debugf("'%s' - '%s' sending game event of '%s'", u.ID(), u.Name(), e)
	if err := validatePayloadKeys(e, "id"); err != nil {
		return err
	}
	gameID, _ := e.Payload["id"]
	if id, ok := gameID.(string); ok {
		if g, ok := s.games[id]; ok {
			g.FromUserHandler(u.ID(), e.Payload)
			return nil
		}
		return fmt.Errorf("gameID '%s' does not exist", id)
	}
	return nil
}
