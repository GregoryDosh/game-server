package games

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/GregoryDosh/game-server/pkg/gsinterfaces"
	uuid "github.com/satori/go.uuid"
)

type moose struct {
	name         string
	id           string
	eventHandler func(playeruuid string, gameuuid string, j json.RawMessage)
}

func (m *moose) AddPlayer(playeruuid string, player gsinterfaces.User) error {
	fmt.Printf("Game %s added player %s\n", m.id, playeruuid)
	return nil
}

func (m *moose) Event(player string, j json.RawMessage) {
	fmt.Printf("Game event %s from %s\n", j, player)
}

func (m *moose) StartGameLoop() {
	for i := 0; i < 5000; i++ {
		fmt.Println(i)
		time.Sleep(time.Second)
	}
}

func NewMoose() *moose {
	u := uuid.Must(uuid.NewV4()).String()
	return &moose{
		name: "Unnamed",
		id:   u,
	}
}
