package hi

import (
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
)

// PlayerInterface defines the interface for a Player
type PlayerInterface interface {
	MessageToPlayer(...*MessageToPlayer) error
	MessageHandler()
	TotalSessions() int
	AddSession(*websocket.Conn) error
	DisconnectSession(*websocket.Conn) error
}

// LobbyPlayer is a generic player in the lobby.
type LobbyPlayer struct {
	Name               string
	MessagesToPlayer   chan *MessageToPlayer
	Sessions           []*websocket.Conn
	stopMessageHandler chan bool
}

// MessageToPlayer will take a pointer to messages and place them on the Messages channel
func (p *LobbyPlayer) MessageToPlayer(msgs ...*MessageToPlayer) error {
	for _, m := range msgs {
		p.MessagesToPlayer <- m
	}
	return nil
}

// MessageHandler should be run as a separate goroutine and handle pulling messages off of the Message channel and sending it to every session a user is part of. To quit it send a message on the stopMessageHandler channel.
func (p *LobbyPlayer) MessageHandler() {
	if p.stopMessageHandler == nil {
		p.stopMessageHandler = make(chan bool)
	}
	for {
		select {
		case msg := <-p.MessagesToPlayer:
			fmt.Print(msg)
		case <-p.stopMessageHandler:
			return
		}
	}
}

// TotalSessions returns the number of sessions a user has open
func (p *LobbyPlayer) TotalSessions() int {
	return len(p.Sessions)
}

// AddSession will add a new websocket.Conn to the list of active sessions
func (p *LobbyPlayer) AddSession(ws *websocket.Conn) error {
	for _, s := range p.Sessions {
		if s == ws {
			return errors.New("websocket already in sessions")
		}
	}
	p.Sessions = append(p.Sessions, ws)
	return nil
}

// DisconnectSession will remove a websocket.Conn from the list of active sessions
func (p *LobbyPlayer) DisconnectSession(ws *websocket.Conn) error {
	if len(p.Sessions) == 0 {
		return errors.New("websocket not in user sessions")
	}
	for i, s := range p.Sessions {
		if s == ws {
			p.Sessions = append(p.Sessions[:i], p.Sessions[i+1:]...)
			return nil
		}
	}
	return errors.New("websocket not in user sessions")
}
