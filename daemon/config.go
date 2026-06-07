package main

// ============================================================
//  config.go — Configuración Central del Daemon
// ============================================================

import "time"

// Config agrupa toda la configuración del daemon
type Config struct {
	// ── Ruta del archivo /proc ───────────────────────────────
	ProcFile string

	// ── Valkey ───────────────────────────────────────────────
	ValkeyAddr string // host:port del servidor Valkey

	// ── Prometheus ───────────────────────────────────────────
	MetricsPort string // Puerto donde se exponen las métricas

	// ── Docker Compose ───────────────────────────────────────
	RutaCompose string // Ruta absoluta al docker-compose.yml

	// ── Scripts ──────────────────────────────────────────────
	RutaScriptSpawn  string // Ruta absoluta a spawn_containers.sh
	RutaScriptModulo string // Ruta absoluta a load_module.sh

	// ── Módulo de kernel ─────────────────────────────────────
	ModuloPath string // Ruta absoluta al .ko compilado

	// ── Loop principal ───────────────────────────────────────
	LoopInterval time.Duration // Cada cuánto corre el ciclo principal

	// ── Reglas de gestión ────────────────────────────────────
	MinBajos int // Mínimo de contenedores de bajo consumo a mantener
	MinAltos int // Mínimo de contenedores de alto consumo a mantener
}

// cfg es la instancia global de configuración.
// Todos los archivos del daemon usan cfg.X para leer valores.
var cfg = Config{
	// *** AJUSTA ESTAS RUTAS A TU MÁQUINA ***
	// Usa rutas absolutas para que funcionen cuando cron ejecuta el script.

	ProcFile: "/proc/continfo_pr1_so1_202308204",

	ValkeyAddr:  "localhost:6379",
	MetricsPort: ":9200",

	// Ruta al docker-compose.yml de la Fase 2
	// Ejemplo: /home/isai/Documentos/Github/202308204_LAB_SO1_VacJun2026/docker/docker-compose.yml
	RutaCompose: "/home/isai/Documentos/Github/202308204_LAB_SO1_VacJun2026/docker/docker-compose.yml",

	// Ruta al script de creación de contenedores (Fase 3)
	RutaScriptSpawn: "/home/isai/Documentos/Github/202308204_LAB_SO1_VacJun2026/scripts/spawn_containers.sh",

	// Ruta al script de carga del módulo (Fase 1)
	RutaScriptModulo: "/home/isai/Documentos/Github/202308204_LAB_SO1_VacJun2026/kernel_module/load_module.sh",

	// Ruta al .ko compilado (fallback si load_module.sh no existe)
	ModuloPath: "/home/isai/Documentos/Github/202308204_LAB_SO1_VacJun2026/kernel_module/sysinfo_module.ko",

	// Intervalo del loop: 30 segundos (dentro del rango 20-60s del enunciado)
	LoopInterval: 30 * time.Second,

	// Reglas del enunciado
	MinBajos: 3,
	MinAltos: 2,
}
