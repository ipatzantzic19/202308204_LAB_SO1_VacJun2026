# Fase 1 — Infraestructura Base
**Proyecto 2 · Q.M.2026.K8s · Isai Patzán · 202308204 · SOPES 1 · Vacaciones Junio 2026**

---

## Archivos de esta fase

```
PROYECTO2/
├── rust-api/
│   ├── src/main.rs
│   ├── Cargo.toml
│   └── Dockerfile
├── go-d1/
│   ├── rest-server/
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── Dockerfile
│   └── grpc-client/
│       ├── main.go
│       ├── go.mod
│       └── Dockerfile
├── locust/
│   ├── locustfile.py
│   └── Dockerfile
├── proto/
│   └── prediction.proto
└── k8s/
    ├── namespace.yaml
    ├── secret-zot.yaml
    ├── rust-api/
    │   ├── deployment.yaml
    │   └── service.yaml
    └── go-d1/
        ├── deployment.yaml
        └── service.yaml
```

---

## Base de conocimiento de esta fase

| Componente | Clase del curso |
|---|---|
| Rust (`cargo`, sintaxis, `Result`) | **Clase 6** |
| Zot (registry), Docker insecure-registries | **Clase 7** |
| containerd / `ctr` | **Clase 7** |
| API REST en Go + gRPC en Go | **Clase 8** |
| GKE, Pods, Deployments, Services, Secrets | **Clase 9** |
| Kubernetes Gateway API | ⚠️ Pendiente (próximas clases) |

---

## Paso 1 — Crear el clúster GKE (Clase 9)

> Seguir el patrón exacto de la **Clase 9**.

### 1.1 Desde la Consola de GCP

1. Abrir [console.cloud.google.com](https://console.cloud.google.com)
2. Menú lateral → **Kubernetes Engine** → **Clusters**
3. Clic en **Crear** → elegir **Estándar**
4. Configuración:
   - **Nombre:** `sopes1-p2-cluster`
   - **Tipo de ubicación:** Zonal (para reducir costos)
   - **Zona:** `us-central1-a` (o la más cercana)
5. En **Grupos de Nodos → Nodos**:
   - **Tipo de máquina:** `N1` → `n1-standard-4`
   - Activar **"Activar nodos con virtualización anidada"** (necesario para KubeVirt en Fase 3)
   - **Tamaño de disco de arranque:** reducir para bajar costos (p.ej. 50 GB)
   - **Cantidad de nodos:** 3

> ⚠️ **Importante:** Las instancias N1 son las únicas que soportan virtualización anidada en GCP.
> Si no habilitas esta opción ahora, tendrás que recrear el clúster para la Fase 3 (KubeVirt).

### 1.2 Conectar kubectl al clúster

```bash
# Instalar gcloud CLI si no lo tienes:
# https://cloud.google.com/sdk/docs/install-sdk?hl=es

# Autenticarse
gcloud auth login

# Conectar kubectl al clúster recién creado
gcloud container clusters get-credentials sopes1-p2-cluster \
  --zone us-central1-a \
  --project <TU-PROJECT-ID>

# Verificar la conexión
kubectl cluster-info
kubectl get nodes
```

**Salida esperada de `kubectl get nodes`:**
```
NAME                                   STATUS   ROLES    AGE   VERSION
gke-sopes1-p2-cluster-default-pool-...   Ready    <none>   2m    v1.x.x
gke-sopes1-p2-cluster-default-pool-...   Ready    <none>   2m    v1.x.x
gke-sopes1-p2-cluster-default-pool-...   Ready    <none>   2m    v1.x.x
```

### 1.3 Crear el Namespace del proyecto

```bash
kubectl create namespace sopes1-p2

# Verificar
kubectl get namespaces
```

---

## Paso 2 — Configurar Zot Registry (Clase 7)

> Patrón exacto de **Clase 7**: Zot corre como contenedor Docker en una VM separada.

### 2.1 Crear la VM para Zot en GCP

```bash
# Desde la Consola de GCP: Compute Engine → Instancias de VM → Crear instancia
# Nombre: zot-registry
# Región/Zona: us-central1-a (misma que el clúster)
# Tipo de máquina: e2-medium (suficiente para Zot)
# Sistema operativo: Ubuntu 22.04 LTS
# Permitir tráfico HTTP y HTTPS

# O desde gcloud CLI:
gcloud compute instances create zot-registry \
  --zone=us-central1-a \
  --machine-type=e2-medium \
  --image-family=ubuntu-2204-lts \
  --image-project=ubuntu-os-cloud \
  --tags=zot-registry \
  --project=<TU-PROJECT-ID>

# Regla de firewall para el puerto 5000 (Zot)
gcloud compute firewall-rules create allow-zot \
  --allow=tcp:5000 \
  --target-tags=zot-registry \
  --project=<TU-PROJECT-ID>
```

### 2.2 Instalar Docker en la VM y levantar Zot

```bash
# Conectarse a la VM de Zot
gcloud compute ssh zot-registry --zone=us-central1-a

# ─── DENTRO DE LA VM DE ZOT ───────────────────────────────────

# Instalar Docker
sudo apt update
sudo apt install -y docker.io
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER
newgrp docker

# Levantar Zot como contenedor Docker (patrón de Clase 7)
docker run -d \
  -p 5000:5000 \
  --name zot \
  --restart unless-stopped \
  ghcr.io/project-zot/zot-linux-amd64:latest

# Verificar que Zot está corriendo
docker ps
curl http://localhost:5000/v2/
# Respuesta esperada: {}
```

```bash
# ─── OBTENER IP EXTERNA DE LA VM ──────────────────────────────
# Salir de la VM
exit

# Obtener la IP
gcloud compute instances describe zot-registry \
  --zone=us-central1-a \
  --format='get(networkInterfaces[0].accessConfigs[0].natIP)'
```

> **Anota esta IP.** La llamaremos `<IP-ZOT>` en todo el resto de la guía.

### 2.3 Configurar Docker local para usar Zot como registro inseguro

> Patrón exacto de **Clase 7**: editar `daemon.json` con `insecure-registries`.

```bash
# EN TU MÁQUINA LOCAL (donde vas a hacer los docker build/push):
sudo nano /etc/docker/daemon.json
```

Contenido del archivo (si está vacío, pegar esto directamente):

```json
{
  "insecure-registries": ["<IP-ZOT>:5000"]
}
```

```bash
# Reiniciar Docker para aplicar los cambios
sudo systemctl restart docker

# Verificar que Zot está accesible desde tu máquina local
curl http://<IP-ZOT>:5000/v2/
# Respuesta: {}

# Ver catálogo de imágenes (vacío por ahora)
curl http://<IP-ZOT>:5000/v2/_catalog
# Respuesta: {"repositories":[]}
```

---

## Paso 3 — Proto gRPC compartido (Clase 8)

> Se genera primero porque tanto **Go D1** como **Go D2** (Fase 2) lo usan.
> Patrón exacto de **Clase 8**: `protoc --go_out=. --go-grpc_out=.`

```bash
# Instalar protoc (Clase 8 — Linux)
sudo apt update && sudo apt upgrade -y
sudo apt install -y build-essential libtool pkg-config protobuf-compiler

# Verificar
protoc --version

# Instalar plugins de Go para protoc (Clase 8)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Agregar el PATH de Go bin si no está
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc
```

```bash
# Crear la carpeta proto
mkdir -p PROYECTO2/proto
```

```protobuf
# PROYECTO2/proto/prediction.proto
# (copiado exactamente del enunciado del proyecto)
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
# Generar el código Go desde el proto (Clase 8)
cd PROYECTO2/proto
protoc --go_out=. --go-grpc_out=. prediction.proto

# Verificar que se generaron los archivos
ls
# prediction.pb.go  prediction_grpc.pb.go  prediction.proto
```

---

## Paso 4 — API REST en Rust (Clase 6 + Clase 7)

> **Clase 6:** estructura de un proyecto Rust con `cargo`, manejo de errores con `Result`.
> **Clase 7:** `Dockerfile` + `docker push` a Zot.

### 4.1 Crear el proyecto Rust

```bash
# Desde la raíz de PROYECTO2/
cargo new rust-api
cd rust-api
```

### 4.2 Editar `Cargo.toml`

```toml
[package]
name    = "rust-api"
version = "0.1.0"
edition = "2021"

[dependencies]
# Framework web async para Rust (equivalente a Go's net/http)
axum    = "0.7"
# Runtime async (necesario para axum)
tokio   = { version = "1", features = ["full"] }
# Serialización/deserialización JSON
serde       = { version = "1", features = ["derive"] }
serde_json  = "1"
# Cliente HTTP para reenviar al Go D1
reqwest = { version = "0.12", features = ["json"] }
# Logs
tracing            = "0.1"
tracing-subscriber = "0.3"
```

### 4.3 Escribir `src/main.rs`

```rust
// src/main.rs
// Rust REST API — recibe predicciones de Locust y las reenvía al Go D1
// Concepto Clase 6: funciones, Result, match, estructuras

use axum::{
    extract::State,
    http::StatusCode,
    routing::post,
    Json, Router,
};
use serde::{Deserialize, Serialize};
use std::{env, sync::Arc};
use tracing::info;

// Estructura del mensaje JSON (Clase 6: tipos de datos)
// #[derive(Deserialize)] → axum deserializa el JSON automáticamente
#[derive(Debug, Deserialize, Serialize, Clone)]
struct Prediction {
    home_team:  String,
    away_team:  String,
    home_goals: i32,
    away_goals: i32,
    username:   String,
    timestamp:  String,
}

// Respuesta que devuelve nuestra API
#[derive(Serialize)]
struct ApiResponse {
    status: String,
}

// Estado compartido entre los handlers (dirección del Go D1)
struct AppState {
    go_d1_url:     String,
    http_client:   reqwest::Client,
}

// Handler del endpoint POST /
// Concepto Clase 6: funciones con Result + Clase 8 REST
async fn receive_prediction(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<Prediction>,
) -> (StatusCode, Json<ApiResponse>) {
    info!("Predicción recibida: {} vs {} ({} - {})",
        payload.home_team, payload.away_team,
        payload.home_goals, payload.away_goals
    );

    // Reenviar al Go D1 (Clase 6: manejo de Result con match)
    let resultado = state.http_client
        .post(&state.go_d1_url)
        .json(&payload)
        .send()
        .await;

    match resultado {
        Ok(resp) => {
            info!("Go D1 respondió con status: {}", resp.status());
            (StatusCode::OK, Json(ApiResponse { status: "ok".to_string() }))
        }
        Err(e) => {
            // Clase 6: Err es un valor, no una excepción
            tracing::error!("Error reenviando al Go D1: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR,
             Json(ApiResponse { status: "error".to_string() }))
        }
    }
}

#[tokio::main]
async fn main() {
    // Inicializar logs
    tracing_subscriber::fmt::init();

    // Leer la URL del Go D1 desde variable de entorno (Clase 9: ConfigMap/env vars)
    let go_d1_url = env::var("GO_D1_URL")
        .unwrap_or_else(|_| "http://go-d1-service:8080".to_string());

    info!("Rust API iniciando. Go D1 URL: {}", go_d1_url);

    let state = Arc::new(AppState {
        go_d1_url,
        http_client: reqwest::Client::new(),
    });

    // Definir rutas (Clase 8: REST endpoint)
    let app = Router::new()
        .route("/", post(receive_prediction))
        .with_state(state);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080")
        .await
        .unwrap();

    info!("Escuchando en 0.0.0.0:8080");
    axum::serve(listener, app).await.unwrap();
}
```

### 4.4 Compilar y probar localmente

```bash
# Compilar en modo release (Clase 6: cargo build --release)
cargo build --release

# Probar localmente (en otra terminal)
GO_D1_URL=http://localhost:9090 cargo run

# Enviar una predicción de prueba
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{
    "home_team": "BRA",
    "away_team": "GTM",
    "home_goals": 3,
    "away_goals": 1,
    "username": "user_42",
    "timestamp": "2026-06-21T13:00:00Z"
  }'
# Esperado: {"status":"error"} (porque Go D1 no está corriendo aún)
# Lo importante es que Rust compiló y recibió el request
```

### 4.5 Dockerizar y publicar en Zot (Clase 7)

```dockerfile
# rust-api/Dockerfile
# Etapa 1: compilar (imagen completa de Rust)
FROM rust:1.78 AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
# Truco para cachear dependencias antes de copiar el código
RUN mkdir src && echo 'fn main() {}' > src/main.rs
RUN cargo build --release
RUN rm src/main.rs
# Ahora copiar el código real y compilar
COPY src ./src
RUN touch src/main.rs && cargo build --release

# Etapa 2: imagen final mínima (sin el compilador de Rust)
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/rust-api /usr/local/bin/rust-api
EXPOSE 8080
CMD ["rust-api"]
```

```bash
# Build de la imagen Docker (Clase 7)
docker build -t rust-api:v1 .

# Etiquetar para Zot (Clase 7)
docker tag rust-api:v1 <IP-ZOT>:5000/sopes1/rust-api:v1

# Push a Zot (Clase 7)
docker push <IP-ZOT>:5000/sopes1/rust-api:v1

# Verificar que está en Zot
curl http://<IP-ZOT>:5000/v2/_catalog
# {"repositories":["sopes1/rust-api"]}

curl http://<IP-ZOT>:5000/v2/sopes1/rust-api/tags/list
# {"name":"sopes1/rust-api","tags":["v1"]}
```

---

## Paso 5 — Go Deployment 1: REST Server (Clase 8)

> Patrón de **Clase 8** (REST en Go) + 2 containers en el mismo Pod (**Clase 9**).
> En Fase 1, el gRPC client **solo loguea** el mensaje (Go D2 no existe aún).

### 5.1 Container A — REST Server

```bash
mkdir -p PROYECTO2/go-d1/rest-server
cd PROYECTO2/go-d1/rest-server
go mod init rest-server
```

```go
// go-d1/rest-server/main.go
// Recibe el JSON de Rust y lo reenvía al gRPC Client (Container B del mismo Pod)
// Patrón REST de Clase 8
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

// Estructura del mensaje (debe coincidir con Rust)
type Prediction struct {
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeGoals int32  `json:"home_goals"`
	AwayGoals int32  `json:"away_goals"`
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
}

type Response struct {
	Status string `json:"status"`
}

func predictionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Leer body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error leyendo body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Deserializar JSON (Clase 8 patrón)
	var pred Prediction
	if err := json.Unmarshal(body, &pred); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	log.Printf("[REST-SERVER] Predicción recibida: %s vs %s (%d-%d) por %s",
		pred.HomeTeam, pred.AwayTeam, pred.HomeGoals, pred.AwayGoals, pred.Username)

	// Reenviar al gRPC Client (Container B, mismo Pod → localhost)
	grpcClientURL := os.Getenv("GRPC_CLIENT_URL")
	if grpcClientURL == "" {
		grpcClientURL = "http://localhost:9000/send"
	}

	resp, err := http.Post(grpcClientURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[REST-SERVER] Error llamando al gRPC client: %v", err)
		// En Fase 1, esto es normal (Go D2 no existe aún)
		// No fallar: devolver OK de todas formas
	} else {
		defer resp.Body.Close()
		log.Printf("[REST-SERVER] gRPC client respondió: %d", resp.StatusCode)
	}

	// Responder al Rust API
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Status: "ok"})
}

func main() {
	http.HandleFunc("/", predictionHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("[REST-SERVER] Escuchando en :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

```bash
go mod tidy
```

```dockerfile
# go-d1/rest-server/Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o rest-server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/rest-server .
EXPOSE 8080
CMD ["./rest-server"]
```

```bash
# Build y push (Clase 7)
docker build -t <IP-ZOT>:5000/sopes1/go-d1-rest:v1 .
docker push <IP-ZOT>:5000/sopes1/go-d1-rest:v1
```

### 5.2 Container B — gRPC Client (stub para Fase 1)

> En Fase 1 este container **recibe** la petición del Container A y **loguea** el mensaje.
> En Fase 2 se actualizará para llamar al gRPC Server (Go D2) cuando exista.

```bash
mkdir -p PROYECTO2/go-d1/grpc-client
cd PROYECTO2/go-d1/grpc-client
go mod init grpc-client
```

```go
// go-d1/grpc-client/main.go
// FASE 1: Stub — recibe de Container A y loguea.
// FASE 2: Se reemplazará por el cliente gRPC real hacia Go D2.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

type Response struct {
	Status string `json:"status"`
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error leyendo body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var pred Prediction
	if err := json.Unmarshal(body, &pred); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	log.Printf("[GRPC-CLIENT][FASE1-STUB] Mensaje recibido de Container A:")
	log.Printf("  Local: %s | Visitante: %s", pred.HomeTeam, pred.AwayTeam)
	log.Printf("  Goles: %d - %d | Usuario: %s", pred.HomeGoals, pred.AwayGoals, pred.Username)
	log.Printf("  Timestamp: %s", pred.Timestamp)
	log.Printf("  → [TODO Fase 2] Aquí se llamará al gRPC Server (Go D2)")

	// Responder OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Status: "ok"})
}

func main() {
	http.HandleFunc("/send", sendHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}

	fmt.Printf("[GRPC-CLIENT] Escuchando en :%s (modo stub Fase 1)\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

```dockerfile
# go-d1/grpc-client/Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o grpc-client .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/grpc-client .
EXPOSE 9000
CMD ["./grpc-client"]
```

```bash
go mod tidy
docker build -t <IP-ZOT>:5000/sopes1/go-d1-grpc-client:v1 .
docker push <IP-ZOT>:5000/sopes1/go-d1-grpc-client:v1

# Verificar que ambas imágenes están en Zot
curl http://<IP-ZOT>:5000/v2/_catalog
# {"repositories":["sopes1/go-d1-grpc-client","sopes1/go-d1-rest","sopes1/rust-api"]}
```

---

## Paso 6 — Locust (generador de tráfico)

> Locust no está en una clase específica del curso, pero es Python puro.
> La estructura del JSON es la definida en el enunciado.

```bash
mkdir -p PROYECTO2/locust
```

```python
# locust/locustfile.py
import random
import time
from locust import HttpUser, task, between

# Equipos válidos del proyecto
TEAMS = ["GTM", "MEX", "BRA", "ARG", "ESP"]

class QuinielaUser(HttpUser):
    # Tiempo de espera entre predicciones (simula usuario real)
    wait_time = between(0.1, 0.5)

    @task
    def enviar_prediccion(self):
        # Elegir equipo local y visitante (distintos)
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

        # POST a la ruta del Gateway API (Fase 1: apuntamos al service de Rust)
        self.client.post(
            "/",   # En Fase 1 usamos / hasta tener Gateway API
            json=payload,
            headers={"Content-Type": "application/json"}
        )
```

```dockerfile
# locust/Dockerfile
FROM locustio/locust:2.20.0
COPY locustfile.py /home/locust/locustfile.py
WORKDIR /home/locust
EXPOSE 8089
ENTRYPOINT ["locust", "-f", "locustfile.py"]
```

```bash
# Build y push a Zot
docker build -t <IP-ZOT>:5000/sopes1/locust:v1 .
docker push <IP-ZOT>:5000/sopes1/locust:v1

# Para correr Locust localmente sin Docker (prueba rápida):
pip install locust
# Primero hacer port-forward del Rust API (ver Paso 7.5)
locust -f locustfile.py --host=http://localhost:8080 \
  --headless --users=10 --spawn-rate=2 --run-time=30s
```

---

## Paso 7 — Manifiestos Kubernetes (Clase 9)

> Patrón exacto de **Clase 9**: Deployments, Services, Secrets.

### 7.1 Namespace

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: sopes1-p2
```

```bash
kubectl apply -f k8s/namespace.yaml
```

### 7.2 Secret para autenticación con Zot (Clase 9 — Secrets)

```bash
# Crear el Secret para que K8s pueda hacer pull desde Zot (Clase 9)
kubectl create secret docker-registry zot-credentials \
  --docker-server=<IP-ZOT>:5000 \
  --docker-username=admin \
  --docker-password=admin \
  --namespace=sopes1-p2

# Si Zot no requiere autenticación (configuración básica de Clase 7), omitir esto
```

### 7.3 Deployment + Service del Rust API

```yaml
# k8s/rust-api/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rust-api
  namespace: sopes1-p2
  labels:
    app: rust-api
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
      # Solo agregar imagePullSecrets si Zot requiere auth
      # imagePullSecrets:
      # - name: zot-credentials
      containers:
      - name: rust-api
        image: <IP-ZOT>:5000/sopes1/rust-api:v1
        ports:
        - containerPort: 8080
        env:
        - name: GO_D1_URL
          # Nombre DNS interno del Service de Go D1 (Clase 9: DNS de Services)
          value: "http://go-d1-service.sopes1-p2.svc.cluster.local:8080"
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "256Mi"
```

```yaml
# k8s/rust-api/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: rust-api-service
  namespace: sopes1-p2
spec:
  type: LoadBalancer   # ← Expone al exterior (Clase 9: LoadBalancer)
  selector:
    app: rust-api
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
```

> **Nota sobre LoadBalancer vs Gateway API:** En esta Fase 1 usamos `LoadBalancer`
> porque Gateway API aún no se ha cubierto en el curso. Cuando se cubra (Clase 10+),
> cambiaremos el Service a `ClusterIP` y agregaremos los objetos `Gateway` + `HTTPRoute`.

### 7.4 Deployment + Service del Go D1 (Pod con 2 containers — Clase 9)

```yaml
# k8s/go-d1/deployment.yaml
# Pod con 2 containers comparten la red → se comunican por localhost (Clase 9)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-d1
  namespace: sopes1-p2
  labels:
    app: go-d1
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
      # Container A: REST Server — escucha en :8080
      - name: rest-server
        image: <IP-ZOT>:5000/sopes1/go-d1-rest:v1
        ports:
        - containerPort: 8080
        env:
        - name: GRPC_CLIENT_URL
          # Container B está en el MISMO pod → localhost
          value: "http://localhost:9000/send"
        resources:
          requests:
            cpu: "50m"
            memory: "64Mi"
          limits:
            cpu: "200m"
            memory: "128Mi"

      # Container B: gRPC Client — escucha en :9000
      - name: grpc-client
        image: <IP-ZOT>:5000/sopes1/go-d1-grpc-client:v1
        ports:
        - containerPort: 9000
        env:
        # En Fase 2 se activará con la dirección de Go D2
        - name: GO_D2_GRPC_ADDR
          value: "go-d2-service.sopes1-p2.svc.cluster.local:50051"
        resources:
          requests:
            cpu: "50m"
            memory: "64Mi"
          limits:
            cpu: "200m"
            memory: "128Mi"
```

```yaml
# k8s/go-d1/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: go-d1-service
  namespace: sopes1-p2
spec:
  type: ClusterIP   # Solo accesible dentro del clúster (Clase 9: ClusterIP)
  selector:
    app: go-d1
  ports:
  - port: 8080
    targetPort: 8080
    protocol: TCP
```

### 7.5 Aplicar todos los manifiestos

```bash
# Aplicar en orden (Clase 9: kubectl apply -f)
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/rust-api/deployment.yaml
kubectl apply -f k8s/rust-api/service.yaml
kubectl apply -f k8s/go-d1/deployment.yaml
kubectl apply -f k8s/go-d1/service.yaml

# Verificar que los Pods están corriendo
kubectl get pods -n sopes1-p2
kubectl get deployments -n sopes1-p2
kubectl get services -n sopes1-p2
```

**Salida esperada:**
```
NAME                        READY   STATUS    RESTARTS   AGE
go-d1-xxxxx-yyyyy           2/2     Running   0          30s   ← 2 containers!
rust-api-xxxxx-yyyyy         1/1     Running   0          30s
```

---

## Paso 8 — Verificación del Flujo de Fase 1

### 8.1 Obtener la IP externa del Rust API

```bash
# El LoadBalancer tarda 1-2 minutos en asignar la IP externa
kubectl get service rust-api-service -n sopes1-p2 -w

# Cuando aparezca la EXTERNAL-IP, ya puedes hacer requests
# NAME               TYPE           CLUSTER-IP   EXTERNAL-IP      PORT(S)
# rust-api-service   LoadBalancer   10.x.x.x     34.xx.xx.xx      80:3xxxx/TCP
```

### 8.2 Probar el flujo completo

```bash
# Guardar la IP en una variable
RUST_IP=$(kubectl get service rust-api-service -n sopes1-p2 \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "IP del Rust API: $RUST_IP"

# Prueba manual — enviar una predicción
curl -X POST "http://$RUST_IP/" \
  -H "Content-Type: application/json" \
  -d '{
    "home_team": "BRA",
    "away_team": "MEX",
    "home_goals": 2,
    "away_goals": 0,
    "username": "user_202308204",
    "timestamp": "2026-06-21T13:00:00Z"
  }'

# Respuesta esperada:
# {"status":"ok"}
```

### 8.3 Ver logs del flujo completo

```bash
# Logs de Rust API — debería mostrar que recibió y reenvió
kubectl logs -n sopes1-p2 -l app=rust-api -f

# Logs del Container A de Go D1 (REST Server)
kubectl logs -n sopes1-p2 -l app=go-d1 -c rest-server -f

# Logs del Container B de Go D1 (gRPC Client stub)
kubectl logs -n sopes1-p2 -l app=go-d1 -c grpc-client -f
```

**Salida esperada en logs de grpc-client:**
```
[GRPC-CLIENT][FASE1-STUB] Mensaje recibido de Container A:
  Local: BRA | Visitante: MEX
  Goles: 2 - 0 | Usuario: user_202308204
  Timestamp: 2026-06-21T13:00:00Z
  → [TODO Fase 2] Aquí se llamará al gRPC Server (Go D2)
```

### 8.4 Prueba de carga básica con Locust

```bash
# Correr Locust localmente apuntando al Rust API
pip install locust

locust -f locust/locustfile.py \
  --host=http://$RUST_IP \
  --headless \
  --users=20 \
  --spawn-rate=5 \
  --run-time=60s

# Ver resultado: RPS, latencia, % errores
# ✅ Si ves < 5% errores → la infraestructura está funcionando
```

---

## Checklist de Fase 1

```
✅ Clúster GKE creado con instancias N1 (virtualización anidada habilitada)
✅ kubectl conectado al clúster
✅ Namespace sopes1-p2 creado
✅ VM de Zot corriendo en GCP con Zot en Docker
✅ Docker local configurado con insecure-registries para <IP-ZOT>:5000
✅ proto/prediction.proto creado y compilado → pb.go generados
✅ rust-api compila con cargo build --release
✅ Imagen rust-api:v1 en Zot
✅ go-d1 rest-server: imagen en Zot
✅ go-d1 grpc-client (stub): imagen en Zot
✅ locust/locustfile.py configurado con el JSON correcto
✅ kubectl apply -f k8s/ → todos los Pods en Running
✅ curl a la IP del Rust API → {"status":"ok"}
✅ Logs de Go D1 muestran el mensaje recibido
✅ Locust corre 60s sin errores graves
```

---

## Errores Comunes y Soluciones

**`ImagePullBackOff` en los Pods**
```bash
kubectl describe pod <nombre-pod> -n sopes1-p2 | grep -A5 Events

# Solución A: Verificar que la imagen existe en Zot
curl http://<IP-ZOT>:5000/v2/_catalog

# Solución B: Agregar la IP de Zot como insecure en el nodo (Clase 7 → Flatcar)
# Los nodos de GKE usan containerd — necesitan configuración especial
# Ver Paso 9 más abajo
```

**Los nodos GKE no pueden hacer pull de Zot (insecure)**
```bash
# Los nodos de GKE tienen containerd como runtime (no Docker)
# Necesitan configurar el registry insecuro en containerd
# Conectarse a un nodo:
gcloud compute ssh <NOMBRE-NODO> --zone=us-central1-a

# Editar la config de containerd
sudo mkdir -p /etc/containerd/certs.d/<IP-ZOT>:5000
cat << 'EOF' | sudo tee /etc/containerd/certs.d/<IP-ZOT>:5000/hosts.toml
server = "http://<IP-ZOT>:5000"

[host."http://<IP-ZOT>:5000"]
  capabilities = ["pull", "resolve"]
  skip_verify = true
EOF

sudo systemctl restart containerd
```

> **Alternativa más simple:** Usar la IP interna de la VM de Zot (no la externa),
> ya que está en la misma red de GCP.

**`CrashLoopBackOff` en rust-api**
```bash
kubectl logs <pod-rust-api> -n sopes1-p2 --previous
# Revisar si hay error de compilación o de configuración
```

**`2/2` no aparece en el Pod de Go D1 (solo `1/2`)**
```bash
kubectl describe pod -l app=go-d1 -n sopes1-p2
# Ver qué container está fallando y por qué
kubectl logs -l app=go-d1 -c grpc-client -n sopes1-p2
```

---

## Paso 9 — Pendiente: Gateway API

> ⚠️ **Gateway API aún no ha sido cubierta en el curso** (Clase 10+).
> Por ahora, el Rust API está expuesto con un `LoadBalancer` service.
>
> Cuando se cubra en clase, se reemplazará por:
> 1. Cambiar `rust-api-service` de `LoadBalancer` a `ClusterIP`
> 2. Crear `GatewayClass` + `Gateway`
> 3. Crear `HTTPRoute` con path `/grpc-202308204` → rust-api-service
>
> **Actualizar esta sección cuando se publique la clase correspondiente.**

---

## Notas de Costos GCP

> ⚠️ Apagar el clúster cuando no estés trabajando.

```bash
# Reducir nodos a 0 para no gastar (el clúster existe pero no hay VMs activas)
gcloud container clusters resize sopes1-p2-cluster \
  --num-nodes=0 \
  --zone=us-central1-a

# Volver a 3 nodos cuando retomes el trabajo
gcloud container clusters resize sopes1-p2-cluster \
  --num-nodes=3 \
  --zone=us-central1-a
```