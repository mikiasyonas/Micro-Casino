package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"sample-miniapp-backend/internal/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler struct {
	gameEngine   *services.GameEngine
	redisService *services.RedisService
	hub          *WebSocketHub
}

type WebSocketHub struct {
	clients    map[int64]*websocket.Conn
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
}

type Client struct {
	UserID int64
	Conn   *websocket.Conn
}

type Message struct {
	Type   string      `json:"type"`
	UserID int64       `json:"user_id,omitempty"`
	GameID string      `json:"game_id,omitempty"`
	Data   interface{} `json:"data"`
}

func NewWebSocketHandler(gameEngine *services.GameEngine, redisService *services.RedisService) *WebSocketHandler {
	hub := &WebSocketHub{
		clients:    make(map[int64]*websocket.Conn),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 100),
	}

	go hub.run()

	return &WebSocketHandler{
		gameEngine:   gameEngine,
		redisService: redisService,
		hub:          hub,
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	userID := c.GetInt64("user_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	client := &Client{
		UserID: userID,
		Conn:   conn,
	}

	h.hub.register <- client

	defer func() {
		h.hub.unregister <- client
		conn.Close()
	}()

	h.sendBalance(client)

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		h.handleMessage(client, &msg)
	}
}

func (h *WebSocketHandler) handleMessage(client *Client, msg *Message) {
	switch msg.Type {
	case "PING":
		h.sendPong(client)
	case "SUBSCRIBE_GAME":
		// Subscribe to game updates
		if gameID, ok := msg.Data.(string); ok {
			h.subscribeToGame(client, gameID)
		}
	case "UNSUBSCRIBE_GAME":
		// Unsubscribe from game updates
		if gameID, ok := msg.Data.(string); ok {
			h.unsubscribeFromGame(client, gameID)
		}
	}
}

func (h *WebSocketHandler) sendBalance(client *Client) {
	wallet, err := h.redisService.GetWallet(client.UserID)
	if err != nil {
		log.Printf("Failed to get wallet for WS: %v", err)
		return
	}

	msg := Message{
		Type: "BALANCE_UPDATE",
		Data: gin.H{
			"balance":       wallet.Balance,
			"locked":        wallet.LockedBalance,
			"available":     wallet.Balance - wallet.LockedBalance,
			"total_wagered": wallet.TotalWagered,
			"total_won":     wallet.TotalWon,
		},
	}

	client.Conn.WriteJSON(msg)
}

func (h *WebSocketHandler) sendPong(client *Client) {
	msg := Message{
		Type: "PONG",
		Data: gin.H{
			"timestamp": time.Now().Unix(),
		},
	}

	client.Conn.WriteJSON(msg)
}

func (h *WebSocketHandler) subscribeToGame(client *Client, gameID string) {
	// Subscribe logic here
	// You'll want to track which clients are watching which games
}

func (h *WebSocketHandler) unsubscribeFromGame(client *Client, gameID string) {
	// Unsubscribe logic here
}

func (hub *WebSocketHub) run() {
	for {
		select {
		case client := <-hub.register:
			hub.clients[client.UserID] = client.Conn
			log.Printf("Client registered: %d", client.UserID)

		case client := <-hub.unregister:
			if _, ok := hub.clients[client.UserID]; ok {
				delete(hub.clients, client.UserID)
				log.Printf("Client unregistered: %d", client.UserID)
			}

		case message := <-hub.broadcast:
			hub.broadcastMessage(message)
		}
	}
}

func (hub *WebSocketHub) broadcastMessage(message *Message) {
	if message.UserID != 0 {
		if conn, ok := hub.clients[message.UserID]; ok {
			conn.WriteJSON(message)
		}
	} else {
		for _, conn := range hub.clients {
			conn.WriteJSON(message)
		}
	}
}

func (h *WebSocketHandler) BroadcastGameUpdate(gameID string, multiplier float64) {
	msg := &Message{
		Type:   "GAME_UPDATE",
		GameID: gameID,
		Data: gin.H{
			"game_id":    gameID,
			"multiplier": multiplier,
			"timestamp":  time.Now().Unix(),
		},
	}

	h.hub.broadcast <- msg
}

func (h *WebSocketHandler) BroadcastGameCrash(gameID string, crashPoint float64) {
	msg := &Message{
		Type:   "GAME_CRASH",
		GameID: gameID,
		Data: gin.H{
			"game_id":     gameID,
			"crash_point": crashPoint,
			"timestamp":   time.Now().Unix(),
		},
	}

	h.hub.broadcast <- msg
}
