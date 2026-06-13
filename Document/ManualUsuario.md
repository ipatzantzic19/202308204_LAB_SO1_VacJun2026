# Manual de Usuario — Proyecto 1 SOPES 1
**Monitor de Contenedores Docker con Telemetría de Kernel**
Universidad San Carlos de Guatemala · Facultad de Ingeniería
Estudiante: 202308204 · Vacaciones Junio 2026

---

## Tabla de Contenidos

1. [Introducción](#1-introducción)
2. [Requisitos del Sistema](#2-requisitos-del-sistema)
3. [Instalación Paso a Paso](#3-instalación-paso-a-paso)
4. [Operación del Sistema](#4-operación-del-sistema)
5. [El Dashboard de Grafana](#5-el-dashboard-de-grafana)
6. [Comandos de Uso Frecuente](#6-comandos-de-uso-frecuente)
7. [Apagado Limpio del Sistema](#7-apagado-limpio-del-sistema)
8. [Interpretación de los Datos](#8-interpretación-de-los-datos)
9. [Preguntas Frecuentes](#9-preguntas-frecuentes)

---

## 1. Introducción

Este sistema monitorea en tiempo real el uso de memoria RAM y los contenedores Docker activos en tu máquina Linux. Funciona de forma completamente autónoma: una vez iniciado, el sistema:

- **Captura métricas** del kernel Linux cada segundo (módulo C en `/proc`).
- **Gestiona contenedores** automáticamente, manteniendo siempre 3 bajos + 2 altos activos.
- **Crea contenedores de prueba** cada 2 minutos para simular carga de trabajo.
- **Visualiza todo** en un dashboard web accesible desde el navegador.

No necesitas interactuar con el sistema durante su operación normal — el daemon Go hace todo el trabajo.

---

## 2. Requisitos del Sistema

Antes de comenzar, verifica que tu máquina tiene todo lo necesario:

```bash
# 1. Sistema operativo Linux (Ubuntu 20.04+ o similar)
uname -r
# Esperado: 5.15.x o superior

# 2. Headers del kernel instalados
ls /usr/src/linux-headers-$(uname -r)
# Si no existen: sudo apt install linux-headers-$(uname -r) build-essential

# 3. Go instalado
go version
# Esperado: go1.21.x o superior

# 4. Docker funcionando
docker --version && docker compose version
# Si no está: sudo apt install docker.io docker-compose-plugin

# 5. Cron activo
systemctl status cron
# Si no está activo: sudo systemctl enable --now cron
```

---

## 3. Instalación Paso a Paso

### Paso 1 — Compilar el módulo de kernel

El módulo de kernel es el sensor que lee las métricas directamente desde el núcleo del sistema operativo.

```bash
cd /ruta/al/proyecto/kernel_module/

# Compilar
make

# Cargar en el kernel
sudo insmod sys_info_module.ko

# Verificar que funciona
cat /proc/continfo_pr1_so1_202308204 | python3 -m json.tool
```

Deberías ver una salida JSON con la memoria RAM y la lista de procesos del sistema. Si aparece, el módulo está funcionando correctamente.

### Paso 2 — Levantar el stack de monitoreo

Este paso inicia Valkey (base de datos), Prometheus (recolector) y Grafana (dashboard) en contenedores Docker.

```bash
cd /ruta/al/proyecto/docker/

# Primera vez: instalación completa
bash setup_entorno.sh

# Veces posteriores: solo levantar
docker compose up -d

# Verificar que todos los servicios están activos
docker compose ps
```

Deberías ver 4 servicios con estado `Up`: `valkey`, `redis_exporter`, `prometheus`, `grafana`.

### Paso 3 — Configurar las rutas del daemon

Antes de ejecutar el daemon, edita el archivo `daemon/config.go` y ajusta las rutas absolutas a tu máquina:

```go
var cfg = Config{
    ProcFile:        "/proc/continfo_pr1_so1_202308204",
    RutaCompose:     "/ruta/completa/a/docker/docker-compose.yml",
    RutaScriptSpawn: "/ruta/completa/a/scripts/spawn_containers.sh",
    RutaScriptModulo: "/ruta/completa/a/kernel_module/load_module.sh",
    ModuloPath:      "/ruta/completa/a/kernel_module/sys_info_module.ko",
    // El resto de valores ya son correctos
}
```

> **Importante:** Usa rutas absolutas (que empiecen con `/`), no rutas relativas.

### Paso 4 — Compilar y ejecutar el daemon

```bash
cd /ruta/al/proyecto/daemon/

# Descargar dependencias (solo la primera vez)
go mod tidy

# Compilar
go build -o daemon_sopes1 .

# Ejecutar (requiere sudo para manejar el módulo de kernel)
sudo ./daemon_sopes1
```

Deberías ver mensajes como estos en la terminal:

```
[MAIN] Paso 1/4 → Iniciando Grafana...
[DOCKER] ✓ Entorno Docker iniciado.
[MAIN] Paso 2/4 → Cargando módulo de kernel...
[KERNEL] ✓ Módulo ya cargado. Continuando.
[MAIN] Paso 3/4 → Registrando cronjob...
[CRON]  ✓ Cronjob registrado: */2 * * * * /ruta/spawn_containers.sh
[MAIN] Paso 3.5/4 → Iniciando servidor Prometheus...
[METRICS] ✓ Servidor Prometheus en http://0.0.0.0:9200/metrics
[MAIN] Paso 3.6/4 → Conectando a Valkey...
[VALKEY] ✓ Conectado a Valkey en localhost:6379
[MAIN] Paso 4/4 → Iniciando loop principal (cada 30s)...
```

### Paso 5 — Abrir el Dashboard

1. Abre tu navegador y ve a `http://localhost:3000`
2. Usuario: `admin` / Contraseña: `admin`
3. En el menú lateral busca **Dashboards** → **SOPES1 P1 - Monitor de Contenedores**

> **Nota:** Espera al menos **2-3 minutos** después de iniciar el daemon para que el cronjob genere contenedores y aparezcan datos en los paneles de tiempo.

---

## 4. Operación del Sistema

### Lo que hace el sistema automáticamente

Una vez iniciado el daemon, todo ocurre de forma autónoma:

| Cada... | Acción automática |
|---|---|
| 2 minutos | El cronjob crea 5 nuevos contenedores Docker aleatorios |
| 30 segundos | El daemon lee las métricas del kernel y evalúa los contenedores |
| 30 segundos | Si hay más de 3 bajos o más de 2 altos, elimina el excedente |
| 30 segundos | Guarda el estado actual en Valkey y actualiza las métricas de Prometheus |
| 15 segundos | Prometheus recoge las métricas del daemon y de Valkey |

### Cómo el daemon decide qué eliminar

El daemon nunca elimina contenedores al azar. Sigue estas reglas:

1. Separa los contenedores en dos grupos: **bajos** (`alpine sleep`) y **altos** (`go-client` o `alpine+bc`).
2. Dentro de cada grupo, ordena por RAM consumida (RSS) de mayor a menor.
3. Elimina los que **más consumen** primero, preservando los más eficientes.
4. Se detiene cuando quedan exactamente 3 bajos y 2 altos.

```
Ejemplo: hay 7 contenedores bajos
  → Ordena por RSS: [450KB, 380KB, 310KB, 290KB, 250KB, 200KB, 180KB]
  → Elimina los 4 de mayor consumo: 450, 380, 310, 290
  → Quedan los 3 de menor consumo: 250, 200, 180  ✓
```

### Monitorear la actividad en tiempo real

Mientras el daemon corre, puedes observar lo que está pasando en terminales separadas:

```bash
# Ver los contenedores del proyecto activos
watch -n 5 'docker ps --filter "name=sopes1_" --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"'

# Ver el log del cronjob
tail -f scripts/spawn_containers_internal.log

# Ver los datos almacenados en Valkey
docker exec -it valkey valkey-cli
> GET memoria:actual
> LLEN eliminados:log
> ZREVRANGE ranking:ram 0 4 WITHSCORES

# Ver métricas que Prometheus está leyendo
curl -s http://localhost:9200/metrics | grep -E "sysinfo_ram|sopes1_"
```

---

## 5. El Dashboard de Grafana

El dashboard tiene **8 paneles** organizados en 3 secciones:

### Sección 1: Métricas de RAM (fila superior)

| Panel | Qué muestra | Colores |
|---|---|---|
| **RAM Total** | Memoria RAM total instalada en GB | Azul fijo |
| **RAM en Uso** | RAM actualmente ocupada en GB | Verde → Amarillo → Rojo según % |
| **Memoria Libre** | RAM disponible en MB | Rojo (poco) → Verde (bastante) |
| **% RAM Usada** | Gauge circular del porcentaje total | Verde (< 60%) → Amarillo → Rojo (> 85%) |

### Sección 2: Evolución en el Tiempo (fila media)

**Uso de RAM a lo largo del tiempo:** Muestra dos líneas:
- Línea **roja** → RAM usada (tenderá a estar alta)
- Línea **verde** → RAM libre (tenderá a estar baja)

La leyenda inferior muestra el valor más reciente (`Last`), el máximo (`Max`) y el mínimo (`Min`) del período visible.

**Contenedores Eliminados en el Tiempo:** Muestra tres series:
- Barras **verdes** → Cuántos contenedores se eliminaron en cada ventana de 2 minutos
- Área **amarilla** → Cuántos contenedores altos hay activos en ese momento
- Área **azul** → Cuántos contenedores bajos hay activos en ese momento

> Las tres series son independientes. No suman entre sí. Los eliminados muestran actividad pasada; los activos muestran el estado actual.

### Sección 3: Top Consumidores (fila inferior)

Dos gráficas de pastel que muestran los 5 procesos con mayor consumo del sistema:

- **Top 5 por RAM** → Los procesos que más memoria física (RSS) están usando en este momento
- **Top 5 por CPU** → Los procesos que más porcentaje de CPU están consumiendo

Incluyen tanto los procesos de los contenedores del proyecto como otros procesos del sistema operativo.

### Controles del Dashboard

- **Rango de tiempo** (esquina superior derecha): Cambia qué período histórico ves. Por defecto: últimos 30 minutos.
- **Refresco automático** (junto al rango): El dashboard se actualiza cada 30 segundos automáticamente.
- **Zoom en gráficas**: Haz clic y arrastra sobre cualquier gráfica de líneas para hacer zoom en ese período.

---

## 6. Comandos de Uso Frecuente

### Verificar el estado del módulo kernel

```bash
# ¿Está cargado?
lsmod | grep sys_info_module

# ¿El archivo /proc existe y tiene datos?
cat /proc/continfo_pr1_so1_202308204 | python3 -m json.tool | head -20

# Ver mensajes del kernel relacionados
sudo dmesg | grep SOPES1 | tail -5
```

### Verificar el estado del stack Docker

```bash
# Estado de los servicios de monitoreo
docker compose -f /ruta/al/proyecto/docker/docker-compose.yml ps

# Logs de Grafana
docker compose -f /ruta/al/proyecto/docker/docker-compose.yml logs grafana --tail 20

# ¿Prometheus puede alcanzar los targets?
curl -s http://localhost:9090/api/v1/targets | python3 -m json.tool | grep -A2 '"health"'
```

### Verificar el cronjob

```bash
# Ver si el cronjob está registrado
crontab -l

# Ver la actividad del script
tail -30 /ruta/al/proyecto/scripts/spawn_containers_internal.log
```

### Verificar datos en Valkey

```bash
# Conectarse al CLI de Valkey
docker exec -it valkey valkey-cli

# Dentro del CLI:
GET memoria:actual                    # Estado actual de RAM (JSON)
LLEN memoria:historia                 # Cuántos snapshots hay
LRANGE eliminados:log 0 4             # Últimos 5 eliminados
GET eliminados:total                  # Total acumulado de eliminaciones
ZREVRANGE ranking:ram 0 4 WITHSCORES  # Top 5 por RAM
ZREVRANGE ranking:cpu 0 4 WITHSCORES  # Top 5 por CPU
KEYS sopes1_*                         # Contenedores activos
```

### Gestión manual de contenedores de prueba

```bash
# Lanzar manualmente 5 contenedores (sin esperar al cronjob)
bash /ruta/al/proyecto/scripts/spawn_containers.sh

# Ver todos los contenedores del proyecto activos
docker ps --filter "name=sopes1_"

# Eliminar todos los contenedores del proyecto manualmente
docker stop $(docker ps -q --filter "name=sopes1_") 2>/dev/null
docker rm $(docker ps -aq --filter "name=sopes1_") 2>/dev/null
```

---

## 7. Apagado Limpio del Sistema

Para detener el sistema de forma correcta:

### Paso 1 — Detener el daemon

En la terminal donde corre el daemon, presiona `Ctrl+C`. Verás:

```
[MAIN] Señal recibida: interrupt. Apagando limpiamente...
[LIMPIEZA] Eliminando cronjob...
[CRON] ✓ Cronjob eliminado.
[LIMPIEZA] ✓ Limpieza completada.
[MAIN] ✓ Daemon detenido correctamente.
```

Esto garantiza que el cronjob queda eliminado del sistema.

### Paso 2 — Verificar que el cronjob se eliminó

```bash
crontab -l
# No debe aparecer ninguna línea con spawn_containers.sh
```

### Paso 3 — Detener el stack Docker (opcional)

Si no necesitas mantener Grafana y Valkey corriendo:

```bash
docker compose -f /ruta/al/proyecto/docker/docker-compose.yml down
# Agrega -v al final para borrar también los datos almacenados
```

### Paso 4 — Descargar el módulo de kernel (opcional)

```bash
sudo rmmod sys_info_module
# Verificar:
lsmod | grep sys_info_module  # No debe aparecer nada
```

> Si necesitas reiniciar el sistema, el módulo kernel **no se recarga automáticamente** al arrancar. Deberás cargarlo manualmente (`sudo insmod sys_info_module.ko`) o ejecutar el daemon (que lo carga solo).

---

## 8. Interpretación de los Datos

### ¿Por qué la RAM en uso es tan alta (97%)?

Esto es normal en sistemas Linux con muchas aplicaciones corriendo. Los contenedores `roldyoran/go-client` consumen cantidades significativas de RAM por diseño, lo cual es el objetivo del proyecto: simular carga real.

### ¿Por qué el % de CPU en los procesos puede ser mayor a 100%?

Los valores de CPU que el módulo del kernel captura son contadores acumulados en nanosegundos (`utime + stime`), no porcentajes instantáneos. Esto es intencional y está permitido por el enunciado del proyecto. El kernel almacena el tiempo de CPU total que ha usado cada proceso desde que arrancó, no la tasa instantánea.

### ¿Por qué los "Top 5" muestran procesos del sistema además de contenedores?

El módulo captura **todos** los procesos del sistema (tal como lo pide el enunciado). Naturalmente, aplicaciones como Grafana, Chrome o VS Code pueden consumir más RAM que los contenedores `alpine sleep`. Los tops muestran los mayores consumidores reales del sistema completo.

### ¿Qué significa "Eliminados (2m)" en la gráfica?

Muestra cuántos contenedores eliminó el daemon durante cada ventana de 2 minutos. Cuando el cronjob crea 5 contenedores nuevos y ya hay más de 3 bajos o 2 altos activos, el daemon elimina el excedente. El pico más alto indica el momento de mayor actividad de limpieza.

### ¿Es normal que "Altos Activos" y "Bajos Activos" no sumen igual que "Eliminados"?

Sí, completamente normal. Son métricas independientes:

- `Eliminados (2m)`: actividad de eliminación en el pasado reciente
- `Altos/Bajos Activos`: cantidad de contenedores **vivos** ahora mismo

El hecho de que 2 altos + 3 bajos = 5 activos no tiene relación directa con cuántos se eliminaron antes.

---

## 9. Preguntas Frecuentes

**¿Qué pasa si cierro la terminal donde corre el daemon sin hacer Ctrl+C?**

El daemon se detiene pero el cronjob queda registrado en crontab. El script seguirá corriendo cada 2 minutos creando contenedores sin que nadie los gestione. Para limpiar:

```bash
crontab -l | grep -v "spawn_containers" | crontab -
```

**¿Puedo cambiar el intervalo del loop del daemon?**

Sí, en `daemon/config.go` modifica `LoopInterval`. El enunciado permite valores entre 20 y 60 segundos:

```go
LoopInterval: 30 * time.Second,  // Cambia a 20s o 60s según prefieras
```

**¿Puedo ver el historial completo de eliminaciones?**

Sí, desde Valkey:

```bash
docker exec -it valkey valkey-cli LRANGE eliminados:log 0 -1
```

También puedes ver los últimos 30 minutos directamente en Grafana en la gráfica "Contenedores Eliminados en el Tiempo".

**¿El sistema funciona si no tengo internet?**

Sí, una vez que las imágenes Docker están descargadas (`roldyoran/go-client` y `alpine`). La única operación que requiere red es el `docker pull` inicial.

**¿Qué pasa si hay menos de 3 contenedores bajos activos?**

El daemon no crea contenedores por sí mismo. Espera a que el cronjob (que corre cada 2 minutos) cree nuevos contenedores. El sistema diseñado es reactivo: gestiona lo que existe, pero no genera carga activa.

**¿Puedo ejecutar el daemon en segundo plano?**

Sí, aunque para el proyecto es recomendable dejarlo en primer plano para ver los logs. Si necesitas fondo:

```bash
sudo nohup ./daemon_sopes1 > daemon.log 2>&1 &
echo $!  # Guarda este PID para matar el proceso después
```

**¿Por qué necesita sudo?**

El daemon carga el módulo de kernel con `insmod`, que requiere privilegios de root. Sin sudo, esta operación falla.