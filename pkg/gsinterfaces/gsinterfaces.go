package gsinterfaces

type Server interface {
	GetUser(uuid string, name string) User
	Shutdown(timeout int)
	DebugAddUser(user User)
	DebugAddGame(game Game)
}

type User interface {
	SetFromHandler(func(userUUID string, b []byte))
	AddConnection(params ...interface{}) error
	RemoveConnection(params ...interface{}) error
	SendData(b []byte)
	SetName(n string) error
	Name() string
	ID() string
	Shutdown()
}

type Game interface {
	ID() string
	Name() string
	StartGameLoop()
	FromUserHandler(uuid string, payload map[string]interface{})
	SetFromGameHandler(func(userUUID string, gameuuid string, e interface{}))
	Shutdown()
}
