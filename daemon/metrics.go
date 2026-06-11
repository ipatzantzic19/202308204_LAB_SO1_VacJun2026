package main

// ============================================================
//  metrics.go — Servidor de Métricas Prometheus
//
//  Patrón idéntico al lector.go de Clase 5 del curso.
//  Expone métricas en http://localhost:9200/metrics
//
//  Métricas del sistema (de /proc):
//    sysinfo_ram_total_kb
//    sysinfo_ram_free_kb
//    sysinfo_ram_used_kb
//    sysinfo_process_count
//    sysinfo_process_vsz_kb        {pid, name, cmdline}
//    sysinfo_process_rss_kb        {pid, name, cmdline}
//    sysinfo_process_memory_percent {pid, name, cmdline}
//    sysinfo_process_cpu_percent   {pid, name, cmdline}
//
//  Métricas de contenedores (actualizadas en cada ciclo):
//    sopes1_containers_eliminated_total  ← counter acumulado
//    sopes1_active_containers_alto       ← gauge actual
//    sopes1_active_containers_bajo       ← gauge actual
// ============================================================

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ── Métricas de contenedores (globales, actualizadas desde main.go) ──────────
var (
	// Counter: se incrementa cada vez que el daemon elimina un contenedor
	metricEliminadosTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sopes1_containers_eliminated_total",
		Help: "Total acumulado de contenedores eliminados por el daemon",
	})

	// Gauges: cantidad actual de cada tipo
	metricContAltos = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sopes1_active_containers_alto",
		Help: "Contenedores de alto consumo activos actualmente",
	})
	metricContBajos = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sopes1_active_containers_bajo",
		Help: "Contenedores de bajo consumo activos actualmente",
	})
)

// actualizarMetricasContenedores es llamada desde main.go al final de cada ciclo.
func actualizarMetricasContenedores(contenedores []ContainerInfo, eliminadosEnCiclo int) {
	var altos, bajos float64
	for _, c := range contenedores {
		if c.Tipo == TipoAlto {
			altos++
		} else {
			bajos++
		}
	}
	metricContAltos.Set(altos)
	metricContBajos.Set(bajos)

	// Incrementar el counter por cada contenedor eliminado en este ciclo
	for i := 0; i < eliminadosEnCiclo; i++ {
		metricEliminadosTotal.Inc()
	}
}

// ── Collector del sistema (patrón de Clase 5 / lector.go) ────────────────────

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

func NewSysInfoCollector() *SysInfoCollector {
	labels := []string{"pid", "name", "cmdline"}
	ns := "sysinfo"
	return &SysInfoCollector{
		totalRAM:   prometheus.NewDesc(ns+"_ram_total_kb", "RAM total en KB", nil, nil),
		freeRAM:    prometheus.NewDesc(ns+"_ram_free_kb", "RAM libre en KB", nil, nil),
		usedRAM:    prometheus.NewDesc(ns+"_ram_used_kb", "RAM usada en KB", nil, nil),
		procCount:  prometheus.NewDesc(ns+"_process_count", "Total de procesos", nil, nil),
		procVSZ:    prometheus.NewDesc(ns+"_process_vsz_kb", "VSZ del proceso en KB", labels, nil),
		procRSS:    prometheus.NewDesc(ns+"_process_rss_kb", "RSS del proceso en KB", labels, nil),
		procMemPct: prometheus.NewDesc(ns+"_process_memory_percent", "% RAM del proceso", labels, nil),
		procCPUPct: prometheus.NewDesc(ns+"_process_cpu_percent", "% CPU del proceso", labels, nil),
	}
}

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

// Collect lee /proc y envía métricas a Prometheus en cada scrape.
// (Patrón idéntico al lector.go de Clase 5)
func (c *SysInfoCollector) Collect(ch chan<- prometheus.Metric) {
	info, err := leerProcFile()
	if err != nil {
		log.Printf("[METRICS] Error en scrape: %v", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalRAM, prometheus.GaugeValue, float64(info.TotalRAM))
	ch <- prometheus.MustNewConstMetric(c.freeRAM, prometheus.GaugeValue, float64(info.FreeRAM))
	ch <- prometheus.MustNewConstMetric(c.usedRAM, prometheus.GaugeValue, float64(info.UsedRAM))
	ch <- prometheus.MustNewConstMetric(c.procCount, prometheus.GaugeValue, float64(info.Procs))

	for _, p := range info.Processes {
		pid := fmt.Sprintf("%d", p.PID)
		ch <- prometheus.MustNewConstMetric(c.procVSZ, prometheus.GaugeValue, float64(p.VSZ), pid, p.Name, p.Cmdline)
		ch <- prometheus.MustNewConstMetric(c.procRSS, prometheus.GaugeValue, float64(p.RSS), pid, p.Name, p.Cmdline)
		ch <- prometheus.MustNewConstMetric(c.procMemPct, prometheus.GaugeValue, p.MemoryUsage, pid, p.Name, p.Cmdline)
		ch <- prometheus.MustNewConstMetric(c.procCPUPct, prometheus.GaugeValue, p.CPUUsage, pid, p.Name, p.Cmdline)
	}
}

// iniciarServidorMetricas lanza el servidor HTTP en :9200 en una goroutine.
func iniciarServidorMetricas() {
	prometheus.MustRegister(NewSysInfoCollector())

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
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

	go func() {
		log.Printf("[METRICS] ✓ Servidor Prometheus en http://0.0.0.0%s/metrics", cfg.MetricsPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[METRICS] Error: %v", err)
		}
	}()
}
