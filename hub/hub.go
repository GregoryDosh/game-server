package hub

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/GregoryDosh/game-server/hub/hubevents"
	"github.com/GregoryDosh/game-server/hub/hubinterfaces"
	log "github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

type hub struct {
	Games map[string]hubinterfaces.GameInterface
	lobby map[hubinterfaces.PlayerInterface]bool
}

func (h *hub) AddGame(g hubinterfaces.GameInterface) (string, error) {
	if g == nil {
		return "", errors.New("invalid game created")
	}
	u := uuid.Must(uuid.NewV4()).String()
	h.Games[u] = g
	go g.AutoStart()
	h.UpdateGamelist()
	log.Debugf("game '%s' created", u)
	return u, nil
}

func (h *hub) RemoveGame(u string) error {
	if u == "" {
		return errors.New("UUID empty")
	}
	g, ok := h.Games[u]
	if ok {
		g.EndGame()
		delete(h.Games, u)
		h.UpdateGamelist()
		log.Debugf("game '%s' deleted", u)
		return nil
	}
	return fmt.Errorf("could not find game with UUID '%s'", u)
}

func (h *hub) UpdateGamelist() {
	games, err := json.Marshal(h.Games)
	if err != nil {
		log.Error(err)
		return
	}
	for p := range h.lobby {
		p.MessagePlayer(&hubevents.MessageToPlayer{
			Type:    "GAME_LIST",
			Message: string(games),
		})
	}
	log.Debugf("send GAME_LIST to %d players", len(h.lobby))
}

// New will initialize a new hub with required params and sane defaults
func New() *hub {
	return &hub{
		Games: make(map[string]hubinterfaces.GameInterface, 0),
		lobby: make(map[hubinterfaces.PlayerInterface]bool, 0),
	}
}
