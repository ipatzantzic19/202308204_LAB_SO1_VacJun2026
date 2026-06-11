package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("╔══════════════════════════════════════════╗")
	log.Println("║    Daemon SOPES1 - Proyecto 1            ║")
	log.Println("║    Sistemas Operativos 1 - USAC 2026     ║")
	log.Println("╚══════════════════════════════════════════╝")
	log.Printf("[MAIN] ProcFile  : %s", cfg.ProcFile)
	log.Printf("[MAIN] Valkey    : %s", cfg.ValkeyAddr)
	log.Printf("[MAIN] Métricas  : http://localhost%s/metrics", cfg.MetricsPort)
	log.Printf("[MAIN] Intervalo : %v", cfg.LoopInterval)
	log.Println("")

	// PASO 1: Iniciar Grafana (docker compose up)
	log.Println("[MAIN] Paso 1/4 → Iniciando Grafana...")
	iniciarGrafana(cfg.RutaCompose)

	// PASO 2: Cargar módulo de kernel
	log.Println("[MAIN] Paso 2/4 → Cargando módulo de kernel...")
	cargarModuloKernel(cfg.RutaScriptModulo)

	if _, err := os.Stat(cfg.ProcFile); err != nil {
		log.Printf("[MAIN] ⚠ Advertencia: %s no existe todavía.", cfg.ProcFile)
	}

	// PASO 3: Registrar cronjob (patrón Clase 4)
	log.Println("[MAIN] Paso 3/4 → Registrando cronjob...")
	registrarCronjob(cfg.RutaScriptSpawn)

	// PASO 3.5: Iniciar servidor Prometheus (patrón Clase 5)
	log.Println("[MAIN] Paso 3.5/4 → Iniciando servidor Prometheus...")
	iniciarServidorMetricas()

	// PASO 3.6: Conectar a Valkey
	log.Println("[MAIN] Paso 3.6/4 → Conectando a Valkey...")
	if err := inicializarValkey(); err != nil {
		log.Printf("[MAIN] ⚠ Valkey no disponible: %v", err)
	}

	// PASO 4: Loop principal
	log.Printf("[MAIN] Paso 4/4 → Iniciando loop principal (cada %v)...", cfg.LoopInterval)
	log.Println("[MAIN] Presiona Ctrl+C para detener limpiamente.")
	log.Println("")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(cfg.LoopInterval)
	defer ticker.Stop()

	// Primer ciclo inmediato
	ejecutarCiclo()

	for {
		select {
		case <-ticker.C:
			ejecutarCiclo()
		case sig := <-sigChan:
			log.Printf("[MAIN] Señal recibida: %v. Apagando limpiamente...", sig)
			limpiarAlSalir()
			log.Println("[MAIN] ✓ Daemon detenido correctamente.")
			return
		}
	}
}

func ejecutarCiclo() {
	log.Println("────────────────────────────────────────────")
	log.Printf("[CICLO] Iniciando ciclo a las %s", time.Now().Format("15:04:05"))

	// 1. Leer /proc
	info, err := leerProcFile()
	if err != nil {
		log.Printf("[CICLO] ✗ Error leyendo /proc: %v", err)
		return
	}

	// 2. Obtener contenedores con métricas del kernel
	contenedores, err := obtenerContenedoresConMetricas(info.Processes)
	if err != nil {
		log.Printf("[CICLO] ✗ Error listando contenedores: %v", err)
	}

	// 3. Gestionar (mantener 3 bajos + 2 altos)
	var eliminados []ContainerInfo
	if len(contenedores) > 0 {
		eliminados = gestionarContenedores(contenedores)
	} else {
		log.Println("[CICLO] Sin contenedores del proyecto activos. Esperando al cronjob.")
	}

	// 4. Guardar en Valkey
	if rdb != nil {
		guardarCiclo(info, contenedores, eliminados)
		total := leerTotalEliminados()
		log.Printf("[CICLO] Total acumulado eliminados: %d", total)
	}

	// 5. Actualizar métricas Prometheus de contenedores ← NUEVO
	actualizarMetricasContenedores(contenedores, len(eliminados))

	log.Printf("[CICLO] ✓ Ciclo completado. Próximo en %v.", cfg.LoopInterval)
}

func limpiarAlSalir() {
	log.Println("[LIMPIEZA] Eliminando cronjob...")
	eliminarCronJob(cfg.RutaScriptSpawn)
	log.Println("[LIMPIEZA] ✓ Limpieza completada.")
}
