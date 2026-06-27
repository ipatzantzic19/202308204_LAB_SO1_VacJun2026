// FASE 1: Stub — recibe de Container A y loguea.
// FASE 2: Se reemplazará por el cliente gRPC real hacia Go D2.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

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

func sendHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error leyendo body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var pred Prediction
	if err := json.Unmarshal(body, &pred); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	log.Printf("[GRPC-CLIENT][FASE1-STUB] Mensaje recibido de Container A:")
	log.Printf("  Local: %s | Visitante: %s", pred.HomeTeam, pred.AwayTeam)
	log.Printf("  Goles: %d - %d | Usuario: %s", pred.HomeGoals, pred.AwayGoals, pred.Username)
	log.Printf("  Timestamp: %s", pred.Timestamp)
	log.Printf("  → [TODO Fase 2] Aquí se llamará al gRPC Server (Go D2)")

	// Responder OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Status: "ok"})
}

func main() {
	http.HandleFunc("/send", sendHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}

	fmt.Printf("[GRPC-CLIENT] Escuchando en :%s (modo stub Fase 1)\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
