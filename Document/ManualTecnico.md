# Manual Técnico — Proyecto 1 SOPES 1
**Sonda de Kernel en C y Daemon en Go para Telemetría de Contenedores**
Universidad San Carlos de Guatemala · Facultad de Ingeniería
Estudiante: 202308204 · Vacaciones Junio 2026

---

## Tabla de Contenidos

1. [Descripción General del Sistema](#1-descripción-general-del-sistema)
2. [Arquitectura del Sistema](#2-arquitectura-del-sistema)
3. [Fase 1 — Módulo de Kernel en C](#3-fase-1--módulo-de-kernel-en-c)
4. [Fase 2 — Entorno Docker](#4-fase-2--entorno-docker)
5. [Fase 3 — Script del Cronjob](#5-fase-3--script-del-cronjob)
6. [Fase 4 — Daemon en Go](#6-fase-4--daemon-en-go)
7. [Fase 5 — Dashboard en Grafana](#7-fase-5--dashboard-en-grafana)
8. [Flujo de Datos Completo](#8-flujo-de-datos-completo)
9. [Guía de Instalación](#9-guía-de-instalación)
10. [Estructura del Repositorio](#10-estructura-del-repositorio)
11. [Dependencias y Versiones](#11-dependencias-y-versiones)
12. [Restricciones y Reglas de Negocio](#12-restricciones-y-reglas-de-negocio)
13. [Solución de Problemas](#13-solución-de-problemas)

---

## 1. Descripción General del Sistema

El proyecto implementa un sistema integral de monitoreo y gestión autónoma de contenedores Docker en entornos Linux. El sistema opera en cuatro capas que se comunican entre sí:

- **Capa 0 (Kernel):** Un módulo C accede directamente a las estructuras internas del kernel Linux (`task_struct`, `si_meminfo`) para capturar métricas en tiempo real de memoria RAM y procesos. Los datos se exponen a través del sistema de archivos virtual `/proc`.

- **Capa 1 (Orquestación):** Un daemon escrito en Go lee periódicamente el archivo `/proc`, toma decisiones autónomas sobre qué contenedores mantener o eliminar, expone métricas para Prometheus y persiste logs en Valkey.

- **Capa 2 (Carga de trabajo simulada):** Un script de shell ejecutado por cron cada 2 minutos crea 5 contenedores Docker aleatorios de tres perfiles de consumo distintos.

- **Capa 3 (Visualización):** Grafana consulta Prometheus como fuente de datos y presenta 8 paneles interactivos con estado del sistema en tiempo real.

---

## 2. Arquitectura del Sistema

### 2.1 Diagrama de Arquitectura General

```mermaid
graph TB
    subgraph KERNEL["Kernel Space"]
        KM["Módulo C<br/>sysinfo_module.ko<br/>task_struct · si_meminfo"]
    end

    subgraph PROC["/proc filesystem"]
        PF["/proc/continfo_pr1_so1_202308204<br/>(JSON: RAM + Procesos)"]
    end

    subgraph HOST["User Space — Host Linux"]
        DG["Daemon Go<br/>:9200/metrics"]
        CRON["Cron<br/>*/2 * * * *"]
        SH["spawn_containers.sh"]
    end

    subgraph DOCKER["Docker Network: monitoring"]
        VK["Valkey<br/>:6379"]
        RE["redis_exporter<br/>:9121"]
        PR["Prometheus<br/>:9090"]
        GR["Grafana<br/>:3000"]
        C1["sopes1_*_1<br/>alpine sleep"]
        C2["sopes1_*_2<br/>go-client"]
        C3["sopes1_*_3<br/>alpine bc"]
    end

    KM -->|"expone JSON"| PF
    PF -->|"os.ReadFile()"| DG
    DG -->|"SET/ZADD/LPUSH"| VK
    DG -->|"Prometheus metrics"| PR
    VK -->|"métricas redis"| RE
    RE -->|"scrape :9121"| PR
    PR -->|"datasource"| GR
    DG -->|"docker stop/rm"| C1
    DG -->|"docker stop/rm"| C2
    DG -->|"docker stop/rm"| C3
    CRON -->|"ejecuta cada 2min"| SH
    SH -->|"docker run"| C1
    SH -->|"docker run"| C2
    SH -->|"docker run"| C3
    DG -->|"registra/elimina"| CRON
```

### 2.2 Diagrama de Componentes

```mermaid
graph LR
    subgraph KERNEL["Fase 1: Módulo Kernel"]
        A1["sysinfo_module.c"]
        A2["Makefile"]
        A3["load_module.sh"]
    end

    subgraph DOCKER_ENV["Fase 2: Docker"]
        B1["docker-compose.yml"]
        B2["prometheus.yml"]
    end

    subgraph CRON_SCRIPT["Fase 3: Cronjob"]
        C1["spawn_containers.sh"]
    end

    subgraph DAEMON["Fase 4: Daemon Go"]
        D1["main.go"]
        D2["config.go"]
        D3["types.go"]
        D4["kernel.go"]
        D5["cronjob.go"]
        D6["proc_reader.go"]
        D7["docker_manager.go"]
        D8["valkey_client.go"]
        D9["metrics.go"]
    end

    subgraph GRAFANA_DASH["Fase 5: Grafana"]
        E1["dashboard.json"]
        E2["setup_grafana.sh"]
    end

    A1 --> D6
    A3 --> D4
    B1 --> D1
    C1 --> D5
    D1 --> D4
    D1 --> D5
    D1 --> D6
    D1 --> D7
    D1 --> D8
    D1 --> D9
    D2 --> D1
    D3 --> D6
    D3 --> D7
    D8 --> E1
    D9 --> E1
```

### 2.3 Diagrama de Secuencia — Ciclo de Vida del Daemon

```mermaid
sequenceDiagram
    participant M as main.go
    participant K as kernel.go
    participant CR as cronjob.go
    participant MT as metrics.go
    participant VK as valkey_client.go
    participant PR as proc_reader.go
    participant DM as docker_manager.go

    Note over M: Inicio del daemon
    M->>K: iniciarGrafana(rutaCompose)
    K-->>M: docker compose up -d ✓
    M->>K: cargarModuloKernel(rutaScript)
    K-->>M: módulo cargado ✓
    M->>CR: registrarCronjob(rutaScript)
    CR-->>M: crontab actualizado ✓
    M->>MT: iniciarServidorMetricas()
    MT-->>M: goroutine :9200 activa ✓
    M->>VK: inicializarValkey()
    VK-->>M: PING → PONG ✓

    loop Cada 30 segundos
        M->>PR: leerProcFile()
        PR-->>M: SysInfo{RAM, Procesos[]}
        M->>DM: obtenerContenedoresConMetricas(procesos)
        DM-->>M: []ContainerInfo{ID,PID,RSS,CPU,Tipo}
        M->>DM: gestionarContenedores(contenedores)
        DM-->>M: []eliminados
        M->>VK: guardarCiclo(info, contenedores, eliminados)
        VK-->>M: persistido en Valkey ✓
        M->>MT: actualizarMetricasContenedores(contenedores, nEliminados)
    end

    Note over M: Ctrl+C / SIGTERM
    M->>CR: eliminarCronJob(rutaScript)
    CR-->>M: crontab limpiado ✓
```

---

## 3. Fase 1 — Módulo de Kernel en C

### 3.1 Descripción

El módulo `sysinfo_module` (compilado como `sys_info_module.ko`) es un Loadable Kernel Module (LKM) que se inyecta en el kernel Linux en tiempo de ejecución. Al cargarse, crea el archivo `/proc/continfo_pr1_so1_202308204` que retorna un JSON con el estado completo del sistema.

### 3.2 Archivo `/proc` — Formato JSON

```json
{
  "Totalram": 12039728,
  "Freeram": 564944,
  "Usedram": 11474784,
  "Procs": 419,
  "Processes": [
    {
      "PID": 1,
      "Name": "systemd",
      "Cmdline": "/sbin/init splash",
      "vsz": 102400,
      "rss": 8192,
      "Memory_Usage": 0.1,
      "CPU_Usage": 0.00
    }
  ]
}
```

Todos los valores de memoria están en **KB**. `Memory_Usage` es porcentaje con 1 decimal. `CPU_Usage` es porcentaje acumulado con 2 decimales (puede exceder 100% por diseño del kernel).

### 3.3 Flujo Interno del Módulo

```mermaid
flowchart TD
    A["insmod sys_info_module.ko"] --> B["sysinfo_init()"]
    B --> C["proc_create('continfo_pr1_so1_202308204', 0444, NULL, &sysinfo_ops)"]
    C --> D["Archivo /proc creado"]

    E["cat /proc/continfo_pr1_so1_202308204"] --> F["sysinfo_open()"]
    F --> G["single_open() → sysinfo_show()"]
    G --> H["si_meminfo(&si) → total/libre/usada en KB"]
    H --> I["rcu_read_lock()"]
    I --> J["for_each_process(task)"]
    J --> K{"task->mm != NULL?"}
    K -->|"Sí (proceso usuario)"| L["VSZ = total_vm << PAGE_SHIFT-10<br/>RSS = get_mm_rss(mm) << PAGE_SHIFT-10"]
    K -->|"No (thread kernel)"| M["vsz=0, rss=0"]
    L --> N["get_process_cmdline(task)"]
    M --> N
    N --> O["cpu = utime+stime×10000/jiffies/CPUs"]
    O --> P["seq_printf → JSON del proceso"]
    P --> J
    J --> Q["rcu_read_unlock()"]
    Q --> R["JSON completo al usuario"]

    S["rmmod sys_info_module"] --> T["sysinfo_exit()"]
    T --> U["remove_proc_entry()"]
```

### 3.4 Decisiones de Diseño

| Decisión | Razón |
|---|---|
| `rcu_read_lock/unlock` | Protege el acceso concurrente a la lista `task_struct` sin bloquear el scheduler |
| `PAGE_SHIFT - 10` para VSZ/RSS | Convierte páginas (4096 bytes) a KB: `× 4096 / 1024 = × 4 = << 2` |
| `get_task_comm()` en vez de `task->comm` directamente | Evita condición de carrera al leer el nombre del proceso |
| `kmalloc / kfree` para cmdline | Asignación dinámica segura en el espacio del kernel |
| `sanitize_for_json()` | Reemplaza `"` y `\` para garantizar JSON válido aunque el nombre de proceso contenga caracteres especiales |
| `single_open` + `seq_file` | Patrón estándar del kernel para archivos `/proc` que generan salida variable |

### 3.5 Comandos del Makefile

| Comando | Función |
|---|---|
| `make` | Compila → genera `sys_info_module.ko` |
| `make load` | Compila + `sudo insmod` |
| `make unload` | `sudo rmmod` |
| `make reload` | `rmmod` + `make` + `insmod` |
| `make test` | `cat /proc/... | python3 -m json.tool` |
| `make log` | `dmesg | grep SOPES1` |
| `make status` | Estado actual en `lsmod` |

---

## 4. Fase 2 — Entorno Docker

### 4.1 Servicios del Docker Compose

```mermaid
graph LR
    subgraph NET["Red: monitoring (bridge)"]
        VK["valkey:6379<br/>Base de datos en memoria<br/>compatible Redis"]
        RE["redis_exporter:9121<br/>Traduce métricas Valkey<br/>a formato Prometheus"]
        PR["prometheus:9090<br/>Recolecta y almacena<br/>series de tiempo"]
        GR["grafana:3000<br/>Dashboards de<br/>visualización"]

        VK -->|"REDIS_ADDR=valkey:6379"| RE
        RE -->|"scrape /metrics"| PR
        PR -->|"datasource"| GR
    end

    HOST["Daemon Go<br/>host:9200/metrics"] -->|"host.docker.internal:9200"| PR
    USER["Usuario<br/>navegador"] -->|"HTTP :3000"| GR
```

### 4.2 Imágenes de Contenedores de Prueba

| Categoría | Imagen | Comando | Perfil |
|---|---|---|---|
| Alto RAM | `roldyoran/go-client` | `docker run -d roldyoran/go-client` | Consumo significativo de RAM |
| Alto CPU | `alpine` | `docker run -d alpine sh -c "while true; do echo '2^20' \| bc > /dev/null; sleep 2; done"` | Bucle matemático intensivo |
| Bajo consumo | `alpine` | `docker run -d alpine sleep 240` | Inactivo durante 4 minutos |

### 4.3 Configuración de Prometheus (`prometheus.yml`)

Dos targets de scraping cada 15 segundos:

- `valkey` → `redis_exporter:9121` (métricas de la base de datos)
- `daemon_go` → `host.docker.internal:9200` (métricas del sistema vía daemon)

La directiva `extra_hosts: host.docker.internal:host-gateway` en el compose permite que Prometheus dentro de Docker alcance el daemon que corre en el host.

---

## 5. Fase 3 — Script del Cronjob

### 5.1 Lógica del Script

```mermaid
flowchart TD
    A["spawn_containers.sh ejecutado por cron"] --> B["i = 1"]
    B --> C["TIPO = RANDOM % 3"]
    C --> D["NOMBRE = sopes1_timestamp_nanosegundos_i"]
    D --> E{TIPO}
    E -->|"0"| F["docker run -d roldyoran/go-client<br/>(ALTO_RAM)"]
    E -->|"1"| G["docker run -d alpine 'bc loop'<br/>(ALTO_CPU)"]
    E -->|"2"| H["docker run -d alpine sleep 240<br/>(BAJO)"]
    F --> I["log → spawn_containers_internal.log"]
    G --> I
    H --> I
    I --> J{"i < 5?"}
    J -->|"Sí"| K["i++"]
    K --> C
    J -->|"No"| L["Resumen en log"]
```

### 5.2 Registro del Cronjob desde Go

El daemon registra y elimina el cronjob programáticamente:

```
Registro:   (crontab -l 2>/dev/null; echo "*/2 * * * * /ruta/spawn_containers.sh >> ...log 2>&1") | crontab -
Eliminación: crontab -l 2>/dev/null | grep -v "spawn_containers.sh" | crontab -
```

El patrón cron `*/2 * * * *` ejecuta el script exactamente cada 2 minutos.

---

## 6. Fase 4 — Daemon en Go

### 6.1 Estructura de Archivos

| Archivo | Responsabilidad |
|---|---|
| `config.go` | Constantes de configuración (rutas, puertos, intervalos) |
| `types.go` | Estructuras de datos: `Process`, `SysInfo`, `ContainerInfo`, `TipoContenedor` |
| `main.go` | Punto de entrada, secuencia de inicialización, loop principal |
| `kernel.go` | Carga del módulo kernel (script + fallback directo) |
| `cronjob.go` | Registro y eliminación del cronjob en crontab |
| `proc_reader.go` | Lectura y parseo del JSON de `/proc` |
| `docker_manager.go` | Listado, clasificación y eliminación de contenedores |
| `valkey_client.go` | Persistencia de métricas en Valkey (Redis protocol) |
| `metrics.go` | Servidor HTTP Prometheus en `:9200` |

### 6.2 Lógica de Gestión de Contenedores

```mermaid
flowchart TD
    A["ejecutarCiclo()"] --> B["leerProcFile() → SysInfo"]
    B --> C["listarContenedoresProyecto()<br/>docker ps --filter name=sopes1_"]
    C --> D["Para cada contenedor:<br/>obtenerPIDContenedor() via docker inspect"]
    D --> E["buscarProcesoPorPID() en SysInfo.Processes"]
    E --> F["clasificarContenedor() → TipoAlto / TipoBajo"]
    F --> G["Separar en: []bajos y []altos"]
    G --> H["ordenarPorRSS() descendente en cada grupo"]
    H --> I{"len(bajos) > 3?"}
    I -->|"Sí"| J["Eliminar bajos[0..n-3]<br/>(mayor consumo primero)"]
    I -->|"No"| K{"len(altos) > 2?"}
    J --> K
    K -->|"Sí"| L["Eliminar altos[0..n-2]<br/>(mayor consumo primero)"]
    K -->|"No"| M["Sistema en equilibrio"]
    L --> N["guardarCiclo() en Valkey"]
    M --> N
    N --> O["actualizarMetricasContenedores()"]
```

### 6.3 Clasificación de Contenedores

| Imagen / Comando | Tipo asignado | Criterio |
|---|---|---|
| Contiene `go-client` o `roldyoran` | `TipoAlto` | Imagen de alto RAM |
| Contiene `bc` o `while` en comando | `TipoAlto` | Alpine con bucle CPU |
| Cualquier otro (alpine sleep) | `TipoBajo` | Bajo consumo por defecto |

### 6.4 Métricas Expuestas en Prometheus (`:9200/metrics`)

**Métricas del sistema (colectadas en cada scrape desde `/proc`):**

| Métrica | Tipo | Descripción |
|---|---|---|
| `sysinfo_ram_total_kb` | Gauge | RAM total en KB |
| `sysinfo_ram_free_kb` | Gauge | RAM libre en KB |
| `sysinfo_ram_used_kb` | Gauge | RAM usada en KB |
| `sysinfo_process_count` | Gauge | Total de procesos activos |
| `sysinfo_process_vsz_kb{pid,name,cmdline}` | Gauge | VSZ por proceso |
| `sysinfo_process_rss_kb{pid,name,cmdline}` | Gauge | RSS por proceso |
| `sysinfo_process_memory_percent{pid,name,cmdline}` | Gauge | % RAM por proceso |
| `sysinfo_process_cpu_percent{pid,name,cmdline}` | Gauge | % CPU por proceso |

**Métricas de contenedores (actualizadas por el daemon en cada ciclo):**

| Métrica | Tipo | Descripción |
|---|---|---|
| `sopes1_containers_eliminated_total` | Counter | Total acumulado de eliminaciones |
| `sopes1_active_containers_alto` | Gauge | Contenedores altos activos |
| `sopes1_active_containers_bajo` | Gauge | Contenedores bajos activos |

### 6.5 Claves en Valkey

| Clave | Tipo | Contenido | TTL |
|---|---|---|---|
| `memoria:actual` | String (JSON) | Snapshot de RAM actual | Sin expiración |
| `memoria:historia` | List (JSON) | Historial de snapshots (máx 1000) | Sin expiración |
| `contenedor:{id}` | String (JSON) | Estado de cada contenedor | 1 hora |
| `ranking:ram` | Sorted Set | Score=RSS, Member=nombre | Sin expiración |
| `ranking:cpu` | Sorted Set | Score=CPU%, Member=nombre | Sin expiración |
| `eliminados:log` | List (JSON) | Registro de eliminaciones (máx 1000) | Sin expiración |
| `eliminados:total` | String (int) | Contador global de eliminaciones | Sin expiración |
| `eliminado:{nano}` | String (JSON) | Registro individual con timestamp | 24 horas |

---

## 7. Fase 5 — Dashboard en Grafana

### 7.1 Paneles del Dashboard

```mermaid
graph TD
    subgraph ROW1["Fila 1: Métricas de RAM"]
        P1["RAM Total<br/>Stat (azul)<br/>sysinfo_ram_total_kb / 1024"]
        P2["RAM en Uso<br/>Stat (semáforo)<br/>sysinfo_ram_used_kb / 1024"]
        P3["Memoria Libre<br/>Stat (semáforo inverso)<br/>sysinfo_ram_free_kb / 1024"]
        P4["% RAM Usada<br/>Gauge (0-100%)<br/>used/total * 100"]
    end

    subgraph ROW2["Fila 2: Evolución Temporal"]
        P5["Uso de RAM a lo largo del tiempo<br/>Time Series<br/>RAM Usada (rojo) + RAM Libre (verde)"]
        P6["Contenedores Eliminados en el Tiempo<br/>Time Series<br/>Eliminados(2m) + Altos Activos + Bajos Activos"]
    end

    subgraph ROW3["Fila 3: Top Consumidores"]
        P7["Top 5 por Consumo de RAM<br/>Pie Chart<br/>topk(5, max by name sysinfo_process_rss_kb)"]
        P8["Top 5 por Consumo de CPU<br/>Pie Chart<br/>topk(5, max by name sysinfo_process_cpu_percent)"]
    end
```

### 7.2 Queries PromQL Principales

```
# Métricas de RAM (instant)
sysinfo_ram_total_kb / 1024           → MB totales
sysinfo_ram_used_kb / 1024            → MB en uso
sysinfo_ram_free_kb / 1024            → MB libres
sysinfo_ram_used_kb / sysinfo_ram_total_kb * 100  → % uso

# Series de tiempo (range)
sysinfo_ram_used_kb / 1024            → evolución RAM usada
sysinfo_ram_free_kb / 1024            → evolución RAM libre
increase(sopes1_containers_eliminated_total[2m])  → eliminados en ventana 2m
sopes1_active_containers_alto         → altos activos actuales
sopes1_active_containers_bajo         → bajos activos actuales

# Top 5 consumidores (instant)
topk(5, max by (name) (sysinfo_process_rss_kb)) / 1024
topk(5, max by (name) (sysinfo_process_cpu_percent))
```

---

## 8. Flujo de Datos Completo

```mermaid
sequenceDiagram
    participant K as Kernel<br/>(task_struct)
    participant P as /proc/continfo
    participant D as Daemon Go
    participant V as Valkey
    participant PM as Prometheus
    participant G as Grafana

    Note over K,G: Cada 30 segundos (ciclo del daemon)

    K->>P: Escribe JSON (RAM + Procesos)
    D->>P: os.ReadFile() → json.Unmarshal()
    P-->>D: SysInfo{TotalRAM, FreeRAM, Processes[]}
    D->>D: docker ps → docker inspect → clasificar
    D->>D: ordenar por RSS → eliminar excedentes
    D->>V: SET memoria:actual
    D->>V: LPUSH memoria:historia
    D->>V: ZAdd ranking:ram / ranking:cpu
    D->>V: LPUSH eliminados:log + INCR eliminados:total

    Note over PM,G: Cada 15 segundos (scrape de Prometheus)

    PM->>D: GET :9200/metrics
    D-->>PM: sysinfo_ram_* + sopes1_containers_*
    PM->>V: redis_exporter GET :9121/metrics
    G->>PM: PromQL queries
    PM-->>G: series de tiempo
    G->>G: Renderiza 8 paneles del dashboard
```

---

## 9. Guía de Instalación

### 9.1 Requisitos Previos

```bash
# Sistema operativo
uname -r    # Linux kernel >= 5.15

# Herramientas de compilación
sudo apt update
sudo apt install -y linux-headers-$(uname -r) build-essential gcc make

# Go 1.21+
go version

# Docker + Docker Compose
docker --version
docker compose version

# Cron activo
systemctl status cron
```

### 9.2 Orden de Instalación

```mermaid
flowchart LR
    A["1. Clonar repositorio"] --> B["2. Compilar módulo kernel<br/>cd kernel_module && make"]
    B --> C["3. Cargar módulo<br/>sudo insmod sys_info_module.ko"]
    C --> D["4. Crear red Docker<br/>docker network create monitoring"]
    D --> E["5. Levantar stack<br/>cd docker && bash setup_entorno.sh"]
    E --> F["6. Ajustar config.go<br/>con rutas absolutas"]
    F --> G["7. Compilar daemon<br/>cd daemon && go mod tidy && go build"]
    G --> H["8. Ejecutar daemon<br/>sudo ./daemon_sopes1"]
    H --> I["9. Esperar 2-4 minutos<br/>para que el cronjob genere datos"]
    I --> J["10. Abrir Grafana<br/>http://localhost:3000"]
```

### 9.3 Verificación por Fase

```bash
# Fase 1 — Módulo kernel
lsmod | grep sys_info_module
cat /proc/continfo_pr1_so1_202308204 | python3 -m json.tool

# Fase 2 — Docker stack
docker compose -f docker/docker-compose.yml ps

# Fase 3 — Script (prueba manual)
bash scripts/spawn_containers.sh
docker ps --filter "name=sopes1_"

# Fase 4 — Daemon y métricas
curl http://localhost:9200/metrics | grep sysinfo_ram
curl http://localhost:9200/health

# Fase 5 — Grafana targets
# Abrir: http://localhost:9090/targets
# Verificar: daemon_go → UP, valkey → UP
```

---

## 10. Estructura del Repositorio

```
202308204_LAB_SO1_VacJun2026/
├── kernel_module/
│   ├── sys_info_module.c       ← Código fuente del módulo
│   ├── Makefile                ← Build system del kernel
│   ├── load_module.sh          ← Script de carga segura
│   ├── unload_module.sh        ← Script de descarga
│   └── test_module.sh          ← Suite de pruebas automáticas
├── docker/
│   ├── docker-compose.yml      ← Stack: Valkey + redis_exporter + Prometheus + Grafana
│   ├── prometheus.yml          ← Configuración de scraping
│   ├── setup_entorno.sh        ← Instalación completa (primera vez)
│   ├── test_entorno.sh         ← Suite de pruebas del stack
│   └── test_imagenes.sh        ← Prueba de los 3 tipos de contenedor
├── scripts/
│   ├── spawn_containers.sh     ← Script del cronjob (crea 5 contenedores)
│   └── test_spawn.sh           ← Suite de pruebas del script
├── daemon/
│   ├── config.go               ← Configuración central (EDITAR rutas)
│   ├── types.go                ← Estructuras de datos
│   ├── main.go                 ← Punto de entrada + loop principal
│   ├── kernel.go               ← Gestión del módulo kernel
│   ├── cronjob.go              ← Gestión del crontab
│   ├── proc_reader.go          ← Lectura de /proc
│   ├── docker_manager.go       ← Gestión de contenedores Docker
│   ├── valkey_client.go        ← Cliente Valkey (go-redis)
│   ├── metrics.go              ← Servidor Prometheus :9200
│   ├── go.mod
│   └── go.sum
├── grafana/
│   ├── dashboard.json          ← Dashboard importable
│   └── setup_grafana.sh        ← Configuración automática vía API
├── SKILL/                      ← Documentación de referencia del proyecto
├── README.md
└── ESTADO_VERIFICACION.md
```

---

## 11. Dependencias y Versiones

| Componente | Versión | Uso |
|---|---|---|
| Linux Kernel | >= 5.15 | API del módulo (proc_ops, task_struct) |
| GCC | cualquier | Compilación del módulo C |
| Go | 1.21 | Compilación del daemon |
| github.com/redis/go-redis/v9 | v9.5.1 | Cliente Valkey desde Go |
| github.com/prometheus/client_golang | v1.19.1 | Servidor de métricas |
| Docker | >= 24 | Contenedores |
| Docker Compose | >= 2.0 | Orquestación del stack |
| Valkey | latest | Base de datos en memoria |
| redis_exporter (oliver006) | latest | Bridge Valkey → Prometheus |
| Prometheus | latest | Almacenamiento de series de tiempo |
| Grafana | latest | Visualización |

---

## 12. Restricciones y Reglas de Negocio

```mermaid
graph TD
    A["Ciclo del daemon detecta<br/>contenedores del proyecto<br/>(prefijo sopes1_*)"] --> B["¿Cuántos bajos hay?"]
    B -->|"> 3"| C["Ordenar bajos por RSS desc<br/>Eliminar los de mayor consumo<br/>Preservar los 3 de menor consumo"]
    B -->|"= 3 o < 3"| D["No eliminar ningún bajo"]
    A --> E["¿Cuántos altos hay?"]
    E -->|"> 2"| F["Ordenar altos por RSS desc<br/>Eliminar los de mayor consumo<br/>Preservar los 2 de menor consumo"]
    E -->|"= 2 o < 2"| G["No eliminar ningún alto"]
    C --> H["NUNCA eliminar contenedor de Grafana"]
    D --> H
    F --> H
    G --> H
```

**Reglas absolutas:**

1. Siempre deben existir exactamente **3 contenedores de bajo consumo** activos.
2. Siempre deben existir exactamente **2 contenedores de alto consumo** activos.
3. El contenedor de Grafana (parte del compose) **nunca se elimina**.
4. Los contenedores del proyecto siempre llevan el prefijo `sopes1_`.
5. El criterio de eliminación es: mayor consumo de RAM (RSS) es eliminado primero.
6. El daemon elimina el cronjob antes de apagarse (limpieza al recibir `SIGTERM`/`SIGINT`).

---

## 13. Solución de Problemas

### 13.1 Árbol de Diagnóstico

```mermaid
flowchart TD
    A["Problema reportado"] --> B{"¿El módulo está cargado?"}
    B -->|"No"| C["sudo insmod kernel_module/sys_info_module.ko<br/>sudo dmesg | grep SOPES1"]
    B -->|"Sí"| D{"¿/proc existe?"}
    D -->|"No"| E["make reload en kernel_module/"]
    D -->|"Sí"| F{"¿El daemon corre?"}
    F -->|"No"| G["sudo ./daemon_sopes1<br/>Verificar config.go rutas"]
    F -->|"Sí"| H{"¿Grafana muestra datos?"}
    H -->|"No"| I{"¿Targets en Prometheus UP?"}
    I -->|"daemon_go DOWN"| J["curl localhost:9200/health<br/>¿Puerto 9200 libre?"]
    I -->|"valkey DOWN"| K["docker compose ps<br/>docker compose restart redis_exporter"]
    I -->|"Ambos UP"| L["Esperar al menos 30s<br/>para que Prometheus acumule datos"]
    H -->|"Sí"| M["Sistema funcionando correctamente ✓"]
```

### 13.2 Errores Comunes y Soluciones

| Error | Causa | Solución |
|---|---|---|
| `insmod: ERROR: could not insert module` | Módulo desactualizado o formato incorrecto | `make clean && make && sudo insmod sys_info_module.ko` |
| `no se pudo leer /proc/continfo_...` | Módulo no cargado | `sudo insmod sys_info_module.ko` |
| `no se pudo conectar a Valkey` | Stack Docker no corre | `docker compose -f docker/docker-compose.yml up -d` |
| `permission denied` al cargar módulo | Falta sudo | Ejecutar daemon con `sudo go run .` |
| Target `daemon_go` DOWN en Prometheus | Puerto 9200 no accesible desde Docker | Verificar `extra_hosts: host.docker.internal:host-gateway` en compose |
| Paneles Grafana sin datos | Prometheus aún no tiene métricas | Esperar 30-60s después de iniciar el daemon |
| `tee: /tmp/spawn_containers.log: Permiso denegado` | Cron ejecuta el script sin permisos sobre `/tmp` | El script usa `spawn_containers_internal.log` en su propio directorio |
| JSON inválido en `/proc` | Caracteres especiales en nombre de proceso | `sanitize_for_json()` los reemplaza — verificar con `make test` |