package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pb "grpc-client/proto"

	"google.golang.org/grpc"
)

type fakePredictionClient struct {
	response *pb.MatchPredictionResponse
	err      error
}

func (f fakePredictionClient) SendPrediction(context.Context, *pb.MatchPredictionRequest, ...grpc.CallOption) (*pb.MatchPredictionResponse, error) {
	return f.response, f.err
}

const validBody = `{"home_team":"BRA","away_team":"MEX","home_goals":2,"away_goals":1,"username":"user_42","timestamp":"2026-06-27T00:00:00Z"}`

func TestSendHandlerReturnsBadGatewayOnGRPCFailure(t *testing.T) {
	s := &server{client: fakePredictionClient{err: errors.New("RabbitMQ unavailable")}}
	req := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(validBody))
	res := httptest.NewRecorder()
	s.sendHandler(res, req)

	if res.Code != http.StatusBadGateway {
		t.Fatalf("status = %d; want %d", res.Code, http.StatusBadGateway)
	}
}

func TestSendHandlerReturnsGRPCStatus(t *testing.T) {
	s := &server{client: fakePredictionClient{response: &pb.MatchPredictionResponse{Status: "published"}}}
	req := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(validBody))
	res := httptest.NewRecorder()
	s.sendHandler(res, req)

	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "published") {
		t.Fatalf("status/body = %d %q; want 200 published", res.Code, res.Body.String())
	}
}

func TestTeamToEnumRejectsUnknownTeam(t *testing.T) {
	if _, ok := teamToEnum("XXX"); ok {
		t.Fatal("teamToEnum accepted an unknown team")
	}
}
