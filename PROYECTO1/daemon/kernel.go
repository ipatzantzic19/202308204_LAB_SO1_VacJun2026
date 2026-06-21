package main

// ============================================================
//  kernel.go — Gestión del Módulo de Kernel
//
//  El daemon carga el módulo al iniciar (si no está cargado)
//  y lo descarga al apagarse.
// ============================================================

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

// cargarModuloKernel carga el módulo de kernel usando el script de la Fase 1.
// Si el módulo ya está cargado, no hace nada.
func cargarModuloKernel(rutaScript string) {
	log.Println("[KERNEL] Verificando módulo de kernel...")

	// Si el módulo ya está cargado, no repetir
	if moduloCargado() {
		log.Println("[KERNEL] Módulo ya cargado. Continuando.")
		return
	}

	// Verificar que el script de carga existe
	if _, err := os.Stat(rutaScript); err != nil {
		log.Printf("[KERNEL] Script no encontrado en %s. Intentando insmod directo.", rutaScript)
		cargarDirecto()
		return
	}

	// Ejecutar el script load_module.sh de la Fase 1
	log.Printf("[KERNEL] Ejecutando: bash %s", rutaScript)
	out, err := exec.Command("bash", rutaScript).CombinedOutput()
	if err != nil {
		log.Printf("[KERNEL] Error ejecutando script: %v\n%s", err, string(out))
		return
	}

	log.Printf("[KERNEL] %s", strings.TrimSpace(string(out)))

	if moduloCargado() {
		log.Println("[KERNEL] ✓ Módulo cargado exitosamente.")
	} else {
		log.Println("[KERNEL] ✗ El módulo no aparece en lsmod después del script.")
	}
}

// cargarDirecto intenta cargar el módulo con insmod sin pasar por el script.
func cargarDirecto() {
	moduloPath := cfg.ModuloPath
	if _, err := os.Stat(moduloPath); err != nil {
		log.Printf("[KERNEL] ✗ Módulo .ko no encontrado en %s", moduloPath)
		return
	}

	out, err := exec.Command("sudo", "insmod", moduloPath).CombinedOutput()
	if err != nil {
		log.Printf("[KERNEL] ✗ Error con insmod: %v\n%s", err, string(out))
	} else {
		log.Println("[KERNEL] ✓ Módulo cargado con insmod directo.")
	}
}

// moduloCargado verifica con lsmod si el módulo está activo.
func moduloCargado() bool {
	out, err := exec.Command("lsmod").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "sysinfo_module") ||
		strings.Contains(string(out), "sys_info_module")
}
