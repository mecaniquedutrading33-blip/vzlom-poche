// Vzlom Poche — WebSocket relay & microservice proxy
// Relai WebSocket pour Agora <-> Bridge <-> Mobile
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

var (
	listenAddr  = flag.String("addr", ":3457", "Adresse d'écoute")
	bridgeURL   = flag.String("bridge", "http://localhost:3456", "URL du bridge Vzlom")
	upgrader    = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// Client représente une connexion WebSocket client
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	userID   string
}

// Hub gère les connexions WebSocket actives
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 1024),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("[POCHE] Client connecté: %s (total: %d)", client.userID, len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("[POCHE] Client déconnecté: %s (total: %d)", client.userID, len(h.clients))
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[POCHE] read error: %v", err)
			}
			break
		}
		// Relayer le message vers tous les autres clients
		hub.broadcast <- message
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for message := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			return
		}
	}
}

func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token requis", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[POCHE] upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: token[:8],
	}
	hub.register <- client

	go client.writePump()
	go client.readPump(hub)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprintf(w, `{"status":"ok","service":"vzlom-poche"}`)
}

func main() {
	flag.Parse()

	log.Printf("[POCHE] Démarrage sur %s → bridge: %s", *listenAddr, *bridgeURL)

	hub := newHub()
	go hub.run()

	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	})

	// Proxy HTTP vers le bridge
	http.HandleFunc("/proxy/", func(w http.ResponseWriter, r *http.Request) {
		// Strip /proxy prefix et forward vers bridge
		r.Header.Set("X-Forwarded-For", r.RemoteAddr)
		http.Redirect(w, r, *bridgeURL+"/"+r.URL.Path[7:], http.StatusMovedPermanently)
	})

	// Arrêt propre
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{Addr: *listenAddr}
	go func() {
		log.Printf("[POCHE] Serveur prêt sur %s", *listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[POCHE] Erreur serveur: %v", err)
		}
	}()

	<-stop
	log.Println("[POCHE] Arrêt en cours...")
	srv.Close()
}