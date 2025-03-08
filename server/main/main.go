package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"server/game"
	"server/network"
	"strconv"
	"time"
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
         	return
       }

	var userData network.UserData

	userData.ClientIP = getClientIP(r)
	log.Printf(getClientIP(r))

	// Try to retrieve the refresh token from cookies
	refreshTokenCookie, err := r.Cookie("refreshToken")
	if err != nil {
		// If the refresh token cookie is not set, proceed directly to WebSocket handler
		//log.Println("Refresh token not set, proceeding without it.")
		network.WsEndpoint(w, r, userData)
		return
	}

	// If the refresh token is set, handle it
	refreshToken := refreshTokenCookie.Value
	//log.Printf("Received refresh token: %s", refreshToken)

	// Fetch user data with the refresh token
	body := map[string]string{"refreshToken": refreshToken}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Printf("Failed to marshal JSON body: %v", err)
		http.Error(w, "Failed to retrieve user data", http.StatusInternalServerError)
		return
	}

	// Create a new POST request
	req, err := http.NewRequest(http.MethodPost, "https://auth.blobl.io/api/user", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		http.Error(w, "Failed to retrieve user data", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	origin := fmt.Sprintf("http://127.0.0.1:%s", PORT)
	req.Header.Set("Origin", origin)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send request to user API: %v", err)
		http.Error(w, "Failed to retrieve user data", http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid or expired refresh token: %s", resp.Status)
		http.Error(w, "Invalid or expired refresh token.", http.StatusForbidden)
		return
	}

	if err := json.NewDecoder(resp.Body).Decode(&userData); err != nil {
		log.Printf("Failed to parse user data: %v", err)
		http.Error(w, "Failed to parse user data", http.StatusInternalServerError)
		return
	}

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
	http.Handle("/", corsMiddleware(http.HandlerFunc(wsEndpoint)))
	http.Handle("/ffa1", corsMiddleware(http.HandlerFunc(wsEndpoint)))
	http.Handle("/ffa2", corsMiddleware(http.HandlerFunc(wsEndpoint)))

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
