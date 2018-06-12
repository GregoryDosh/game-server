package game

import (
	events "github.com/GregoryDosh/game-server/hub/events"
)

// GameInterface holds the interface required for a Game to be served up by the server
type GameInterface interface {
	AddPlayer(PlayerInterface) (interface{}, error)
	RemovePlayer(PlayerInterface) error
	PlayerEvent(PlayerInterface, events.PlayerEvent) error
	Name() string
	Status() string
	StartGame() error
	EndGame() error
	AutoStart()
}

// PlayerInterface defines the interface for a Player
type PlayerInterface interface {
	MessagePlayer(...*events.MessageToPlayer) error
}

// LobbyPlayer is a generic player in the lobby
type LobbyPlayer struct {
	Name string
}

func (p *LobbyPlayer) MessagePlayer(...*events.MessageToPlayer) error {
	return nil
}
