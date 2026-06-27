# Fase 1 — Infraestructura base

**Proyecto 2 · 202308204 · SOPES 1 · Vacaciones Junio 2026**

## Estado

Completada y corregida después de auditoría el 27 de junio de 2026.

- GKE Standard: tres nodos `n1-standard-4`, discos de 50 GB y virtualización anidada.
- Zot: VM externa `zot-registry`, IP estática `35.226.224.23` y HTTPS confiable.
- Registry: `zot.35-226-224-23.sslip.io`.
- Gateway API: ruta pública `/grpc-202308204`.
- Rust y Go D1 desplegados desde Zot.

## Arquitectura de esta fase

```text
Locust -> Gateway API -> Rust API -> Go D1
                                  |- REST server :8080
                                  `- gRPC client :9000

Zot HTTPS (VM externa) <- push/pull de imágenes
```

Go D1 ya contiene el cliente gRPC real. El stub usado durante el desarrollo inicial fue
retirado; la integración con Go D2 se documenta en `Fase2.md`.

## GKE

```bash
gcloud container clusters get-credentials sopes1-p2-cluster \
  --zone us-central1-a \
  --project project-20c03ab3-bd8f-4b4b-aae

kubectl get nodes
kubectl apply -f PROYECTO2/infra/kubernetes/namespace.yaml
```

El clúster usa el namespace `sopes1-p2`. Los manifiestos contienen requests, limits y
probes; las credenciales no se almacenan en Git.

## Zot con HTTPS

### Diseño vigente

- VM: `zot-registry`, fuera de GKE.
- Dirección regional reservada: `zot-registry-ip` (`35.226.224.23`).
- DNS: `zot.35-226-224-23.sslip.io` resuelve a la IP estática.
- Caddy escucha en 80/443, obtiene y renueva el certificado público y reenvía a Zot.
- La regla `allow-zot` expone únicamente TCP 80 y 443.
- Zot no publica TCP 5000 en la interfaz de la VM; su HTTP solo existe en la red
  Docker privada compartida con Caddy.
- El catálogo vive en `/var/lib/zot` sobre el disco persistente estándar de la VM,
  ampliado de 10 a 20 GB y con `auto-delete` desactivado. Sumado al PVC estándar de
  10 GB, el proyecto utiliza exactamente los 30 GB-mes incluidos en Always Free.
- Zot usa la versión fija `v2.1.18`; Caddy usa `2.10.2`.
- Los nodos GKE confían en el certificado público; no requieren configuración manual de
  containerd.

El archivo de Caddy y la configuración reproducible de la VM están agrupados en
`PROYECTO2/infra/zot/`.

### Promover la IP efímera existente

Este paso ya fue ejecutado y se conserva como referencia reproducible:

```bash
gcloud compute addresses create zot-registry-ip \
  --addresses=35.226.224.23 \
  --region=us-central1
```

### Provisionar almacenamiento, Zot y el proxy TLS

La receta es idempotente: valida que el proyecto no exceda 30 GB de `pd-standard`,
amplía el disco existente y su filesystem, copia el catálogo de un contenedor legado
detenido y recrea Zot con almacenamiento persistente. El contenedor anterior se
conserva detenido como `zot-legacy` hasta que se valide el catálogo.

```bash
cd PROYECTO2
make zot-provision

gcloud compute firewall-rules update allow-zot \
  --allow=tcp:80,tcp:443
```

### Validación TLS y catálogo

```bash
curl --fail https://zot.35-226-224-23.sslip.io/v2/
curl --fail https://zot.35-226-224-23.sslip.io/v2/_catalog
openssl s_client -connect zot.35-226-224-23.sslip.io:443 \
  -servername zot.35-226-224-23.sslip.io </dev/null

gcloud compute ssh zot-registry --zone=us-central1-a --command='\
  findmnt /var/lib/zot; \
  sudo docker inspect zot --format="mounts={{json .Mounts}} ports={{json .HostConfig.PortBindings}}"'
```

Tras comprobar que todos los repositorios y tags están presentes, el respaldo legado
puede eliminarse en la VM con
`sudo CONFIRM_DELETE_LEGACY=yes zot-cleanup-legacy`.

La ampliación no añade costo de disco bajo el estado auditado de la cuenta: el único
proyecto vinculado pasa de 20 a 30 GB totales de `pd-standard`. Esto no implica que la
arquitectura completa sea gratuita; los tres nodos `n1-standard-4` deben mantenerse en
cero fuera de las prácticas y demostraciones.

No se configura `insecure-registries`, no se usa `skip_verify` y no se modifica
containerd en cada nodo.

## Componentes

### Rust API

- `POST /` recibe la predicción y la reenvía a Go D1.
- `GET /health` sirve como health check del balanceador.
- Solo responde `200` si Go D1 responde con éxito.
- Usa timeout HTTP de siete segundos.

```bash
docker build \
  -t zot.35-226-224-23.sslip.io/sopes1/rust-api:v3 \
  PROYECTO2/rust-api
docker push zot.35-226-224-23.sslip.io/sopes1/rust-api:v3
```

### Go D1

El Pod contiene:

1. `rest-server`: recibe REST de Rust y llama al bridge local.
2. `grpc-client`: valida equipos y ejecuta `SendPrediction` contra Go D2.

Ambas capas propagan fallos. Si el bridge o Go D2 no están disponibles, Go D1 devuelve
`502`; nunca convierte ese fallo en `200`.

```bash
docker build \
  -t zot.35-226-224-23.sslip.io/sopes1/go-d1-rest:v3 \
  PROYECTO2/go-d1/rest-server
docker push zot.35-226-224-23.sslip.io/sopes1/go-d1-rest:v3

docker build \
  -f PROYECTO2/go-d1/grpc-client/Dockerfile \
  -t zot.35-226-224-23.sslip.io/sopes1/go-d1-grpc-client:v3 \
  PROYECTO2
docker push zot.35-226-224-23.sslip.io/sopes1/go-d1-grpc-client:v3
```

### Locust

`PROYECTO2/locust/locustfile.py` genera equipos distintos, goles entre 0 y 5,
`user_N` y timestamp UTC. Envía a `/grpc-202308204` y registra como error cualquier
respuesta no exitosa.

## Despliegue

```bash
kubectl apply -k PROYECTO2/infra/kubernetes

kubectl rollout status deployment/rust-api -n sopes1-p2
kubectl rollout status deployment/go-d1 -n sopes1-p2
```

Las imágenes aplicadas son:

```text
zot.35-226-224-23.sslip.io/sopes1/rust-api:v3
zot.35-226-224-23.sslip.io/sopes1/go-d1-rest:v3
zot.35-226-224-23.sslip.io/sopes1/go-d1-grpc-client:v3
```

## Pruebas

```bash
(cd PROYECTO2/rust-api && cargo test)
(cd PROYECTO2/go-d1/rest-server && go test ./...)
(cd PROYECTO2/go-d1/grpc-client && go test ./...)
# O ejecutar toda la suite de forma modular:
make -C PROYECTO2 test
```

Las pruebas unitarias cubren health, errores HTTP posteriores, fallos gRPC, equipos
inválidos y respuestas exitosas.

Prueba pública:

```bash
curl -X POST http://136.68.202.37/grpc-202308204 \
  -H 'Content-Type: application/json' \
  -d '{"home_team":"BRA","away_team":"MEX","home_goals":2,"away_goals":1,"username":"user_202308204","timestamp":"2026-06-27T15:00:00Z"}'
```

## Checklist verificado

- [x] Tres nodos N1 con virtualización anidada.
- [x] Zot corre en una VM externa a GKE.
- [x] IP de Zot reservada como estática.
- [x] Dominio estable y certificado TLS público confiable.
- [x] Firewall público limitado a 80/443.
- [x] Manifiestos sin IP interna ni puerto inseguro.
- [x] Rust y los dos contenedores de Go D1 consumidos desde Zot.
- [x] Gateway API y `/grpc-202308204` funcionando.
- [x] Pruebas unitarias con casos de error reales.
- [x] Configuración manual por nodo eliminada.
