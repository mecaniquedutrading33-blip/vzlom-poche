// Package poche fournit les utilitaires de relai WebSocket pour Vzlom
package poche

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Config représente la configuration de la poche
type Config struct {
	Addr      string `json:"addr"`
	BridgeURL string `json:"bridge_url"`
	Token     string `json:"-"`
}

// Status représente l'état du service
type Status struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	StartedAt time.Time `json:"started_at"`
	Clients   int       `json:"clients"`
}

// Response est le format standard de réponse JSON
type Response struct {
	OK      bool        `json:"ok"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Version string      `json:"version"`
}

// WriteJSON envoie une réponse JSON propre
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteResponse formate et envoie une réponse standard
func WriteResponse(w http.ResponseWriter, status int, ok bool, data interface{}, errMsg string) {
	resp := Response{
		OK:      ok,
		Data:    data,
		Error:   errMsg,
		Version: "1.0.0",
	}
	WriteJSON(w, status, resp)
}

// HealthCheck retourne l'état du service
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, http.StatusOK, true, map[string]string{
		"service": "vzlom-poche",
		"uptime":  time.Since(startedAt).String(),
	}, "")
}

var startedAt = time.Now()

// LogRequest journalise une requête entrante (sans exposer les tokens)
func LogRequest(r *http.Request) {
	fmt.Printf("[%s] %s %s (from: %s)\n",
		time.Now().Format("15:04:05"),
		r.Method,
		r.URL.Path,
		r.RemoteAddr,
	)
}