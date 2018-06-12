package hubevents

// MessageToPlayer holds the type required for games to pass around as a shared type object
type MessageToPlayer struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// PlayerEvent is an event generated by a player in a game to be fed into a game for processing.
type PlayerEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}