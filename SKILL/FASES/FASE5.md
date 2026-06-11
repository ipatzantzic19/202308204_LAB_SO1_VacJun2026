# Fase 5 — Dashboard en Grafana
**Proyecto 1 · Sistemas Operativos 1 · USAC Vacaciones Junio 2026**

---

## Archivos de esta fase

```
grafana/
├── dashboard.json      ← Dashboard completo listo para importar
└── setup_grafana.sh    ← Script de configuración automática

daemon/ (archivos actualizados)
├── metrics.go          ← Agrega contadores de contenedores para Grafana
└── main.go             ← Llama a actualizarMetricasContenedores() por ciclo
```

---

## ¿Qué hace esta fase?

Configura Grafana para visualizar en tiempo real todas las métricas que el daemon recolecta. El flujo de datos es el mismo que enseña el curso en Clase 5:

```
Módulo Kernel → /proc → Daemon Go → :9200/metrics → Prometheus → Grafana
                                  → Valkey (logs)
```

Grafana no lee Valkey directamente. Consulta **Prometheus** como única fuente de datos. Los paneles usan las mismas queries PromQL documentadas en el `Readme.md` de Clase 5.

---

## Paneles del Dashboard

El dashboard tiene 8 paneles organizados en 3 filas:

### Fila 1 — 📊 Métricas de RAM

| Panel | Tipo | Query PromQL |
|---|---|---|
| RAM Total | Stat (azul) | `sysinfo_ram_total_kb / 1024` |
| RAM en Uso | Stat (semáforo) | `sysinfo_ram_used_kb / 1024` |
| Memoria Libre | Stat (semáforo) | `sysinfo_ram_free_kb / 1024` |
| % RAM Usada | Gauge | `sysinfo_ram_used_kb / sysinfo_ram_total_kb * 100` |

### Fila 2 — 📈 Evolución en el Tiempo

| Panel | Tipo | Queries |
|---|---|---|
| Uso de RAM a lo largo del tiempo | Time series | `sysinfo_ram_used_kb / 1024` + `sysinfo_ram_free_kb / 1024` |
| Contenedores Eliminados en el Tiempo | Time series | `increase(sopes1_containers_eliminated_total[2m])` + activos alto/bajo |

### Fila 3 — 🏆 Top Consumidores

| Panel | Tipo | Query PromQL |
|---|---|---|
| Top 5 por Consumo de RAM | Pie Chart | `topk(5, max by (name) (sysinfo_process_rss_kb)) / 1024` |
| Top 5 por Consumo de CPU | Pie Chart | `topk(5, max by (name) (sysinfo_process_cpu_percent))` |

> **Nota sobre `max by (name)`:** Esta cláusula, tomada directamente del Readme de Clase 5, colapsa todos los PIDs del mismo proceso en un solo valor. Sin ella, Grafana mostraría múltiples entradas para el mismo proceso con diferentes PIDs.

---

## Instalación paso a paso

### Paso 1: Reemplazar los archivos del daemon actualizados

```bash
# Desde la carpeta donde descargaste los archivos:
cp metrics.go ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/daemon/
cp main.go    ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/daemon/
```

Los cambios en estos archivos agregan 3 métricas nuevas:
- `sopes1_containers_eliminated_total` → counter que sube con cada eliminación
- `sopes1_active_containers_alto` → gauge de contenedores altos activos
- `sopes1_active_containers_bajo` → gauge de contenedores bajos activos

### Paso 2: Copiar archivos de Grafana al proyecto

```bash
mkdir -p ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/grafana/
cp dashboard.json   ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/grafana/
cp setup_grafana.sh ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/grafana/
```

### Paso 3: Verificar que el compose está corriendo

```bash
cd ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/docker/
docker compose ps
# Deben estar Up: grafana, prometheus, redis_exporter, valkey
```

### Paso 4: Ejecutar el setup automático

```bash
cd ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/grafana/
bash setup_grafana.sh
```

El script hace 3 cosas vía la API de Grafana:
1. Espera a que Grafana esté listo (máx 30s)
2. Crea el datasource Prometheus apuntando a `http://prometheus:9090`
3. Importa el `dashboard.json` completo

### Paso 5: Iniciar el daemon con los archivos actualizados

```bash
cd ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/daemon/
sudo go run .
```

### Paso 6: Esperar datos y abrir Grafana

```bash
# Abrir en el navegador
http://localhost:3000
# Usuario: admin | Contraseña: admin

# Navegar al dashboard:
# Dashboards → SOPES1 P1 - Monitor de Contenedores
```

Espera al menos **2 minutos** para que el cronjob genere contenedores y el daemon haga 2-3 ciclos. Los paneles de Pie Chart necesitan que haya procesos activos para mostrar datos.

---

## Importar el dashboard manualmente (si el script falla)

Si `setup_grafana.sh` no funciona, importa el dashboard a mano:

1. Abre `http://localhost:3000` → usuario `admin`, contraseña `admin`
2. En el menú lateral: **Connections → Data sources → Add new data source**
3. Selecciona **Prometheus**
4. En URL escribe exactamente: `http://prometheus:9090` *(nombre del servicio Docker, no localhost)*
5. Clic en **Save & test** → debe aparecer verde
6. Menú lateral: **Dashboards → Import**
7. Clic en **Upload dashboard JSON file**
8. Selecciona el archivo `grafana/dashboard.json`
9. En el selector de datasource elige **Prometheus**
10. Clic en **Import**

---

## Verificar que los paneles tienen datos

Una vez con el daemon corriendo, verifica cada panel:

```bash
# Confirmar que Prometheus recibe métricas del daemon
curl -s http://localhost:9200/metrics | grep -E "sysinfo_ram|sopes1_"

# Confirmar que Prometheus tiene los targets UP
# http://localhost:9090/targets
# → daemon_go: UP
# → valkey:    UP

# Confirmar queries directamente en Prometheus
# http://localhost:9090/graph
# Escribe: sysinfo_ram_used_kb / 1024  → debe dar un número
# Escribe: topk(5, max by (name) (sysinfo_process_rss_kb))  → debe listar 5 procesos
```

---

## Configuración de auto-refresco

El dashboard está configurado para refrescarse automáticamente cada **30 segundos** (igual al intervalo del loop del daemon). Puedes cambiarlo en la esquina superior derecha de Grafana con el selector de intervalo.

---

## Estructura del repositorio hasta esta fase

```
202308204_LAB_SO1_VacJun2026/
├── kernel_module/
│   ├── sysinfo_module.c        ✅ Fase 1
│   ├── Makefile                ✅ Fase 1
│   ├── load_module.sh          ✅ Fase 1
│   └── unload_module.sh        ✅ Fase 1
├── docker/
│   ├── docker-compose.yml      ✅ Fase 2
│   └── prometheus.yml          ✅ Fase 2
├── scripts/
│   └── spawn_containers.sh     ✅ Fase 3
├── daemon/
│   ├── main.go                 ✅ Fase 4 (actualizado Fase 5)
│   ├── config.go               ✅ Fase 4
│   ├── types.go                ✅ Fase 4
│   ├── cronjob.go              ✅ Fase 4
│   ├── kernel.go               ✅ Fase 4
│   ├── proc_reader.go          ✅ Fase 4
│   ├── docker_manager.go       ✅ Fase 4
│   ├── valkey_client.go        ✅ Fase 4
│   ├── metrics.go              ✅ Fase 4 (actualizado Fase 5)
│   ├── go.mod / go.sum
│   └── README_FASE4.md
├── grafana/
│   ├── dashboard.json          ✅ Fase 5
│   ├── setup_grafana.sh        ✅ Fase 5
│   └── README_FASE5.md         ✅ Fase 5
└── README.md
```

---

## Errores comunes

**Los paneles muestran "No data"**
→ El daemon no está corriendo: `sudo go run .` en la carpeta `daemon/`
→ Prometheus no tiene el target UP: revisa `http://localhost:9090/targets`

**"Datasource not found" al importar**
→ Primero crea el datasource manualmente (paso 4 de importación manual arriba)

**El pie chart muestra más de 5 procesos**
→ Asegúrate de que la query tiene `topk(5, ...)` y que en las opciones de la query está marcado **Instant** (no Range)

**Los contenedores eliminados no aparecen en la gráfica**
→ Normal al inicio. Deja correr el daemon + cronjob durante 4-6 minutos para que el daemon tenga que eliminar contenedores excedentes y el contador suba