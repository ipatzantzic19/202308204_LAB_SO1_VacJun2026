# Planificación — Proyecto 2 Q.M.2026.K8s
**Isai Patzán · 202308204 · SOPES 1 · Vacaciones Junio 2026**

---

## Estado General

| Fase | Nombre | Estado | Clase asociada |
|---|---|---|---|
| **Fase 1** | Infraestructura Base | ✅ Corregida y verificada | Clases 6-9 y 12 |
| **Fase 2** | Comunicación Interna y Mensajería | ✅ Corregida y verificada | Clases 8, 11 y 12 |
| **Fase 3** | Consumo y Persistencia en VM | ✅ Implementada y verificada | Clases 11 y 13 |
| **Fase 4** | Visualización y Pruebas de Carga | ✅ Implementada y verificada | Clases 9, 11-13 |
| **Fase 5** | Documentación y Entrega | 🔄 Lista técnicamente; faltan controles de GitHub | Validación final |

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
> **Estado:** completada y revalidada el 28 de junio de 2026.

### 3.1 Consumer en Go

- [x] Nuevo módulo Go: `go-consumer`
- [x] Conectarse a RabbitMQ y consumir mensajes de la queue
- [x] Parsear el mensaje JSON
- [x] Guardar datos en Valkey con las claves apropiadas para el dashboard
- [x] Dockerizar y push a Zot
- [x] Deployment de K8s: `deployment-consumer.yaml`

### 3.2 KubeVirt

- [x] Instalar KubeVirt operator en el clúster GKE
- [x] Verificar que los nodos N1 soportan virtualización anidada
- [x] Instalar `virtctl` (CLI de KubeVirt)

### 3.3 VM 1 — Valkey

- [x] Crear `VirtualMachine` en K8s (YAML) para VM de Valkey
- [x] Dentro de la VM: instalar `containerd`
- [x] Crear contenedor con Valkey usando containerd (no Docker)
- [x] Exponer Valkey vía Service de K8s para que Consumer pueda conectar
- [x] Acotar series recientes a 10,000 elementos para evitar crecimiento ilimitado

### 3.4 Integrar Consumer con Valkey

- [x] Variable de entorno `VALKEY_ADDR` apuntando al service de Valkey
- [x] Prueba: enviar predicción y verificar con `valkey-cli GET <clave>`

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

- [x] Predicciones fluyen hasta Valkey
- [x] `valkey-cli KEYS "prediction:bra:*"` muestra claves con datos
- [x] Consumer no pierde mensajes (manejar acknowledgment de RabbitMQ)

---

> Nota: `valkey-vm` está `Running/Ready=True` y `valkey-service` responde funcionalmente. No se validó internamente con `virtctl ssh` porque la VM no tiene SSH activo en puerto 22. La evidencia funcional se obtuvo con `valkey-cli` desde Kubernetes.

## Fase 4 — Visualización y Pruebas de Carga

> **Clase asociada:** Clase 12+ del curso
> **Estado:** completada y revalidada el 28 de junio de 2026.

### 4.1 VM 2 — Grafana

- [x] Crear segunda `VirtualMachine` en K8s para Grafana (independiente de VM 1)
- [x] Instalar containerd dentro de la VM
- [x] Crear contenedores con Grafana y Prometheus usando containerd
- [x] Configurar Prometheus como datasource, alimentado por exporter propio desde Valkey
- [x] Exponer Grafana vía Service

### 4.2 Dashboard BRA

Crear paneles en Grafana:

- [x] Mayor goles local en un partido (Stat)
- [x] Menor goles local en un partido (Stat)
- [x] Mayor goles visitante en un partido (Stat)
- [x] Menor goles visitante en un partido (Stat)
- [x] Top equipos con más victorias predichas (Bar chart)
- [x] Top usuarios más activos (Bar chart)
- [x] Moda de goles predichos local (Stat)
- [x] Moda de goles predichos visitante (Stat)
- [x] Serie temporal BRA: goles local y visitante
- [x] Nombre del equipo: "BRA" (Text)
- [x] Total predicciones para BRA (Stat)

### 4.3 HPA para Rust

- [x] Crear `HorizontalPodAutoscaler` para el Deployment de Rust:
  - `minReplicas: 1`, `maxReplicas: 3`, `targetCPUUtilizationPercentage: 30`
- [x] Verificar que el Deployment tiene requests/limits de CPU configurados
- [x] Generar carga con Locust y observar el autoscaling 1 → 3 → 1

### 4.4 Pruebas de Carga con Locust

- [x] Prueba con **1 réplica** en Go D2 (registrar métricas)
- [x] Prueba con **2 réplicas** en Go D2 (registrar métricas)
- [x] Comparar throughput, latencia, errores
- [x] Verificar HPA del Rust actuando bajo carga

### 4.5 Pruebas de Fase 4

- [x] Dashboard Grafana provisionado con todos los paneles y métricas reales
- [x] HPA escala Rust correctamente bajo carga
- [x] Locust reporta < 1% de errores bajo carga sostenida
- [x] Analizar diferencia de rendimiento 1 vs 2 réplicas Go D2

---

## Fase 5 — Documentación y Entrega

### 5.1 Manual Técnico (Markdown)

- [x] Arquitectura general con diagrama
- [x] Flujo completo de datos (Locust → ... → Grafana)
- [x] Configuración de Gateway API (GatewayClass, Gateway, HTTPRoute)
- [x] Comunicación REST (Locust→Rust→GoD1) y gRPC (GoD1→GoD2)
- [x] Uso de RabbitMQ (queue y persistencia)
- [x] Despliegue de Valkey y Grafana en containerd sobre KubeVirt
- [x] Configuración de HPA
- [x] Publicación y consumo de imágenes desde Zot
- [x] OCI Artifact adicional documentado; requisito dispensado por el auxiliar
- [x] Pruebas realizadas y conclusiones (análisis 1 vs 2 réplicas)
- [x] Captura final del dashboard después del arranque limpio de la VM

### 5.2 Repositorio GitHub

- [x] Estructura modular de carpetas
- [x] README.md en raíz de PROYECTO2/
- [x] Todos los Dockerfiles presentes
- [x] Manifiestos organizados por componente con Kustomize
- [x] Manual técnico completo en Markdown
- [x] Documento integral de aprendizaje y explicación del flujo
- [x] Checklist reproducible de validación y entrega
- [ ] @CamiloSincal agregado como colaborador

### 5.3 Validación Final

- [x] Reaplicar manifiestos y validar el sistema siguiendo la guía
- [x] Locust genera tráfico y el exporter expone datos actualizados
- [x] Imágenes en Zot son consumidas correctamente
- [x] HPA funciona correctamente
- [x] KubeVirt VMs corren Valkey y Grafana

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
| 2026-06-28 | Fases 3 y 4 | Valkey, Grafana, dashboard y HPA validados | Dos VMI Ready; HPA 1→3→1 |
| 2026-06-28 | Rendimiento | Comparación Go D2 con 1 y 2 réplicas | 0 errores; +1.21% RPS con 2 réplicas |
| 2026-06-28 | Fase 5 | Manual, metodología y guía de calificación | Falta confirmar colaborador GitHub |

> Llena esta tabla conforme avances en el proyecto.
