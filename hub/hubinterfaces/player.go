package hi

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
)

// PlayerInterface defines the interface for a Player
type PlayerInterface interface {
	AddSession(*websocket.Conn) error
	DisconnectSession(*websocket.Conn) error
	SessionHandler(time.Duration)
	TotalSessions() int
	MessageToPlayer(...*MessageToPlayer) error
	MessageFromPlayerHandler(*websocket.Conn)
}

// LobbyPlayer is a generic player in the lobby.
type LobbyPlayer struct {
	Name                string
	MessagesToPlayer    chan *MessageToPlayer
	MessagesFromPlayer  chan []byte
	AddWSChannel        chan *websocket.Conn
	DisconnectWSChannel chan *websocket.Conn
	StopRoutines        chan bool
	sessions            map[*websocket.Conn]bool
	lenSessions         int
}

// AddSession will place the websocket.Conn on the AddWSChannel channel for the SessionsHandler to manage
func (p *LobbyPlayer) AddSession(ws *websocket.Conn) error {
	if p.AddWSChannel == nil {
		return errors.New("AddWSChannel cannot be nil")
	}
	log.Debugf("adding session to '%s'", p.Name)
	go func() {
		p.MessageFromPlayerHandler(ws)
		p.DisconnectSession(ws)
	}()
	select {
	case p.AddWSChannel <- ws:
	case <-time.After(250 * time.Millisecond):
		return fmt.Errorf("could not add session for '%s' as channel was blocked", p.Name)
	}
	return nil
}

// DisconnectSession place the websocket.Conn on the DisconnectWSChannel channel for the SessionsHandler to manage
func (p *LobbyPlayer) DisconnectSession(ws *websocket.Conn) error {
	if p.DisconnectWSChannel == nil {
		return errors.New("DisconnectWSChannel cannot be nil")
	}
	log.Debugf("disconnecting session of '%s'", p.Name)
	select {
	case p.DisconnectWSChannel <- ws:
	case <-time.After(250 * time.Millisecond):
		return fmt.Errorf("could not disconnect session for '%s' as channel was blocked", p.Name)
	}
	return nil
}

// SessionHandler will handle the adding/disconnecting of websocket sessions, and the message sending in a hopefully concurrent safe manner
func (p *LobbyPlayer) SessionHandler(pingPeriod time.Duration) {
	log.Debugf("✅ Starting SessionHandler for '%s'", p.Name)
	defer log.Debugf("🛑 Stopping SessionHandler for '%s'", p.Name)
	if p.sessions == nil {
		p.sessions = make(map[*websocket.Conn]bool)
	}
	if p.StopRoutines == nil {
		p.StopRoutines = make(chan bool)
	}
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		// Handle adding sessions atomically
		case ws := <-p.AddWSChannel:
			p.sessions[ws] = true
			p.lenSessions = len(p.sessions)
		// Handle removing sessions atomically
		case ws := <-p.DisconnectWSChannel:
			delete(p.sessions, ws)
			p.lenSessions = len(p.sessions)
		// Handle messages to all the sessions
		case msg := <-p.MessagesToPlayer:
			binaryMessage, _ := json.Marshal(msg)
			log.Printf("Sending '%s' to %d sessions", binaryMessage, p.TotalSessions())
			for s := range p.sessions {
				s.WriteMessage(websocket.TextMessage, binaryMessage)
			}
		// Let the clients know the server is still alive
		case <-ticker.C:
			for s := range p.sessions {
				if s.UnderlyingConn() != nil {
					s.SetWriteDeadline(time.Now().Add(4 * time.Second))
					if err := s.WriteMessage(websocket.PingMessage, nil); err != nil {
						p.DisconnectWSChannel <- s
					}
				}
			}
		// Handle shutting down the handler
		case <-p.StopRoutines:
			return
		}
	}
}

// TotalSessions returns the number of sessions a user has open
func (p *LobbyPlayer) TotalSessions() int {
	return p.lenSessions
}

// MessageToPlayer will take a pointer to messages and place them on the Messages channel
func (p *LobbyPlayer) MessageToPlayer(msgs ...*MessageToPlayer) error {
	if p.MessagesToPlayer == nil {
		return errors.New("missing MessagesToPlayer channel on player")
	}
	for _, m := range msgs {
		if m.EventChannel == "" {
			return errors.New("missing EventChannel on MessageToPlayer")
		}
		select {
		case p.MessagesToPlayer <- m:
		case <-time.After(250 * time.Millisecond):
			return errors.New("timeout sending message(s)")
		}
	}
	return nil
}

// MessageFromPlayerHandler should be run as a separate goroutine and will pool messages from connection into MessageFromPlayerAggregator.
func (p *LobbyPlayer) MessageFromPlayerHandler(ws *websocket.Conn) {
	log.Debugf("✅ Starting MessageFromPlayerHandler for '%s'", p.Name)
	// log.Debugf("✅ Starting MessageFromPlayerHandler for '%s': total ws '%d'", p.Name, p.TotalSessions())
	defer log.Debugf("🛑 Stopping MessageFromPlayerHandler for '%s'", p.Name)
	if ws.UnderlyingConn() != nil {
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error(err)
				}
				p.DisconnectWSChannel <- ws
				break
			}
			log.Debugf("received '%s'", message)
			p.MessagesFromPlayer <- message
			err = p.MessageToPlayer(&MessageToPlayer{
				Type:         "Echo",
				EventChannel: ChannelGlobal,
				Message:      string(message),
			})
			if err != nil {
				log.Error(err)
			}
		}
	}
}
