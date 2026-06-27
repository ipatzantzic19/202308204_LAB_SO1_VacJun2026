# Fase 2 — Comunicación interna y publicación en RabbitMQ

**Proyecto 2 · 202308204 · SOPES 1 · Vacaciones Junio 2026**

## Objetivo

Completar en GCP el flujo:

```text
Locust -> Gateway /grpc-202308204 -> Rust -> Go D1
       -> gRPC -> Go D2 -> writer AMQP -> RabbitMQ
```

## Patrones oficiales usados

- Clase 8: contrato protobuf, cliente y servidor gRPC en Go.
- Clase 11: RabbitMQ Cluster Operator, cola durable y variables de entorno AMQP.
- Clase 12: Pods con múltiples contenedores, DNS de Services y Gateway API de GKE.

El ejemplo de Clase 12 integra gRPC y AMQP en un proceso. Aquí se separan porque el
enunciado exige que Go D2 tenga dos contenedores.

## Estructura

```text
PROYECTO2/
├── go-d2/
│   ├── grpc-server/
│   └── rabbit-writer/
└── infra/kubernetes/
    ├── rabbitmq/
    ├── go-d2/
    └── gateway/
```

## Paso 1 — RabbitMQ Cluster Operator

```bash
kubectl create namespace rabbitmq-system
kubectl apply -f https://github.com/rabbitmq/cluster-operator/releases/download/v2.21.1/cluster-operator.yml
kubectl rollout status deployment/rabbitmq-cluster-operator -n rabbitmq-system
kubectl apply -f PROYECTO2/infra/kubernetes/rabbitmq/cluster.yaml
kubectl get rabbitmqcluster,pods,svc -n rabbitmq-system
```

El cluster usa una réplica, imagen management, PVC de 10 GiB, requests/limits y Service
ClusterIP. Las credenciales se leen del Secret creado por el operador y se copian como
Secret al namespace `sopes1-p2`; nunca se guardan en Git.

## Paso 2 — Go D2 con dos contenedores

### Container A: gRPC server

- Escucha `:50051`.
- Implementa `MatchPredictionService.SendPrediction`.
- Convierte el protobuf a JSON.
- Envía el JSON por HTTP a `http://localhost:9100/publish`.
- Devuelve `published` solo si el writer confirma la publicación.

### Container B: RabbitMQ writer

- Escucha `:9100` dentro del mismo Pod.
- Lee `RABBITMQ_URL` y `RABBITMQ_QUEUE`.
- Declara la cola durable `predictions`.
- Publica mensajes persistentes en el default exchange.
- Si una conexión establecida se pierde, reconecta y reintenta la publicación una vez
  sin depender de que Kubernetes reinicie el contenedor.
- Expone `/health` para readiness de RabbitMQ y `/live` para verificar el proceso sin
  provocar reinicios durante una desconexión recuperable.

## Paso 3 — Go D1 gRPC real

El container `grpc-client` conserva su puente HTTP `/send`, transforma los equipos a
`Teams` y llama a:

```text
go-d2-service.sopes1-p2.svc.cluster.local:50051
```

Debe usar timeout y propagar error HTTP si gRPC falla.

## Paso 4 — Imágenes Zot por HTTPS

```text
zot.35-226-224-23.sslip.io/sopes1/go-d1-grpc-client:v3
zot.35-226-224-23.sslip.io/sopes1/go-d2-grpc-server:v2
zot.35-226-224-23.sslip.io/sopes1/go-d2-rabbit-writer:v4
zot.35-226-224-23.sslip.io/library/rabbitmq:3.13-management
```

Todas se publican o espejan en Zot y se verifican mediante HTTPS en
`/v2/.../tags/list`. RabbitMQ no se descarga directamente de Docker Hub en GKE.

## Paso 5 — Manifiestos GKE

- Go D2: Deployment de una réplica con dos contenedores y Service TCP 50051.
- RabbitMQ URL desde Secret (`secretKeyRef`).
- Requests/limits en ambos contenedores.
- Readiness y liveness en el writer.
- Go D1 usa el cliente gRPC `v3` y Go D2 el servidor gRPC `v2`, ambos enlazados al
  módulo compartido `PROYECTO2/proto`.
- RabbitMQ usa la imagen `library/rabbitmq:3.13-management` alojada en Zot.
- El writer `v4` verifica y recupera la conexión RabbitMQ cada cinco segundos sin
  depender del tráfico ni del reinicio del Pod.

## Paso 6 — Gateway API

Patrón de Clase 12:

- `GatewayClass`: `gke-l7-global-external-managed`.
- `Gateway` HTTP puerto 80 en `sopes1-p2`.
- `HealthCheckPolicy` apuntando a `/health` de Rust.
- `HTTPRoute` con `PathPrefix /grpc-202308204`.
- `URLRewrite` reemplaza el prefijo por `/` antes de llegar a Rust.
- `rust-api-service` cambia a `ClusterIP`.

Rust debe exponer `GET /health` para el balanceador.

## Paso 7 — Validación en GCP

```bash
kubectl get pods -n rabbitmq-system
kubectl get pods -n sopes1-p2
kubectl get gateway,httproute -n sopes1-p2
kubectl logs -n sopes1-p2 deployment/go-d2 -c grpc-server
kubectl logs -n sopes1-p2 deployment/go-d2 -c rabbit-writer
```

Prueba pública:

```bash
curl -X POST http://GATEWAY_IP/grpc-202308204 \
  -H 'Content-Type: application/json' \
  -d '{"home_team":"BRA","away_team":"MEX","home_goals":2,"away_goals":1,"username":"user_202308204","timestamp":"2026-06-27T00:00:00Z"}'
```

La cola se verifica con la API de management o `rabbitmqctl list_queues name messages`.

Las pruebas unitarias de ambos componentes de Go D2 se ejecutan con `make test` y
cubren la conversión gRPC, los fallos del writer, validación HTTP y health.

## Criterio de finalización

- RabbitMQ y su PVC están `Ready` en GKE.
- Go D1 y Go D2 están `2/2 Running`.
- Gateway y HTTPRoute están `Programmed/Accepted`.
- Una petición pública devuelve `ok`.
- Los logs prueban REST -> gRPC -> writer.
- La cola `predictions` aumenta al enviar predicciones.
- Al detener Go D2, la ruta pública devuelve `502` y Locust contabiliza el fallo.

## Resultado verificado en GCP — 2026-06-27, después de auditoría

- RabbitMQ Cluster Operator y `rabbitmq-cluster` disponibles con PVC de 10 GiB.
- Rust `1/1`, Go D1 `2/2` y Go D2 `2/2` en estado Running.
- Gateway `gke-l7-global-external-managed`: `136.68.202.37`, `Programmed=True`.
- Ruta pública: `POST /grpc-202308204`.
- La cola `predictions` sobrevivió al escalado del node pool a cero.
- Locust: 2,759 requests, 0 errores, 46.18 req/s, promedio 123 ms.
- Zot responde por HTTPS con certificado confiable e IP estática.
- RabbitMQ fue espejado en Zot y el Pod lo descargó desde ese registry.
- Prueba negativa: Go D2 en cero réplicas produjo HTTP `502` en la ruta pública.
- Prueba de recuperación: Go D2 restaurado produjo HTTP `200`.
- Pruebas unitarias: Rust (2), Go D1 REST (3) y Go D1 gRPC client (3), todas exitosas.

Los nodos nuevos descargan las imágenes mediante HTTPS estándar. Ya no existe un script
para modificar containerd por nodo.
