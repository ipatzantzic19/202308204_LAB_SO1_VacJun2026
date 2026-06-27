package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakePublisher struct {
	body    []byte
	err     error
	isAlive bool
}

func (f *fakePublisher) publishMessage(_ context.Context, body []byte) error {
	f.body = append([]byte(nil), body...)
	return f.err
}

func (f *fakePublisher) healthy() bool { return f.isAlive }

const validPrediction = `{"home_team":"BRA","away_team":"MEX","home_goals":2,"away_goals":1,"username":"user_42","timestamp":"2026-06-27T00:00:00Z"}`

func TestPublishAcceptsValidPrediction(t *testing.T) {
	fake := &fakePublisher{isAlive: true}
	a := &api{publisher: fake}
	request := httptest.NewRequest(http.MethodPost, "/publish", strings.NewReader(validPrediction))
	response := httptest.NewRecorder()

	a.publish(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "published") {
		t.Fatalf("status/body = %d %q; want 200 published", response.Code, response.Body.String())
	}
	if !strings.Contains(string(fake.body), `"username":"user_42"`) {
		t.Fatalf("published body = %q; username missing", fake.body)
	}
}

func TestPublishReportsBrokerFailure(t *testing.T) {
	a := &api{publisher: &fakePublisher{err: errors.New("connection closed")}}
	request := httptest.NewRequest(http.MethodPost, "/publish", strings.NewReader(validPrediction))
	response := httptest.NewRecorder()

	a.publish(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestPublishRejectsInvalidInput(t *testing.T) {
	a := &api{publisher: &fakePublisher{}}

	for name, test := range map[string]struct {
		method string
		body   string
		want   int
	}{
		"method": {http.MethodGet, validPrediction, http.StatusMethodNotAllowed},
		"json":   {http.MethodPost, "{", http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/publish", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			a.publish(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d; want %d", response.Code, test.want)
			}
		})
	}
}

func TestHealthReflectsRabbitMQState(t *testing.T) {
	for _, test := range []struct {
		alive bool
		want  int
	}{{true, http.StatusOK}, {false, http.StatusServiceUnavailable}} {
		a := &api{publisher: &fakePublisher{isAlive: test.alive}}
		response := httptest.NewRecorder()
		a.health(response, httptest.NewRequest(http.MethodGet, "/health", nil))
		if response.Code != test.want {
			t.Fatalf("alive=%v: status=%d; want %d", test.alive, response.Code, test.want)
		}
	}
}

func TestLiveDoesNotDependOnRabbitMQ(t *testing.T) {
	a := &api{publisher: &fakePublisher{isAlive: false}}
	response := httptest.NewRecorder()
	a.live(response, httptest.NewRequest(http.MethodGet, "/live", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
	}
}
