package channels

const (
	// Global events are server global like new game, purge chat, etc.
	Global = "GLOBAL"
	// Player events are player specific like change username, appear offline, etc.
	Player = "PLAYER"
	// Game events are game specific like ready, start, finish, etc.
	Game = "GAME"
)
