package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/GregoryDosh/game-server/pkg/event"
	"github.com/GregoryDosh/game-server/pkg/gsinterfaces"
	ws "github.com/GregoryDosh/game-server/pkg/websocket"
	log "github.com/Sirupsen/logrus"
)

type server struct {
	umtx  sync.RWMutex
	users map[string]gsinterfaces.User
	gmtx  sync.RWMutex
	games map[string]gsinterfaces.Game
}

func New() gsinterfaces.Server {
	return &server{
		users: make(map[string]gsinterfaces.User),
		games: make(map[string]gsinterfaces.Game),
	}
}

func (s *server) GetUser(uuid, name string) gsinterfaces.User {
	s.umtx.RLock()
	u, ok := s.users[uuid]
	s.umtx.RUnlock()
	if !ok {
		nu := ws.NewUser(uuid, name)
		nu.SetFromHandler(s.eventFromUserHandler)
		s.umtx.Lock()
		s.users[uuid] = nu
		s.umtx.Unlock()
		return nu
	}
	return u
}

func (s *server) Shutdown(timeout int) {
	timeoutTicker := time.NewTicker(time.Duration(timeout) * time.Second)
	done := make(chan bool)
	go func(done chan bool) {
		s.umtx.RLock()
		for _, u := range s.users {
			u.Shutdown()
		}
		s.umtx.RUnlock()
		s.gmtx.RLock()
		for _, g := range s.games {
			g.Shutdown()
		}
		s.gmtx.RUnlock()
		close(done)
	}(done)
	for {
		select {
		case <-done:
			log.Info("closed all connections")
			return
		case <-timeoutTicker.C:
			log.Warn("not all connections closed before timeout")
			return
		}
	}
}

func (s *server) DebugAddUser(n string, u gsinterfaces.User) {
	s.umtx.Lock()
	s.users[n] = u
	s.umtx.Unlock()
}

func (s *server) eventFromUserHandler(userUUID string, b []byte) {
	u := s.GetUser(userUUID, "")
	log.Debugf("Received from '%s' this message: %s", u.Name(), b)
	e := &event.General{}
	if err := json.Unmarshal(b, &e); err != nil {
		log.Error(err)
	}
	log.Warn(e)
	switch e.Event {
	case "BROADCAST":
		if err := s.broadcastHandler(u, e); err != nil {
			u.SendData(event.WrapError(err))
		}
	case "CREATE_GAME":
		if err := s.createGameHandler(u, e); err != nil {
			u.SendData(event.WrapError(err))
		}
	case "GAME":
		if err := s.gameEventHandler(u, e); err != nil {
			u.SendData(event.WrapError(err))
		}
	case "CHANGE_USERNAME":
		if err := s.changeUsernameHandler(u, e); err != nil {
			u.SendData(event.WrapError(err))
		}
	default:
		m := fmt.Sprintf("unknown event from '%s': '%s'", userUUID, e.Event)
		log.Infof(m)
		u.SendData(event.WrapError(errors.New(m)))
	}
}

func (s *server) eventFromGameHandler(userUUID string, gameUUID string, e interface{}) {
	log.Debugf("game %s wants to send to %s: %s", gameUUID, userUUID, e)
	u := s.GetUser(userUUID, "")
	u.SendData(event.WrapValues("GAME_EVENT", map[string]interface{}{
		"id":            gameUUID,
		"event_details": e,
	}))
}
