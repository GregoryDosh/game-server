package moose

import (
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	hi "github.com/GregoryDosh/game-server/hub/hubinterfaces"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// Game Defaults and Statuses
const (
	StatusCreated   = "Created"
	StatusStarted   = "Started"
	StatusFinished  = "Finished"
	DefaultGameName = "Untitled"
)

// PlayerSecretMoose holds additional information about a player
type PlayerSecretMoose struct {
	IsMoose          bool
	IsReady          bool
	IsFirstPresident bool
	LobbyPlayer      hi.PlayerInterface
}

// GameSecretMoose holds data necessary for a game of Secret Moose
type GameSecretMoose struct {
	GameName        string               `json:"name"`
	GameStatus      string               `json:"status"`
	GameTimeCreated time.Time            `json:"created"`
	Players         []*PlayerSecretMoose `json:"players"`
	Fascists        []*PlayerSecretMoose `json:"fascists"`
	Liberals        []*PlayerSecretMoose `json:"liberals"`
	FirstPresident  *PlayerSecretMoose   `json:"first_president"`
	Moose           *PlayerSecretMoose   `json:"moose"`
	checkAutostart  chan bool
}

// PartialGame holds the structure for marshaling output of an incomplete game.
type PartialGame struct {
	Name    string            `json:"name"`
	Status  string            `json:"status"`
	Players []json.RawMessage `json:"players"`
	Created time.Time         `json:"created"`
}

// FinishedGame holds the structure for marshaling output of a finished game.
type FinishedGame struct {
	Name           string            `json:"name"`
	Status         string            `json:"status"`
	Players        []json.RawMessage `json:"players"`
	Created        time.Time         `json:"created"`
	Liberals       []json.RawMessage `json:"liberals"`
	Fascists       []json.RawMessage `json:"fascists"`
	FirstPresident json.RawMessage   `json:"first_president"`
	Moose          json.RawMessage   `json:"moose"`
}

// MarshalJSON should hide information based on the status of the game
func (g *GameSecretMoose) MarshalJSON() ([]byte, error) {
	p := make([]json.RawMessage, 0)
	for _, ep := range g.Players {
		lp, err := json.Marshal(ep.LobbyPlayer)
		if err != nil {
			return nil, err
		}
		p = append(p, lp)
	}
	if g.Status() == StatusFinished {
		l, f := make([]json.RawMessage, 0), make([]json.RawMessage, 0)
		for _, ep := range g.Fascists {
			fasp, err := json.Marshal(ep.LobbyPlayer)
			if err != nil {
				return nil, err
			}
			f = append(f, fasp)
		}
		for _, ep := range g.Liberals {
			libp, err := json.Marshal(ep.LobbyPlayer)
			if err != nil {
				return nil, err
			}
			l = append(l, libp)
		}
		fp, err := json.Marshal(g.FirstPresident.LobbyPlayer)
		if err != nil {
			return nil, err
		}
		m, err := json.Marshal(g.Moose.LobbyPlayer)
		if err != nil {
			return nil, err
		}
		return json.Marshal(&FinishedGame{
			Name:           g.GameName,
			Status:         g.Status(),
			Players:        p,
			Fascists:       f,
			Liberals:       l,
			FirstPresident: fp,
			Moose:          m,
		})
	}
	return json.Marshal(&PartialGame{
		Name:    g.GameName,
		Status:  g.Status(),
		Players: p,
	})
}

// Name will return the game name
func (g *GameSecretMoose) Name() string {
	if g.GameName == "" {
		return DefaultGameName
	}
	return g.GameName
}

// Status will return the game name
func (g *GameSecretMoose) Status() string {
	if g.GameStatus == "" {
		return StatusCreated
	}
	return g.GameStatus
}

// StartGame will handle all of the pieces required to start a Secret Moose game
func (g *GameSecretMoose) StartGame() error {
	if len(g.Players) < 5 {
		return errors.New("not enough players to start")
	} else if len(g.Players) > 10 {
		return errors.New("too many players to start")
	}
	for _, p := range g.Players {
		if !p.IsReady {
			return errors.New("players not ready to start")
		}
	}
	var totalFascists int
	switch len(g.Players) {
	case 5, 6:
		totalFascists = 2
	case 7, 8:
		totalFascists = 3
	case 9, 10:
		totalFascists = 4
	}
	for a, b := range rand.Perm(len(g.Players)) {
		g.Players[a], g.Players[b] = g.Players[b], g.Players[a]
	}
	g.Fascists = append(g.Fascists, g.Players[:totalFascists]...)
	g.Liberals = append(g.Liberals, g.Players[totalFascists:]...)
	g.Moose = g.Players[0]

	// Since fascists were stacked at the beginning, shuffle them throughout
	for a, b := range rand.Perm(len(g.Players)) {
		g.Players[a], g.Players[b] = g.Players[b], g.Players[a]
	}
	g.FirstPresident = g.Players[0]
	g.GameStatus = StatusStarted
	return nil
}

// EndGame will handle all of the pieces required to end a Secret Moose game
func (g *GameSecretMoose) EndGame() error {
	g.GameStatus = StatusFinished
	if g.checkAutostart != nil {
		close(g.checkAutostart)
	}
	return nil
}

// AddPlayer will handle all of the pieces required to add a player to a Secret Moose game
func (g *GameSecretMoose) AddPlayer(p hi.PlayerInterface) (interface{}, error) {
	if len(g.Players) >= 10 {
		return nil, errors.New("cannot add player as game is full")
	}
	sp := &PlayerSecretMoose{
		LobbyPlayer: p,
	}
	if g.Status() != StatusCreated {
		return nil, errors.New("cannot add player after game started")
	}
	g.Players = append(g.Players, sp)
	return sp, nil
}

// RemovePlayer will handle all of the pieces required to remove a player from a Secret Moose game
func (g *GameSecretMoose) RemovePlayer(p hi.PlayerInterface) error {
	if g.Status() != StatusCreated {
		return errors.New("cannot remove player after game started")
	}
	for i, ep := range g.Players {
		if ep.LobbyPlayer == p {
			g.Players = append(g.Players[:i], g.Players[i+1:]...)
			return nil
		}
	}
	return errors.New("could not remove unknown player")
}

// AutoStart will handle starting the game when all players are ready
func (g *GameSecretMoose) AutoStart() {
	if g.checkAutostart == nil {
		g.checkAutostart = make(chan bool, 0)
	}
	for {
		select {
		case _, ok := <-g.checkAutostart:
			if !ok {
				return
			}
			if err := g.StartGame(); err == nil {
				return
			}
		}
	}
}

// PlayerEvent will handle player events in the game
func (g *GameSecretMoose) PlayerEvent(p hi.PlayerInterface, e *hi.MessageFromPlayer) error {
	validPlayer := false
	var gamePlayer *PlayerSecretMoose
	for _, ep := range g.Players {
		if ep.LobbyPlayer == p {
			validPlayer = true
			gamePlayer = ep
			break
		}
	}
	if !validPlayer || p == nil {
		return errors.New("PlayerEvent require an active player")
	}
	switch e.Type {
	case "ToggleReady":
		gamePlayer.IsReady = !gamePlayer.IsReady
		select {
		case g.checkAutostart <- true:
		case <-time.After(25 * time.Millisecond):
			errors.New("checkAutostart timeout ToggleReady")
		}
	}
	return nil
}
