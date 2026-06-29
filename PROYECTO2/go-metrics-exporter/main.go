package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	assignedTeam       = "BRA"
	retainedWindow     = 10000
	currentMatchWindow = 5
)

type valkeyReader interface {
	Get(context.Context, string) *redis.StringCmd
	ZRevRangeWithScores(context.Context, string, int64, int64) *redis.ZSliceCmd
	Ping(context.Context) *redis.StatusCmd
}

type exporter struct {
	valkey valkeyReader
}

type prediction struct {
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeGoals int    `json:"home_goals"`
	AwayGoals int    `json:"away_goals"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

type summary struct {
	LocalMax  int
	LocalMin  int
	AwayMax   int
	AwayMin   int
	LocalMode int
	AwayMode  int
	Latest    prediction
	HasLatest bool
	MatchSize int
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func validNumber(raw string) string {
	if _, err := strconv.ParseFloat(raw, 64); err != nil {
		return "0"
	}
	return raw
}

func (e *exporter) scalar(ctx context.Context, key string) (string, error) {
	value, err := e.valkey.Get(ctx, key).Result()
	if err == redis.Nil {
		return "0", nil
	}
	if err != nil {
		return "", err
	}
	return validNumber(value), nil
}

func (e *exporter) recent(ctx context.Context) ([]prediction, error) {
	items, err := e.valkey.ZRevRangeWithScores(ctx, "prediction:bra:recent", 0, retainedWindow-1).Result()
	if err != nil {
		return nil, fmt.Errorf("leyendo prediction:bra:recent: %w", err)
	}
	predictions := make([]prediction, 0, len(items))
	for _, item := range items {
		parts := strings.SplitN(fmt.Sprint(item.Member), "|", 2)
		if len(parts) != 2 {
			continue
		}
		var p prediction
		if err := json.Unmarshal([]byte(parts[1]), &p); err != nil {
			continue
		}
		predictions = append(predictions, p)
	}
	return predictions, nil
}

func mode(predictions []prediction, local bool) int {
	frequency := map[int]int{}
	bestValue, bestCount := 0, -1
	for _, p := range predictions {
		value := p.AwayGoals
		if local {
			value = p.HomeGoals
		}
		frequency[value]++
		if frequency[value] > bestCount || (frequency[value] == bestCount && value < bestValue) {
			bestValue, bestCount = value, frequency[value]
		}
	}
	return bestValue
}

func summarize(predictions []prediction) summary {
	s := summary{}
	if len(predictions) == 0 {
		return s
	}
	s.Latest, s.HasLatest = predictions[0], true

	currentMatch := make([]prediction, 0, len(predictions))
	for _, p := range predictions {
		if p.HomeTeam == s.Latest.HomeTeam && p.AwayTeam == s.Latest.AwayTeam {
			currentMatch = append(currentMatch, p)
			if len(currentMatch) == currentMatchWindow {
				break
			}
		}
	}
	s.MatchSize = len(currentMatch)
	s.LocalMax, s.LocalMin = currentMatch[0].HomeGoals, currentMatch[0].HomeGoals
	s.AwayMax, s.AwayMin = currentMatch[0].AwayGoals, currentMatch[0].AwayGoals
	for _, p := range currentMatch[1:] {
		if p.HomeGoals > s.LocalMax {
			s.LocalMax = p.HomeGoals
		}
		if p.HomeGoals < s.LocalMin {
			s.LocalMin = p.HomeGoals
		}
		if p.AwayGoals > s.AwayMax {
			s.AwayMax = p.AwayGoals
		}
		if p.AwayGoals < s.AwayMin {
			s.AwayMin = p.AwayGoals
		}
	}

	s.LocalMode, s.AwayMode = mode(currentMatch, true), mode(currentMatch, false)
	return s
}

func (e *exporter) historicalRanking(ctx context.Context, key string) (map[string]int, error) {
	items, err := e.valkey.ZRevRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("leyendo %s: %w", key, err)
	}
	values := make(map[string]int, len(items))
	for _, item := range items {
		values[fmt.Sprint(item.Member)] = int(item.Score)
	}
	return values, nil
}

func writeHeader(b *strings.Builder, name, help, metricType string) {
	fmt.Fprintf(b, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, metricType)
}

func writeValue(b *strings.Builder, name, help, metricType string, value int) {
	writeHeader(b, name, help, metricType)
	fmt.Fprintf(b, "%s %d\n", name, value)
}

type rankingItem struct {
	Name  string
	Count int
}

func writeRanking(b *strings.Builder, values map[string]int, name, label, help string) {
	items := make([]rankingItem, 0, len(values))
	for itemName, count := range values {
		items = append(items, rankingItem{Name: itemName, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Name < items[j].Name
		}
		return items[i].Count > items[j].Count
	})
	writeHeader(b, name, help, "gauge")
	for _, item := range items {
		fmt.Fprintf(b, "%s{%s=\"%s\"} %d\n", name, label, escapeLabel(item.Name), item.Count)
	}
}

func writeWinPercentages(b *strings.Builder, wins map[string]int) {
	total := 0
	for _, count := range wins {
		total += count
	}
	writeHeader(b, "quiniela_team_win_percentage", "Porcentaje historico sobre todas las victorias predichas.", "gauge")
	if total == 0 {
		return
	}
	items := make([]rankingItem, 0, len(wins))
	for team, count := range wins {
		items = append(items, rankingItem{Name: team, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Name < items[j].Name
		}
		return items[i].Count > items[j].Count
	})
	for _, item := range items {
		percentage := float64(item.Count) * 100 / float64(total)
		fmt.Fprintf(b, "quiniela_team_win_percentage{team=\"%s\"} %.2f\n", escapeLabel(item.Name), percentage)
	}
}

func (e *exporter) render(ctx context.Context) (string, error) {
	braTotal, err := e.scalar(ctx, "prediction:bra:count")
	if err != nil {
		return "", err
	}
	globalTotal, err := e.scalar(ctx, "stats:predictions:total")
	if err != nil {
		return "", err
	}
	predictions, err := e.recent(ctx)
	if err != nil {
		return "", err
	}
	s := summarize(predictions)
	historicalWins, err := e.historicalRanking(ctx, "stats:wins")
	if err != nil {
		return "", err
	}
	historicalUsers, err := e.historicalRanking(ctx, "stats:users")
	if err != nil {
		return "", err
	}

	var b strings.Builder
	writeHeader(&b, "quiniela_bra_predictions_total", "Total exacto acumulado de predicciones relacionadas con BRA.", "counter")
	fmt.Fprintf(&b, "quiniela_bra_predictions_total %s\n", braTotal)
	writeHeader(&b, "quiniela_predictions_total", "Total global de predicciones procesadas.", "counter")
	fmt.Fprintf(&b, "quiniela_predictions_total %s\n", globalTotal)
	writeValue(&b, "quiniela_local_goals_max", "Maximo local en las ultimas 5 predicciones del enfrentamiento actual.", "gauge", s.LocalMax)
	writeValue(&b, "quiniela_local_goals_min", "Minimo local en las ultimas 5 predicciones del enfrentamiento actual.", "gauge", s.LocalMin)
	writeValue(&b, "quiniela_away_goals_max", "Maximo visitante en las ultimas 5 predicciones del enfrentamiento actual.", "gauge", s.AwayMax)
	writeValue(&b, "quiniela_away_goals_min", "Minimo visitante en las ultimas 5 predicciones del enfrentamiento actual.", "gauge", s.AwayMin)
	writeValue(&b, "quiniela_local_goals_mode", "Moda local en las ultimas 5 predicciones del enfrentamiento actual.", "gauge", s.LocalMode)
	writeValue(&b, "quiniela_away_goals_mode", "Moda visitante en las ultimas 5 predicciones del enfrentamiento actual.", "gauge", s.AwayMode)
	writeValue(&b, "quiniela_recent_sample_size", "Predicciones recientes disponibles para el dashboard.", "gauge", len(predictions))
	writeValue(&b, "quiniela_current_match_sample_size", "Muestra usada del enfrentamiento actual, hasta 5 predicciones.", "gauge", s.MatchSize)

	writeHeader(&b, "quiniela_assigned_team_info", "Equipo asignado y carnet del proyecto.", "gauge")
	fmt.Fprintf(&b, "quiniela_assigned_team_info{team=\"%s\",carnet=\"202308204\"} 1\n", assignedTeam)
	writeRanking(&b, historicalWins, "quiniela_team_wins", "team", "Victorias predichas acumuladas historicamente.")
	writeWinPercentages(&b, historicalWins)
	writeRanking(&b, historicalUsers, "quiniela_user_predictions", "username", "Predicciones enviadas acumuladas historicamente.")

	writeHeader(&b, "quiniela_latest_match_info", "Equipos de la prediccion mas reciente relacionada con BRA.", "gauge")
	if s.HasLatest {
		fmt.Fprintf(&b, "quiniela_latest_match_info{home_team=\"%s\",away_team=\"%s\"} 1\n", escapeLabel(s.Latest.HomeTeam), escapeLabel(s.Latest.AwayTeam))
	}
	writeValue(&b, "quiniela_bra_local_goals", "Goles del equipo local en la prediccion BRA mas reciente.", "gauge", s.Latest.HomeGoals)
	writeValue(&b, "quiniela_bra_away_goals", "Goles del equipo visitante en la prediccion BRA mas reciente.", "gauge", s.Latest.AwayGoals)
	return b.String(), nil
}

func (e *exporter) metrics(w http.ResponseWriter, _ *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	body, err := e.render(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (e *exporter) health(w http.ResponseWriter, _ *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := e.valkey.Ping(ctx).Err(); err != nil {
		http.Error(w, "valkey unavailable", http.StatusServiceUnavailable)
		return
	}
	_, _ = w.Write([]byte("ok"))
}

func live(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	client := redis.NewClient(&redis.Options{Addr: getenv("VALKEY_ADDR", "valkey-service.sopes1-p2.svc.cluster.local:6379"), Password: os.Getenv("VALKEY_PASSWORD")})
	exp := &exporter{valkey: client}
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", exp.metrics)
	mux.HandleFunc("/health", exp.health)
	mux.HandleFunc("/live", live)
	addr := getenv("LISTEN_ADDR", ":9100")
	log.Printf("[EXPORTER] Escuchando en %s, equipo=%s", addr, assignedTeam)
	log.Fatal(http.ListenAndServe(addr, mux))
}
