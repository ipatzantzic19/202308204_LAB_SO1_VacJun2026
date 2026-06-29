# Q.M.2026.K8s — Proyecto 2

Implementación modular del flujo de predicciones del Mundial 2026 en GKE.

## Estructura

```text
PROYECTO2/
├── go-d1/                 # REST server y cliente gRPC
├── go-d2/                 # servidor gRPC y writer RabbitMQ
├── go-metrics-exporter/   # métricas Prometheus leídas desde Valkey
├── rust-api/              # API pública
├── locust/                # generador de carga
├── proto/                 # única fuente del contrato gRPC
├── infra/
│   ├── kubernetes/        # recursos por componente + Kustomize
│   └── zot/               # TLS y configuración de la VM registry
├── scripts/               # build, tests, despliegue y validación
└── Makefile               # interfaz operativa
```

Cada ejecutable conserva su propio módulo y Dockerfile. La infraestructura no contiene
código de aplicación. `proto/` es un módulo Go compartido y contiene tanto el contrato
editable como su único código generado; se regenera con `make proto`.

## Operación

```bash
make proto       # regenerar protobuf
make test        # ejecutar pruebas de todos los componentes
make build       # construir imágenes
make push        # publicar imágenes ya construidas
make images      # build + push
make artifact    # publicar y verificar prediction.proto como OCI Artifact (adicional)
make artifact-pull # descargar el contrato desde Zot
make deploy      # operador, RabbitMQ, Secret y aplicaciones
make validate    # manifiestos, registry y estado del clúster
make load-test   # evidencia Locust con 1 y 2 réplicas de Go D2
make validate-hpa # evidencia del ciclo automático 1 → 3 → 1
make zot-provision # ampliar disco gratuito, migrar y configurar Zot
```

Las tareas de imágenes aceptan `REGISTRY`. Las tareas de clúster aceptan `NAMESPACE`,
`RABBITMQ_NAMESPACE` y `RABBITMQ_OPERATOR_VERSION`.

`make zot-provision` amplía de 10 a 20 GB el disco `pd-standard` existente de la VM,
desactiva su `auto-delete`, migra de forma segura el catálogo legado y recrea Zot sin
publicar el puerto 5000 en la VPC. Antes de actuar comprueba que el total proyectado
no supere los 30 GB de disco estándar incluidos en Always Free. Puede parametrizarse
con `VM_NAME`, `ZONE`, `TARGET_BOOT_GB` y `ZOT_DOMAIN`.

Esta comprobación evita costo **adicional de almacenamiento** con el estado actual de
la cuenta. No convierte toda la plataforma en Always Free: los nodos
`n1-standard-4`, la VM `e2-medium`, el balanceador y las IPv4 públicas pueden consumir
créditos o generar cargos mientras estén activos. El node pool debe escalarse a cero
cuando no se esté trabajando en el proyecto.

El despliegue usa `zot.35-226-224-23.sslip.io` por HTTPS. No se guardan credenciales en
archivos: `sync-rabbitmq-secret.sh` copia el Secret generado por el operador mediante un
pipe y elimina metadatos específicos del namespace de origen.

## Observabilidad de fase 4

El flujo observado es `Valkey → go-metrics-exporter → Prometheus → Grafana`. El exporter
publica las métricas obligatorias de BRA en el puerto 9100. Prometheus y Grafana se ejecutan
con `containerd` dentro de `grafana-vm`; el datasource y el dashboard se provisionan al
arrancar. El JSON exportable está en
`infra/kubernetes/grafana/dashboards/quiniela-bra.json`.

El consumer conserva hasta 10,000 predicciones recientes relacionadas con BRA. El dashboard
calcula máximos, mínimos y modas usando hasta cinco predicciones del enfrentamiento más
reciente; los rankings de victorias y usuarios son históricos. El total de predicciones de
BRA permanece como contador exacto.

El Secret de cloud-init no se versiona. `scripts/render-grafana-cloudinit.sh` combina la
plantilla con el dashboard y `make deploy` crea el Secret directamente en Kubernetes.
La VM deshabilita autenticación SSH por contraseña, genera una contraseña aleatoria para
el administrador de Grafana y permite visualizar dashboards anónimamente con rol `Viewer`.

### Justificación de los PVC como discos `virtio`

Los PVC `grafana-data` y `prometheus-data` se adjuntan a la VM como discos de bloque con bus
`virtio` y se formatean/montan dentro del guest. Esta decisión sustituye deliberadamente
`virtiofs`: los volúmenes `standard-rwo` de GKE son discos persistentes de bloque y su uso
directo evita el proceso adicional de `virtiofsd`, reduce dependencias entre host y guest y
conserva semántica nativa de `fsync` para las bases de datos de Grafana y Prometheus. La
persistencia requerida se mantiene porque los datos viven en PVC, no en el `containerDisk`.
La contrapartida es que la VMI no admite migración en vivo mientras use PVC RWO; para este
proyecto se priorizan persistencia y simplicidad sobre live migration.

### Pruebas de carga

`make load-test` obtiene la IP pública del Gateway, fija temporalmente Rust en una réplica,
compara Go D2 con una y dos réplicas y restaura el HPA obligatorio 1–3. Los parámetros `USERS`,
`SPAWN_RATE`, `DURATION` y `RESULTS_DIR` permiten repetir el experimento. Los CSV, HTML y
la comparación Markdown se guardan bajo `evidence/locust/`.

El Deployment `locust` mantiene cinco usuarios simulados activos en GKE contra el Gateway
público. Se puede detener o reanudar sin eliminarlo con:

```bash
kubectl scale deployment/locust -n sopes1-p2 --replicas=0
kubectl scale deployment/locust -n sopes1-p2 --replicas=1
```

`make validate-hpa` ejecuta una carga más intensa, registra cada cinco segundos las
réplicas y CPU observadas, y exige completar el ciclo automático `1 → 3 → 1`. La evidencia
queda bajo `evidence/hpa/`.

## Documentación de entrega

- [Manual técnico](docs/ManualTecnico.md)
- [Aprendizaje del proyecto](docs/Aprendizaje.md)
- [Guía de calificación](docs/GuiaCalificacion.md)
- [Metodología](docs/Metodologia.md)
- [Auditoría de requisitos](docs/AuditoriaRequisitos.md)
- [Checklist de entrega](docs/ChecklistEntrega.md)

La documentación detallada por fase se encuentra en `../SKILL/FASES/`.
