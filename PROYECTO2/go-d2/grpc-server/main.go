package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	pb "github.com/ipatzantzic19/202308204_LAB_SO1_VacJun2026/PROYECTO2/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type prediction struct {
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeGoals int32  `json:"home_goals"`
	AwayGoals int32  `json:"away_goals"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

type server struct {
	pb.UnimplementedMatchPredictionServiceServer
	writerURL string
	client    *http.Client
}

func (s *server) SendPrediction(ctx context.Context, req *pb.MatchPredictionRequest) (*pb.MatchPredictionResponse, error) {
	payload := prediction{
		HomeTeam: req.HomeTeam.String(), AwayTeam: req.AwayTeam.String(),
		HomeGoals: req.HomeGoals, AwayGoals: req.AwayGoals,
		Username: req.Username, Timestamp: req.Timestamp,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Error(codes.Internal, "no se pudo serializar la predicción")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.writerURL, bytes.NewReader(body))
	if err != nil {
		return nil, status.Error(codes.Internal, "no se pudo crear request al writer")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "writer RabbitMQ no disponible: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, status.Errorf(codes.Unavailable, "writer respondió %s", resp.Status)
	}

	log.Printf("[GRPC-SERVER] Publicada %s vs %s por %s", payload.HomeTeam, payload.AwayTeam, payload.Username)
	return &pb.MatchPredictionResponse{Status: "published"}, nil
}

func main() {
	writerURL := os.Getenv("RABBIT_WRITER_URL")
	if writerURL == "" {
		writerURL = "http://localhost:9100/publish"
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("No se pudo escuchar en :50051: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMatchPredictionServiceServer(grpcServer, &server{
		writerURL: writerURL,
		client:    &http.Client{Timeout: 5 * time.Second},
	})
	log.Printf("[GRPC-SERVER] Escuchando :50051 -> %s", writerURL)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(fmt.Errorf("gRPC terminó: %w", err))
	}
}
