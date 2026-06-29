# Guía de calificación

Esta guía traduce los comandos sugeridos por el auxiliar a las etiquetas reales del proyecto.
Ejecutar desde la raíz del repositorio.

## 1. Estado general

```bash
kubectl get nodes
kubectl get pods -A
kubectl get deployments -A
kubectl get services -A
kubectl get hpa -A
```

Resultado esperado: Pods del proyecto `Running`, Go D1 y Go D2 `2/2`, VMs `Ready=True` y HPA
de Rust con rango 1–3 y objetivo CPU 30%.

## 2. Gateway y ruta

```bash
kubectl get gateway,httproute -A
kubectl describe gateway rust-api-gateway -n sopes1-p2
kubectl describe httproute rust-api-route -n sopes1-p2
```

Debe observarse `Programmed=True`, ruta aceptada y prefijo `/grpc-202308204`.

## 3. Prueba pública

```bash
curl http://136.68.202.37/health

curl -X POST http://136.68.202.37/grpc-202308204 \
  -H 'Content-Type: application/json' \
  -d '{"home_team":"BRA","away_team":"MEX","home_goals":2,"away_goals":1,"username":"user_202308204","timestamp":"2026-06-28T23:01:00Z"}'
```

Ambas solicitudes deben responder exitosamente.

## 4. Logs del flujo

Las etiquetas del ejemplo del auxiliar no coinciden literalmente con este repositorio. Usar:

```bash
# Rust API
kubectl logs -n sopes1-p2 -l app=rust-api -f --prefix

# Go D1: REST y cliente gRPC
kubectl logs -n sopes1-p2 -l app=go-d1 -c rest-server -f --prefix
kubectl logs -n sopes1-p2 -l app=go-d1 -c grpc-client -f --prefix

# Go D2: servidor gRPC y writer RabbitMQ
kubectl logs -n sopes1-p2 -l app=go-d2 -c grpc-server -f --prefix
kubectl logs -n sopes1-p2 -l app=go-d2 -c rabbit-writer -f --prefix

# RabbitMQ (el operador usa esta etiqueta)
kubectl logs -n rabbitmq-system \
  -l app.kubernetes.io/name=rabbitmq-cluster -c rabbitmq -f --prefix

# Consumer
kubectl logs -n sopes1-p2 -l app=go-consumer -f --prefix
```

Abrir los logs en terminales separadas y luego ejecutar el POST público.

## 5. RabbitMQ

```bash
kubectl exec -n rabbitmq-system rabbitmq-cluster-server-0 -- \
  rabbitmqctl list_queues name durable messages messages_ready messages_unacknowledged
```

La cola debe llamarse `predictions`, ser durable y regresar a cero después de que el consumer
procese el evento.

## 6. KubeVirt y containerd

```bash
kubectl get kubevirt -n kubevirt
kubectl get vm,vmi -n sopes1-p2
kubectl describe vm valkey-vm -n sopes1-p2
kubectl describe vm grafana-vm -n sopes1-p2
```

Las VMI deben estar `Running/Ready=True`. En los `describe` se observan VMs independientes,
PVC y cloud-init. Los manifiestos muestran las unidades systemd que ejecutan `ctr run`:

```bash
rg -n 'containerd|ctr run' PROYECTO2/infra/kubernetes/{valkey,grafana}
```

## 7. Grafana

```bash
kubectl port-forward -n sopes1-p2 service/grafana-service 3000:3000
```

Abrir `http://localhost:3000/d/quiniela-bra/quiniela-mundial-2026-brasil-bra` y mostrar los
paneles. En otra terminal puede comprobarse Prometheus:

```bash
kubectl port-forward -n sopes1-p2 service/grafana-service 9090:9090
```

## 8. HPA y pruebas

```bash
cat PROYECTO2/evidence/hpa/20260628T145848Z/RESULTADO.md
cat PROYECTO2/evidence/locust/20260628T225747Z/COMPARACION.md
```

Para repetirlos:

```bash
make -C PROYECTO2 load-test
make -C PROYECTO2 validate-hpa
```

## 9. Validación final rápida

```bash
make -C PROYECTO2 test
make -C PROYECTO2 validate
```

Antes de presentar, confirmar manualmente que `CamiloSincal` sea colaborador del repositorio
privado; esta máquina no dispone de GitHub CLI para comprobarlo.
