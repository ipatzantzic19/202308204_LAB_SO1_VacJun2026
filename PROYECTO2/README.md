# Q.M.2026.K8s — Proyecto 2

Implementación modular del flujo de predicciones del Mundial 2026 en GKE.

## Estructura

```text
PROYECTO2/
├── go-d1/                 # REST server y cliente gRPC
├── go-d2/                 # servidor gRPC y writer RabbitMQ
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
make deploy      # operador, RabbitMQ, Secret y aplicaciones
make validate    # manifiestos, registry y estado del clúster
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

La documentación detallada por fase se encuentra en `../SKILL/FASES/`.
