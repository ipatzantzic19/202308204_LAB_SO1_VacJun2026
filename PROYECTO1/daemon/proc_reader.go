package main

// ============================================================
//  proc_reader.go — Lectura del archivo /proc
//
//  Lee el JSON que expone el módulo de kernel y lo convierte
//  a structs de Go.
// ============================================================

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// leerProcFile lee el archivo /proc del módulo de kernel y retorna
// el snapshot parseado del sistema. Retorna error si el módulo
// no está cargado o si el JSON es inválido.
func leerProcFile() (*SysInfo, error) {
	// Leer el archivo completo como bytes (esperamos un JSON)
	raw, err := os.ReadFile(cfg.ProcFile)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer %s: %w\n"+
			"  ¿Está el módulo de kernel cargado? Verifica con: lsmod | grep sysinfo",
			cfg.ProcFile, err)
	}

	// Parsear el JSON a la estructura SysInfo
	var info SysInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("JSON inválido en %s: %w", cfg.ProcFile, err)
	}

	log.Printf("[PROC] Leído: %d KB total | %d KB usada | %d procesos",
		info.TotalRAM, info.UsedRAM, info.Procs)

	return &info, nil
}

// buscarProcesoPorPID busca un proceso en la lista por su PID.
// Retorna el proceso encontrado y true, o un proceso vacío y false.
func buscarProcesoPorPID(procesos []Process, pid int) (Process, bool) {
	for _, p := range procesos {
		if p.PID == pid {
			return p, true
		}
	}
	return Process{}, false
}
