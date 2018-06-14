package hub

import (
	"encoding/json"
	"errors"
	"fmt"

	hi "github.com/GregoryDosh/game-server/hub/hubinterfaces"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

// Hub holds the data structures for the methods below.  Use New instead of calling this directly.
type Hub struct {
	// Map UUID for GameIDs to a specific GameInterface
	games map[string]hi.GameInterface
	// Map UUID for PlayerIDs to a PlayerInterface
	lobby map[string]hi.PlayerInterface
}

// AddGame takes a GameInterface, places it in the games map with a UUID, and then calls UpdateGameList
func (h *Hub) AddGame(g hi.GameInterface) (string, error) {
	if g == nil {
		return "", errors.New("invalid game created")
	}
	u := uuid.Must(uuid.NewV4()).String()
	h.games[u] = g
	go g.AutoStart()
	log.Debugf("game '%s' created", u)
	return u, h.UpdateGameList()
}

// RemoveGame takes a UUID string and removes it from the game map, then calls UpdateGameList.
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
		log.Debugf("game '%s' deleted", u)
		return h.UpdateGameList()
	}
	return fmt.Errorf("could not find game with UUID '%s'", u)
}

// ConnectSession will take a UUID and either create a PlayerInterface and place it in the lobby, or return the existing PlayerInterface from the lobby.
func (h *Hub) ConnectSession(u string, ws *websocket.Conn) (hi.PlayerInterface, error) {
	if ws == nil {
		return nil, errors.New("missing websocket connection")
	}
	if s, ok := h.lobby[u]; ok {
		return s, s.AddSession(ws)
	}
	s := &hi.LobbyPlayer{
		Name:               "",
		MessagesToPlayer:   make(chan *hi.MessageToPlayer),
		MessagesFromPlayer: make(chan []byte, 1024),
		Sessions:           make(map[*websocket.Conn]bool),
	}
	h.lobby[u] = s
	// Since this is a new player, spawn new thread to handle sending messages.
	go s.MessageToPlayerHandler()
	go s.MessageFromPlayerAggregator()
	return s, s.AddSession(ws)
}

// DisconnectSession will take a UUID and a websocket and remove it from the lobby.  If this is the last remaining websocket it will remove the player from the lobby.
func (h *Hub) DisconnectSession(u string, ws *websocket.Conn) error {
	if ws == nil {
		return errors.New("cannot disconnect nil websocket")
	}
	if s, ok := h.lobby[u]; ok {
		return s.DisconnectSession(ws)
	}
	return fmt.Errorf("player with uuid '%s' not in lobby", u)
}

// UpdateGameList will marshall the games into JSON and send it to all players in the lobby with the Type GAME_LIST.
func (h *Hub) UpdateGameList() error {
	games, err := json.Marshal(h.games)
	if err != nil {
		return err
	}
	for _, p := range h.lobby {
		err := p.MessageToPlayer(&hi.MessageToPlayer{
			Type:         "GAME_LIST",
			EventChannel: hi.ChannelGlobal,
			Message:      string(games),
		})
		if err != nil {
			return err
		}
	}
	log.Debugf("send GAME_LIST to %d players", len(h.lobby))
	return nil
}

// New will initialize a new hub with required params and sane defaults
func New() *Hub {
	return &Hub{
		games: make(map[string]hi.GameInterface, 0),
		lobby: make(map[string]hi.PlayerInterface, 0),
	}
}
