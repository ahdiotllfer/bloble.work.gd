package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"server/game"
	"server/network"
	"strconv"
)

var PORT string

// Handler to return player count
func playerCountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	// Respond with the current player count
	response := map[string]int{"player_count": len(game.State.Players)}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func serverRebootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Parse the query parameters
	query := r.URL.Query()
	minutesLeftStr := query.Get("minutesLeft")
	if minutesLeftStr == "" {
		http.Error(w, "Missing 'minutesLeft' query parameter", http.StatusBadRequest)
		return
	}

	// Convert minutesLeft to an integer
	minutesLeft, err := strconv.Atoi(minutesLeftStr)
	if err != nil || minutesLeft <= 0 {
		http.Error(w, "Invalid 'minutesLeft' query parameter", http.StatusBadRequest)
		return
	}

	// Broadcast reboot alert
	network.BroadcastRebootAlert(byte(minutesLeft))

	network.SERVER_REBOOTING = true
}

// CORS Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow requests from specific origins
		if origin == "https://blubber.run.place" || origin == "http://localhost" || origin == "http://127.0.0.1" || origin == "http://localhost:5502" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			
		} else {
			log.Printf("cors middleware err", origin)
		}

		// Handle OPTIONS requests
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Pass the request to the next handler
		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
	// Log the client's IP address
	clientIP := r.Header.Get("X-Real-IP")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	return clientIP
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
       origine := r.Header.Get("Origin")
       // Allow only requests from "https://blubber.run.place"
       if origine != "https://blubber.run.place" {
       		http.Error(w, "Forbidden", http.StatusForbidden)
			log.Printf("origin forbidden", origine)
         	return
       }

	var userData network.UserData

	userData.ClientIP = getClientIP(r)
	log.Printf(getClientIP(r))

	// Pass the request to the WebSocket handler
	network.WsEndpoint(w, r, userData)
}

func main() {
	// Get the port from the environment variable
	PORT = os.Getenv("PORT")
	if PORT == "" {
		PORT = "8080" // Default port
		log.Printf("Port not specified. Defaulting to port %s\n", PORT)
	}

	game.Start()

	// Define WebSocket endpoint handlers with session checks
	http.HandleFunc("/", wsEndpoint)
	http.HandleFunc("/ffa1", wsEndpoint)
	http.HandleFunc("/ffa2", wsEndpoint)

	http.HandleFunc("/playercount", playerCountHandler)
	http.HandleFunc("/reboot", serverRebootHandler)

	// Log server start
	address := fmt.Sprintf("localhost:%s", PORT)
	log.Printf("Blobl.io Server starting on %s\n", address)

	// Start the server
	if err := http.ListenAndServe("localhost:"+PORT, nil); err != nil {
		log.Fatal(err)
	}
}
