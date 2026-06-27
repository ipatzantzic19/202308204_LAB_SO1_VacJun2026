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
	rawURL  string
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
	mu      sync.Mutex
}

type publisher interface {
	publishMessage(context.Context, []byte) error
	healthy() bool
}

type api struct {
	publisher publisher
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
	w := &writer{rawURL: rawURL, queue: queue}
	if err := w.reconnectLocked(); err != nil {
		return nil, err
	}
	return w, nil
}

func open(rawURL, queue string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(rawURL)
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	if _, err = ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, err
	}
	return conn, ch, nil
}

func (w *writer) reconnectLocked() error {
	if w.channel != nil {
		_ = w.channel.Close()
	}
	if w.conn != nil {
		_ = w.conn.Close()
	}

	conn, ch, err := open(w.rawURL, w.queue)
	if err != nil {
		w.conn = nil
		w.channel = nil
		return err
	}
	w.conn = conn
	w.channel = ch
	return nil
}

func (w *writer) ensureConnected() (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil && w.channel != nil && !w.conn.IsClosed() && !w.channel.IsClosed() {
		return false, nil
	}
	return true, w.reconnectLocked()
}

func (w *writer) reconnectLoop(interval time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			reconnected, err := w.ensureConnected()
			if err != nil {
				log.Printf("[RABBIT-WRITER] RabbitMQ sigue no disponible: %v", err)
			} else if reconnected {
				log.Printf("[RABBIT-WRITER] conexión RabbitMQ recuperada")
			}
		case <-done:
			return
		}
	}
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

func (w *writer) publishMessage(ctx context.Context, body []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn == nil || w.channel == nil || w.conn.IsClosed() || w.channel.IsClosed() {
		if err := w.reconnectLocked(); err != nil {
			return fmt.Errorf("reconectando RabbitMQ: %w", err)
		}
	}

	publish := func() error {
		return w.channel.PublishWithContext(ctx, "", w.queue, false, false, amqp.Publishing{
			ContentType: "application/json", DeliveryMode: amqp.Persistent,
			Timestamp: time.Now().UTC(), Body: body,
		})
	}
	if err := publish(); err != nil {
		log.Printf("[RABBIT-WRITER] conexión perdida; reconectando: %v", err)
		if reconnectErr := w.reconnectLocked(); reconnectErr != nil {
			return fmt.Errorf("publicando: %v; reconectando: %w", err, reconnectErr)
		}
		if err := publish(); err != nil {
			return fmt.Errorf("publicando después de reconectar: %w", err)
		}
	}
	return nil
}

func (w *writer) healthy() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn != nil && w.channel != nil && !w.conn.IsClosed() && !w.channel.IsClosed()
}

func (w *writer) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.channel != nil {
		_ = w.channel.Close()
	}
	if w.conn != nil {
		_ = w.conn.Close()
	}
}

func (a *api) publish(rw http.ResponseWriter, r *http.Request) {
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
	if err = a.publisher.publishMessage(ctx, body); err != nil {
		log.Printf("[RABBIT-WRITER] Error publicando: %v", err)
		http.Error(rw, "Error publicando", http.StatusServiceUnavailable)
		return
	}

	log.Printf("[RABBIT-WRITER] Mensaje persistente publicado: %s", body)
	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write([]byte(`{"status":"published"}`))
}

func (a *api) health(rw http.ResponseWriter, _ *http.Request) {
	if !a.publisher.healthy() {
		http.Error(rw, "rabbitmq disconnected", http.StatusServiceUnavailable)
		return
	}
	_, _ = fmt.Fprint(rw, "ok")
}

func (a *api) live(rw http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprint(rw, "ok")
}

func main() {
	queue := getenv("RABBITMQ_QUEUE", "predictions")
	w := connectWithRetry(rabbitURL(), queue)
	defer w.close()
	done := make(chan struct{})
	defer close(done)
	go w.reconnectLoop(5*time.Second, done)
	a := &api{publisher: w}

	http.HandleFunc("/publish", a.publish)
	http.HandleFunc("/health", a.health)
	http.HandleFunc("/live", a.live)
	log.Printf("[RABBIT-WRITER] Escuchando :9100; cola=%s", queue)
	log.Fatal(http.ListenAndServe(":9100", nil))
}
