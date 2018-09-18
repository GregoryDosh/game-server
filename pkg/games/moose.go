package games

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	namesgenerator "github.com/moby/moby/pkg/namesgenerator"
	uuid "github.com/satori/go.uuid"
)

type moose struct {
	fromGameHandler func(useruuid string, gameuuid string, e interface{})
	profilemtx      sync.RWMutex
	name            string
	id              string
	gameEvents      chan []byte
}

type gameError struct {
	UserID string `json:"gameid"`
	Error  string `json:"error"`
}

type readyToggled struct {
	UserID string `json:"userid"`
}

func (m *moose) ID() string {
	m.profilemtx.RLock()
	defer m.profilemtx.RUnlock()
	return m.id
}

func (m *moose) Name() string {
	m.profilemtx.RLock()
	defer m.profilemtx.RUnlock()
	return m.name
}

func (m *moose) SetFromGameHandler(h func(u string, g string, e interface{})) {
	if h != nil {
		m.fromGameHandler = h
	}
}

func (m *moose) FromUserHandler(u string, p map[string]interface{}) {
	log.Debugf("event from %s: %s", u, p)
	t, ok := p["type"]
	if !ok {
		m.fromGameHandler(u, "INVALID_EVENT", &gameError{
			Error: "type missing from keys",
		})
		return
	}
	log.Warn(t)
	switch t {
	case "TOGGLE_READY":
		m.fromGameHandler(u, "TOGGLED_READY", &readyToggled{
			UserID: u,
		})
	default:
		m.fromGameHandler(u, "UNKNOWN_EVENT", &gameError{
			Error: fmt.Sprintf("unknown type '%s'", t),
		})
	}
}

func (m *moose) StartGameLoop() {
	timeoutTicker := time.NewTicker(2 * time.Hour)
	for {
		select {
		case e := <-m.gameEvents:
			log.Errorf("Whoa, event!? %s", e)
		case <-timeoutTicker.C:
			log.Error("Game Timed Out")
			return
		}
	}
}

func (m *moose) Shutdown() {
	log.Warnf("Received shutdown notification in game %s", m.Name())
}

func NewMoose(name string) *moose {
	id := uuid.Must(uuid.NewV4()).String()
	if name == "" {
		genName := strings.Split(namesgenerator.GetRandomName(0), "_")
		name = fmt.Sprintf("%s %s", strings.Title(genName[0]), strings.Title(genName[1]))
	}
	g := &moose{
		name:       name,
		id:         id,
		gameEvents: make(chan []byte, 50),
	}
	go g.StartGameLoop()
	return g
}
