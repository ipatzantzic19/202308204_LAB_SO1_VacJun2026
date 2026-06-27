package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	pb "github.com/ipatzantzic19/202308204_LAB_SO1_VacJun2026/PROYECTO2/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Prediction struct {
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeGoals int32  `json:"home_goals"`
	AwayGoals int32  `json:"away_goals"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

type server struct {
	client pb.MatchPredictionServiceClient
}

func teamToEnum(team string) (pb.Teams, bool) {
	value, ok := pb.Teams_value[strings.ToUpper(team)]
	if !ok || value == int32(pb.Teams_TEAMS_UNKNOWN) {
		return pb.Teams_TEAMS_UNKNOWN, false
	}
	return pb.Teams(value), true
}

func (s *server) sendHandler(w http.ResponseWriter, r *http.Request) {
	var pred Prediction
	if err := json.NewDecoder(r.Body).Decode(&pred); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	home, homeOK := teamToEnum(pred.HomeTeam)
	away, awayOK := teamToEnum(pred.AwayTeam)
	if !homeOK || !awayOK || home == away {
		http.Error(w, "Equipos inválidos", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := s.client.SendPrediction(ctx, &pb.MatchPredictionRequest{
		HomeTeam: home, AwayTeam: away,
		HomeGoals: pred.HomeGoals, AwayGoals: pred.AwayGoals,
		Username: pred.Username, Timestamp: pred.Timestamp,
	})
	if err != nil {
		log.Printf("[GRPC-CLIENT] Error llamando Go D2: %v", err)
		http.Error(w, "Error gRPC", http.StatusBadGateway)
		return
	}

	log.Printf("[GRPC-CLIENT] %s vs %s por %s -> %s",
		pred.HomeTeam, pred.AwayTeam, pred.Username, resp.Status)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": resp.Status})
}

func main() {
	address := os.Getenv("GO_D2_GRPC_ADDR")
	if address == "" {
		address = "go-d2-service.sopes1-p2.svc.cluster.local:50051"
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("No se pudo crear cliente gRPC: %v", err)
	}
	defer conn.Close()

	s := &server{client: pb.NewMatchPredictionServiceClient(conn)}
	http.HandleFunc("/send", s.sendHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	log.Printf("[GRPC-CLIENT] HTTP bridge :9000 -> %s", address)
	log.Fatal(http.ListenAndServe(":9000", nil))
}
