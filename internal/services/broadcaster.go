package services

type Broadcaster interface {
	BroadcastGameUpdate(gameID string, multiplier float64)
	BroadcastGameCrash(gameID string, crashPoint float64)
}
