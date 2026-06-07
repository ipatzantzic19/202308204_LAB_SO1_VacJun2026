package main

// ============================================================
//  metrics.go — Servidor de Métricas Prometheus

//  Expone métricas en http://localhost:9200/metrics
//  Prometheus hace scraping de este endpoint cada 15s
//  (configurado en docker/prometheus.yml).
//
//  Métricas expuestas:
//    sysinfo_ram_total_kb          → RAM total del sistema
//    sysinfo_ram_free_kb           → RAM libre
//    sysinfo_ram_used_kb           → RAM usada
//    sysinfo_process_count         → Total de procesos
//    sysinfo_process_vsz_kb        → VSZ por proceso
//    sysinfo_process_rss_kb        → RSS por proceso
//    sysinfo_process_memory_percent → %RAM por proceso
//    sysinfo_process_cpu_percent   → %CPU por proceso
// ============================================================

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SysInfoCollector implementa prometheus.Collector.
type SysInfoCollector struct {
	totalRAM   *prometheus.Desc
	freeRAM    *prometheus.Desc
	usedRAM    *prometheus.Desc
	procCount  *prometheus.Desc
	procVSZ    *prometheus.Desc
	procRSS    *prometheus.Desc
	procMemPct *prometheus.Desc
	procCPUPct *prometheus.Desc
}

// NewSysInfoCollector crea el collector con todos sus descriptores.
// (Copiado de lector.go — Clase 5)
func NewSysInfoCollector() *SysInfoCollector {
	labels := []string{"pid", "name", "cmdline"}
	ns := "sysinfo"

	return &SysInfoCollector{
		totalRAM: prometheus.NewDesc(
			ns+"_ram_total_kb", "RAM total del sistema en KB", nil, nil,
		),
		freeRAM: prometheus.NewDesc(
			ns+"_ram_free_kb", "RAM libre del sistema en KB", nil, nil,
		),
		usedRAM: prometheus.NewDesc(
			ns+"_ram_used_kb", "RAM usada del sistema en KB", nil, nil,
		),
		procCount: prometheus.NewDesc(
			ns+"_process_count", "Número total de procesos", nil, nil,
		),
		procVSZ: prometheus.NewDesc(
			ns+"_process_vsz_kb", "Memoria virtual del proceso en KB", labels, nil,
		),
		procRSS: prometheus.NewDesc(
			ns+"_process_rss_kb", "RSS del proceso en KB", labels, nil,
		),
		procMemPct: prometheus.NewDesc(
			ns+"_process_memory_percent", "Porcentaje de memoria del proceso", labels, nil,
		),
		procCPUPct: prometheus.NewDesc(
			ns+"_process_cpu_percent", "Porcentaje de CPU del proceso", labels, nil,
		),
	}
}

// Describe envía los descriptores de métricas a Prometheus.
// (Copiado de lector.go — Clase 5)
func (c *SysInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalRAM
	ch <- c.freeRAM
	ch <- c.usedRAM
	ch <- c.procCount
	ch <- c.procVSZ
	ch <- c.procRSS
	ch <- c.procMemPct
	ch <- c.procCPUPct
}

// Collect lee /proc y envía los valores de cada métrica.
// Prometheus llama este método en cada scrape (cada 15s).
// (Copiado de lector.go — Clase 5, solo cambia procPath → cfg.ProcFile)
func (c *SysInfoCollector) Collect(ch chan<- prometheus.Metric) {
	// Leer y parsear el archivo /proc
	info, err := leerProcFile()
	if err != nil {
		log.Printf("[METRICS] Error en scrape: %v", err)
		return
	}

	// ── Métricas globales ───────────────────────────────────────
	ch <- prometheus.MustNewConstMetric(c.totalRAM, prometheus.GaugeValue, float64(info.TotalRAM))
	ch <- prometheus.MustNewConstMetric(c.freeRAM, prometheus.GaugeValue, float64(info.FreeRAM))
	ch <- prometheus.MustNewConstMetric(c.usedRAM, prometheus.GaugeValue, float64(info.UsedRAM))
	ch <- prometheus.MustNewConstMetric(c.procCount, prometheus.GaugeValue, float64(info.Procs))

	// ── Métricas por proceso ────────────────────────────────────
	for _, p := range info.Processes {
		pid := fmt.Sprintf("%d", p.PID)
		ch <- prometheus.MustNewConstMetric(c.procVSZ, prometheus.GaugeValue, float64(p.VSZ), pid, p.Name, p.Cmdline)
		ch <- prometheus.MustNewConstMetric(c.procRSS, prometheus.GaugeValue, float64(p.RSS), pid, p.Name, p.Cmdline)
		ch <- prometheus.MustNewConstMetric(c.procMemPct, prometheus.GaugeValue, p.MemoryUsage, pid, p.Name, p.Cmdline)
		ch <- prometheus.MustNewConstMetric(c.procCPUPct, prometheus.GaugeValue, p.CPUUsage, pid, p.Name, p.Cmdline)
	}
}

// iniciarServidorMetricas lanza el servidor HTTP en :9200 en una goroutine.
// No bloquea: el servidor corre en segundo plano mientras el loop principal continúa.
func iniciarServidorMetricas() {
	// Registrar el collector en Prometheus (igual que en lector.go)
	collector := NewSysInfoCollector()
	prometheus.MustRegister(collector)

	// Configurar rutas HTTP
	mux := http.NewServeMux()

	// /metrics → Prometheus lo consulta cada 15s (ver prometheus.yml)
	mux.Handle("/metrics", promhttp.Handler())

	// /health → útil para verificar que el daemon está vivo
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("daemon sopes1 ok"))
	})

	srv := &http.Server{
		Addr:         cfg.MetricsPort,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Lanzar en goroutine para no bloquear main()
	go func() {
		log.Printf("[METRICS] ✓ Servidor Prometheus en http://0.0.0.0%s/metrics", cfg.MetricsPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[METRICS] Error en servidor: %v", err)
		}
	}()
}
