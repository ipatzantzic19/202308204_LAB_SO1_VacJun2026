# -*- coding: utf-8 -*-Planificación del Proyecto 1 — SOPES 1
**Nombre: Sonda de Kernel en C y Daemon en Go para Telemetría de Contenedores**
**Estudiante: [TU_NOMBRE_AQUÍ]**
Universidad San Carlos de Guatemala | Vacaciones de Junio 2026 | Ponderación: 40 pts

---

## 🎯 Objetivo General

Desarrollar un sistema integral de monitoreo y gestión autónoma de contenedores Docker compuesto por:
- Un **módulo de kernel en C** que capture métricas del sistema.
- Un **Daemon en Go** que procese esas métricas y tome decisiones.
- Un **Cronjob** que simule carga de trabajo con contenedores.
- Un **Dashboard en Grafana** para visualización en tiempo real.

---

## 📦 Entregables Finales

| Entregable | Descripción |
|---|---|
| Repositorio GitHub | `Carnet#_LAB_P1_SO1_VacJun2026` (privado, colaborador: CamiloSincal) |
| Módulo Kernel (C) | Archivo `.c` + `Makefile` |
| Daemon Go | Código fuente del servicio principal |
| Docker Compose | Archivo para Grafana + Valkey |
| Script Cronjob | Shell script de generación de contenedores |
| Manual Técnico | Guía de instalación y documentación |
| Evidencia Funcional | Capturas de pantalla del sistema funcionando |

---

## 🗂️ Fases del Proyecto

### FASE 1 — Módulo de Kernel en C + Interfaz /proc
**Objetivo:** Crear el sensor de bajo nivel que expone métricas del sistema.

**Tareas:**
- [ ] Instalar herramientas de desarrollo del kernel (`linux-headers`, `build-essential`).
- [ ] Crear el archivo `sysinfo_module.c` con las funciones básicas de init/exit.
- [ ] Implementar lectura de métricas de **memoria RAM** (total, libre, en uso) usando `si_meminfo`.
- [ ] Implementar iteración por procesos usando `task_struct` para capturar: PID, Nombre, Cmdline, VSZ, RSS, %CPU, %Mem.
- [ ] Crear la entrada `/proc/continfo_pr1_so1_#CARNET` que retorna un JSON.
- [ ] Crear el `Makefile` para compilar el módulo.
- [ ] Probar carga (`insmod`) y descarga (`rmmod`) del módulo.
- [ ] Verificar lectura del archivo: `cat /proc/continfo_pr1_so1_#CARNET`.

**Criterio de éxito:** El archivo `/proc` retorna datos válidos en JSON con memoria y procesos.

---

### FASE 2 — Entorno Docker: Imágenes + Compose
**Objetivo:** Preparar el ambiente donde correrá el sistema.

**Tareas:**
- [ ] Crear la imagen Docker de **alto consumo de RAM** (`go-client` o usar `roldyoran/go-client`).
- [ ] Crear el comando para imagen de **alto consumo de CPU** (alpine con bucle de cálculos).
- [ ] Crear el comando para imagen de **bajo consumo** (alpine con `sleep 240`).
- [ ] Crear el `docker-compose.yml` con servicios:
  - `valkey` (base de datos clave-valor, puerto 6379).
  - `grafana` (dashboard, puerto 3000).
- [ ] Verificar que Grafana se levanta y puede conectarse a Valkey como datasource.

**Criterio de éxito:** `docker compose up -d` levanta Grafana accesible en `localhost:3000`.

---

### FASE 3 — Script del Cronjob
**Objetivo:** Automatizar la creación de contenedores de prueba cada 2 minutos.

**Tareas:**
- [ ] Crear el script `spawn_containers.sh`.
- [ ] El script debe lanzar exactamente **5 contenedores** por ejecución.
- [ ] La selección de imagen debe ser **aleatoria** entre las 3 categorías definidas.
- [ ] Verificar que el script funciona manualmente antes de registrarlo en cron.

**Criterio de éxito:** Ejecutar el script genera 5 contenedores en ejecución.

---

### FASE 4 — Daemon en Go (Núcleo del Proyecto)
**Objetivo:** Desarrollar el servicio central que orquesta todo el sistema.

**Sub-fases:**

#### 4a — Estructura Base del Proyecto Go
- [ ] Inicializar módulo Go: `go mod init daemon_sopes1`.
- [ ] Instalar dependencias: cliente Valkey/Redis (`go-redis`).
- [ ] Definir las estructuras de datos (structs) para procesos y memoria.

#### 4b — Inicio del Servicio
- [ ] Función para ejecutar script que carga el módulo de kernel.
- [ ] Función para crear el contenedor de Grafana vía Docker.
- [ ] Función para registrar el cronjob en el sistema operativo.

#### 4c — Loop Principal (cada 20-60 segundos)
- [ ] Leer el archivo `/proc/continfo_pr1_so1_#CARNET`.
- [ ] Parsear/deserializar el JSON.
- [ ] Filtrar procesos que pertenecen a contenedores Docker.
- [ ] Clasificar contenedores en "alto consumo" y "bajo consumo".
- [ ] Aplicar lógica de gestión:
  - Mantener siempre **3 contenedores de bajo consumo**.
  - Mantener siempre **2 contenedores de alto consumo**.
  - Ordenar por RAM, VSZ, RSS, CPU para decidir cuáles eliminar.
  - Ejecutar `docker stop` + `docker rm` en los excedentes.
- [ ] Guardar registro (log) del estado en Valkey con timestamp.

#### 4d — Finalización del Servicio
- [ ] Capturar señal de interrupción (SIGTERM/SIGINT).
- [ ] Eliminar el cronjob registrado.
- [ ] Liberar recursos y cerrar conexiones.

**Criterio de éxito:** El daemon corre indefinidamente, regula los contenedores y almacena logs en Valkey.

---

### FASE 5 — Dashboard en Grafana
**Objetivo:** Visualizar en tiempo real el estado del sistema.

**Paneles a crear:**

| Panel | Tipo | Fuente de datos |
|---|---|---|
| Total de RAM | Card / Indicador | Valkey |
| RAM en uso | Card / Indicador | Valkey |
| Memoria libre | Card / Indicador | Valkey |
| Uso de RAM a lo largo del tiempo | Time Series (línea) | Valkey |
| Contenedores eliminados en el tiempo | Time Series (barras) | Valkey |
| Top 5 por consumo de RAM | Pie Chart | Valkey |
| Top 5 por consumo de CPU | Pie Chart | Valkey |

**Tareas:**
- [ ] Configurar Valkey como datasource en Grafana.
- [ ] Crear nuevo dashboard y agregar cada panel.
- [ ] Conectar cada panel a la clave correspondiente en Valkey.
- [ ] Verificar que los datos se actualizan conforme el daemon guarda logs.

**Criterio de éxito:** Dashboard muestra datos en tiempo real sin errores.

---

### FASE 6 — Pruebas Integrales y Documentación
**Objetivo:** Validar el sistema completo y documentar para la entrega.

**Tareas:**
- [ ] Prueba de flujo completo: encender daemon → esperar cronjob → verificar gestión de contenedores → verificar Grafana.
- [ ] Captura de pantalla: `/proc` mostrando procesos con PID, nombre, memoria y CPU.
- [ ] Captura de pantalla: Grafana con todos los paneles con datos reales.
- [ ] Captura de pantalla: Terminal mostrando logs del daemon.
- [ ] Redactar `README.md` con instrucciones de instalación.
- [ ] Redactar manual técnico.
- [ ] Subir todo al repositorio privado y agregar a `CamiloSincal` como colaborador.

---

## 📅 Cronograma Sugerido

```
Día 1-2:   FASE 1 — Módulo de Kernel (lo más crítico y nuevo)
Día 3:     FASE 2 — Entorno Docker + Compose
Día 3:     FASE 3 — Script de Cronjob
Día 4-6:   FASE 4 — Daemon en Go (la parte más grande)
Día 7:     FASE 5 — Dashboard en Grafana
Día 8:     FASE 6 — Pruebas + Documentación
```

---

## ⚠️ Restricciones Clave (Nunca Olvidar)

1. **Siempre deben existir** 3 contenedores de **bajo consumo** activos.
2. **Siempre deben existir** 2 contenedores de **alto consumo** activos.
3. **NUNCA eliminar** el contenedor de Grafana.
4. El archivo `/proc` debe llamarse exactamente `/proc/continfo_pr1_so1_#CARNET` (reemplazar `#CARNET` con tu número de carnet).
5. El repositorio GitHub debe ser **privado** y agregar a `CamiloSincal` como colaborador.

---

## 🛠️ Stack Tecnológico

| Tecnología | Rol en el proyecto |
|---|---|
| **C (Kernel Module)** | Sensor de bajo nivel — captura métricas del OS |
| **Go** | Daemon de gestión — cerebro del sistema |
| **Docker / Docker Compose** | Plataforma de contenedores |
| **Valkey** | Base de datos para logs y métricas |
| **Grafana** | Dashboard de visualización |
| **Linux /proc filesystem** | Canal de comunicación Kernel ↔ Usuario |
| **Cron** | Programador de tareas periódicas |

---

## 📁 Estructura de Carpetas Propuesta

```
Carnet#_LAB_P1_SO1_VacJun2026/
├── kernel_module/
│   ├── sysinfo_module.c        ← Módulo de kernel
│   └── Makefile                ← Script de compilación
├── daemon/
│   ├── main.go                 ← Punto de entrada del daemon
│   ├── go.mod                  ← Módulo Go
│   ├── go.sum
│   └── (otros archivos .go)
├── scripts/
│   ├── spawn_containers.sh     ← Script del cronjob
│   └── load_module.sh          ← Script para cargar el módulo
├── docker/
│   ├── docker-compose.yml      ← Grafana + Valkey
│   └── Dockerfile.go-client    ← (si creas imagen propia)
├── grafana/
│   └── dashboard.json          ← Exportación del dashboard
├── docs/
│   ├── manual_tecnico.md
│   └── screenshots/
└── README.md
```