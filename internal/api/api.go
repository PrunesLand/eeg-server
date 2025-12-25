package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/PrunesLand/eeg-server.git/internal/settings"
)

// StartServer starts the HTTP API server in a background goroutine.
func StartServer(s *settings.Settings) {
	mux := http.NewServeMux()

	// GET/POST /api/gain
	mux.HandleFunc("/api/gain", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetGain(w, r, s)
		case http.MethodPost:
			handleSetGain(w, r, s)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("🌍 API Server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("API Server failed: %v", err)
	}
}

func handleGetGain(w http.ResponseWriter, r *http.Request, s *settings.Settings) {
	response := map[string]float64{
		"gain": s.GetGain(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleSetGain(w http.ResponseWriter, r *http.Request, s *settings.Settings) {
	var body struct {
		Gain float64 `json:"gain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Basic validation (gain shouldn't be 0 to avoid division by zero)
	if body.Gain == 0 {
		http.Error(w, "Gain cannot be 0", http.StatusBadRequest)
		return
	}

	s.SetGain(body.Gain)
	log.Printf("🎛️ Gain updated to: %.2f", body.Gain)

	response := map[string]interface{}{
		"status": "success",
		"gain":   s.GetGain(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
