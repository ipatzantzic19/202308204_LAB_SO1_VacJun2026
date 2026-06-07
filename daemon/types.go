package main

// ============================================================
//  types.go — Estructuras de datos del Daemon
//
//  Los campos JSON deben coincidir EXACTAMENTE con los que
//  produce el módulo de kernel (sysinfo_module.c).
// ============================================================

// ── Estructuras que mapean el JSON de /proc ───────────────────

// Process representa un proceso capturado por el módulo de kernel.
// Los tags `json:"..."` le dicen a Go cómo leer cada campo del JSON.
type Process struct {
	PID         int     `json:"PID"`          // ID único del proceso
	Name        string  `json:"Name"`         // Nombre del ejecutable (ej: "nginx")
	Cmdline     string  `json:"Cmdline"`      // Comando completo con argumentos
	VSZ         uint64  `json:"vsz"`          // Memoria virtual total en KB
	RSS         uint64  `json:"rss"`          // RAM física en uso en KB
	MemoryUsage float64 `json:"Memory_Usage"` // % de RAM total que consume
	CPUUsage    float64 `json:"CPU_Usage"`    // % de CPU que está usando
}

// SysInfo es el JSON completo que retorna el archivo /proc del módulo.
type SysInfo struct {
	TotalRAM  uint64    `json:"Totalram"`  // RAM total del sistema en KB
	FreeRAM   uint64    `json:"Freeram"`   // RAM libre en KB
	UsedRAM   uint64    `json:"Usedram"`   // RAM usada en KB
	Procs     int       `json:"Procs"`     // Total de procesos activos
	Processes []Process `json:"Processes"` // Lista de todos los procesos
}

// ── Estructuras de contenedores Docker ───────────────────────

// TipoContenedor clasifica el consumo esperado del contenedor
type TipoContenedor string

const (
	TipoAlto TipoContenedor = "alto" // roldyoran/go-client o alpine+CPU
	TipoBajo TipoContenedor = "bajo" // alpine sleep 240
)

// ContainerInfo reúne los datos de Docker con los datos del módulo de kernel
type ContainerInfo struct {
	ID      string         // ID corto del contenedor (12 chars)
	Nombre  string         // Nombre asignado (ej: sopes1_1749..._1)
	Imagen  string         // Imagen Docker usada
	Comando string         // Comando con el que corre el contenedor
	PID     int            // PID del proceso principal (de docker inspect)
	VSZ     uint64         // Memoria virtual en KB (del módulo kernel)
	RSS     uint64         // RAM física en KB (del módulo kernel)
	MemPct  float64        // % de memoria (del módulo kernel)
	CPU     float64        // % de CPU (del módulo kernel)
	Tipo    TipoContenedor // Clasificación: "alto" o "bajo"
}

// ── Estructura para logs en Valkey ────────────────────────────

// LogEliminacion registra cada contenedor que el daemon elimina
type LogEliminacion struct {
	ID        string  `json:"id"`
	Nombre    string  `json:"nombre"`
	Imagen    string  `json:"imagen"`
	Tipo      string  `json:"tipo"`
	RSS       uint64  `json:"rss_kb"`
	CPU       float64 `json:"cpu_pct"`
	Timestamp string  `json:"timestamp"`
	Razon     string  `json:"razon"` // ej: "excede_limite_bajo"
}

// LogMemoria guarda el estado de memoria en cada ciclo
type LogMemoria struct {
	TotalKB   uint64 `json:"total_kb"`
	LibreKB   uint64 `json:"libre_kb"`
	UsadaKB   uint64 `json:"usada_kb"`
	Timestamp int64  `json:"timestamp"`
}
