# Guía Paso a Paso — Proyecto 2 Q.M.2026.K8s
**Isai Patzán · 202308204 · SOPES 1 · Vacaciones Junio 2026**

> Esta guía se actualiza conforme se avanza en las fases y se publican nuevas clases.
> Sigue los pasos en orden. No avances a la siguiente fase sin verificar la actual.

---

## PRE-REQUISITOS — Herramientas Locales

```bash
# Verificar gcloud CLI
gcloud --version

# Si no está instalado:
# https://cloud.google.com/sdk/docs/install

# Verificar kubectl
kubectl version --client

# Verificar Docker
docker --version

# Verificar Go
go version    # necesitas Go 1.21+

# Verificar Rust
cargo --version  # Si no lo tienes: https://rustup.rs/

# Verificar Helm (para RabbitMQ más adelante)
helm version
```

---

## FASE 1 — Infraestructura Base

### Paso 1.1 — Configurar GCP y crear el Clúster GKE

```bash
# Autenticarse en GCP
gcloud auth login

# Configurar el proyecto
gcloud config set project <TU-PROJECT-ID>

# Habilitar APIs necesarias
gcloud services enable container.googleapis.com
gcloud services enable compute.googleapis.com

# Crear el clúster GKE con instancias N1
# IMPORTANTE: Las instancias N1 son necesarias para KubeVirt (virtualización anidada)
gcloud container clusters create sopes1-p2-cluster \
  --machine-type=n1-standard-4 \
  --num-nodes=3 \
  --zone=us-central1-a \
  --enable-nested-virtualization \
  --image-type=UBUNTU_CONTAINERD

# Obtener credenciales para kubectl
gcloud container clusters get-credentials sopes1-p2-cluster \
  --zone=us-central1-a

# Verificar que funciona
kubectl get nodes

# Crear namespace del proyecto
kubectl create namespace sopes1-p2

# Verificar
kubectl get ns
```

> **Nota sobre virtualización anidada:** KubeVirt (Fase 3) requiere que los nodos
> soporten virtualización anidada. En GCP esto se habilita con `--enable-nested-virtualization`
> en instancias N1 o superiores. Verificar en clase la configuración exacta.

---

### Paso 1.2 — Configurar Zot Registry (VM externa)

```bash
# Crear VM para Zot en GCP (fuera del clúster)
gcloud compute instances create zot-registry \
  --machine-type=n1-standard-2 \
  --zone=us-central1-a \
  --image-family=ubuntu-2204-lts \
  --image-project=ubuntu-os-cloud \
  --tags=zot-registry

# Regla de firewall para HTTPS (puerto 443) y registry (5000)
gcloud compute firewall-rules create allow-zot \
  --allow=tcp:443,tcp:5000 \
  --target-tags=zot-registry

# Obtener IP externa de la VM
gcloud compute instances describe zot-registry \
  --zone=us-central1-a \
  --format='get(networkInterfaces[0].accessConfigs[0].natIP)'

# SSH a la VM de Zot
gcloud compute ssh zot-registry --zone=us-central1-a
```

```bash
# DENTRO DE LA VM DE ZOT:

# Instalar Zot
wget https://github.com/project-zot/zot/releases/download/v2.0.0-rc7/zot-linux-amd64
chmod +x zot-linux-amd64
sudo mv zot-linux-amd64 /usr/local/bin/zot

# Crear configuración básica de Zot
mkdir -p /etc/zot
cat > /etc/zot/config.json << 'EOF'
{
  "distSpecVersion": "1.1.0",
  "storage": {
    "rootDirectory": "/var/lib/zot"
  },
  "http": {
    "address": "0.0.0.0",
    "port": "5000"
  },
  "log": {
    "level": "info"
  }
}
EOF

# Crear directorio de almacenamiento
sudo mkdir -p /var/lib/zot

# Iniciar Zot (en background para prueba)
zot serve /etc/zot/config.json &

# Verificar que responde
curl http://localhost:5000/v2/
```

```bash
# DESDE TU MÁQUINA LOCAL:
# Login al registry Zot
docker login <IP-EXTERNA-ZOT>:5000

# Prueba push de imagen
docker pull alpine
docker tag alpine <IP-EXTERNA-ZOT>:5000/test/alpine:latest
docker push <IP-EXTERNA-ZOT>:5000/test/alpine:latest
```

> **Nota HTTPS:** El enunciado pide HTTPS. Configura certificados TLS usando Let's Encrypt
> o un certificado autofirmado. Ver documentación de Zot: https://zotregistry.dev/

---

### Paso 1.3 — Desarrollar la API REST en Rust

```bash
# Crear proyecto Rust
cargo new rust-api
cd rust-api

# Editar Cargo.toml — agregar dependencias
```

```toml
[package]
name = "rust-api"
version = "0.1.0"
edition = "2021"

[dependencies]
tokio = { version = "1", features = ["full"] }
axum = "0.7"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
reqwest = { version = "0.12", features = ["json"] }
tracing = "0.1"
tracing-subscriber = "0.3"
```

```rust
// src/main.rs
use axum::{routing::post, Json, Router};
use serde::{Deserialize, Serialize};
use std::env;

#[derive(Debug, Deserialize, Serialize, Clone)]
struct Prediction {
    home_team:  String,
    away_team:  String,
    home_goals: u32,
    away_goals: u32,
    username:   String,
    timestamp:  String,
}

#[derive(Serialize)]
struct ApiResponse {
    status: String,
}

async fn receive_prediction(
    Json(payload): Json<Prediction>,
) -> Json<ApiResponse> {
    let go_d1_url = env::var("GO_D1_URL")
        .unwrap_or_else(|_| "http://go-d1-service:8080".to_string());

    let client = reqwest::Client::new();
    let result = client
        .post(&go_d1_url)
        .json(&payload)
        .send()
        .await;

    match result {
        Ok(_) => Json(ApiResponse { status: "ok".to_string() }),
        Err(e) => {
            eprintln!("Error forwarding to Go D1: {}", e);
            Json(ApiResponse { status: "error".to_string() })
        }
    }
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt::init();

    let app = Router::new()
        .route("/", post(receive_prediction));

    let addr = "0.0.0.0:8080";
    println!("Rust API listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
```

```dockerfile
# Dockerfile para Rust API
FROM rust:1.78 AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src ./src
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/rust-api /usr/local/bin/rust-api
EXPOSE 8080
CMD ["rust-api"]
```

```bash
# Build y push a Zot
docker build -t <IP-ZOT>:5000/sopes1/rust-api:latest .
docker push <IP-ZOT>:5000/sopes1/rust-api:latest
```

---

### Paso 1.4 — Proto gRPC (compartido entre Go D1 y Go D2)

```bash
# Crear carpeta proto compartida
mkdir -p proto
```

```protobuf
// proto/prediction.proto
syntax = "proto3";
package worldcup2026;
option go_package = "./proto";

message MatchPredictionRequest {
  Teams  home_team  = 1;
  Teams  away_team  = 2;
  int32  home_goals = 3;
  int32  away_goals = 4;
  string username   = 5;
  string timestamp  = 6;
}

enum Teams {
  TEAMS_UNKNOWN = 0;
  GTM = 1;
  MEX = 2;
  BRA = 3;
  ARG = 4;
  ESP = 5;
}

message MatchPredictionResponse {
  string status = 1;
}

service MatchPredictionService {
  rpc SendPrediction (MatchPredictionRequest) returns (MatchPredictionResponse);
}
```

```bash
# Instalar protoc y plugins de Go
sudo apt install -y protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generar código Go
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/prediction.proto
```

---

### Paso 1.5 — Go Deployment 1 (REST Receiver + gRPC Client)

```bash
mkdir -p go-d1
cd go-d1
go mod init go-d1
go get google.golang.org/grpc
go get google.golang.org/protobuf
```

El Go D1 tiene **2 containers** en un mismo Pod. La comunicación entre ellos puede
hacerse via `localhost` (comparten el mismo namespace de red dentro del Pod).

```go
// go-d1/rest-server/main.go
// Container A: recibe de Rust via HTTP y reenvía al gRPC client (Container B)
// via localhost:9000 (puerto interno del pod)
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "bytes"
    "os"
)

type Prediction struct {
    HomeTeam  string `json:"home_team"`
    AwayTeam  string `json:"away_team"`
    HomeGoals int32  `json:"home_goals"`
    AwayGoals int32  `json:"away_goals"`
    Username  string `json:"username"`
    Timestamp string `json:"timestamp"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var pred Prediction
    if err := json.NewDecoder(r.Body).Decode(&pred); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Reenviar al gRPC client (Container B) via localhost
    grpcClientURL := os.Getenv("GRPC_CLIENT_URL")
    if grpcClientURL == "" {
        grpcClientURL = "http://localhost:9000/send"
    }

    data, _ := json.Marshal(pred)
    resp, err := http.Post(grpcClientURL, "application/json", bytes.NewBuffer(data))
    if err != nil {
        log.Printf("Error calling grpc client: %v", err)
        http.Error(w, "error", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
}

func main() {
    http.HandleFunc("/", handler)
    log.Println("Go D1 REST Server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

```go
// go-d1/grpc-client/main.go  
// Container B: recibe de Container A y llama al gRPC server (Go D2)
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "strings"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "../proto"
)

var grpcConn *grpc.ClientConn
var grpcClient pb.MatchPredictionServiceClient

func init() {
    gd2Addr := os.Getenv("GO_D2_GRPC_ADDR")
    if gd2Addr == "" {
        gd2Addr = "go-d2-service:50051"
    }
    var err error
    grpcConn, err = grpc.NewClient(gd2Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect to gRPC server: %v", err)
    }
    grpcClient = pb.NewMatchPredictionServiceClient(grpcConn)
}

type Prediction struct {
    HomeTeam  string `json:"home_team"`
    AwayTeam  string `json:"away_team"`
    HomeGoals int32  `json:"home_goals"`
    AwayGoals int32  `json:"away_goals"`
    Username  string `json:"username"`
    Timestamp string `json:"timestamp"`
}

func teamToEnum(team string) pb.Teams {
    switch strings.ToUpper(team) {
    case "GTM": return pb.Teams_GTM
    case "MEX": return pb.Teams_MEX
    case "BRA": return pb.Teams_BRA
    case "ARG": return pb.Teams_ARG
    case "ESP": return pb.Teams_ESP
    default: return pb.Teams_TEAMS_UNKNOWN
    }
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
    var pred Prediction
    json.NewDecoder(r.Body).Decode(&pred)

    req := &pb.MatchPredictionRequest{
        HomeTeam:  teamToEnum(pred.HomeTeam),
        AwayTeam:  teamToEnum(pred.AwayTeam),
        HomeGoals: pred.HomeGoals,
        AwayGoals: pred.AwayGoals,
        Username:  pred.Username,
        Timestamp: pred.Timestamp,
    }

    resp, err := grpcClient.SendPrediction(context.Background(), req)
    if err != nil {
        log.Printf("gRPC error: %v", err)
        http.Error(w, "grpc error", http.StatusInternalServerError)
        return
    }
    log.Printf("gRPC response: %s", resp.Status)
    w.Write([]byte(`{"status":"ok"}`))
}

func main() {
    http.HandleFunc("/send", sendHandler)
    log.Println("gRPC Client HTTP bridge listening on :9000")
    log.Fatal(http.ListenAndServe(":9000", nil))
}
```

---

### Paso 1.6 — Locust

```python
# locust/locustfile.py
import random
import time
from locust import HttpUser, task, between

TEAMS = ["GTM", "MEX", "BRA", "ARG", "ESP"]

class QuinielaUser(HttpUser):
    wait_time = between(0.1, 0.5)

    @task
    def send_prediction(self):
        home_team = random.choice(TEAMS)
        away_team = random.choice([t for t in TEAMS if t != home_team])

        payload = {
            "home_team":  home_team,
            "away_team":  away_team,
            "home_goals": random.randint(0, 5),
            "away_goals": random.randint(0, 5),
            "username":   f"user_{random.randint(1, 1000)}",
            "timestamp":  time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        }

        self.client.post(
            "/grpc-202308204",
            json=payload,
            headers={"Content-Type": "application/json"}
        )
```

```dockerfile
# locust/Dockerfile
FROM locustio/locust:2.20.0
COPY locustfile.py /home/locust/locustfile.py
WORKDIR /home/locust
```

```bash
# Para correr Locust localmente apuntando al Gateway
locust -f locustfile.py --host=http://<IP-GATEWAY> --headless \
  --users 50 --spawn-rate 5 --run-time 60s
```

---

### Paso 1.7 — Manifiestos Kubernetes (Fase 1)

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: sopes1-p2
```

```yaml
# k8s/rust/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rust-api
  namespace: sopes1-p2
spec:
  replicas: 1
  selector:
    matchLabels:
      app: rust-api
  template:
    metadata:
      labels:
        app: rust-api
    spec:
      containers:
      - name: rust-api
        image: <IP-ZOT>:5000/sopes1/rust-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: GO_D1_URL
          value: "http://go-d1-service.sopes1-p2.svc.cluster.local:8080"
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "256Mi"
---
apiVersion: v1
kind: Service
metadata:
  name: rust-api-service
  namespace: sopes1-p2
spec:
  selector:
    app: rust-api
  ports:
  - port: 80
    targetPort: 8080
```

```yaml
# k8s/go-d1/deployment.yaml
# Deployment con 2 containers en el mismo pod
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-d1
  namespace: sopes1-p2
spec:
  replicas: 1
  selector:
    matchLabels:
      app: go-d1
  template:
    metadata:
      labels:
        app: go-d1
    spec:
      containers:
      - name: rest-server
        image: <IP-ZOT>:5000/sopes1/go-d1-rest:latest
        ports:
        - containerPort: 8080
        env:
        - name: GRPC_CLIENT_URL
          value: "http://localhost:9000/send"
        resources:
          requests:
            cpu: "100m"
            memory: "64Mi"
      - name: grpc-client
        image: <IP-ZOT>:5000/sopes1/go-d1-grpc-client:latest
        ports:
        - containerPort: 9000
        env:
        - name: GO_D2_GRPC_ADDR
          value: "go-d2-service.sopes1-p2.svc.cluster.local:50051"
        resources:
          requests:
            cpu: "100m"
            memory: "64Mi"
---
apiVersion: v1
kind: Service
metadata:
  name: go-d1-service
  namespace: sopes1-p2
spec:
  selector:
    app: go-d1
  ports:
  - port: 8080
    targetPort: 8080
```

```yaml
# k8s/gateway/gateway.yaml
# Instalar primero los CRDs de Gateway API:
# kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml

apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: sopes1-gateway-class
spec:
  controllerName: "example.com/foo-bar-controller"
  # Cambiar según el proveedor que se enseñe en clase
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: sopes1-gateway
  namespace: sopes1-p2
spec:
  gatewayClassName: sopes1-gateway-class
  listeners:
  - name: http
    port: 80
    protocol: HTTP
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: quiniela-route
  namespace: sopes1-p2
spec:
  parentRefs:
  - name: sopes1-gateway
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /grpc-202308204
    backendRefs:
    - name: rust-api-service
      port: 80
```

```bash
# Aplicar todos los manifiestos de Fase 1
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/gateway/
kubectl apply -f k8s/rust/
kubectl apply -f k8s/go-d1/

# Verificar
kubectl get pods -n sopes1-p2
kubectl get services -n sopes1-p2
kubectl get httproutes -n sopes1-p2

# Obtener la IP del Gateway
kubectl get gateway sopes1-gateway -n sopes1-p2
```

### Verificación Fase 1

```bash
# Obtener IP del Gateway
GATEWAY_IP=$(kubectl get gateway sopes1-gateway -n sopes1-p2 \
  -o jsonpath='{.status.addresses[0].value}')

# Prueba manual
curl -X POST "http://$GATEWAY_IP/grpc-202308204" \
  -H "Content-Type: application/json" \
  -d '{
    "home_team": "GTM",
    "away_team": "BRA",
    "home_goals": 2,
    "away_goals": 1,
    "username": "user_42",
    "timestamp": "2026-06-21T13:00:00Z"
  }'

# Ver logs de Rust API
kubectl logs -n sopes1-p2 -l app=rust-api

# Ver logs de Go D1
kubectl logs -n sopes1-p2 -l app=go-d1 -c rest-server
kubectl logs -n sopes1-p2 -l app=go-d1 -c grpc-client

# ✅ Si ves el mensaje en los logs de Go D1 → Fase 1 completada
```

---

## FASE 2 — Comunicación Interna y Mensajería

> ⚠️ **PENDIENTE** — Se completará cuando la Clase 8-9 sea publicada en el curso.
> Esta sección cubre: **RabbitMQ** + **Go Deployment 2 (gRPC Server + Publisher)**

### Paso 2.1 — RabbitMQ en GKE

```bash
# Usando Helm (patrón típico para RabbitMQ en K8s)
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install rabbitmq bitnami/rabbitmq \
  --namespace sopes1-p2 \
  --set auth.username=admin \
  --set auth.password=adminpassword

# Verificar
kubectl get pods -n sopes1-p2 -l app.kubernetes.io/name=rabbitmq

# Ver credenciales
kubectl get secret rabbitmq -n sopes1-p2 -o yaml
```

> **TODO:** Revisar el patrón de la Clase 8-9 del curso para RabbitMQ y adaptar.

### Paso 2.2 — Go Deployment 2

> **TODO:** Completar con el patrón de gRPC Server de la clase correspondiente.

```go
// Estructura básica — completar con patrón del curso
// go-d2/grpc-server/main.go

package main
// Importar el proto generado
// Implementar MatchPredictionService.SendPrediction
// Al recibir: publicar en RabbitMQ via Container B (localhost:9001)
```

---

## FASE 3 — Consumo y Persistencia en VM

> ⚠️ **PENDIENTE** — Se completará cuando la Clase 10-11 sea publicada.
> Esta sección cubre: **Consumer Go**, **KubeVirt**, **Valkey en containerd en VM**

### Paso 3.1 — KubeVirt

```bash
# Instalar KubeVirt operator — VERIFICAR VERSIÓN CON EL CURSO
export KUBEVIRT_VERSION=$(curl -s https://api.github.com/repos/kubevirt/kubevirt/releases/latest | jq -r .tag_name)
kubectl apply -f "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml"
kubectl apply -f "https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-cr.yaml"

# Esperar que esté listo
kubectl wait --for=condition=Available kubevirt/kubevirt -n kubevirt --timeout=300s

# Instalar virtctl
VERSION=$(kubectl get kubevirt.kubevirt.io/kubevirt -n kubevirt \
  -o=jsonpath="{.status.observedKubeVirtVersion}")
curl -L -o virtctl "https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/virtctl-${VERSION}-linux-amd64"
chmod +x virtctl
sudo mv virtctl /usr/local/bin/
```

```yaml
# k8s/kubevirt/vm-valkey.yaml — PENDIENTE DE DETALLES DEL CURSO
# VirtualMachine con Ubuntu, containerd instalado, Valkey en contenedor
apiVersion: kubevirt.io/v1
kind: VirtualMachine
metadata:
  name: vm-valkey
  namespace: sopes1-p2
spec:
  # ... completar con el patrón del curso
```

> **TODO:** Completar con el YAML exacto de KubeVirt que se vea en clase.

---

## FASE 4 — Visualización y Pruebas de Carga

> ⚠️ **PENDIENTE** — Se completará con las clases finales.

### Paso 4.1 — HPA para Rust

```yaml
# k8s/rust/hpa.yaml
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
        averageUtilization: 30
```

```bash
kubectl apply -f k8s/rust/hpa.yaml
kubectl get hpa -n sopes1-p2

# Observar el scaling bajo carga
kubectl get hpa -n sopes1-p2 -w
```

### Paso 4.2 — Pruebas de Carga (para análisis)

```bash
# Prueba 1: Go D2 con 1 réplica
kubectl scale deployment go-d2 --replicas=1 -n sopes1-p2
locust -f locustfile.py --host=http://$GATEWAY_IP \
  --headless --users=100 --spawn-rate=10 --run-time=120s \
  --csv=resultados-1-replica

# Prueba 2: Go D2 con 2 réplicas
kubectl scale deployment go-d2 --replicas=2 -n sopes1-p2
locust -f locustfile.py --host=http://$GATEWAY_IP \
  --headless --users=100 --spawn-rate=10 --run-time=120s \
  --csv=resultados-2-replicas

# Comparar los CSV resultantes
```

---

## Comandos Útiles de Kubernetes

```bash
# Ver todos los recursos del namespace
kubectl get all -n sopes1-p2

# Ver logs de un pod específico
kubectl logs -n sopes1-p2 <nombre-pod> -c <nombre-container>

# Seguir logs en tiempo real
kubectl logs -n sopes1-p2 <nombre-pod> -f

# Describir un pod (útil para ver errores de imagen)
kubectl describe pod <nombre-pod> -n sopes1-p2

# Ejecutar comando dentro de un pod
kubectl exec -it <nombre-pod> -n sopes1-p2 -- /bin/sh

# Ver eventos del namespace
kubectl get events -n sopes1-p2 --sort-by=.metadata.creationTimestamp

# Reiniciar un deployment
kubectl rollout restart deployment/<nombre> -n sopes1-p2

# Ver el estado de un rollout
kubectl rollout status deployment/<nombre> -n sopes1-p2

# Actualizar imagen de un container
kubectl set image deployment/<nombre> <container>=<imagen>:<tag> -n sopes1-p2

# Port-forward para acceder localmente a un servicio
kubectl port-forward svc/<servicio> 8080:80 -n sopes1-p2
```

## Comandos Útiles de Docker y Zot

```bash
# Build con tag para Zot
docker build -t <IP-ZOT>:5000/sopes1/<imagen>:<tag> .

# Push a Zot
docker push <IP-ZOT>:5000/sopes1/<imagen>:<tag>

# Ver imágenes en Zot via API
curl https://<IP-ZOT>:5000/v2/_catalog

# OCI Artifact — push de un archivo (ej: el proto)
oras push <IP-ZOT>:5000/sopes1/artifacts/prediction-proto:latest \
  proto/prediction.proto:application/vnd.worldcup.proto.v1

# OCI Artifact — pull
oras pull <IP-ZOT>:5000/sopes1/artifacts/prediction-proto:latest
```

> **Nota `oras`:** Instalar ORAS CLI para manejar OCI Artifacts:
> ```bash
> curl -LO "https://github.com/oras-project/oras/releases/download/v1.1.0/oras_1.1.0_linux_amd64.tar.gz"
> tar -zxvf oras_1.1.0_linux_amd64.tar.gz oras
> sudo mv oras /usr/local/bin/
> ```

## Troubleshooting Común

```bash
# Pod en estado ImagePullBackOff → problema con Zot
kubectl describe pod <pod> -n sopes1-p2
# Verificar que imagePullSecrets está configurado si Zot requiere auth

# Crear Secret para pull desde Zot
kubectl create secret docker-registry zot-credentials \
  --docker-server=<IP-ZOT>:5000 \
  --docker-username=<user> \
  --docker-password=<password> \
  -n sopes1-p2

# Agregar al deployment:
# spec.template.spec.imagePullSecrets:
# - name: zot-credentials

# Pod en CrashLoopBackOff → ver logs del crash
kubectl logs <pod> -n sopes1-p2 --previous

# gRPC connection refused → verificar service y puerto
kubectl get svc -n sopes1-p2
kubectl port-forward svc/go-d2-service 50051:50051 -n sopes1-p2
# luego probar con grpcurl desde local

# RabbitMQ no conecta → verificar credenciales en Secret
kubectl get secret rabbitmq -n sopes1-p2 -o jsonpath='{.data.rabbitmq-password}' | base64 -d
```