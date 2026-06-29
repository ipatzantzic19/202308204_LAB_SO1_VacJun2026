# Manual técnico — Q.M.2026.K8s

**Estudiante:** Isai Patzán · **Carnet:** 202308204  
**Curso:** Sistemas Operativos 1 · Vacaciones junio 2026

## 1. Objetivo

El sistema recibe predicciones del Mundial 2026, las valida, las transporta mediante REST,
gRPC y RabbitMQ, las persiste en Valkey y presenta métricas del equipo asignado, BRA, en
Grafana. La solución se ejecuta en GKE y usa KubeVirt para alojar Valkey y Grafana en VMs
independientes cuyos servicios se administran con containerd.

## 2. Arquitectura

```text
Locust
  │ POST /grpc-202308204
  ▼
Gateway API ──► Rust API (HPA 1–3, CPU 30%)
                    │ REST
                    ▼
               Go D1 Pod
             ┌─────────────┐
             │ REST server │──localhost──►│ gRPC client │
             └──────────────────────────────────────────┘
                                      │ gRPC
                                      ▼
                                 Go D2 Pod
                           ┌────────────────────┐
                           │ gRPC server        │
                           │ RabbitMQ writer    │
                           └─────────┬──────────┘
                                     │ AMQP
                                     ▼
                                  RabbitMQ
                                     │ ACK posterior al guardado
                                     ▼
                                 Go consumer
                                     │ RESP
                                     ▼
                  KubeVirt VM: Valkey sobre containerd
                                     │
                                     ▼
                           Go metrics exporter
                                     │ /metrics
                                     ▼
             KubeVirt VM: Prometheus + Grafana sobre containerd

Zot HTTPS (VM externa a GKE) ──► imágenes de todos los componentes
```

Los recursos de aplicación usan el namespace `sopes1-p2`. El operador y el servidor de
RabbitMQ usan `rabbitmq-system`, y KubeVirt usa `kubevirt`.

## 3. Contrato de datos

```json
{
  "home_team": "BRA",
  "away_team": "MEX",
  "home_goals": 2,
  "away_goals": 1,
  "username": "user_202308204",
  "timestamp": "2026-06-28T23:01:00Z"
}
```

Los equipos válidos son `GTM`, `MEX`, `BRA`, `ARG` y `ESP`; local y visitante deben ser
distintos. Los goles están entre 0 y 5 y el timestamp usa RFC 3339. El contrato gRPC se define
una sola vez en `proto/prediction.proto`.

## 4. Flujo completo

1. Locust genera predicciones aleatorias y las envía al Gateway.
2. El `HTTPRoute` reconoce `/grpc-202308204`, reemplaza el prefijo por `/` y dirige la
   solicitud a `rust-api-service`.
3. Rust valida el JSON y lo reenvía por REST a Go D1. Un error posterior se propaga como
   error HTTP; no se reporta un falso éxito.
4. El REST server de Go D1 llama por localhost al segundo contenedor del mismo Pod. El cliente
   transforma el mensaje al protobuf y llama a `SendPrediction` en Go D2.
5. El servidor gRPC de Go D2 transforma el protobuf a JSON y solicita al writer local la
   publicación AMQP.
6. El writer declara la cola durable `predictions` y publica mensajes persistentes.
7. El consumer usa `autoAck=false`; solo confirma el mensaje después de una escritura atómica
   exitosa en Valkey. Ante un fallo de Valkey realiza `NACK` con reencolado.
8. El exporter consulta Valkey y expone métricas Prometheus. Prometheus las recolecta y
   Grafana construye el dashboard BRA.

## 5. Gateway API

Se usa `GatewayClass gke-l7-global-external-managed`, un `Gateway` HTTP y un `HTTPRoute`; no
se utiliza Ingress. La política de health check consulta `/health` en Rust. El estado esperado
es `Programmed=True` para el Gateway y `Accepted=True`, `ResolvedRefs=True` para la ruta.

La dirección verificada el 28 de junio de 2026 fue `136.68.202.37`.

## 6. Servicios Go y RabbitMQ

Go D1 y Go D2 cumplen el requisito de dos contenedores por Deployment. La comunicación entre
Pods usa DNS de Services; la comunicación entre los dos contenedores de cada Pod usa localhost.
Todos poseen requests y limits.

RabbitMQ se instala con RabbitMQ Cluster Operator, una réplica y un PVC de 10 GiB. Las
credenciales provienen del Secret generado por el operador y se copian sin versionar valores
sensibles. La cola `predictions` es durable y al finalizar las pruebas mostró cero mensajes
pendientes, porque el consumer había confirmado todo lo persistido.

## 7. KubeVirt, containerd y persistencia

Los nodos destinados a las VMs tienen la etiqueta `nested-virtualization=enabled`. Las dos
VMs son recursos independientes:

| VM | Proceso administrado por containerd | Persistencia | Service |
|---|---|---|---|
| `valkey-vm` | Valkey 8 | PVC `valkey-data`, AOF | `valkey-service:6379` |
| `grafana-vm` | Grafana y Prometheus | PVC separados | `grafana-service:3000,9090` |

Cloud-init instala containerd y crea unidades systemd cuyo `ExecStart` usa `ctr run`. Las
imágenes se descargan de Zot por HTTPS. Valkey usa AOF con `appendfsync everysec`. Las series
recientes se acotan a 10,000 elementos para limitar crecimiento; los contadores y rankings
históricos se conservan.

La comprobación dentro del guest está registrada en
`evidence/runtime/20260628-containerd.md`. El acceso de diagnóstico usa una clave SSH generada
localmente en `.bin/`; la clave privada nunca se versiona.

## 8. Dashboard BRA

El flujo de observabilidad es `Valkey → go-metrics-exporter → Prometheus → Grafana`.
Prometheus es una capa de consulta; Valkey sigue siendo el almacenamiento obligatorio.

El dashboard contiene máximo y mínimo de goles local/visitante, modas, top de equipos por
victorias, top de usuarios, identificación de BRA, total de predicciones y evolución temporal
como local y visitante. Máximos, mínimos y modas usan hasta cinco registros del enfrentamiento
más reciente; rankings y total son históricos.

El JSON provisionable está en `infra/kubernetes/grafana/dashboards/quiniela-bra.json`.

## 9. HPA

El HPA de `rust-api` usa `minReplicas: 1`, `maxReplicas: 3` y objetivo promedio de CPU de 30%.
El Deployment define requests de CPU, requisito para que la utilización pueda calcularse.
La prueba automatizada observó el ciclo `1 → 3 → 1`.

Resultado guardado en `evidence/hpa/20260628T145848Z/RESULTADO.md`: 12,334 solicitudes,
6 errores (0.0486%), máximo de 3 réplicas y retorno a 1.

## 10. Pruebas de carga

La comparación reproducible mantiene Rust en una réplica para aislar la variable y ejecuta el
mismo escenario con Go D2 en una y dos réplicas.

| Métrica | Go D2: 1 réplica | Go D2: 2 réplicas |
|---|---:|---:|
| Requests/s | 112.62 | 113.99 |
| Promedio | 127.24 ms | 123.21 ms |
| p95 | 150 ms | 150 ms |
| p99 | 230 ms | 200 ms |
| Solicitudes | 6,653 | 6,733 |
| Errores | 0 | 0 |

Dos réplicas mejoraron 1.21% el throughput y redujeron ligeramente promedio y p99. Bajo esta
carga Go D2 no fue el cuello de botella: una réplica ya procesaba el tráfico disponible. La
evidencia fuente está en `evidence/locust/20260628T225747Z/`.

## 11. Zot e imágenes

Zot se ejecuta en una VM de GCP fuera de GKE y se publica por HTTPS como
`zot.35-226-224-23.sslip.io`. Las APIs, Locust, consumer, exporter, RabbitMQ, Valkey, Grafana,
Prometheus y el disco base de Ubuntu se consumen desde este registry. No se configura un
registry inseguro en los nodos.

El requisito de OCI Artifact fue dispensado verbalmente por el auxiliar. Como evidencia
adicional, el repositorio también permite publicar y descargar `prediction.proto` como
`sopes1/prediction-proto:v1` mediante `make artifact` y `make artifact-pull`.

## 12. Reproducibilidad

```bash
cd PROYECTO2
make test
make images
make artifact        # adicional; no requerido según indicación del auxiliar
make deploy
make validate
make load-test
make validate-hpa
```

El despliegue crea el Secret de cloud-init de Grafana desde una plantilla y genera una
contraseña aleatoria; no guarda credenciales en Git. El Secret de RabbitMQ también se obtiene
del operador.

## 13. Conclusiones

- El flujo obligatorio fue validado extremo a extremo con REST, gRPC, AMQP y persistencia.
- Los ACK posteriores al guardado evitan perder mensajes cuando Valkey falla temporalmente.
- KubeVirt permite mantener Valkey y Grafana aislados en VMs sin abandonar la administración
  declarativa de Kubernetes.
- El HPA respondió a carga real y volvió a su mínimo tras el enfriamiento.
- Duplicar Go D2 no produjo una mejora material con 50 usuarios; escalar debe responder a una
  medición, no asumirse como beneficio automático.
- Dapr y k3s no se implementaron porque corresponden a punteo opcional.
