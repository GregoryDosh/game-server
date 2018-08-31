package websocket

import (
	"errors"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

type us struct {
	name           string
	id             string
	eventHandler   func(playeruuid string, b []byte)
	messagesToUser chan []byte
	badConnections chan *websocket.Conn
	connmtx        sync.RWMutex
	connections    map[*websocket.Conn]bool
}

func (u *us) ID() string   { return u.id }
func (u *us) Name() string { return u.name }

func (u *us) SendEvent(b []byte) {
	if u.messagesToUser != nil {
		select {
		case u.messagesToUser <- b:
		default:
			log.Warnf("lost message %s", b)
		}
	}
}

func (u *us) SetFromHandler(h func(playeruuid string, b []byte)) {
	if h != nil {
		u.eventHandler = h
	}
}

func (u *us) AddConnection(ps ...interface{}) error {
	if len(ps) != 1 {
		return errors.New("invalid number parameters for this type of user")
	}
	c, ok := ps[0].(*websocket.Conn)
	if !ok {
		return errors.New("wrong parameter for this type of user")
	}
	u.connmtx.RLock()
	if _, ok := u.connections[c]; ok {
		u.connmtx.RUnlock()
		return errors.New("connection already added")
	}
	u.connmtx.RUnlock()

	u.connmtx.Lock()
	u.connections[c] = true
	u.connmtx.Unlock()
	go u.messageFromUserHandler(c)
	return nil
}

func (u *us) RemoveConnection(ps ...interface{}) error {
	if len(ps) != 1 {
		return errors.New("invalid number parameters")
	}
	c, ok := ps[0].(*websocket.Conn)
	if !ok {
		return errors.New("wrong parameter for this type of user")
	}
	u.connmtx.RLock()
	if _, ok := u.connections[c]; ok {
		u.connmtx.RUnlock()
		return errors.New("connection already added")
	}
	u.badConnections <- c
	return nil
}

func (u *us) messageToUserHandler() {
	log.Debugf("➡️📪 started messageToUserHandler for %s %s", u.ID(), u.Name())
	defer log.Debugf("🛑 ➡️📪 started messageToUserHandler for %s %s", u.ID(), u.Name())
	pingTicker := time.NewTicker(5 * time.Second)
	defer func() {
		pingTicker.Stop()
	}()
	for {
		select {
		case msg := <-u.messagesToUser:
			u.connmtx.RLock()
			for c := range u.connections {
				c.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Error(err)
				}
			}
			log.Debugf("📪➡️😀 successfully sent %s", msg)
			u.connmtx.RUnlock()
			if msg == nil {
				return
			}
		case <-pingTicker.C:
			u.connmtx.RLock()
			for c := range u.connections {
				c.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Errorf("Something ? %s", err)
					// Assume client disconnected and add them to the badConnections queue to be cleaned up
					u.badConnections <- c
				}
			}
			u.connmtx.RUnlock()
		}
	}
}

func (u *us) badConnectionHandler() {
	log.Debugf("📪➡️ started badConnectionHandler for %s %s", u.ID(), u.Name())
	defer log.Debugf("🛑 📪➡️ stopped badConnectionHandler for %s %s", u.ID(), u.Name())
	for {
		c := <-u.badConnections
		log.Debugf("closing connection for %s %s", u.ID(), u.Name())
		u.connmtx.Lock()
		delete(u.connections, c)
		u.connmtx.Unlock()
		c.Close()
	}
}

func (u *us) messageFromUserHandler(c *websocket.Conn) {
	log.Debugf("📪➡️ started messageFromUserHandler for %s %s", u.ID(), u.Name())
	defer log.Debugf("🛑 📪➡️ stopped messageFromUserHandler for %s %s", u.ID(), u.Name())
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Errorf("Totes: %s", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error(err)
			}
			u.badConnections <- c
			return
		}
		u.eventHandler(u.ID(), msg)
	}
}

// These aren't part of the normal user interface.  They extend the functionality.
// Since we're hooking into this system via websockets, it has some other considerations.
func (u *us) simulateInternalEvent(b []byte) {
	if u.eventHandler != nil {
		u.eventHandler(u.id, b)
	}
}

func NewUser(id string) *us {
	if id == "" {
		id = uuid.Must(uuid.NewV4()).String()
	}
	u := &us{
		name:           "Unknown",
		id:             id,
		eventHandler:   nil,
		messagesToUser: make(chan []byte, 5),
		connections:    make(map[*websocket.Conn]bool, 0),
		badConnections: make(chan *websocket.Conn, 5),
	}
	go u.messageToUserHandler()
	go u.badConnectionHandler()
	return u
}
