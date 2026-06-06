# Fase 2 — Entorno Docker
**Proyecto 1 · Sistemas Operativos 1 · USAC Vacaciones Junio 2026**

---

## Archivos de esta fase

```
docker/
├── docker-compose.yml    ← Stack completo (Valkey + Prometheus + Grafana)
├── prometheus.yml        ← Config de scraping de Prometheus
├── setup_entorno.sh      ← Script de instalación completa (ejecutar primero)
├── test_entorno.sh       ← Suite de pruebas automáticas
└── test_imagenes.sh      ← Prueba los 3 tipos de contenedor del proyecto
```

---

## Arquitectura de esta fase

```
                    ┌────────────────────────────────────────┐
                    │         Red Docker: monitoring         │
                    │                                        │
  Daemon Go         │  ┌────────┐     ┌────────────────────┐ │
  (Fase 4)  ──────→ │  │ Valkey │────>│  redis_exporter    │ │
  escribe en         │  │ :6379  │     │  :9121/metrics     │ │
  Valkey             │  └────────┘     └────────┬───────────┘ │
                    │                           │ scrape       │
  Daemon Go         │  ┌─────────────────────┐  │             │
  expone            │  │    Prometheus        │◄─┘             │
  :9200/metrics ───>│  │    :9090             │                │
                    │  └──────────┬──────────┘                │
                    │             │ datasource                 │
                    │  ┌──────────▼──────────┐                │
                    │  │      Grafana         │                │
                    │  │      :3000           │                │
                    │  └─────────────────────┘                │
                    └────────────────────────────────────────┘
```

---

## Instalación (ejecútalo todo en orden)

```bash
# Paso 1: Instalar Docker y Docker Compose (si no los tienes)
sudo apt update
sudo apt install -y docker.io docker-compose-plugin
sudo usermod -aG docker $USER   # Para no usar sudo en cada docker
newgrp docker                   # Aplicar el grupo sin cerrar sesión

# Paso 2: Ir al directorio del proyecto
cd ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/docker

# Paso 3: Setup completo (una sola vez)
bash setup_entorno.sh

# Paso 4: Verificar que todo funciona
bash test_entorno.sh

# Paso 5: Probar los 3 tipos de contenedor
bash test_imagenes.sh
```

---

## Comandos del día a día

```bash
# Levantar todo el stack
docker compose up -d

# Ver estado de los servicios
docker compose ps

# Ver logs en tiempo real (todos)
docker compose logs -f

# Ver logs de un servicio específico
docker compose logs -f grafana
docker compose logs -f prometheus

# Detener todo (sin borrar datos)
docker compose down

# Detener y borrar TODOS los datos (¡cuidado!)
docker compose down -v

# Reiniciar un servicio específico
docker compose restart grafana

# Abrir la CLI de Valkey
docker exec -it valkey valkey-cli

# Ver qué hay almacenado en Valkey
docker exec -it valkey valkey-cli KEYS "*"
```

---

## Las 3 imágenes de contenedores del proyecto

| Tipo | Imagen | Comando |
|---|---|---|
| Alto consumo RAM | `roldyoran/go-client` | `docker run -d roldyoran/go-client` |
| Alto consumo CPU | `alpine` | `docker run -d alpine sh -c "while true; do echo '2^20' \| bc > /dev/null; sleep 2; done"` |
| Bajo consumo | `alpine` | `docker run -d alpine sleep 240` |

Estos comandos los usará el script del Cronjob (Fase 3) de forma aleatoria.

---

## Configurar Prometheus como datasource en Grafana

1. Abrir `http://localhost:3000` → usuario: `admin`, contraseña: `admin`
2. Ir a **Connections → Data sources → Add new data source**
3. Seleccionar **Prometheus**
4. En el campo **URL** escribir:
   ```
   http://prometheus:9090
   ```
   *(usa el nombre del servicio Docker, no localhost)*
5. Clic en **Save & test** → debe aparecer verde

---

## Queries de Prometheus para verificar datos (Fase 5)

Una vez que el Daemon esté corriendo (Fase 4), estas queries en Grafana mostrarán datos:

```promql
# RAM total en MB
sysinfo_ram_total_kb / 1024

# RAM usada en MB
sysinfo_ram_used_kb / 1024

# RAM libre en MB
sysinfo_ram_free_kb / 1024

# Total de procesos
sysinfo_process_count

# Top 5 por RAM
topk(5, max by (name) (sysinfo_process_memory_percent))

# Top 5 por CPU
topk(5, max by (name) (sysinfo_process_cpu_percent))
```

---

## Errores comunes

**`network monitoring not found`**
→ Ejecuta: `docker network create monitoring`

**`port is already allocated`**
→ Algún servicio ya usa ese puerto. Para ver cuál: `sudo lsof -i :3000`
→ Para liberar: `docker stop $(docker ps -q)` o cambiar el puerto en el compose

**`Error response from daemon: conflict`**
→ Ya existe un contenedor con ese nombre. Borra con: `docker rm -f nombre_contenedor`

**Grafana no abre**
→ Espera 30s más después de `docker compose up`. Grafana tarda en arrancar.
→ Ver logs: `docker compose logs grafana`

**Prometheus no muestra datos del Daemon**
→ Normal en esta fase. El target `daemon_go` estará DOWN hasta que implementes la Fase 4.
→ Solo `valkey` target debe estar UP en esta fase.

---

## Conexión con las otras fases

```
Fase 1 (kernel module) → expone /proc/continfo_pr1_so1_202308204
                                        │
Fase 3 (script)  ──────────────────────>│ genera contenedores
                                        │
Fase 4 (daemon)  ──────── lee /proc ───>│
                 │         escribe ──────────────> Valkey (esta fase)
                 └─ expone :9200/metrics ────────> Prometheus (esta fase)
                                                          │
Fase 5 (Grafana) ◄─────────────────────────────── Prometheus (esta fase)
```