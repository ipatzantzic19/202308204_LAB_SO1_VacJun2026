package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	pb "github.com/ipatzantzic19/202308204_LAB_SO1_VacJun2026/PROYECTO2/proto"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const assignedTeam = "BRA"

type prediction struct {
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeGoals int32  `json:"home_goals"`
	AwayGoals int32  `json:"away_goals"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

type store interface {
	Save(context.Context, prediction) error
	Ping(context.Context) error
	Close() error
}

type valkeyStore struct {
	client    *redis.Client
	maxPoints int64
}

// Una sola ejecución de Lua mantiene consistentes todas las métricas antes del ACK.
var savePrediction = redis.NewScript(`
local function update_extreme(key, value, comparator)
  local current = redis.call('GET', key)
  if (not current) or comparator(tonumber(value), tonumber(current)) then
    redis.call('SET', key, value)
  end
end

local function update_mode(hash_key, mode_key, count_key, value)
  local count = redis.call('HINCRBY', hash_key, value, 1)
  local mode_count = tonumber(redis.call('GET', count_key) or '-1')
  local mode = tonumber(redis.call('GET', mode_key) or '999')
  if count > mode_count or (count == mode_count and tonumber(value) < mode) then
    redis.call('SET', mode_key, value)
    redis.call('SET', count_key, count)
  end
end

local home_team, away_team = ARGV[1], ARGV[2]
local home_goals, away_goals = tonumber(ARGV[3]), tonumber(ARGV[4])
local username, timestamp = ARGV[5], ARGV[6]
local score, event_id = tonumber(ARGV[7]), ARGV[8]
local assigned, max_points = ARGV[9], tonumber(ARGV[10])

redis.call('INCR', 'stats:predictions:total')
update_extreme('stats:local:goals:max', home_goals, function(a,b) return a > b end)
update_extreme('stats:local:goals:min', home_goals, function(a,b) return a < b end)
update_extreme('stats:away:goals:max', away_goals, function(a,b) return a > b end)
update_extreme('stats:away:goals:min', away_goals, function(a,b) return a < b end)
update_mode('stats:local:goals:frequency', 'stats:local:goals:mode', 'stats:local:goals:mode_count', home_goals)
update_mode('stats:away:goals:frequency', 'stats:away:goals:mode', 'stats:away:goals:mode_count', away_goals)
redis.call('ZINCRBY', 'stats:users', 1, username)

if home_goals > away_goals then
  redis.call('ZINCRBY', 'stats:wins', 1, home_team)
elseif away_goals > home_goals then
  redis.call('ZINCRBY', 'stats:wins', 1, away_team)
end

if home_team == assigned or away_team == assigned then
  local prefix = 'prediction:' .. string.lower(assigned)
  redis.call('INCR', prefix .. ':count')

  -- Ventana reproducible con la predicción completa. El consecutivo evita
  -- colisiones cuando varios usuarios envían en el mismo segundo.
  local sequence = redis.call('INCR', 'stats:event:sequence')
  redis.call('ZADD', prefix .. ':recent', score, tostring(sequence) .. '|' .. ARGV[11])
  local recent_size = redis.call('ZCARD', prefix .. ':recent')
  if recent_size > max_points then
    redis.call('ZREMRANGEBYRANK', prefix .. ':recent', 0, recent_size - max_points - 1)
  end

  local side, goals
  if home_team == assigned then side, goals = 'local', home_goals else side, goals = 'away', away_goals end
  local key = prefix .. ':timeseries:' .. side
  local member = timestamp .. '|' .. goals .. '|' .. event_id
  redis.call('ZADD', key, score, member)
  local size = redis.call('ZCARD', key)
  if size > max_points then redis.call('ZREMRANGEBYRANK', key, 0, size - max_points - 1) end
end
return 1
`)

func (s *valkeyStore) Save(ctx context.Context, p prediction) error {
	t, err := time.Parse(time.RFC3339, p.Timestamp)
	if err != nil {
		return fmt.Errorf("timestamp inválido: %w", err)
	}
	eventID := fmt.Sprintf("%d-%s-%s-%s", t.UnixNano(), p.HomeTeam, p.AwayTeam, p.Username)
	rawPrediction, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("serializando predicción: %w", err)
	}
	_, err = savePrediction.Run(ctx, s.client, nil,
		p.HomeTeam, p.AwayTeam, p.HomeGoals, p.AwayGoals, p.Username, p.Timestamp,
		t.UnixMilli(), eventID, assignedTeam, s.maxPoints, string(rawPrediction),
	).Result()
	if err != nil {
		return fmt.Errorf("guardando métricas en Valkey: %w", err)
	}
	return nil
}

func (s *valkeyStore) Ping(ctx context.Context) error { return s.client.Ping(ctx).Err() }
func (s *valkeyStore) Close() error                   { return s.client.Close() }

func decodePrediction(body []byte) (prediction, error) {
	var p prediction
	if err := json.Unmarshal(body, &p); err != nil {
		return p, fmt.Errorf("JSON inválido: %w", err)
	}
	p.HomeTeam = strings.ToUpper(strings.TrimSpace(p.HomeTeam))
	p.AwayTeam = strings.ToUpper(strings.TrimSpace(p.AwayTeam))
	p.Username = strings.TrimSpace(p.Username)
	if _, ok := pb.Teams_value[p.HomeTeam]; !ok || p.HomeTeam == pb.Teams_TEAMS_UNKNOWN.String() {
		return p, fmt.Errorf("home_team inválido: %q", p.HomeTeam)
	}
	if _, ok := pb.Teams_value[p.AwayTeam]; !ok || p.AwayTeam == pb.Teams_TEAMS_UNKNOWN.String() {
		return p, fmt.Errorf("away_team inválido: %q", p.AwayTeam)
	}
	if p.HomeTeam == p.AwayTeam {
		return p, errors.New("los equipos deben ser distintos")
	}
	if p.HomeGoals < 0 || p.HomeGoals > 5 || p.AwayGoals < 0 || p.AwayGoals > 5 {
		return p, errors.New("los goles deben estar entre 0 y 5")
	}
	if p.Username == "" {
		return p, errors.New("username es obligatorio")
	}
	if _, err := time.Parse(time.RFC3339, p.Timestamp); err != nil {
		return p, fmt.Errorf("timestamp inválido: %w", err)
	}
	return p, nil
}

func rabbitURL() string {
	if raw := os.Getenv("RABBITMQ_URL"); raw != "" {
		return raw
	}
	u := &url.URL{Scheme: "amqp", Host: getenv("RABBITMQ_HOST", "rabbitmq-cluster.rabbitmq-system.svc.cluster.local:5672"), Path: "/"}
	u.User = url.UserPassword(os.Getenv("RABBITMQ_USER"), os.Getenv("RABBITMQ_PASSWORD"))
	return u.String()
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type health struct {
	amqpReady   atomic.Bool
	valkeyReady atomic.Bool
}

func (h *health) serve(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/live", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		if !h.amqpReady.Load() || !h.valkeyReady.Load() {
			http.Error(w, "dependencies disconnected", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("[CONSUMER] Health server en %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("health server: %v", err)
	}
}

func consumeSession(ctx context.Context, rawURL, queue string, s store, h *health) error {
	conn, err := amqp.Dial(rawURL)
	if err != nil {
		return fmt.Errorf("conectando RabbitMQ: %w", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("abriendo canal: %w", err)
	}
	defer ch.Close()
	if _, err = ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declarando cola: %w", err)
	}
	if err = ch.Qos(20, 0, false); err != nil {
		return fmt.Errorf("configurando QoS: %w", err)
	}
	deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("iniciando consumo: %w", err)
	}
	closed := ch.NotifyClose(make(chan *amqp.Error, 1))
	h.amqpReady.Store(true)
	defer h.amqpReady.Store(false)
	log.Printf("[CONSUMER] Consumiendo cola=%s con autoAck=false", queue)

	for {
		select {
		case <-ctx.Done():
			return nil
		case closeErr := <-closed:
			if closeErr == nil {
				return errors.New("canal AMQP cerrado")
			}
			return fmt.Errorf("canal AMQP cerrado: %w", closeErr)
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("canal de entregas cerrado")
			}
			p, decodeErr := decodePrediction(d.Body)
			if decodeErr != nil {
				log.Printf("[CONSUMER] Mensaje descartado: %v", decodeErr)
				_ = d.Nack(false, false)
				continue
			}
			saveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = s.Save(saveCtx, p)
			cancel()
			if err != nil {
				h.valkeyReady.Store(false)
				log.Printf("[CONSUMER] Valkey no disponible; mensaje reencolado: %v", err)
				_ = d.Nack(false, true)
				time.Sleep(time.Second)
				continue
			}
			h.valkeyReady.Store(true)
			if err = d.Ack(false); err != nil {
				return fmt.Errorf("confirmando mensaje: %w", err)
			}
			log.Printf("[CONSUMER] Persistida %s-%s de %s", p.HomeTeam, p.AwayTeam, p.Username)
		}
	}
}

func main() {
	maxPoints, err := strconv.ParseInt(getenv("TIMESERIES_MAX_POINTS", "10000"), 10, 64)
	if err != nil || maxPoints < 1 {
		log.Fatal("TIMESERIES_MAX_POINTS debe ser un entero positivo")
	}
	client := redis.NewClient(&redis.Options{Addr: getenv("VALKEY_ADDR", "valkey-service.sopes1-p2.svc.cluster.local:6379"), Password: os.Getenv("VALKEY_PASSWORD")})
	s := &valkeyStore{client: client, maxPoints: maxPoints}
	defer s.Close()
	h := &health{}
	go h.serve(":8080")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	queue := getenv("RABBITMQ_QUEUE", "predictions")
	for attempt := 1; ctx.Err() == nil; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = s.Ping(pingCtx)
		cancel()
		h.valkeyReady.Store(err == nil)
		if err != nil {
			log.Printf("[CONSUMER] Valkey no disponible (intento %d): %v", attempt, err)
			time.Sleep(5 * time.Second)
			continue
		}
		if err = consumeSession(ctx, rabbitURL(), queue, s, h); err != nil {
			log.Printf("[CONSUMER] Sesión terminada; reconectando: %v", err)
			time.Sleep(5 * time.Second)
		}
	}
}
