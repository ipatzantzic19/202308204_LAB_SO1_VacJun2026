# Fase 4 — Grafana, Prometheus, Observabilidad y Pruebas de Carga

> **Proyecto:** SOPES1 Proyecto 2  
> **Carnet:** 202308204  
> **Equipo asignado para dashboard:** BRA  
> **Estado:** 🟡 En progreso  
> **Criterio principal:** Grafana debe ejecutarse dentro de una VM independiente de KubeVirt usando `containerd`, consumiendo imágenes desde Zot y mostrando datos reales del flujo obligatorio.

---

## Objetivo

Visualizar los datos reales generados por las predicciones, habilitar escalamiento automático en Rust API y ejecutar pruebas de carga reproducibles para comparar el comportamiento del sistema con una y dos réplicas.

El flujo obligatorio se mantiene como:

```text
Rust API
  → RabbitMQ
  → go-consumer
  → Valkey en valkey-vm
```

Para visualización y observabilidad se agrega:

```text
Valkey
  → go-metrics-exporter
  → Prometheus
  → Grafana en grafana-vm
```

Prometheus se utiliza como capa de métricas para Grafana. Valkey continúa siendo la fuente de persistencia del proyecto; Prometheus solo expone los datos en formato consultable para dashboards.

---

## Decisión técnica de Fase 4

Se usará Prometheus como datasource principal de Grafana, alimentado por un exporter propio que lee las claves reales desde Valkey.

Esta decisión evita depender del plugin Redis directo de Grafana y permite construir paneles con métricas claras, manteniendo Valkey como almacenamiento obligatorio.

---

## 4.1 VM 2 — Grafana y Prometheus

- [ ] Crear carpeta `infra/kubernetes/grafana`
- [ ] Crear PVC `grafana-data`
- [ ] Crear `VirtualMachine` independiente `grafana-vm`
- [ ] Programar la VM en nodos con `nested-virtualization=enabled`
- [ ] Agregar toleration para el taint `kubevirt.io/dedicated=virtualization:NoSchedule`
- [ ] Montar PVC mediante `virtiofs`
- [ ] Instalar `containerd` dentro de la VM con `cloud-init`
- [ ] Ejecutar Grafana con `ctr`
- [ ] Ejecutar Prometheus con `ctr`
- [ ] Consumir imágenes desde Zot
- [ ] Persistir `/var/lib/grafana`
- [ ] Validar que la VM quede `Running` y `Ready=True`

### Imágenes esperadas desde Zot

```text
zot.35-226-224-23.sslip.io/library/ubuntu-container-disk:24.04
zot.35-226-224-23.sslip.io/library/grafana-oss:11.5.2
zot.35-226-224-23.sslip.io/library/prometheus:v2.55.1
```

> Nota: las imágenes externas deben espejarse a Zot antes de ser consumidas por la VM, ya que el proyecto exige consumir imágenes desde Zot por HTTPS.

---

## 4.2 Service para Grafana

- [ ] Crear `grafana-service`
- [ ] Exponer puerto `3000`
- [ ] Validar endpoint hacia `grafana-vm`
- [ ] Acceder con `kubectl port-forward`
- [ ] Confirmar login en Grafana

---

## 4.3 Exporter de métricas desde Valkey

- [ ] Crear módulo `go-metrics-exporter`
- [ ] Conectarse a `valkey-service.sopes1-p2.svc.cluster.local:6379`
- [ ] Leer claves reales de Valkey
- [ ] Exponer endpoint `/metrics`
- [ ] Publicar imagen del exporter en Zot
- [ ] Crear Deployment del exporter
- [ ] Crear Service del exporter
- [ ] Validar que Prometheus pueda scrapear el exporter

### Claves reales validadas en Valkey

```text
prediction:bra:count
prediction:bra:timeseries:local
prediction:bra:timeseries:away

stats:predictions:total
stats:wins
stats:users
stats:local:goals:max
stats:local:goals:min
stats:local:goals:frequency
stats:local:goals:mode
stats:local:goals:mode_count
stats:away:goals:max
stats:away:goals:min
stats:away:goals:frequency
stats:away:goals:mode
stats:away:goals:mode_count
```

### Métricas Prometheus esperadas

```text
quiniela_bra_predictions_total
quiniela_predictions_total
quiniela_local_goals_max
quiniela_local_goals_min
quiniela_away_goals_max
quiniela_away_goals_min
quiniela_local_goals_mode
quiniela_away_goals_mode
quiniela_team_wins{team="BRA"}
quiniela_user_predictions{username="user_202308204"}
quiniela_bra_local_goals
quiniela_bra_away_goals
```

---

## 4.4 Datasource Grafana

- [ ] Configurar Prometheus como datasource
- [ ] Validar conexión Grafana → Prometheus
- [ ] Usar PromQL para construir los paneles obligatorios

---

## 4.5 Dashboard obligatorio BRA

El dashboard debe mostrar datos reales para el equipo BRA:

- [ ] Nombre del equipo: BRA
- [ ] Total de predicciones relacionadas con BRA
- [ ] Máximo de goles local
- [ ] Mínimo de goles local
- [ ] Máximo de goles visitante
- [ ] Mínimo de goles visitante
- [ ] Top de equipos con victorias predichas
- [ ] Top de usuarios activos
- [ ] Moda de goles local
- [ ] Moda de goles visitante
- [ ] Serie temporal BRA como local
- [ ] Serie temporal BRA como visitante

---

## 4.6 HPA para Rust API

- [ ] Crear HPA para `rust-api`
- [ ] Configurar mínimo `1` réplica
- [ ] Configurar máximo `3` réplicas
- [ ] Configurar objetivo CPU `30%`
- [ ] Validar escalamiento con carga
- [ ] Validar retorno a una réplica después de la carga

---

## 4.7 Pruebas de carga con Locust

- [ ] Crear escenario de carga hacia `/grpc-202308204`
- [ ] Ejecutar prueba con una réplica
- [ ] Registrar RPS
- [ ] Registrar percentiles
- [ ] Registrar errores
- [ ] Ejecutar prueba con dos réplicas
- [ ] Comparar resultados 1 vs 2 réplicas
- [ ] Guardar evidencia reproducible

---

## 4.8 Dapr opcional

- [ ] Evaluar implementación opcional de `/dapr-202308204`
- [ ] No reemplazar RabbitMQ en el flujo obligatorio
- [ ] Documentar Dapr únicamente como flujo alternativo si se implementa

---

## Criterio de finalización de Fase 4

La Fase 4 se considera completa cuando:

- [ ] `grafana-vm` corre como VM independiente en KubeVirt
- [ ] Grafana corre dentro de la VM usando `containerd`
- [ ] Prometheus corre dentro de la VM usando `containerd`
- [ ] Las imágenes usadas por la VM se consumen desde Zot
- [ ] Grafana muestra todos los paneles obligatorios de BRA con datos reales
- [ ] HPA de Rust escala hasta máximo 3 réplicas y regresa a 1
- [ ] Existen resultados de Locust para 1 vs 2 réplicas
- [ ] El flujo obligatorio mantiene una tasa de error aceptable