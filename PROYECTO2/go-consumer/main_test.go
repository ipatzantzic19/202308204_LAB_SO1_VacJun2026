package main

import "testing"

func TestDecodePrediction(t *testing.T) {
	p, err := decodePrediction([]byte(`{"home_team":"bra","away_team":"MEX","home_goals":2,"away_goals":1,"username":" user_1 ","timestamp":"2026-06-27T12:00:00Z"}`))
	if err != nil {
		t.Fatalf("decodePrediction() error = %v", err)
	}
	if p.HomeTeam != "BRA" || p.Username != "user_1" {
		t.Fatalf("normalización inesperada: %+v", p)
	}
}

func TestDecodePredictionRejectsInvalidEvents(t *testing.T) {
	tests := []string{
		`{"home_team":"BRA","away_team":"BRA","home_goals":1,"away_goals":0,"username":"u","timestamp":"2026-06-27T12:00:00Z"}`,
		`{"home_team":"BRA","away_team":"MEX","home_goals":6,"away_goals":0,"username":"u","timestamp":"2026-06-27T12:00:00Z"}`,
		`{"home_team":"XXX","away_team":"MEX","home_goals":1,"away_goals":0,"username":"u","timestamp":"2026-06-27T12:00:00Z"}`,
		`{"home_team":"BRA","away_team":"MEX","home_goals":1,"away_goals":0,"username":"","timestamp":"bad"}`,
	}
	for _, body := range tests {
		if _, err := decodePrediction([]byte(body)); err == nil {
			t.Errorf("se aceptó evento inválido: %s", body)
		}
	}
}
