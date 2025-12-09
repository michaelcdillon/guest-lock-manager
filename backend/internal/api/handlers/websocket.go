package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	ws "github.com/guest-lock-manager/backend/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from Home Assistant ingress
		return true
	},
}

// WebSocketUpgrade returns a handler that upgrades HTTP connections to WebSocket.
func WebSocketUpgrade(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		client := ws.NewClient(hub)
		hub.Register(client)

		// Start read and write pumps
		go writePump(conn, client)
		go readPump(conn, client, hub)
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
func writePump(conn *websocket.Conn, client *ws.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send():
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the WebSocket connection to the hub.
func readPump(conn *websocket.Conn, client *ws.Client, hub *ws.Hub) {
	defer func() {
		hub.Unregister(client)
		conn.Close()
	}()

	conn.SetReadLimit(65536)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Handle client commands (subscribe, ping, etc.)
		handleClientMessage(conn, message, client, hub)
	}
}

// handleClientMessage processes incoming client messages.
func handleClientMessage(conn *websocket.Conn, message []byte, client *ws.Client, hub *ws.Hub) {
	// For now, just acknowledge receipt
	// TODO: Implement subscribe/unsubscribe commands
	log.Printf("Received WebSocket message: %s", message)
}


