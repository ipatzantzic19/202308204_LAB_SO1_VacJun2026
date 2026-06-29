# Auditoría de requisitos

Auditoría realizada contra `SKILL/SKILL.md` el 28 de junio de 2026.

| Requisito | Estado | Evidencia principal |
|---|---|---|
| GCP, GKE Standard y nodos N1 con virtualización anidada | Cumple | Fases 1 y 3; KubeVirt estuvo `Deployed` |
| Locust y JSON aleatorio requerido | Cumple | `locust/locustfile.py`, evidencia Locust |
| Gateway API, sin Ingress, ruta `/grpc-202308204` | Cumple | `gateway/gateway.yaml`; Gateway `Programmed=True` |
| Rust REST y HPA 1–3 al 30% CPU | Cumple | `rust-api/`, `rust-api/hpa.yaml`, evidencia HPA |
| Go D1 con REST server + cliente gRPC | Cumple | Deployment de dos contenedores |
| Go D2 con servidor gRPC + writer AMQP | Cumple | Deployment de dos contenedores |
| RabbitMQ obligatorio, cola durable y mensajes persistentes | Cumple | Cluster Operator; cola `predictions` |
| Consumer Go y ACK posterior al guardado | Cumple | `go-consumer/main.go` |
| Valkey bajo containerd en VM KubeVirt | Cumple | `valkey/vm.yaml`; Service y persistencia funcional |
| Grafana bajo containerd en VM independiente | Cumple | `grafana/vm.yaml`, cloud-init y evidencia runtime |
| Dashboard del equipo BRA con paneles requeridos | Cumple técnicamente | Dashboard JSON y exporter; falta renovar captura final |
| Zot en VM externa, HTTPS e imágenes consumidas desde Zot | Cumple | Catálogo Zot y referencias de imágenes |
| Comparación de 1 vs. 2 réplicas | Cumple | Go D2: `evidence/locust/20260628T225747Z/` |
| Manual técnico Markdown, ejecución, metodología y conclusiones | Cumple | Carpeta `docs/` |
| OCI Artifact | Dispensado; además implementado | `prediction-proto:v1`, `make artifact` |
| Dapr | No implementado, opcional | No afecta los 60 puntos del proyecto |
| k3s local | No implementado, opcional | No afecta los 60 puntos del proyecto |

## Controles manuales restantes antes de presentar

1. Encender los node pools y ejecutar `make validate`.
2. Abrir Grafana y guardar una captura final del dashboard con datos.
3. Confirmar en GitHub que el repositorio sea privado y que `CamiloSincal` sea colaborador.
4. Confirmar que todos los cambios de entrega estén versionados y enviados al remoto.

No se reactivaron los node pools durante el cierre de esta auditoría para evitar generar costo
sin autorización. La última validación activa previa al apagado mostró Pods `Running`, ambas
VMI `Ready=True`, Gateway programado, cola vacía y flujo público HTTP 200.
