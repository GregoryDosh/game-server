package gsinterfaces

type Server interface {
	GetUser(uuid string) User
	EventHandler(uuid string, b []byte)
}

type User interface {
	SetFromHandler(func(playeruuid string, b []byte))
	AddConnection(params ...interface{}) error
	RemoveConnection(params ...interface{}) error
	SendEvent(b []byte)
	Name() string
	ID() string
}

type Game interface {
	// SetEventHandler(func(playeruuid string, gameuuid string, j json.RawMessage))
	// AddPlayer(string, user.Interface) error
	// RemovePlayer(string) error
	// Event(string, json.RawMessage)
	// StartGameLoop()
	// TerminateGame()
	// ID() string
	// String() string
}
