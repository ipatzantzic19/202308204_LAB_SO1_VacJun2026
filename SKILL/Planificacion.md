# Planificación — Proyecto 2 Q.M.2026.K8s
**Isai Patzán · 202308204 · SOPES 1 · Vacaciones Junio 2026**

---

## Estado General

| Fase | Nombre | Estado | Clase asociada |
|---|---|---|---|
| **Fase 1** | Infraestructura Base | ✅ Corregida y verificada | Clases 6-9 y 12 |
| **Fase 2** | Comunicación Interna y Mensajería | ✅ Corregida y verificada | Clases 8, 11 y 12 |
| **Fase 3** | Consumo y Persistencia en VM | ⬜ Planificada | Clases 11 y 13 |
| **Fase 4** | Visualización y Pruebas de Carga | ⬜ Planificada | Clases 9, 11-13 |
| **Fase 5** | Documentación y Entrega | ⬜ Planificada | Validación final |

> **Leyenda:** ✅ Completado · 🔄 En progreso · ⬜ Pendiente · 🔒 Bloqueado
>
> Fuente operativa: `SKILL/FASES/Fase1.md` a `Fase5.md`. Los patrones se toman de
> `CamiloSincal/EJEMPLOS_SOPES1_VACJUN2026`, sin contradecir el enunciado.

---

## Arquitectura Objetivo

```
Internet
   │
   ▼
[Locust]  ──POST /grpc-202308204──►  [Gateway API - GKE]
                                            │
                                            ▼
                                     [Rust API - HPA]
                                      (1-3 réplicas)
                                            │ POST JSON
                                            ▼
                                   [Go Deployment 1]
                                   ┌──────────────────┐
                                   │ Container A: REST │
                                   │ Container B: gRPC │
                                   │         Client    │
                                   └─────────┬────────┘
                                             │ gRPC SendPrediction
                                             ▼
                                   [Go Deployment 2]
                                   ┌──────────────────┐
                                   │ Container A: gRPC │
                                   │         Server    │
                                   │ Container B: MQ   │
                                   │       Publisher   │
                                   └─────────┬────────┘
                                             │ AMQP
                                             ▼
                                        [RabbitMQ]
                                             │ consume
                                             ▼
                                     [Go Consumer]
                                             │ SET/ZADD
                                             ▼
                              ┌──────────────────────────┐
                              │     KubeVirt VM 1        │
                              │  containerd → [Valkey]   │
                              └──────────────────────────┘
                                             │ datasource
                                             ▼
                              ┌──────────────────────────┐
                              │     KubeVirt VM 2        │
                              │  containerd → [Grafana]  │
                              └──────────────────────────┘

[Zot Registry] ← fuera del clúster, VM GCP externa
  ↑ push / pull imágenes de todos los componentes
```

---

## Fase 1 — Infraestructura Base

> **Estado:** completada y revalidada el 27 de junio de 2026.

- [x] GKE Standard con tres nodos `n1-standard-4`, 50 GB y virtualización anidada.
- [x] Namespace `sopes1-p2` y contexto kubectl configurados.
- [x] Zot en VM externa con IP estática `35.226.224.23`.
- [x] Registry `zot.35-226-224-23.sslip.io` con certificado TLS público confiable.
- [x] Firewall de Zot limitado públicamente a TCP 80/443.
- [x] Rust API con health check, timeout y propagación de errores.
- [x] Go D1 con REST server y cliente gRPC real en dos contenedores.
- [x] Locust genera el JSON requerido y usa `/grpc-202308204`.
- [x] Gateway API `Programmed/Accepted`, IP `136.68.202.37`.
- [x] Imágenes publicadas y consumidas desde Zot mediante HTTPS.
- [x] Pruebas unitarias Rust y Go D1 exitosas.
- [x] No se modifica containerd en los nodos GKE.

---

## Fase 2 — Comunicación Interna y Mensajería

> **Estado:** completada y revalidada el 27 de junio de 2026.

- [x] RabbitMQ Cluster Operator, una réplica y PVC de 10 GiB.
- [x] RabbitMQ `3.13-management` espejado y consumido desde Zot.
- [x] Credenciales generadas por el operador y usadas mediante Secret.
- [x] Cola durable `predictions` y mensajes persistentes.
- [x] Go D2 con gRPC server y writer AMQP en dos contenedores.
- [x] Go D1 llama a Go D2 mediante gRPC real con timeout.
- [x] `/grpc-202308204` publica en RabbitMQ.
- [x] Prueba negativa: Go D2 detenido produce HTTP `502` público.
- [x] Prueba de recuperación: Go D2 restaurado produce HTTP `200`.
- [x] Todos los Pods actuales están `Running`.

---

## Fase 3 — Consumo y Persistencia en VM

> **Clase asociada:** Clase 10-11 del curso (KubeVirt + containerd)
> **Estado:** 🔒 Actualizar cuando se vean estas clases

### 3.1 Consumer en Go

- [ ] Nuevo módulo Go: `go-consumer`
- [ ] Conectarse a RabbitMQ y consumir mensajes de la queue
- [ ] Parsear el mensaje JSON
- [ ] Guardar datos en Valkey con las claves apropiadas para el dashboard
- [ ] Dockerizar y push a Zot
- [ ] Deployment de K8s: `deployment-consumer.yaml`

### 3.2 KubeVirt

- [ ] Instalar KubeVirt operator en el clúster GKE
- [ ] Verificar que los nodos N1 soportan virtualización anidada
- [ ] Instalar `virtctl` (CLI de KubeVirt)

### 3.3 VM 1 — Valkey

- [ ] Crear `VirtualMachine` en K8s (YAML) para VM de Valkey
- [ ] Dentro de la VM: instalar `containerd`
- [ ] Crear contenedor con Valkey usando containerd (no Docker)
- [ ] Exponer Valkey vía Service de K8s para que Consumer pueda conectar
- [ ] Configurar TTL en Valkey para evitar saturación

### 3.4 Integrar Consumer con Valkey

- [ ] Variable de entorno `VALKEY_ADDR` apuntando al service de Valkey
- [ ] Prueba: enviar predicción y verificar con `valkey-cli GET <clave>`

### 3.5 Claves Valkey para el Dashboard

Diseñar la estructura de claves que permita las queries del dashboard de BRA:

```
# Predicciones del equipo BRA
prediction:bra:count          → contador total
prediction:bra:local:goals    → sorted set de goles locales
prediction:bra:away:goals     → sorted set de goles visitante
prediction:bra:timeseries     → lista con timestamp y goles

# Estadísticas globales
stats:wins:{equipo}           → contador de victorias predichas
stats:users:{username}        → contador de predicciones por usuario
stats:local:goals:max         → máximo de goles locales en un partido
stats:local:goals:min         → mínimo de goles locales en un partido
stats:away:goals:max          → máximo de goles visitante en un partido
stats:away:goals:min          → mínimo de goles visitante en un partido
```

### 3.6 Pruebas de Fase 3

- [ ] Predicciones fluyen hasta Valkey
- [ ] `valkey-cli KEYS "prediction:bra:*"` muestra claves con datos
- [ ] Consumer no pierde mensajes (manejar acknowledgment de RabbitMQ)

---

## Fase 4 — Visualización y Pruebas de Carga

> **Clase asociada:** Clase 12+ del curso
> **Estado:** 🔒 Actualizar cuando se vean estas clases

### 4.1 VM 2 — Grafana

- [ ] Crear segunda `VirtualMachine` en K8s para Grafana (independiente de VM 1)
- [ ] Instalar containerd dentro de la VM
- [ ] Crear contenedor con Grafana usando containerd
- [ ] Configurar datasource: Valkey (plugin Redis o Infinity)
  - [ ] URL apuntando al Service de la VM 1
- [ ] Exponer Grafana vía Service (o via Gateway API)

### 4.2 Dashboard BRA

Crear paneles en Grafana:

- [ ] Mayor goles local en un partido (Stat)
- [ ] Menor goles local en un partido (Stat)
- [ ] Mayor goles visitante en un partido (Stat)
- [ ] Menor goles visitante en un partido (Stat)
- [ ] Top equipos con más victorias predichas (Bar chart)
- [ ] Top usuarios más activos (Bar chart)
- [ ] Moda de goles predichos local (Stat)
- [ ] Moda de goles predichos visitante (Stat)
- [ ] Serie temporal BRA: goles local y visitante (Time Series)
- [ ] Nombre del equipo: "BRA" (Text)
- [ ] Total predicciones para BRA (Stat)

### 4.3 HPA para Rust

- [ ] Crear `HorizontalPodAutoscaler` para el Deployment de Rust:
  - `minReplicas: 1`, `maxReplicas: 3`, `targetCPUUtilizationPercentage: 30`
- [ ] Verificar que el Deployment tiene requests/limits de CPU configurados
- [ ] Generar carga con Locust y observar el autoscaling

### 4.4 Pruebas de Carga con Locust

- [ ] Prueba con **1 réplica** en Go D2 (registrar métricas)
- [ ] Prueba con **2 réplicas** en Go D2 (registrar métricas)
- [ ] Comparar throughput, latencia, errores
- [ ] Verificar HPA del Rust actuando bajo carga

### 4.5 Pruebas de Fase 4

- [ ] Dashboard Grafana muestra todos los paneles con datos reales
- [ ] HPA escala Rust correctamente bajo carga
- [ ] Locust reporta < 1% de errores bajo carga sostenida
- [ ] Analizar diferencia de rendimiento 1 vs 2 réplicas Go D2

---

## Fase 5 — Documentación y Entrega

### 5.1 Manual Técnico (Markdown)

- [ ] Arquitectura general con diagrama
- [ ] Flujo completo de datos (Locust → ... → Grafana)
- [ ] Configuración de Gateway API (GatewayClass, Gateway, HTTPRoute)
- [ ] Comunicación REST (Locust→Rust→GoD1) y gRPC (GoD1→GoD2)
- [ ] Uso de RabbitMQ (queue, exchange, bindings)
- [ ] Despliegue de Valkey y Grafana en containerd sobre KubeVirt
- [ ] Configuración de HPA
- [ ] Publicación y consumo de imágenes desde Zot
- [ ] OCI Artifact: qué archivo se distribuyó y cómo se usó
- [ ] Pruebas realizadas y conclusiones (análisis 1 vs 2 réplicas)
- [ ] Screenshots de evidencia

### 5.2 Repositorio GitHub

- [x] Estructura modular de carpetas
- [x] README.md en raíz de PROYECTO2/
- [x] Todos los Dockerfiles presentes
- [x] Manifiestos organizados por componente con Kustomize
- [ ] Manual técnico completo en Markdown
- [ ] @CamiloSincal agregado como colaborador

### 5.3 Validación Final

- [ ] Levantar el sistema desde cero siguiendo la guía
- [ ] Locust genera tráfico → Grafana muestra datos actualizados
- [ ] Imágenes en Zot son consumidas correctamente
- [ ] HPA funciona correctamente
- [ ] KubeVirt VMs corren Valkey y Grafana

---

## Estructura del Repositorio

```
202308204_LAB_SO1_VacJun2026/
├── SKILL/                  ← enunciado, fases y planificación
└── PROYECTO2/
    ├── README.md           ← entrada operativa
    ├── Makefile            ← comandos estables
    ├── rust-api/
    │   ├── src/
    │   ├── Cargo.toml
    │   └── Dockerfile
    ├── go-d1/
    │   ├── rest-server/
    │   └── grpc-client/
    ├── go-d2/
    │   ├── grpc-server/
    │   └── rabbit-writer/
    ├── locust/
    │   ├── locustfile.py
    │   └── Dockerfile
    ├── proto/
    │   └── prediction.proto
    ├── infra/
    │   ├── kubernetes/     ← Kustomize y recursos por componente
    │   └── zot/            ← Caddy y configuración TLS
    └── scripts/            ← build, tests, deploy y validación
```

---

## Notas de Costos GCP

> ⚠️ Tener cuidado con los costos al trabajar en GCP.

- **Apagar el clúster** cuando no estés trabajando activamente
- Instancias N1 en GKE pueden costar si se dejan corriendo
- Usar la cuenta de créditos de estudiante
- Borrar recursos que no necesites (VMs, discos, IPs estáticas)
- Comando para detener el clúster: `gcloud container clusters resize <NOMBRE> --num-nodes=0 --zone=<ZONA>`

---

## Log de Avance

| Fecha | Fase | Tarea completada | Notas |
|---|---|---|---|
| 2026-06-26 | Fase 1 | GKE, Zot, imágenes, Rust y Go D1 desplegados | 3 nodos Ready; prueba Locust 2896 requests, 0 errores |
| 2026-06-26 | Fase 2 | Inicio de implementación | Referencias de clases 8, 11 y 12 auditadas |
| 2026-06-27 | Fase 2 | RabbitMQ, Go D2 y Gateway verificados | 2759 requests, 0 errores; cola persistente |
| 2026-06-27 | Auditoría | HTTPS, Zot, RabbitMQ y errores corregidos | TLS confiable; prueba 502/200; tests unitarios |
| 2026-06-27 | Modularización | Infraestructura y operación reorganizadas | Kustomize, Makefile, scripts y proto único |

> Llena esta tabla conforme avances en el proyecto.
