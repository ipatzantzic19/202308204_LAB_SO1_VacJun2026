package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const validPrediction = `{"home_team":"BRA","away_team":"MEX","home_goals":2,"away_goals":1,"username":"user_42","timestamp":"2026-06-27T00:00:00Z"}`

func TestPredictionHandlerPropagatesDownstreamFailure(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "RabbitMQ unavailable", http.StatusBadGateway)
	}))
	defer downstream.Close()

	s := &server{grpcClientURL: downstream.URL, httpClient: &http.Client{Timeout: time.Second}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(validPrediction))
	res := httptest.NewRecorder()
	s.predictionHandler(res, req)

	if res.Code != http.StatusBadGateway {
		t.Fatalf("status = %d; want %d", res.Code, http.StatusBadGateway)
	}
}

func TestPredictionHandlerReturnsOKOnlyOnDownstreamSuccess(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	s := &server{grpcClientURL: downstream.URL, httpClient: downstream.Client()}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(validPrediction))
	res := httptest.NewRecorder()
	s.predictionHandler(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", res.Code, http.StatusOK)
	}
}

func TestPredictionHandlerRejectsInvalidJSON(t *testing.T) {
	s := &server{grpcClientURL: "http://unused", httpClient: &http.Client{Timeout: time.Second}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"broken"`))
	res := httptest.NewRecorder()
	s.predictionHandler(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", res.Code, http.StatusBadRequest)
	}
}
