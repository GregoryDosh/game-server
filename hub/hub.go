package hub

import (
	"encoding/json"
	"errors"
	"fmt"

	hi "github.com/GregoryDosh/game-server/hub/hubinterfaces"
	log "github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

// Hub holds the data structures for the methods below.  Use New instead of calling this directly.
type Hub struct {
	// Map UUID for GameIDs to a specific GameInterface
	games map[string]hi.GameInterface
	// Map UUID for PlayerIDs to a PlayerInterface
	lobby map[string]hi.PlayerInterface
}

func (h *Hub) AddGame(g hi.GameInterface) (string, error) {
	if g == nil {
		return "", errors.New("invalid game created")
	}
	u := uuid.Must(uuid.NewV4()).String()
	h.games[u] = g
	go g.AutoStart()
	h.UpdateGamelist()
	log.Debugf("game '%s' created", u)
	return u, nil
}

func (h *Hub) RemoveGame(u string) error {
	if u == "" {
		return errors.New("UUID empty")
	}
	g, ok := h.games[u]
	if ok {
		err := g.EndGame()
		if err != nil {
			log.Error(err)
		}
		delete(h.games, u)
		h.UpdateGamelist()
		log.Debugf("game '%s' deleted", u)
		return nil
	}
	return fmt.Errorf("could not find game with UUID '%s'", u)
}

func (h *Hub) UpdateGamelist() {
	games, err := json.Marshal(h.games)
	if err != nil {
		log.Error(err)
		return
	}
	for _, p := range h.lobby {
		err := p.MessagePlayer(&hi.MessageToPlayer{
			Type:    "GAME_LIST",
			Message: string(games),
		})
		if err != nil {
			log.Error(err)
		}
	}
	log.Debugf("send GAME_LIST to %d players", len(h.lobby))
}

func (h *Hub) ConnectSession(u string) hi.PlayerInterface {
	if s, ok := h.lobby[u]; ok {
		return s
	}
	s := &hi.LobbyPlayer{}
	h.lobby[u] = s
	return s
}

// New will initialize a new hub with required params and sane defaults
func New() *Hub {
	return &Hub{
		games: make(map[string]hi.GameInterface, 0),
		lobby: make(map[string]hi.PlayerInterface, 0),
	}
}
