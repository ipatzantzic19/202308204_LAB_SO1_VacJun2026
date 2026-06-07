package main

// ============================================================
//  docker_manager.go — Gestión de Contenedores Docker
//
//  Funciones para:
//   - Iniciar Grafana (compose up)
//   - Listar contenedores del proyecto
//   - Clasificar en "alto" / "bajo" consumo
//   - Cruzar con datos del módulo de kernel (PID)
//   - Eliminar contenedores excedentes
// ============================================================

import (
	"encoding/json"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────
// INICIAR GRAFANA
// ─────────────────────────────────────────────────────────────

// iniciarGrafana levanta el stack completo con docker compose.
// (El enunciado dice "crear un contenedor de Grafana al inicializar")
func iniciarGrafana(rutaCompose string) {
	log.Println("[DOCKER] Iniciando entorno Docker (Grafana + Valkey + Prometheus)...")

	out, err := exec.Command(
		"docker", "compose",
		"-f", rutaCompose,
		"up", "-d",
	).CombinedOutput()

	if err != nil {
		log.Printf("[DOCKER] Advertencia al levantar compose: %v\n%s", err, string(out))
	} else {
		log.Printf("[DOCKER] ✓ Entorno Docker iniciado.\n%s", strings.TrimSpace(string(out)))
	}
}

// ─────────────────────────────────────────────────────────────
// LISTAR Y CLASIFICAR CONTENEDORES
// ─────────────────────────────────────────────────────────────

// dockerPsEntry es el resultado de parsear una línea de docker ps
type dockerPsEntry struct {
	ID      string
	Nombre  string
	Imagen  string
	Comando string
}

// listarContenedoresProyecto retorna los contenedores activos
// que pertenecen al proyecto (prefijo "sopes1_") sin incluir
// los servicios del compose (grafana, valkey, etc.)
func listarContenedoresProyecto() ([]dockerPsEntry, error) {
	// Formato personalizado: campos separados por |
	out, err := exec.Command(
		"docker", "ps",
		"--filter", "name=sopes1_",
		"--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.Command}}",
	).Output()

	if err != nil {
		return nil, err
	}

	var resultado []dockerPsEntry
	lineas := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, linea := range lineas {
		if linea == "" {
			continue
		}
		partes := strings.SplitN(linea, "|", 4)
		if len(partes) < 4 {
			continue
		}
		resultado = append(resultado, dockerPsEntry{
			ID:      partes[0],
			Nombre:  partes[1],
			Imagen:  partes[2],
			Comando: partes[3],
		})
	}

	return resultado, nil
}

// obtenerPIDContenedor obtiene el PID del proceso principal
// de un contenedor via docker inspect.
func obtenerPIDContenedor(containerID string) int {
	out, err := exec.Command(
		"docker", "inspect",
		"--format", "{{.State.Pid}}",
		containerID,
	).Output()

	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0
	}
	return pid
}

// clasificarContenedor determina si un contenedor es de alto o bajo consumo.
// Reglas (según enunciado):
//   - roldyoran/go-client          → TipoAlto (alto RAM)
//   - alpine + comando con "bc"    → TipoAlto (alto CPU)
//   - alpine + comando con "sleep" → TipoBajo
func clasificarContenedor(entrada dockerPsEntry) TipoContenedor {
	imagen := strings.ToLower(entrada.Imagen)
	cmd := strings.ToLower(entrada.Comando)

	// Alto RAM: imagen go-client
	if strings.Contains(imagen, "go-client") || strings.Contains(imagen, "roldyoran") {
		return TipoAlto
	}

	// Alto CPU: alpine con bucle de cálculo
	if strings.Contains(cmd, "bc") || strings.Contains(cmd, "while") {
		return TipoAlto
	}

	// Bajo consumo: alpine con sleep
	return TipoBajo
}

// construirContainerInfo arma la estructura completa de un contenedor
// cruzando los datos de docker ps con los del módulo de kernel.
func construirContainerInfo(entrada dockerPsEntry, procesos []Process) ContainerInfo {
	info := ContainerInfo{
		ID:      entrada.ID,
		Nombre:  entrada.Nombre,
		Imagen:  entrada.Imagen,
		Comando: entrada.Comando,
		Tipo:    clasificarContenedor(entrada),
	}

	// Obtener el PID del proceso principal del contenedor
	pid := obtenerPIDContenedor(entrada.ID)
	info.PID = pid

	// Buscar ese PID en la lista de procesos del módulo de kernel
	if pid > 0 {
		if proc, encontrado := buscarProcesoPorPID(procesos, pid); encontrado {
			info.VSZ = proc.VSZ
			info.RSS = proc.RSS
			info.MemPct = proc.MemoryUsage
			info.CPU = proc.CPUUsage
		}
	}

	return info
}

// ─────────────────────────────────────────────────────────────
// GESTIÓN: mantener 3 bajos + 2 altos, matar el resto
// ─────────────────────────────────────────────────────────────

// gestionarContenedores aplica las reglas del enunciado:
//   - Siempre mantener MIN_BAJO contenedores de bajo consumo
//   - Siempre mantener MIN_ALTO contenedores de alto consumo
//   - Los excedentes se eliminan empezando por el de mayor consumo de RAM
//
// Retorna los contenedores que fueron eliminados (para guardar en Valkey).
func gestionarContenedores(contenedores []ContainerInfo) []ContainerInfo {
	var bajos []ContainerInfo
	var altos []ContainerInfo

	for _, c := range contenedores {
		if c.Tipo == TipoBajo {
			bajos = append(bajos, c)
		} else {
			altos = append(altos, c)
		}
	}

	log.Printf("[GESTIÓN] Activos → Bajos: %d (mínimo: %d) | Altos: %d (mínimo: %d)",
		len(bajos), cfg.MinBajos, len(altos), cfg.MinAltos)

	var eliminados []ContainerInfo

	// Ordenar por RSS descendente: los de MAYOR consumo van primero
	// → se eliminan los que más consumen, preservando los más eficientes
	ordenarPorRSS(bajos)
	ordenarPorRSS(altos)

	// Eliminar excedentes de bajos
	if len(bajos) > cfg.MinBajos {
		aEliminar := len(bajos) - cfg.MinBajos
		log.Printf("[GESTIÓN] Eliminando %d contenedor(es) bajo excedente(s)...", aEliminar)
		for i := 0; i < aEliminar; i++ {
			if eliminarContenedor(bajos[i], "excede_limite_bajo") {
				eliminados = append(eliminados, bajos[i])
			}
		}
	}

	// Eliminar excedentes de altos
	if len(altos) > cfg.MinAltos {
		aEliminar := len(altos) - cfg.MinAltos
		log.Printf("[GESTIÓN] Eliminando %d contenedor(es) alto excedente(s)...", aEliminar)
		for i := 0; i < aEliminar; i++ {
			if eliminarContenedor(altos[i], "excede_limite_alto") {
				eliminados = append(eliminados, altos[i])
			}
		}
	}

	if len(eliminados) == 0 {
		log.Println("[GESTIÓN] Nada que eliminar. Sistema en equilibrio.")
	}

	return eliminados
}

// eliminarContenedor ejecuta docker stop + docker rm.
// Retorna true si se eliminó exitosamente.
func eliminarContenedor(c ContainerInfo, razon string) bool {
	log.Printf("[ELIMINAR] → %s (imagen: %s, RSS: %d KB, razón: %s)",
		c.Nombre, c.Imagen, c.RSS, razon)

	// docker stop: envía SIGTERM y espera hasta 10s
	out, err := exec.Command("docker", "stop", c.ID).CombinedOutput()
	if err != nil {
		log.Printf("[ELIMINAR] ✗ Error en stop %s: %v\n%s", c.ID, err, string(out))
		return false
	}

	// docker rm: elimina el contenedor detenido
	out, err = exec.Command("docker", "rm", c.ID).CombinedOutput()
	if err != nil {
		log.Printf("[ELIMINAR] ✗ Error en rm %s: %v\n%s", c.ID, err, string(out))
		return false
	}

	log.Printf("[ELIMINAR] ✓ Eliminado: %s", c.Nombre)
	return true
}

// ordenarPorRSS ordena una lista de contenedores de mayor a menor RSS.
// (Insertion sort simple, la lista siempre es pequeña)
func ordenarPorRSS(lista []ContainerInfo) {
	for i := 1; i < len(lista); i++ {
		key := lista[i]
		j := i - 1
		for j >= 0 && lista[j].RSS < key.RSS {
			lista[j+1] = lista[j]
			j--
		}
		lista[j+1] = key
	}
}

// ─────────────────────────────────────────────────────────────
// HELPERS: obtener top 5 para Grafana
// ─────────────────────────────────────────────────────────────

// top5PorRAM retorna los 5 contenedores con mayor RSS.
func top5PorRAM(contenedores []ContainerInfo) []ContainerInfo {
	copia := make([]ContainerInfo, len(contenedores))
	copy(copia, contenedores)
	ordenarPorRSS(copia)
	if len(copia) > 5 {
		return copia[:5]
	}
	return copia
}

// top5PorCPU retorna los 5 contenedores con mayor CPU.
func top5PorCPU(contenedores []ContainerInfo) []ContainerInfo {
	copia := make([]ContainerInfo, len(contenedores))
	copy(copia, contenedores)
	// Ordenar por CPU descendente
	for i := 1; i < len(copia); i++ {
		key := copia[i]
		j := i - 1
		for j >= 0 && copia[j].CPU < key.CPU {
			copia[j+1] = copia[j]
			j--
		}
		copia[j+1] = key
	}
	if len(copia) > 5 {
		return copia[:5]
	}
	return copia
}

// ─────────────────────────────────────────────────────────────
// OBTENER LISTA COMPLETA CON DATOS DEL KERNEL
// ─────────────────────────────────────────────────────────────

// obtenerContenedoresConMetricas combina docker ps + kernel /proc
// en una sola lista de ContainerInfo enriquecida.
func obtenerContenedoresConMetricas(procesos []Process) ([]ContainerInfo, error) {
	entradas, err := listarContenedoresProyecto()
	if err != nil {
		return nil, err
	}

	var resultado []ContainerInfo
	for _, entrada := range entradas {
		ci := construirContainerInfo(entrada, procesos)
		resultado = append(resultado, ci)
	}

	log.Printf("[DOCKER] %d contenedor(es) del proyecto detectado(s).", len(resultado))
	return resultado, nil
}

// ─────────────────────────────────────────────────────────────
// SERIALIZACIÓN PARA LOGS
// ─────────────────────────────────────────────────────────────

// contenedorAJSON serializa un ContainerInfo para guardarlo en Valkey.
func contenedorAJSON(c ContainerInfo) string {
	data := map[string]interface{}{
		"id":        c.ID,
		"nombre":    c.Nombre,
		"imagen":    c.Imagen,
		"tipo":      string(c.Tipo),
		"rss_kb":    c.RSS,
		"vsz_kb":    c.VSZ,
		"mem_pct":   c.MemPct,
		"cpu_pct":   c.CPU,
		"timestamp": time.Now().Unix(),
	}
	b, _ := json.Marshal(data)
	return string(b)
}
