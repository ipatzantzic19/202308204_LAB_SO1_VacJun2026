package main

// ============================================================
//  valkey_client.go — Cliente de Valkey
//
//  Almacena todas las métricas del daemon en Valkey.
//  Valkey es compatible con Redis → usamos go-redis.
//
//  Claves utilizadas:
//    memoria:actual         → JSON del último snapshot de RAM
//    memoria:historia       → Lista de snapshots (últimos 1000)
//    contenedor:{id}        → JSON del estado de cada contenedor
//    ranking:ram            → Sorted set por RSS
//    ranking:cpu            → Sorted set por CPU
//    eliminados:log         → Lista de todos los contenedores eliminados
//    eliminados:total       → Contador global de eliminaciones
// ============================================================

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	rdb *redis.Client
	ctx = context.Background()
)

// inicializarValkey conecta al servidor Valkey y verifica la conexión.
func inicializarValkey() error {
	rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.ValkeyAddr,
		Password: "",
		DB:       0,
	})

	// Verificar conexión con PING
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("no se pudo conectar a Valkey en %s: %w\n"+
			"  ¿Está corriendo el contenedor? Verifica con: docker compose ps",
			cfg.ValkeyAddr, err)
	}

	log.Printf("[VALKEY] ✓ Conectado a Valkey en %s", cfg.ValkeyAddr)
	return nil
}

// guardarMemoria guarda el snapshot de RAM en Valkey.
// Mantiene el estado actual y un historial de hasta 1000 entradas.
func guardarMemoria(info *SysInfo) {
	log := logMemoriaDesde(info)
	jsonData, err := json.Marshal(log)
	if err != nil {
		return
	}

	data := string(jsonData)

	// Estado actual (sobreescribe siempre)
	rdb.Set(ctx, "memoria:actual", data, 0)

	// Historial: insertar al inicio, mantener últimos 1000
	rdb.LPush(ctx, "memoria:historia", data)
	rdb.LTrim(ctx, "memoria:historia", 0, 999)
}

// guardarContenedores persiste el estado actual de cada contenedor
// y actualiza los rankings de RAM y CPU.
func guardarContenedores(contenedores []ContainerInfo) {
	// Limpiar rankings anteriores para que reflejen solo los contenedores activos
	rdb.Del(ctx, "ranking:ram")
	rdb.Del(ctx, "ranking:cpu")

	for _, c := range contenedores {
		// Estado individual del contenedor
		key := fmt.Sprintf("contenedor:%s", c.ID)
		rdb.Set(ctx, key, contenedorAJSON(c), time.Hour)

		// Rankings (sorted sets: score = valor de la métrica)
		rdb.ZAdd(ctx, "ranking:ram", redis.Z{
			Score:  float64(c.RSS),
			Member: c.Nombre,
		})
		rdb.ZAdd(ctx, "ranking:cpu", redis.Z{
			Score:  c.CPU,
			Member: c.Nombre,
		})
	}
}

// guardarEliminacion registra en Valkey cada contenedor que el daemon eliminó.
func guardarEliminacion(c ContainerInfo, razon string) {
	registro := LogEliminacion{
		ID:        c.ID,
		Nombre:    c.Nombre,
		Imagen:    c.Imagen,
		Tipo:      string(c.Tipo),
		RSS:       c.RSS,
		CPU:       c.CPU,
		Timestamp: time.Now().Format(time.RFC3339),
		Razon:     razon,
	}

	jsonData, err := json.Marshal(registro)
	if err != nil {
		return
	}

	data := string(jsonData)

	// Lista de eliminados (historial)
	rdb.LPush(ctx, "eliminados:log", data)
	rdb.LTrim(ctx, "eliminados:log", 0, 999) // Máximo 1000 registros

	// Contador global
	rdb.Incr(ctx, "eliminados:total")

	// Registro con timestamp para series de tiempo en Grafana
	tsKey := fmt.Sprintf("eliminado:%d", time.Now().UnixNano())
	rdb.Set(ctx, tsKey, data, 24*time.Hour)
}

// guardarCiclo es el punto de entrada principal:
// guarda todo el estado de un ciclo del daemon en Valkey.
func guardarCiclo(info *SysInfo, contenedores []ContainerInfo, eliminados []ContainerInfo) {
	guardarMemoria(info)
	guardarContenedores(contenedores)

	for _, c := range eliminados {
		guardarEliminacion(c, "excede_limite")
	}

	log.Printf("[VALKEY] ✓ Ciclo guardado. Eliminados en este ciclo: %d", len(eliminados))
}

// ── Helpers de lectura (útiles para debugging) ────────────────

// logMemoriaDesde construye un LogMemoria desde un SysInfo.
func logMemoriaDesde(info *SysInfo) LogMemoria {
	return LogMemoria{
		TotalKB:   info.TotalRAM,
		LibreKB:   info.FreeRAM,
		UsadaKB:   info.UsedRAM,
		Timestamp: time.Now().Unix(),
	}
}

// leerTotalEliminados lee el contador de eliminaciones de Valkey.
func leerTotalEliminados() int64 {
	val, err := rdb.Get(ctx, "eliminados:total").Int64()
	if err != nil {
		return 0
	}
	return val
}
