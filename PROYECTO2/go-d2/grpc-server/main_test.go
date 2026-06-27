package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pb "github.com/ipatzantzic19/202308204_LAB_SO1_VacJun2026/PROYECTO2/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func request() *pb.MatchPredictionRequest {
	return &pb.MatchPredictionRequest{
		HomeTeam: pb.Teams_BRA, AwayTeam: pb.Teams_MEX,
		HomeGoals: 2, AwayGoals: 1, Username: "user_42",
		Timestamp: "2026-06-27T00:00:00Z",
	}
}

func TestSendPredictionPublishesJSON(t *testing.T) {
	var received prediction
	writer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("request = %s content-type=%q", r.Method, r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		rw.WriteHeader(http.StatusOK)
	}))
	defer writer.Close()

	s := &server{writerURL: writer.URL, client: writer.Client()}
	response, err := s.SendPrediction(context.Background(), request())

	if err != nil || response.GetStatus() != "published" {
		t.Fatalf("response/error = %v, %v; want published", response, err)
	}
	if received.HomeTeam != "BRA" || received.AwayTeam != "MEX" || received.Username != "user_42" {
		t.Fatalf("payload = %+v; unexpected conversion", received)
	}
}

func TestSendPredictionPropagatesWriterFailure(t *testing.T) {
	writer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		http.Error(rw, "broker unavailable", http.StatusServiceUnavailable)
	}))
	defer writer.Close()

	s := &server{writerURL: writer.URL, client: writer.Client()}
	_, err := s.SendPrediction(context.Background(), request())
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("code = %s; want %s (error=%v)", status.Code(err), codes.Unavailable, err)
	}
}
