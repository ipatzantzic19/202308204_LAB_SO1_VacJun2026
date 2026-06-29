package main

import (
	"net/http/httptest"
	"testing"
)

func TestLiveDoesNotDependOnValkey(t *testing.T) {
	recorder := httptest.NewRecorder()
	live(recorder, httptest.NewRequest("GET", "/live", nil))
	if recorder.Code != 200 || recorder.Body.String() != "ok" {
		t.Fatalf("live = status %d body %q", recorder.Code, recorder.Body.String())
	}
}

func TestEscapeLabel(t *testing.T) {
	got := escapeLabel("user\\\"one\n")
	want := `user\\\"one\n`
	if got != want {
		t.Fatalf("escapeLabel() = %q, want %q", got, want)
	}
}

func TestValidNumber(t *testing.T) {
	for input, want := range map[string]string{"12": "12", "2.5": "2.5", "bad": "0", "": "0"} {
		if got := validNumber(input); got != want {
			t.Errorf("validNumber(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSummarizeUsesOnlyCurrentMatch(t *testing.T) {
	predictions := []prediction{
		{HomeTeam: "BRA", AwayTeam: "MEX", HomeGoals: 2, AwayGoals: 1, Username: "ana"},
		{HomeTeam: "ARG", AwayTeam: "BRA", HomeGoals: 5, AwayGoals: 5, Username: "luis"},
		{HomeTeam: "BRA", AwayTeam: "MEX", HomeGoals: 4, AwayGoals: 0, Username: "ana"},
		{HomeTeam: "BRA", AwayTeam: "ESP", HomeGoals: 0, AwayGoals: 5, Username: "ana"},
		{HomeTeam: "BRA", AwayTeam: "MEX", HomeGoals: 2, AwayGoals: 3, Username: "luis"},
	}
	s := summarize(predictions)
	if !s.HasLatest || s.Latest.HomeTeam != "BRA" || s.Latest.AwayTeam != "MEX" {
		t.Fatalf("último partido incorrecto: %+v", s.Latest)
	}
	if s.LocalMax != 4 || s.LocalMin != 2 || s.AwayMax != 3 || s.AwayMin != 0 {
		t.Fatalf("extremos incorrectos: %+v", s)
	}
	if s.LocalMode != 2 || s.AwayMode != 0 {
		t.Fatalf("modas incorrectas: local=%d visitante=%d", s.LocalMode, s.AwayMode)
	}
}

func TestSummarizeLimitsCurrentMatchToFiveNewestPredictions(t *testing.T) {
	predictions := []prediction{
		{HomeTeam: "BRA", AwayTeam: "ARG", HomeGoals: 1, AwayGoals: 2},
		{HomeTeam: "BRA", AwayTeam: "ARG", HomeGoals: 2, AwayGoals: 2},
		{HomeTeam: "BRA", AwayTeam: "ARG", HomeGoals: 2, AwayGoals: 3},
		{HomeTeam: "BRA", AwayTeam: "ARG", HomeGoals: 3, AwayGoals: 3},
		{HomeTeam: "BRA", AwayTeam: "ARG", HomeGoals: 4, AwayGoals: 4},
		{HomeTeam: "BRA", AwayTeam: "ARG", HomeGoals: 5, AwayGoals: 0},
	}
	s := summarize(predictions)
	if s.MatchSize != 5 {
		t.Fatalf("tamaño de muestra = %d, want 5", s.MatchSize)
	}
	if s.LocalMax != 4 || s.LocalMin != 1 || s.AwayMax != 4 || s.AwayMin != 2 {
		t.Fatalf("la sexta predicción no debe afectar extremos: %+v", s)
	}
}
