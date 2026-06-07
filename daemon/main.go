package main

// ============================================================
//  main.go — Daemon Principal SOPES1 Proyecto 1
//
//  Estructura basada en daemon_basico/main.go de Clase 4:
//    1. Iniciar Grafana (docker compose up)
//    2. Cargar módulo de kernel
//    3. Registrar cronjob (patrón de Clase 4/cronjob/main.go)
//    4. Loop de lectura de /proc y gestión de contenedores
//
//  Uso:
//    sudo go run .          (en desarrollo)
//    sudo ./daemon_sopes1   (binario compilado)
//
//  *** CONFIGURA LAS RUTAS EN config.go ANTES DE EJECUTAR ***
// ============================================================

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

	// ── PASO 1: Iniciar Grafana (y el resto del compose) ─────────
	// (Equivalente al paso 1 de daemon_basico/main.go del curso)
	log.Println("[MAIN] Paso 1/4 → Iniciando Grafana...")
	iniciarGrafana(cfg.RutaCompose)

	// ── PASO 2: Cargar módulo de kernel ───────────────────────────
	// (Equivalente al paso 3 de daemon_basico/main.go del curso)
	log.Println("[MAIN] Paso 2/4 → Cargando módulo de kernel...")
	cargarModuloKernel(cfg.RutaScriptModulo)

	// Verificar que /proc existe antes de continuar
	if _, err := os.Stat(cfg.ProcFile); err != nil {
		log.Printf("[MAIN] ⚠ Advertencia: %s no existe todavía.", cfg.ProcFile)
		log.Println("[MAIN]   Verifica que el módulo esté cargado: lsmod | grep sysinfo")
		log.Println("[MAIN]   El daemon continuará pero los ciclos fallarán hasta que el módulo esté activo.")
	}

	// ── PASO 3: Registrar cronjob ──────────────────────────────────
	// (Equivalente al paso 2 de daemon_basico/main.go del curso)
	// Usa el patrón EXACTO de Clase 4/cronjob/main.go
	log.Println("[MAIN] Paso 3/4 → Registrando cronjob...")
	registrarCronjob(cfg.RutaScriptSpawn)

	// ── PASO 3.5: Iniciar servidor de métricas Prometheus ─────────
	// (Patrón de Clase 5/Go_lector_escritor/lector.go)
	log.Println("[MAIN] Paso 3.5/4 → Iniciando servidor Prometheus...")
	iniciarServidorMetricas()

	// ── PASO 3.6: Conectar a Valkey ────────────────────────────────
	log.Println("[MAIN] Paso 3.6/4 → Conectando a Valkey...")
	if err := inicializarValkey(); err != nil {
		log.Printf("[MAIN] ⚠ Valkey no disponible: %v", err)
		log.Println("[MAIN]   Continuando sin almacenamiento. Levanta Valkey con: docker compose up -d")
	}

	// ── PASO 4: Loop principal ─────────────────────────────────────
	// (Equivalente al paso 4 de daemon_basico/main.go del curso)
	log.Printf("[MAIN] Paso 4/4 → Iniciando loop principal (cada %v)...", cfg.LoopInterval)
	log.Println("[MAIN] Presiona Ctrl+C para detener limpiamente.")
	log.Println("")

	// Canal para recibir señales de apagado del sistema operativo
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Ticker que dispara el ciclo cada LoopInterval segundos
	ticker := time.NewTicker(cfg.LoopInterval)
	defer ticker.Stop()

	// Ejecutar el primer ciclo inmediatamente (sin esperar el primer tick)
	ejecutarCiclo()

	for {
		select {
		case <-ticker.C:
			// Cada LoopInterval segundos
			ejecutarCiclo()

		case sig := <-sigChan:
			// Señal de apagado recibida (Ctrl+C o systemctl stop)
			log.Printf("[MAIN] Señal recibida: %v. Apagando limpiamente...", sig)
			limpiarAlSalir()
			log.Println("[MAIN] ✓ Daemon detenido correctamente.")
			return
		}
	}
}

// ejecutarCiclo es el cuerpo del loop principal.
// Se llama cada LoopInterval segundos.
func ejecutarCiclo() {
	log.Println("────────────────────────────────────────────")
	log.Printf("[CICLO] Iniciando ciclo a las %s", time.Now().Format("15:04:05"))

	// ── 1. Leer el archivo /proc del módulo de kernel ─────────────
	info, err := leerProcFile()
	if err != nil {
		log.Printf("[CICLO] ✗ Error leyendo /proc: %v", err)
		log.Println("[CICLO] Saltando este ciclo.")
		return
	}

	// ── 2. Obtener contenedores del proyecto con métricas del kernel
	contenedores, err := obtenerContenedoresConMetricas(info.Processes)
	if err != nil {
		log.Printf("[CICLO] ✗ Error listando contenedores: %v", err)
		// No retornar: guardamos la memoria aunque fallen los contenedores
	}

	// ── 3. Aplicar reglas de gestión (mantener 3 bajos + 2 altos) ─
	var eliminados []ContainerInfo
	if len(contenedores) > 0 {
		eliminados = gestionarContenedores(contenedores)
	} else {
		log.Println("[CICLO] Sin contenedores del proyecto activos. Esperando al cronjob.")
	}

	// ── 4. Guardar todo en Valkey ──────────────────────────────────
	if rdb != nil {
		// Reconectar si Valkey estaba caído y volvió
		guardarCiclo(info, contenedores, eliminados)
		total := leerTotalEliminados()
		log.Printf("[CICLO] Total acumulado eliminados: %d", total)
	}

	log.Printf("[CICLO] ✓ Ciclo completado. Próximo en %v.", cfg.LoopInterval)
}

// limpiarAlSalir elimina el cronjob al apagar el daemon.
// Equivalente a lo descrito en el enunciado del proyecto.
func limpiarAlSalir() {
	log.Println("[LIMPIEZA] Eliminando cronjob...")
	eliminarCronJob(cfg.RutaScriptSpawn)
	log.Println("[LIMPIEZA] ✓ Limpieza completada.")
}
