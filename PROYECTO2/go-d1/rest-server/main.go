// go-d1/rest-server/main.go
// Recibe el JSON de Rust y lo reenvía al gRPC Client (Container B del mismo Pod)
// Patrón REST de Clase 8
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// Estructura del mensaje (debe coincidir con Rust)
type Prediction struct {
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeGoals int32  `json:"home_goals"`
	AwayGoals int32  `json:"away_goals"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

type Response struct {
	Status string `json:"status"`
}

func predictionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Leer body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error leyendo body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Deserializar JSON (Clase 8 patrón)
	var pred Prediction
	if err := json.Unmarshal(body, &pred); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	log.Printf("[REST-SERVER] Predicción recibida: %s vs %s (%d-%d) por %s",
		pred.HomeTeam, pred.AwayTeam, pred.HomeGoals, pred.AwayGoals, pred.Username)

	// Reenviar al gRPC Client (Container B, mismo Pod → localhost)
	grpcClientURL := os.Getenv("GRPC_CLIENT_URL")
	if grpcClientURL == "" {
		grpcClientURL = "http://localhost:9000/send"
	}

	resp, err := http.Post(grpcClientURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[REST-SERVER] Error llamando al gRPC client: %v", err)
		// En Fase 1, esto es normal (Go D2 no existe aún)
		// No fallar: devolver OK de todas formas
	} else {
		defer resp.Body.Close()
		log.Printf("[REST-SERVER] gRPC client respondió: %d", resp.StatusCode)
	}

	// Responder al Rust API
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Status: "ok"})
}

func main() {
	http.HandleFunc("/", predictionHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("[REST-SERVER] Escuchando en :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
