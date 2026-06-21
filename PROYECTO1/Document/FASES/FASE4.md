# Fase 4 — Daemon en Go
**Proyecto 1 · Sistemas Operativos 1 · USAC Vacaciones Junio 2026**

---

## Archivos de esta fase

```
daemon/
├── config.go          ← ⚠ EDITA ESTE PRIMERO (rutas de tu sistema)
├── types.go           ← Estructuras de datos (mapean el JSON del kernel)
├── main.go            ← Punto de entrada + loop principal
├── cronjob.go         ← Gestión del cron (patrón Clase 4)
├── kernel.go          ← Carga del módulo de kernel
├── proc_reader.go     ← Lectura del /proc
├── docker_manager.go  ← Gestión de contenedores Docker
├── valkey_client.go   ← Almacenamiento en Valkey
├── metrics.go         ← Servidor Prometheus :9200 (patrón Clase 5)
└── go.mod             ← Módulo y dependencias
```

---

## ⚠️ Paso 1 obligatorio: editar config.go

Abre `config.go` y ajusta **todas** las rutas a tu máquina:

```go
var cfg = Config{
    ProcFile:         "/proc/continfo_pr1_so1_202308204",  // ← tu carnet
    RutaCompose:      "/home/isai/Documentos/Github/202308204_LAB_SO1_VacJun2026/docker/docker-compose.yml",
    RutaScriptSpawn:  "/home/isai/Documentos/Github/202308204_LAB_SO1_VacJun2026/scripts/spawn_containers.sh",
    RutaScriptModulo: "/home/isai/Documentos/Github/202308204_LAB_SO1_VacJun2026/kernel_module/load_module.sh",
    ModuloPath:       "/home/isai/Documentos/Github/202308204_LAB_SO1_VacJun2026/kernel_module/sysinfo_module.ko",
    // ... el resto de campos ya tienen valores correctos
}
```

---

## Instalación y ejecución

```bash
# 1. Instalar Go (si no lo tienes)
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version   # debe mostrar go1.22.x

# 2. Ir al directorio del daemon
cd ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/daemon/

# 3. Editar config.go con tus rutas reales (ver arriba)

# 4. Descargar dependencias
go mod tidy

# 5. Verificar que el docker compose está corriendo
docker compose -f ../docker/docker-compose.yml ps

# 6. Verificar que el módulo de kernel está cargado
lsmod | grep sysinfo_module

# 7. Ejecutar el daemon (necesita sudo para insmod)
sudo go run .

# --- Alternativa: compilar y ejecutar el binario ---
go build -o daemon_sopes1 .
sudo ./daemon_sopes1
```

---

## ¿Qué verás en la consola al arrancar?

```
[MAIN] Paso 1/4 → Iniciando Grafana...
[DOCKER] ✓ Entorno Docker iniciado.
[MAIN] Paso 2/4 → Cargando módulo de kernel...
[KERNEL] ✓ Módulo ya cargado. Continuando.
[MAIN] Paso 3/4 → Registrando cronjob...
[CRON]  ✓ Cronjob registrado: */2 * * * * /home/.../spawn_containers.sh
[CRON]  === Crontab actual ===
        */2 * * * * /home/.../spawn_containers.sh >> ...
[MAIN] Paso 3.5/4 → Iniciando servidor Prometheus...
[METRICS] ✓ Servidor Prometheus en http://0.0.0.0:9200/metrics
[MAIN] Paso 3.6/4 → Conectando a Valkey...
[VALKEY] ✓ Conectado a Valkey en localhost:6379
[MAIN] Paso 4/4 → Iniciando loop principal (cada 30s)...

────────────────────────────────────────────
[CICLO] Iniciando ciclo a las 14:32:00
[PROC]  Leído: 12039732 KB total | 10019196 KB usada | 377 procesos
[DOCKER] 5 contenedor(es) del proyecto detectado(s).
[GESTIÓN] Activos → Bajos: 3 (mínimo: 3) | Altos: 2 (mínimo: 2)
[GESTIÓN] Nada que eliminar. Sistema en equilibrio.
[VALKEY] ✓ Ciclo guardado. Eliminados en este ciclo: 0
[CICLO] ✓ Ciclo completado. Próximo en 30s.
```

---

## Verificar que todo funciona

```bash
# En otra terminal mientras el daemon corre:

# 1. Ver métricas Prometheus
curl http://localhost:9200/metrics | grep sysinfo_ram

# 2. Ver datos en Valkey
docker exec -it valkey valkey-cli
> GET memoria:actual
> LRANGE eliminados:log 0 4
> ZREVRANGE ranking:ram 0 4 WITHSCORES

# 3. Ver targets en Prometheus
# http://localhost:9090/targets
# → daemon_go debe estar UP

# 4. Ver que el cronjob está registrado
crontab -l

# 5. Ver logs del spawn_containers
tail -f /tmp/spawn_containers.log

# 6. Ver contenedores del proyecto
watch -n 5 'docker ps --filter "name=sopes1_"'
```

---

## Flujo completo del daemon

```
Al iniciar:
  iniciarGrafana()       → docker compose up -d
  cargarModuloKernel()   → bash load_module.sh
  registrarCronjob()     → crontab (patrón Clase 4)
  iniciarMetricas()      → goroutine :9200/metrics (patrón Clase 5)
  inicializarValkey()    → connect localhost:6379

Cada 30s (ejecutarCiclo):
  leerProcFile()              → lee y parsea /proc/continfo_pr1_so1_202308204
  obtenerContenedoresConMetricas() → docker ps + docker inspect + cruzar con /proc
  gestionarContenedores()     → mantiene 3 bajos + 2 altos, mata excedentes
  guardarCiclo()              → escribe en Valkey (memoria + contenedores + eliminados)

Al Ctrl+C:
  eliminarCronJob()      → crontab -l | grep -v spawn | crontab -
```

---

## Errores comunes

**`no se pudo leer /proc/continfo_pr1_so1_...`**
→ El módulo no está cargado: `sudo insmod kernel_module/sysinfo_module.ko`

**`no se pudo conectar a Valkey`**
→ El compose no está corriendo: `docker compose -f docker/docker-compose.yml up -d`

**`Error agregando cronjob`**
→ Verifica que `cron` esté activo: `systemctl status cron`

**El target `daemon_go` en Prometheus está DOWN**
→ Verifica que el puerto 9200 no esté bloqueado: `curl http://localhost:9200/health`
→ Prometheus está dentro de Docker y llega al host via `host.docker.internal` (configurado en el compose)

**`permission denied` en insmod**
→ Ejecuta el daemon con `sudo`: `sudo go run .`