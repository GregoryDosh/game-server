package hi

import (
	"encoding/json"
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
)

// PlayerInterface defines the interface for a Player
type PlayerInterface interface {
	MessageToPlayer(...*MessageToPlayer) error
	MessageToPlayerHandler()
	MessageFromPlayerAggregator()
	MessageFromPlayerHandler(*websocket.Conn)
	TotalSessions() int
	AddSession(*websocket.Conn) error
	DisconnectSession(*websocket.Conn) error
}

// LobbyPlayer is a generic player in the lobby.
type LobbyPlayer struct {
	Name               string
	MessagesToPlayer   chan *MessageToPlayer
	MessagesFromPlayer chan []byte
	Sessions           map[*websocket.Conn]bool
}

// MessageToPlayer will take a pointer to messages and place them on the Messages channel
func (p *LobbyPlayer) MessageToPlayer(msgs ...*MessageToPlayer) error {
	for _, m := range msgs {
		if m.EventChannel == "" {
			return errors.New("missing EventChannel on MessageToPlayer")
		}
		p.MessagesToPlayer <- m
	}
	return nil
}

// MessageToPlayerHandler should be run as a separate goroutine and handle pulling messages off of the Message channel and sending it to every session a user is part of.
func (p *LobbyPlayer) MessageToPlayerHandler() {
	log.Debugf("✅ Starting MessageToPlayerHandler for '%s'", p.Name)
	defer log.Debugf("🛑 Stopping MessageToPlayerHandler for '%s'", p.Name)
	for {
		select {
		case msg, ok := <-p.MessagesToPlayer:
			if !ok {
				return
			}
			binaryMessage, _ := json.Marshal(msg)
			log.Printf("Sending %s", binaryMessage)
			for s := range p.Sessions {
				s.WriteMessage(websocket.TextMessage, binaryMessage)
			}
		}
	}
}

// MessageFromPlayerAggregator should be run as a separate goroutine and handle pulling messages off of the Message channel and sending it to every session a user is part of.
func (p *LobbyPlayer) MessageFromPlayerAggregator() {
	log.Debugf("✅ Starting MessageFromPlayerAggregator for '%s'", p.Name)
	defer log.Debugf("🛑 Stopping MessageFromPlayerAggregator for '%s'", p.Name)
	for {
		select {
		case msg, ok := <-p.MessagesFromPlayer:
			if !ok {
				return
			}
			log.Printf("Received %s", msg)
			p.MessageToPlayer(&MessageToPlayer{
				Type:         string(msg),
				EventChannel: ChannelGlobal,
			})
		}
	}
}

// MessageFromPlayerHandler should be run as a separate goroutine and will pool messages from connection into MessageFromPlayerAggregator.
func (p *LobbyPlayer) MessageFromPlayerHandler(ws *websocket.Conn) {
	log.Debugf("✅ Starting MessageFromPlayerHandler for '%s'", p.Name)
	defer log.Debugf("🛑 Stopping MessageFromPlayerHandler for '%s'", p.Name)
	if ws.UnderlyingConn() != nil {
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error(err)
				}
				break
			}
			p.MessagesFromPlayer <- message
		}
	}
}

// TotalSessions returns the number of sessions a user has open
func (p *LobbyPlayer) TotalSessions() int {
	return len(p.Sessions)
}

// AddSession will add a new websocket.Conn to the list of active sessions
func (p *LobbyPlayer) AddSession(ws *websocket.Conn) error {
	if _, ok := p.Sessions[ws]; ok {
		return errors.New("websocket already in sessions")
	}
	p.Sessions[ws] = true
	// Since this is a new session, spawn new thread to handle reading messages.
	go p.MessageFromPlayerHandler(ws)
	return nil
}

// DisconnectSession will remove a websocket.Conn from the list of active sessions
func (p *LobbyPlayer) DisconnectSession(ws *websocket.Conn) error {
	if len(p.Sessions) == 0 {
		return errors.New("websocket not in user sessions")
	}
	if _, ok := p.Sessions[ws]; ok {
		delete(p.Sessions, ws)
		if ws.UnderlyingConn() != nil {
			if err := ws.Close(); err != nil {
				return err
			}
		}
		return nil
	}
	return errors.New("websocket not in user sessions")
}
