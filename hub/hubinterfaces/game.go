package hi

// GameInterface holds the interface required for a Game to be served up by the server
type GameInterface interface {
	AddPlayer(PlayerInterface) (interface{}, error)
	RemovePlayer(PlayerInterface) error
	PlayerEvent(PlayerInterface, *MessageFromPlayer) error
	Name() string
	Status() string
	StartGame() error
	EndGame() error
	AutoStart()
}
