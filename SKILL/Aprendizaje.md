# Guía de Aprendizaje — Proyecto 2 Q.M.2026.K8s
**Isai Patzán · 202308204 · SOPES 1 · Vacaciones Junio 2026**

> Esta guía explica desde cero cada tecnología y concepto del Proyecto 2.
> Parte del conocimiento del Proyecto 1 (Go, Docker, Valkey, Grafana) y construye
> sobre él con Kubernetes, Rust, gRPC, RabbitMQ y KubeVirt.

---

## Tabla de Contenidos

1. [Del Proyecto 1 al Proyecto 2 — Qué cambia](#1-del-proyecto-1-al-proyecto-2--qué-cambia)
2. [Concepto 1: Kubernetes — Orquestación de Contenedores](#2-concepto-1-kubernetes--orquestación-de-contenedores)
3. [Concepto 2: Google Kubernetes Engine (GKE) y GCP](#3-concepto-2-google-kubernetes-engine-gke-y-gcp)
4. [Concepto 3: Kubernetes Gateway API](#4-concepto-3-kubernetes-gateway-api)
5. [Concepto 4: Rust — API REST de Alto Rendimiento](#5-concepto-4-rust--api-rest-de-alto-rendimiento)
6. [Concepto 5: gRPC — Comunicación entre Microservicios](#6-concepto-5-grpc--comunicación-entre-microservicios)
7. [Concepto 6: RabbitMQ — Broker de Mensajería](#7-concepto-6-rabbitmq--broker-de-mensajería)
8. [Concepto 7: KubeVirt — VMs dentro de Kubernetes](#8-concepto-7-kubevirt--vms-dentro-de-kubernetes)
9. [Concepto 8: containerd — Runtime de Contenedores](#9-concepto-8-containerd--runtime-de-contenedores)
10. [Concepto 9: Zot — Registry OCI](#10-concepto-9-zot--registry-oci)
11. [Concepto 10: HPA — Escalado Automático](#11-concepto-10-hpa--escalado-automático)
12. [Concepto 11: Locust — Pruebas de Carga](#12-concepto-11-locust--pruebas-de-carga)
13. [Concepto 12: Pods con Múltiples Contenedores](#13-concepto-12-pods-con-múltiples-contenedores)
14. [El Flujo Completo Integrado](#14-el-flujo-completo-integrado)
15. [Preguntas de Defensa y Respuestas](#15-preguntas-de-defensa-y-respuestas)

---

## 1. Del Proyecto 1 al Proyecto 2 — Qué cambia

En el Proyecto 1 construiste un sistema de observabilidad que corría **en una sola máquina Linux** con Docker Compose. El Daemon Go leía un módulo de kernel y guardaba datos en Valkey para visualizarlos en Grafana.

El Proyecto 2 eleva todo eso a **cloud y escala**:

| Aspecto | Proyecto 1 | Proyecto 2 |
|---|---|---|
| Infraestructura | Una máquina Linux local | GCP + GKE (clúster en la nube) |
| Orquestación | Docker Compose | Kubernetes |
| Exposición | localhost | Kubernetes Gateway API (IP pública) |
| Lenguajes | Go + C | **Rust** + Go + Python |
| Comunicación entre servicios | HTTP local | **gRPC** |
| Mensajería | — | **RabbitMQ** (broker dedicado) |
| Almacenamiento | Valkey en contenedor local | Valkey en **VM dentro de KubeVirt** |
| Visualización | Grafana en Docker | Grafana en **VM dentro de KubeVirt** |
| Registry de imágenes | Docker Hub / local | **Zot** (propio, externo al clúster) |
| Escalado | Manual | **HPA** automático |
| Carga de trabajo | Módulo de kernel | **Locust** genera tráfico simulado |

Lo que aprendiste en el Proyecto 1 que **reutilizas directamente**:
- Go para servicios del backend
- Valkey (comandos, tipos de datos)
- Grafana (dashboards, datasources, queries)
- Docker (build, Dockerfile, imágenes)
- JSON y serialización

---

## 2. Concepto 1: Kubernetes — Orquestación de Contenedores

### ¿Qué problema resuelve Kubernetes?

En el Proyecto 1 tenías un `docker-compose.yml` que levantaba 4 servicios en una máquina. Funciona bien para desarrollo, pero en producción necesitas:

- **Alta disponibilidad:** si un contenedor cae, que se reinicie solo
- **Escalado:** si hay mucho tráfico, lanzar más réplicas automáticamente
- **Distribución:** correr en múltiples máquinas (nodos) sin gestión manual
- **Despliegues sin tiempo de inactividad:** actualizar la app sin cortar el servicio

Kubernetes (K8s) es el estándar de la industria para resolver todo esto.

### Conceptos clave de Kubernetes

```
CLÚSTER
│
├── Nodo 1 (VM en GCP)
│   ├── Pod A → Container(s)
│   └── Pod B → Container(s)
│
├── Nodo 2 (VM en GCP)
│   └── Pod C → Container(s)
│
└── Control Plane (gestiona todo)
    ├── API Server
    ├── Scheduler
    └── Controller Manager
```

**Pod:** Unidad mínima de despliegue. Puede tener 1 o más contenedores que comparten red y almacenamiento. En este proyecto, Go D1 y Go D2 son Pods con **2 contenedores cada uno**.

**Deployment:** Define cuántas réplicas de un Pod deben existir y qué imagen usar. Si un Pod muere, el Deployment lo recrea.

**Service:** Expone un Deployment con un nombre DNS estable. Los Pods de diferentes Deployments se comunican por nombre de Service, no por IP (las IPs de los Pods cambian).

**Namespace:** Espacio lógico para agrupar recursos. Este proyecto usa `sopes1-p2`.

**ConfigMap / Secret:** Almacena configuración y credenciales que los Pods consumen como variables de entorno o archivos.

```yaml
# Analogía con Docker Compose → Kubernetes
docker-compose.yml       → Deployment + Service (YAML)
services:                → spec.template.spec.containers:
  nombre:                →   - name:
    image:               →     image:
    ports:               →     ports:
    environment:         →     env:
    volumes:             →     volumeMounts:
networks:                → Service (ClusterIP, NodePort, LoadBalancer)
```

### Comandos fundamentales

```bash
# Ver qué hay en el clúster
kubectl get pods -n sopes1-p2
kubectl get deployments -n sopes1-p2
kubectl get services -n sopes1-p2
kubectl get all -n sopes1-p2

# Ciclo de vida básico
kubectl apply -f archivo.yaml     # Crear/actualizar recursos
kubectl delete -f archivo.yaml    # Eliminar recursos
kubectl describe pod <nombre>     # Detalles y eventos
kubectl logs <pod> -c <container> # Ver logs
kubectl exec -it <pod> -- sh      # Entrar al contenedor
```

---

## 3. Concepto 2: Google Kubernetes Engine (GKE) y GCP

### ¿Por qué GCP y no local?

KubeVirt (Fase 3) requiere **virtualización anidada**: correr VMs dentro de un clúster Kubernetes, que a su vez corre en VMs. Solo algunos proveedores cloud lo soportan. GCP con instancias **N1** lo permite nativamente.

### Instancias N1 y virtualización anidada

Las instancias N1 de GCP son la familia de máquinas virtuales más clásica de Google Cloud. Son las únicas en GCP que soportan la flag `--enable-nested-virtualization`, necesaria para que KubeVirt pueda crear VMs dentro de los nodos del clúster.

```
GCP (Nube)
└── VM N1 (Nodo de GKE) ← --enable-nested-virtualization
    └── Kubernetes
        └── KubeVirt
            └── VM de KubeVirt (VM dentro de VM)
                └── containerd
                    └── Contenedor Valkey
```

Esto es el concepto de **virtualización anidada** o nested virtualization: una máquina virtual que puede ejecutar otras máquinas virtuales.

### GKE vs Kubernetes "a mano"

GKE (Google Kubernetes Engine) es Kubernetes gestionado: Google se encarga del control plane, actualizaciones, certificados TLS del API server, etc. Tú solo gestionas los nodos (workers) y lo que despliegas encima.

```bash
# Todo lo que necesitas para tener un clúster
gcloud container clusters create mi-cluster \
  --machine-type=n1-standard-4 \
  --num-nodes=3 \
  --enable-nested-virtualization
# GKE crea 3 VMs N1, instala Kubernetes, configura networking. Listo.
```

---

## 4. Concepto 3: Kubernetes Gateway API

### La evolución del Ingress

En Kubernetes, la forma clásica de exponer servicios al exterior es **Ingress**. Pero tiene limitaciones: difícil de extender, comportamiento inconsistente entre proveedores. La **Gateway API** es su sucesor oficial, más expresivo y extensible.

### Los tres objetos de Gateway API

```
Internet
    │
    ▼
GatewayClass         ← "Qué tipo de gateway usar" (el proveedor)
    │
    ▼
Gateway              ← "La puerta de entrada" (IP pública + puerto)
    │
    ▼
HTTPRoute            ← "Las reglas de routing" (qué path va a qué servicio)
```

```yaml
# GatewayClass: define el controlador que implementa el gateway
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: mi-gateway-class
spec:
  controllerName: "..."  # Depende del proveedor (GKE, nginx, etc.)

# Gateway: la entrada principal
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: mi-gateway
spec:
  gatewayClassName: mi-gateway-class
  listeners:
  - name: http
    port: 80
    protocol: HTTP

# HTTPRoute: reglas de enrutamiento
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: quiniela-route
spec:
  parentRefs:
  - name: mi-gateway
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /grpc-202308204     ← nuestra ruta
    backendRefs:
    - name: rust-api-service      ← va a Rust
      port: 80
```

### Por qué `/grpc-202308204` se llama "grpc" si usa HTTP

El nombre de la ruta referencia al mecanismo de comunicación **interno** que usa ese flujo (gRPC entre los servicios Go), no al protocolo externo (que sí es HTTP/REST entre Locust y Rust). Es una convención de nomenclatura del proyecto para diferenciar el flujo gRPC del flujo Dapr (que no implementamos).

---

## 5. Concepto 4: Rust — API REST de Alto Rendimiento

### ¿Por qué Rust para la API de entrada?

Rust está diseñado para ser rápido (comparable a C/C++) y seguro en memoria sin necesidad de garbage collector. Para un API que recibe el tráfico de Locust (potencialmente cientos de requests/segundo), Rust es una elección excelente.

En contraste con Go (que tiene GC) o Python (GIL), Rust garantiza rendimiento predecible bajo carga.

### Conceptos clave de Rust para este proyecto

```rust
// El ownership: cada valor tiene exactamente un dueño
let s = String::from("hello"); // s es el dueño
let s2 = s;                    // s2 es el nuevo dueño, s ya no vale

// Los Result: manejo de errores explícito (como Go, pero más estricto)
match resultado {
    Ok(valor) => { /* éxito */ },
    Err(e)    => { /* error */ },
}

// async/await: igual que en JavaScript/Go pero con tokio como runtime
async fn mi_funcion() -> Result<String, Error> {
    let resp = cliente.get(url).send().await?;  // el ? propaga el error
    Ok(resp.text().await?)
}
```

### El framework Axum

Axum es el framework web de Rust más usado actualmente. Se integra perfectamente con `tokio` (el runtime async de Rust):

```rust
// Definir un handler (función que maneja requests)
async fn handle_prediction(Json(payload): Json<Prediction>) -> Json<Response> {
    // payload ya está deserializado automáticamente
    // ...
    Json(Response { status: "ok".to_string() })
}

// Registrar la ruta
let app = Router::new()
    .route("/", post(handle_prediction));  // POST / → handle_prediction
```

### Comparativa de frameworks web

| Aspecto | Axum (Rust) | axum | Gin (Go) | FastAPI (Python) |
|---|---|---|---|---|
| Rendimiento | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| Seguridad memoria | Garantizada | Garantizada | GC | GC + GIL |
| Curva aprendizaje | Alta | Alta | Baja | Muy baja |

---

## 6. Concepto 5: gRPC — Comunicación entre Microservicios

### HTTP/REST vs gRPC

En el Proyecto 1, el Daemon Go se comunicaba con Docker usando comandos. Aquí los microservicios se comunican con **gRPC**.

```
REST (HTTP)
  - Texto plano (JSON)
  - Cualquier cliente puede llamarlo
  - Latencia un poco mayor por serialización JSON
  - Autodescripción limitada

gRPC (HTTP/2 + Protocol Buffers)
  - Binario (más compacto)
  - Contrato estricto definido en .proto
  - Serialización más eficiente
  - Streaming bidireccional posible
  - Ideal para comunicación interna entre microservicios
```

### Protocol Buffers (protobuf) — El contrato

El archivo `.proto` define la "interfaz" que deben cumplir el cliente y el servidor. Si cambias el proto, debes recompilar ambos lados.

```protobuf
// Esto define:
// 1. El mensaje que se envía (MatchPredictionRequest)
// 2. Los valores válidos para los equipos (Teams)
// 3. La respuesta (MatchPredictionResponse)
// 4. El servicio y sus métodos (MatchPredictionService.SendPrediction)

service MatchPredictionService {
  rpc SendPrediction (MatchPredictionRequest) returns (MatchPredictionResponse);
}
```

### Flujo de uso de gRPC en Go

```
1. Escribir prediction.proto
2. Compilar con protoc → genera prediction.pb.go y prediction_grpc.pb.go
3. Servidor (Go D2): implementar la interfaz generada
4. Cliente (Go D1): usar el stub generado para llamar al servidor

// En el servidor:
type server struct {
    pb.UnimplementedMatchPredictionServiceServer
}

func (s *server) SendPrediction(ctx context.Context, req *pb.MatchPredictionRequest) (*pb.MatchPredictionResponse, error) {
    // Aquí procesas la predicción y la publicas en RabbitMQ
    log.Printf("Recibido: %s vs %s", req.HomeTeam, req.AwayTeam)
    return &pb.MatchPredictionResponse{Status: "ok"}, nil
}

// En el cliente:
conn, _ := grpc.NewClient("go-d2-service:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
client := pb.NewMatchPredictionServiceClient(conn)
resp, _ := client.SendPrediction(ctx, &pb.MatchPredictionRequest{...})
```

### gRPC dentro del clúster de Kubernetes

Un aspecto importante: en Kubernetes, los Pods se comunican por nombre de **Service**. El cliente gRPC de Go D1 debe apuntar a `go-d2-service.sopes1-p2.svc.cluster.local:50051`.

Esto funciona porque Kubernetes tiene un DNS interno que resuelve nombres de Service a IPs de ClusterIP.

---

## 7. Concepto 6: RabbitMQ — Broker de Mensajería

### ¿Qué es un Message Broker?

Un broker de mensajería es un intermediario entre servicios. En lugar de que el Publicador llame directamente al Consumidor (acoplamiento directo), ambos hablan con RabbitMQ:

```
Sin broker:                    Con broker:
Publicador ──directo──► Consumidor    Publicador ──► [RabbitMQ] ──► Consumidor
  - Si el consumidor cae, se pierden mensajes     - Mensajes persisten en la cola
  - El publicador espera al consumidor           - El publicador no espera
  - Difícil escalar                              - Múltiples consumidores posibles
```

### Conceptos de RabbitMQ

**Exchange:** El punto de entrada donde los publicadores envían mensajes. El exchange decide a qué cola(s) enviar el mensaje (según el tipo y el routing key).

**Queue:** La cola donde se almacenan los mensajes hasta que un consumidor los procese.

**Binding:** La regla que conecta un exchange con una queue.

**Consumer:** El proceso que lee mensajes de la queue y los procesa.

```
Go D2 Publisher ──AMQP──► [Exchange] ──binding──► [Queue] ──► Go Consumer
                                                              └─► Go Consumer (escalado)
```

### AMQP vs HTTP

RabbitMQ usa el protocolo **AMQP** (Advanced Message Queuing Protocol). En Go se usa la librería `amqp091-go`:

```go
import amqp "github.com/rabbitmq/amqp091-go"

// Conectar
conn, _ := amqp.Dial("amqp://admin:password@rabbitmq-service:5672/")
ch, _ := conn.Channel()

// Declarar queue (idempotente)
q, _ := ch.QueueDeclare("predicciones", true, false, false, false, nil)

// Publicar mensaje
ch.Publish("", q.Name, false, false, amqp.Publishing{
    ContentType: "application/json",
    Body:        []byte(mensajeJSON),
})

// Consumir (en el Consumer)
msgs, _ := ch.Consume(q.Name, "", false, false, false, false, nil)
for msg := range msgs {
    // procesar msg.Body
    msg.Ack(false) // confirmar que se procesó
}
```

### Acknowledgment — Por qué es crítico

Cuando el Consumer lee un mensaje, RabbitMQ lo mantiene "en vuelo" hasta recibir un **Ack** (acknowledgment). Si el Consumer falla antes de hacer Ack, RabbitMQ reencola el mensaje automáticamente. Esto garantiza que **ningún mensaje se pierde**.

```go
// Correcto: Ack solo DESPUÉS de guardar en Valkey
data := procesarMensaje(msg.Body)
guardarEnValkey(data)
msg.Ack(false)  ← solo aquí

// Incorrecto: Ack inmediato (si falla el guardado, el mensaje se pierde)
msg.Ack(false)
guardarEnValkey(data)  ← si esto falla, el mensaje se perdió
```

---

## 8. Concepto 7: KubeVirt — VMs dentro de Kubernetes

### ¿Qué es KubeVirt?

KubeVirt extiende Kubernetes para poder gestionar **máquinas virtuales completas** junto con los contenedores normales. Una VM de KubeVirt es un objeto de Kubernetes como cualquier Deployment o Service, pero en lugar de correr contenedores, corre una VM completa.

### ¿Por qué VMs y no solo contenedores para Valkey y Grafana?

El enunciado lo requiere explícitamente para practicar el concepto de **virtualización dentro de Kubernetes**. En la industria, KubeVirt se usa para migrar workloads de VMs a Kubernetes sin reescribir todo: la VM corre en K8s pero los procesos dentro siguen siendo procesos normales de Linux.

### Arquitectura KubeVirt en este proyecto

```
GKE Cluster
└── Nodo N1 (con nested virtualization)
    └── KubeVirt Operator
        ├── VirtualMachine: vm-valkey
        │   └── (VM Ubuntu/Debian corriendo en el nodo)
        │       └── containerd (instalado dentro de la VM)
        │           └── Contenedor Valkey
        │               └── proceso valkey-server
        │
        └── VirtualMachine: vm-grafana
            └── (VM Ubuntu/Debian corriendo en el nodo)
                └── containerd (instalado dentro de la VM)
                    └── Contenedor Grafana
                        └── proceso grafana-server
```

### Objetos de Kubernetes que agrega KubeVirt

```yaml
# VirtualMachine: define la VM (como un Deployment, pero para VMs)
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: vm-valkey
spec:
  running: true
  template:
    spec:
      domain:
        devices:
          disks: [...]
        resources:
          requests:
            memory: "2Gi"
      volumes: [...]

# VirtualMachineInstance: la instancia corriendo (como un Pod)
# Se crea automáticamente cuando VirtualMachine.running = true
```

```bash
# Comandos virtctl (análogo a kubectl pero para VMs)
virtctl start vm-valkey        # Encender VM
virtctl stop vm-valkey         # Apagar VM
virtctl ssh vm-valkey          # SSH a la VM
virtctl console vm-valkey      # Consola serie de la VM
kubectl get vmi                # Ver instancias de VM corriendo
```

---

## 9. Concepto 8: containerd — Runtime de Contenedores

### ¿Qué es containerd?

`containerd` es el **runtime de contenedores** que usa Kubernetes internamente (y GKE). Es el componente que realmente descarga imágenes y ejecuta contenedores. Docker también usa containerd internamente.

La diferencia práctica: en el Proyecto 1 usabas `docker run` para lanzar contenedores. En las VMs de KubeVirt de este proyecto, usarás `containerd` directamente (sin la capa de Docker).

### Comandos de containerd dentro de la VM

```bash
# Dentro de la VM de KubeVirt (via virtctl ssh o virtctl console):

# Herramienta CLI de containerd: ctr (nerdctl es la alternativa amigable)
# Instalar nerdctl (más parecido a docker)
wget https://github.com/containerd/nerdctl/releases/download/v1.7.0/nerdctl-1.7.0-linux-amd64.tar.gz
tar -xzf nerdctl-*.tar.gz
sudo mv nerdctl /usr/local/bin/

# Pull de imagen desde Zot
sudo nerdctl pull <IP-ZOT>:5000/sopes1/valkey:latest

# Correr contenedor de Valkey
sudo nerdctl run -d \
  --name valkey \
  -p 6379:6379 \
  <IP-ZOT>:5000/sopes1/valkey:latest

# Verificar
sudo nerdctl ps
sudo nerdctl logs valkey

# Equivalente con ctr (más bajo nivel)
sudo ctr images pull <IP-ZOT>:5000/sopes1/valkey:latest
sudo ctr run -d <IP-ZOT>:5000/sopes1/valkey:latest valkey-container
```

### La cadena de abstracción

```
Docker CLI (lo que usabas en P1)
    ↓
dockerd (daemon de Docker)
    ↓
containerd (runtime de contenedores) ← lo usas directamente en las VMs
    ↓
runc (runtime de bajo nivel, ejecuta el proceso)
    ↓
Proceso del contenedor (Valkey, Grafana, etc.)
```

---

## 10. Concepto 9: Zot — Registry OCI

### ¿Qué es un Container Registry?

Es un servidor donde se almacenan las imágenes Docker (y otros artefactos OCI). Docker Hub es el registry público más conocido. En este proyecto usamos **Zot**, un registry privado propio que corre en una VM de GCP.

### ¿Por qué un registry propio?

- **Privacidad:** las imágenes del proyecto no son públicas
- **Velocidad:** el clúster GKE está en la misma región que la VM de Zot (menos latencia que Docker Hub)
- **Control:** tú controlas qué imágenes existen y cuánto tiempo se mantienen
- **Requisito del enunciado:** practicar la gestión de un registry privado

### OCI — Open Container Initiative

OCI es el estándar que define el formato de las imágenes de contenedores y cómo se distribuyen. Cualquier herramienta compatible con OCI (Docker, podman, containerd, nerdctl) puede usar un registry OCI como Zot.

### OCI Artifact — Más que imágenes

Un **OCI Artifact** es cualquier archivo distribuido usando el protocolo de registry OCI. No tiene que ser una imagen Docker. En este proyecto, el proto file (`prediction.proto`) puede distribuirse como un OCI Artifact:

```bash
# Push del proto como artefacto
oras push <IP-ZOT>:5000/sopes1/proto:v1 \
  proto/prediction.proto:application/vnd.worldcup.proto.v1

# Pull del proto (en otro servicio que lo necesite)
oras pull <IP-ZOT>:5000/sopes1/proto:v1
# → descarga prediction.proto en el directorio actual
```

**¿Por qué es útil?** En vez de copiar el archivo proto manualmente a cada servicio Go, todos lo descargan del registry en tiempo de build. Así hay una única fuente de verdad para el contrato gRPC.

---

## 11. Concepto 10: HPA — Escalado Automático

### ¿Qué es el Horizontal Pod Autoscaler?

HPA es un controlador de Kubernetes que **ajusta automáticamente el número de réplicas** de un Deployment según métricas (típicamente uso de CPU o memoria).

```
HPA observa: CPU promedio de los Pods del Deployment de Rust
  → CPU < 30%: mantiene las réplicas actuales (mínimo 1)
  → CPU > 30%: aumenta réplicas (hasta máximo 3)
  → CPU < 30% por un tiempo: reduce réplicas
```

### Por qué Rust escala con HPA y los Go no

La API de Rust recibe **todo el tráfico** de Locust directamente. Es el punto de mayor presión en el sistema. Los deployments de Go procesan mensajes de forma más distribuida.

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: rust-api-hpa
  namespace: sopes1-p2
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: rust-api
  minReplicas: 1
  maxReplicas: 3
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 30    ← escala si CPU > 30%
```

**Requisito importante:** Para que el HPA funcione, el Deployment **debe** tener `resources.requests.cpu` configurado. Sin eso, Kubernetes no puede calcular el porcentaje de uso.

```yaml
resources:
  requests:
    cpu: "100m"      ← HPA necesita esto para calcular el %
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "256Mi"
```

### Análisis de 1 vs 2 réplicas (Go D2)

El enunciado pide comparar el rendimiento con **1 réplica** y **2 réplicas** en Go D2 (gRPC Server + Publisher). Lo que se espera observar:

| Métrica | 1 Réplica | 2 Réplicas |
|---|---|---|
| Throughput (req/s) | Menor | Mayor (carga distribuida) |
| Latencia promedio | Mayor bajo carga | Menor |
| Errores | Más posibles | Menos |
| Mensajes en RabbitMQ (acumulados) | Más | Menos |

El análisis forma parte del manual técnico del proyecto.

---

## 12. Concepto 11: Locust — Pruebas de Carga

### ¿Qué es Locust?

Locust es una herramienta de pruebas de carga escrita en Python. Define usuarios virtuales que envían requests al sistema y mide el rendimiento bajo carga.

### Por qué Locust en este proyecto

Locust simula a los usuarios que envían predicciones de quiniela. En vez de esperar a que usuarios reales usen el sistema, Locust genera tráfico artificial para:
- Probar que el sistema aguanta carga
- Activar el HPA de Rust
- Comparar rendimiento con 1 vs 2 réplicas

### Estructura de un locustfile

```python
from locust import HttpUser, task, between
import random
import time

class QuinielaUser(HttpUser):
    # Tiempo de espera entre tasks (simula usuario real)
    wait_time = between(0.1, 0.5)

    @task
    def enviar_prediccion(self):
        # Cada llamada a @task es un "request" que el usuario hace
        self.client.post("/grpc-202308204", json={...})

    @task(2)   # weight=2: este task se ejecuta el doble de veces
    def otra_accion(self):
        pass
```

### Métricas que reporta Locust

| Métrica | Significado | Qué observar |
|---|---|---|
| RPS (req/s) | Requests por segundo | Cuánto throughput tiene el sistema |
| Response time (ms) | Tiempo de respuesta | P50, P95, P99 |
| Failure % | Porcentaje de errores | < 1% es aceptable |
| Users | Usuarios virtuales activos | El número que configuraste |

---

## 13. Concepto 12: Pods con Múltiples Contenedores

### El patrón Sidecar

En Kubernetes, un Pod puede tener **múltiples contenedores** que comparten:
- La misma red (se comunican via `localhost`)
- Los mismos volúmenes

Esto permite separar responsabilidades dentro de un mismo "unidad de despliegue". En este proyecto:

```
Pod de Go D1:
├── Container A (rest-server)
│   - Escucha en :8080 (recibe de Rust)
│   - Llama a localhost:9000 (Container B)
│
└── Container B (grpc-client)
    - Escucha en :9000 (recibe de Container A)
    - Llama a go-d2-service:50051 (gRPC)

Pod de Go D2:
├── Container A (grpc-server)
│   - Escucha en :50051 (gRPC)
│   - Llama a localhost:9001 (Container B)
│
└── Container B (mq-publisher)
    - Escucha en :9001 (recibe de Container A)
    - Publica en RabbitMQ
```

### Ventajas del diseño multi-container

Separar el servidor gRPC del publisher de RabbitMQ permite:
- **Desarrollar y actualizar cada parte por separado**
- **Escalar el pod completo** cuando se necesita más capacidad
- **Responsabilidad única** por container (principio de diseño limpio)

---

## 14. El Flujo Completo Integrado

```
[1] Locust genera 100 usuarios virtuales enviando predicciones cada 0.1-0.5 segundos.

[2] Cada request es: POST http://<GATEWAY-IP>/grpc-202308204
    Body: { home_team:"BRA", away_team:"GTM", home_goals:2, away_goals:1, ... }

[3] Gateway API recibe la request y la enruta a rust-api-service (según el HTTPRoute).

[4] El Pod de Rust (escalando entre 1-3 réplicas con HPA):
    - Deserializa el JSON
    - Hace POST al servicio de Go D1

[5] Pod de Go D1 Container A (REST server):
    - Recibe el JSON de Rust
    - Lo reenvía a Container B via localhost:9000

[6] Pod de Go D1 Container B (gRPC client):
    - Convierte el JSON a MatchPredictionRequest (proto)
    - Llama via gRPC a go-d2-service:50051

[7] Pod de Go D2 Container A (gRPC server):
    - Recibe el MatchPredictionRequest
    - Lo reenvía a Container B via localhost:9001

[8] Pod de Go D2 Container B (RabbitMQ publisher):
    - Serializa el mensaje como JSON
    - Lo publica en la queue "predicciones" de RabbitMQ

[9] RabbitMQ almacena el mensaje en la queue.

[10] Pod del Consumer (Go):
    - Consume el mensaje de la queue
    - Hace Ack para confirmar recepción
    - Procesa: extrae equipo, goles, usuario
    - Guarda en Valkey (en VM de KubeVirt):
        * INCR prediction:bra:count (si home o away es BRA)
        * ZADD stats:wins:BRA <score>
        * ZADD stats:users:<username> <count>
        * LPUSH prediction:bra:timeseries <json>

[11] Grafana (en VM de KubeVirt):
    - Consulta Valkey via datasource Redis
    - Muestra los 11 paneles del dashboard cada 30s
    - Serie temporal de BRA actualizada en tiempo real
```

---

## 15. Preguntas de Defensa y Respuestas

**P: ¿Por qué usar Gateway API en lugar de Ingress?**

R: Gateway API es el sucesor oficial de Ingress en Kubernetes. Es más expresivo (permite definir GatewayClass, Gateway y HTTPRoute como objetos separados), más fácil de extender, y tiene comportamiento más consistente entre proveedores. Ingress tenía anotaciones específicas por proveedor que no eran portables.

**P: ¿Por qué Rust para la API de entrada y no Go?**

R: Rust tiene el mayor rendimiento posible ya que no tiene garbage collector, lo cual significa tiempos de respuesta predecibles sin pauses de GC bajo carga. Como la API de Rust es el punto de entrada que recibe todo el tráfico de Locust, el rendimiento es crítico. También el proyecto busca exponer al estudiante a un lenguaje de systems programming moderno.

**P: ¿Qué diferencia hay entre gRPC y REST en este proyecto?**

R: La comunicación externa (Locust → Rust → Go D1) usa REST/HTTP con JSON porque es el protocolo más universal. La comunicación interna entre microservicios (Go D1 → Go D2) usa gRPC con Protocol Buffers porque es más eficiente (binario en lugar de texto), tiene un contrato estricto definido en el `.proto`, y es el estándar de la industria para comunicación interna en microservicios.

**P: ¿Por qué necesitamos RabbitMQ? ¿No podría Go D2 llamar directamente al Consumer?**

R: RabbitMQ desacopla el publicador del consumidor. Si el Consumer cae (por actualización, error, etc.), los mensajes se acumulan en la queue y se procesan cuando el Consumer vuelve. Si la llamada fuera directa y el Consumer cayera, se perderían mensajes. Además, RabbitMQ permite escalar el Consumer a múltiples réplicas que comparten la carga de la queue sin duplicar mensajes.

**P: ¿Por qué Valkey y Grafana corren en VMs de KubeVirt y no en Pods normales?**

R: Es un requisito explícito del proyecto para practicar virtualización dentro de Kubernetes (concepto de Sistemas Operativos). En la industria, KubeVirt se usa para migrar workloads que no pueden ser fácilmente contenerizados (como bases de datos con requerimientos de estado complejos) a un entorno Kubernetes sin reescribirlos.

**P: ¿Qué es containerd y por qué lo usamos dentro de la VM en lugar de Docker?**

R: containerd es el runtime de contenedores que usa Kubernetes internamente. Cuando GKE crea un Pod, internamente usa containerd (no Docker) para ejecutar los contenedores. Al usar containerd directamente dentro de las VMs de KubeVirt, estamos usando la misma capa de tecnología que K8s, sin la capa adicional de dockerd. Además, el enunciado lo especifica explícitamente como práctica de conocimiento del runtime.

**P: ¿Qué es un OCI Artifact y qué archivo distribuimos así?**

R: Un OCI Artifact es cualquier archivo que se distribuye usando el protocolo de registry de contenedores OCI, no solo imágenes Docker. En este proyecto, distribuimos el archivo `prediction.proto` como OCI Artifact. Esto garantiza que todos los servicios que necesitan compilar el proto (Go D1, Go D2, Consumer) usen exactamente la misma versión del contrato gRPC, en lugar de copiar el archivo manualmente. Se usa la herramienta `oras` para push y pull.

**P: ¿Cómo funciona el HPA y por qué el threshold es 30% de CPU?**

R: El HPA monitorea el uso promedio de CPU de los Pods del Deployment de Rust. Cuando el promedio supera el 30%, escala añadiendo réplicas (hasta máximo 3). El 30% fue elegido para que el HPA actúe relativamente pronto bajo carga, demostrando el autoscaling durante las pruebas de Locust. Para que funcione, los Pods deben tener `resources.requests.cpu` definido, ya que el porcentaje se calcula como `cpu_usage / cpu_requested * 100`.

**P: ¿Qué diferencia observaste entre 1 y 2 réplicas en Go D2?**

R: Con 2 réplicas, el gRPC server y el publisher de RabbitMQ tienen el doble de capacidad de procesamiento. Locust debería mostrar mayor throughput (más requests/segundo exitosos), menor latencia promedio (P50 y P95 más bajos), y menor acumulación de mensajes en la queue de RabbitMQ. Esto se observa en los reportes CSV de Locust y en la UI de gestión de RabbitMQ.

**P: ¿Cómo se conecta el Consumer a Valkey si está en una VM de KubeVirt?**

R: La VM de KubeVirt (con Valkey) tiene un Service de Kubernetes que expone el puerto 6379. Kubernetes asigna a la VM una IP dentro del clúster (igual que a cualquier Pod). El Consumer, corriendo como Pod normal en el clúster, se conecta al Service de la VM usando el nombre DNS interno: `valkey-vm-service.sopes1-p2.svc.cluster.local:6379`. KubeVirt integra las VMs en el mismo plano de red que los Pods.

**P: ¿Por qué el equipo asignado es BRA?**

R: El enunciado asigna un equipo basado en el último dígito del carnet. Mi carnet es 202308204, cuyo último dígito es **4**. Los dígitos 4 y 5 corresponden al equipo **BRA (Brasil)**. El dashboard de Grafana muestra visualizaciones específicas para BRA: serie temporal de goles predichos como local y visitante, nombre del equipo, y total de predicciones recibidas para BRA.