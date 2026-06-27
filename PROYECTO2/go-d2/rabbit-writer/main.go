package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type prediction struct {
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeGoals int32  `json:"home_goals"`
	AwayGoals int32  `json:"away_goals"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

type writer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
	mu      sync.Mutex
}

func rabbitURL() string {
	if raw := os.Getenv("RABBITMQ_URL"); raw != "" {
		return raw
	}
	host := getenv("RABBITMQ_HOST", "rabbitmq-cluster.rabbitmq-system.svc.cluster.local:5672")
	u := &url.URL{Scheme: "amqp", Host: host, Path: "/"}
	u.User = url.UserPassword(os.Getenv("RABBITMQ_USER"), os.Getenv("RABBITMQ_PASSWORD"))
	return u.String()
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func connect(rawURL, queue string) (*writer, error) {
	conn, err := amqp.Dial(rawURL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if _, err = ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	return &writer{conn: conn, channel: ch, queue: queue}, nil
}

func connectWithRetry(rawURL, queue string) *writer {
	for attempt := 1; ; attempt++ {
		w, err := connect(rawURL, queue)
		if err == nil {
			return w
		}
		log.Printf("[RABBIT-WRITER] RabbitMQ no disponible (intento %d): %v", attempt, err)
		time.Sleep(5 * time.Second)
	}
}

func (w *writer) publish(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var pred prediction
	if err := json.NewDecoder(r.Body).Decode(&pred); err != nil {
		http.Error(rw, "JSON inválido", http.StatusBadRequest)
		return
	}
	body, err := json.Marshal(pred)
	if err != nil {
		http.Error(rw, "Error serializando", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	w.mu.Lock()
	err = w.channel.PublishWithContext(ctx, "", w.queue, false, false, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent,
		Timestamp: time.Now().UTC(), Body: body,
	})
	w.mu.Unlock()
	if err != nil {
		http.Error(rw, "Error publicando", http.StatusServiceUnavailable)
		return
	}

	log.Printf("[RABBIT-WRITER] Mensaje persistente publicado en %q: %s", w.queue, body)
	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write([]byte(`{"status":"published"}`))
}

func (w *writer) health(rw http.ResponseWriter, _ *http.Request) {
	if w.conn.IsClosed() || w.channel.IsClosed() {
		http.Error(rw, "rabbitmq disconnected", http.StatusServiceUnavailable)
		return
	}
	_, _ = fmt.Fprint(rw, "ok")
}

func main() {
	queue := getenv("RABBITMQ_QUEUE", "predictions")
	w := connectWithRetry(rabbitURL(), queue)
	defer w.conn.Close()
	defer w.channel.Close()

	http.HandleFunc("/publish", w.publish)
	http.HandleFunc("/health", w.health)
	log.Printf("[RABBIT-WRITER] Escuchando :9100; cola=%s", queue)
	log.Fatal(http.ListenAndServe(":9100", nil))
}
