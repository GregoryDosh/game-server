package websocket

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GregoryDosh/game-server/pkg/event"
	namesgenerator "github.com/moby/moby/pkg/namesgenerator"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

type User struct {
	id             string
	eventHandler   func(userUUID string, b []byte)
	messagesToUser chan []byte
	badConnections chan *websocket.Conn
	connmtx        sync.RWMutex
	connections    map[*websocket.Conn]bool
	profilemtx     sync.RWMutex
	name           string
}

func (u *User) ID() string {
	u.profilemtx.RLock()
	defer u.profilemtx.RUnlock()
	return u.id
}

func (u *User) Name() string {
	u.profilemtx.RLock()
	defer u.profilemtx.RUnlock()
	return u.name
}

func (u *User) SetName(n string) error {
	if n != "" {
		u.profilemtx.Lock()
		u.name = n
		u.profilemtx.Unlock()
	} else {
		return errors.New("invalid username")
	}
	return nil
}

func (u *User) SendData(b []byte) {
	if u.messagesToUser != nil {
		select {
		case u.messagesToUser <- b:
		default:
			log.Warnf("lost message %s", b)
		}
	}
}

func (u *User) SetFromHandler(h func(userUUID string, b []byte)) {
	if h != nil {
		u.eventHandler = h
	}
}

func (u *User) AddConnection(ps ...interface{}) error {
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
	u.messagesToUser <- event.WrapValue("GREETING", "message", fmt.Sprintf("Hello %s", u.Name()))
	u.messagesToUser <- event.WrapValue("ANNOUNCEMENTS", "message", "Nothing new to report here.")
	return nil
}

func (u *User) RemoveConnection(ps ...interface{}) error {
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

func (u *User) Shutdown() {
	log.Warnf("Received shutdown notification for user %s", u.Name())
	u.connmtx.Lock()
	for c := range u.connections {
		c.Close()
	}
	u.connmtx.Unlock()
}

func (u *User) messageToUserHandler() {
	log.Debugf("➡️📪 started messageToUserHandler for %s", u.Name())
	defer log.Debugf("🛑 ➡️📪 started messageToUserHandler for %s", u.Name())
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
		case <-pingTicker.C:
			u.connmtx.RLock()
			for c := range u.connections {
				c.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
					// Assume client disconnected and add them to the badConnections queue to be cleaned up
					u.badConnections <- c
				}
			}
			u.connmtx.RUnlock()
		}
	}
}

func (u *User) badConnectionHandler() {
	log.Debugf("📪➡️ started badConnectionHandler for %s", u.Name())
	defer log.Debugf("🛑 📪➡️ stopped badConnectionHandler for %s", u.Name())
	for {
		c := <-u.badConnections
		log.Debugf("closing connection for %s", u.Name())
		u.connmtx.Lock()
		delete(u.connections, c)
		u.connmtx.Unlock()
		c.Close()
	}
}

func (u *User) messageFromUserHandler(c *websocket.Conn) {
	log.Debugf("📪➡️ started messageFromUserHandler for %s", u.Name())
	defer log.Debugf("🛑 📪➡️ stopped messageFromUserHandler for %s", u.Name())
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error(err)
			}
			u.badConnections <- c
			return
		}
		u.eventHandler(u.ID(), msg)
	}
}

func NewUser(id string, name string) *User {
	if id == "" {
		id = uuid.Must(uuid.NewV4()).String()
	}
	if name == "" {
		gen_name := strings.Split(namesgenerator.GetRandomName(0), "_")
		name = fmt.Sprintf("%s %s", strings.Title(gen_name[0]), strings.Title(gen_name[1]))
	}
	u := &User{
		name:           name,
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
