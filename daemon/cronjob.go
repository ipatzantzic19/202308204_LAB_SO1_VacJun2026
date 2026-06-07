package main

// ============================================================
//  cronjob.go — Gestión del Cronjob
//
//  Patrón tomado DIRECTAMENTE del ejemplo del curso:
//  Clase 4/cronjob/main.go
//
//  El daemon registra el cronjob al iniciar y lo elimina
//  al recibir SIGTERM/SIGINT (apagado limpio).
// ============================================================

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// registrarCronjob hace el script ejecutable y lo agrega al crontab.
// Sigue exactamente el patrón de Clase 4/cronjob/main.go.
func registrarCronjob(rutaScript string) {
	hacerEjecutable(rutaScript)
	agregarCronJob(rutaScript)
	verificarCronJob(rutaScript)
}

// hacerEjecutable asigna permisos de ejecución al script.
// (Copiado de Clase 4/cronjob/main.go)
func hacerEjecutable(scriptPath string) {
	// 0755: dueño puede leer/escribir/ejecutar (7),
	//       grupos y otros solo leer/ejecutar (5)
	err := os.Chmod(scriptPath, 0755)
	if err != nil {
		log.Printf("[CRON] Advertencia: no se pudo chmod %s: %v", scriptPath, err)
		return
	}
	log.Printf("[CRON] Script %s marcado como ejecutable.", scriptPath)
}

// agregarCronJob registra el script en el crontab del usuario.
// (Copiado de Clase 4/cronjob/main.go, solo cambia la expresión cron)
func agregarCronJob(rutaScript string) {
	// Verificar que el script no esté ya registrado
	if cronjobExiste(rutaScript) {
		log.Println("[CRON] Cronjob ya registrado. No se duplica.")
		return
	}

	// "*/2 * * * *" → ejecutar cada 2 minutos (según enunciado del proyecto)
	expresionCron := "*/2 * * * *"

	// ">> archivo.log 2>&1" redirige stdout y stderr al log
	comandoCron := fmt.Sprintf("%s %s >> %s.log 2>&1",
		expresionCron, rutaScript, rutaScript)

	// Encadenar:
	// 1. crontab -l   → lista los cronjobs existentes
	// 2. echo "..."   → agrega la nueva línea
	// 3. crontab -    → escribe la nueva lista al crontab
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf(`(crontab -l 2>/dev/null; echo "%s") | crontab -`, comandoCron))

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[CRON] Error agregando cronjob: %v\n%s", err, string(output))
		return
	}

	log.Printf("[CRON] ✓ Cronjob registrado: %s", comandoCron)
}

// verificarCronJob imprime el crontab actual para confirmar el registro.
// (Copiado de Clase 4/cronjob/main.go)
func verificarCronJob(rutaScript string) {
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[CRON] No se pudo listar el crontab: %v", err)
		return
	}
	log.Printf("[CRON] === Crontab actual ===\n%s=== Fin ===", string(output))
}

// eliminarCronJob borra la entrada del crontab al apagarse el daemon.
// Se llama desde limpiarAlSalir() en main.go.
func eliminarCronJob(rutaScript string) {
	log.Println("[CRON] Eliminando cronjob...")

	// Filtrar las líneas que contienen el script y reescribir el crontab
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf(`crontab -l 2>/dev/null | grep -v "%s" | crontab -`, rutaScript))

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[CRON] Error eliminando cronjob: %v\n%s", err, string(output))
		return
	}

	log.Println("[CRON] ✓ Cronjob eliminado.")
}

// cronjobExiste comprueba si el script ya está en el crontab.
func cronjobExiste(rutaScript string) bool {
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), rutaScript)
}
