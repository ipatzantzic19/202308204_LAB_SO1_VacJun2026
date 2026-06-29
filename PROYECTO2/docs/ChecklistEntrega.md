# Checklist de validación y entrega

Este documento operacionaliza los puntos 9.2, 9.3 y 9.4 del enunciado.

## 1. Verificación automática

Desde la raíz del repositorio:

```bash
make -C PROYECTO2 test
make -C PROYECTO2 deploy
make -C PROYECTO2 validate
```

`test` valida unidades Go/Rust, sintaxis de Locust, JSON del dashboard, render de cloud-init y
ausencia de credenciales estáticas. `deploy` aplica la infraestructura respetando el orden de
dependencias. `validate` comprueba recursos, Grafana, Prometheus, Zot y el artefacto adicional.

## 2. Estado esperado

- [x] Tres nodos `Ready`, incluido el pool de KubeVirt.
- [x] Pods de `sopes1-p2` en `Running` y contenedores Ready.
- [x] Go D1 y Go D2 muestran `2/2`.
- [x] Go D2 y exporter actuales muestran cero reinicios.
- [x] `grafana-vm` y `valkey-vm` están `Running/Ready=True`.
- [x] Gateway `Programmed=True` y HTTPRoute aceptada.
- [x] HPA Rust usa mínimo 1, máximo 3 y CPU objetivo 30%.
- [x] Health público responde `ok`.
- [x] POST público responde `{"status":"ok"}`.
- [x] Cola `predictions` es durable y vuelve a cero.
- [x] Grafana 11.5.2 y Prometheus responden dentro de la VM.
- [x] Dashboard accesible sin login mediante puerto local 13000.

## 3. Validación de imágenes Zot

```bash
curl -fsS https://zot.35-226-224-23.sslip.io/v2/
curl -fsS https://zot.35-226-224-23.sslip.io/v2/_catalog | jq
```

Para comprobar un tag:

```bash
curl -fsS \
  https://zot.35-226-224-23.sslip.io/v2/sopes1/go-metrics-exporter/tags/list | jq
```

El catálogo validado contiene todas las imágenes enumeradas en el manual. La prueba definitiva
de consumo es que los Pods están `Running` con esas referencias y que cloud-init ejecuta
Grafana, Prometheus y Valkey descargados desde el mismo dominio.

## 4. Prueba extremo a extremo

```bash
curl -X POST http://136.68.202.37/grpc-202308204 \
  -H 'Content-Type: application/json' \
  -d '{"home_team":"BRA","away_team":"MEX","home_goals":2,"away_goals":1,"username":"user_202308204","timestamp":"2026-06-29T03:50:00Z"}'

kubectl exec -n rabbitmq-system rabbitmq-cluster-server-0 -- \
  rabbitmqctl list_queues name durable messages messages_ready messages_unacknowledged
```

La petición debe devolver éxito. La cola puede aumentar brevemente, pero debe volver a cero
cuando el consumer persista y confirme el mensaje.

## 5. Dashboard

```bash
PROYECTO2/.bin/virtctl port-forward -n sopes1-p2 \
  vmi/grafana-vm 13000:3000
```

Abrir:

```text
http://localhost:13000/d/quiniela-bra-202308204/quiniela-mundial-2026-brasil-bra
```

No usar `localhost:3000` en esta máquina: ese puerto pertenece a un Grafana local diferente.

## 6. Evidencia que debe estar versionada

- [x] Manual técnico Markdown.
- [x] Documento de aprendizaje.
- [x] Metodología.
- [x] Guía de calificación.
- [x] Auditoría de requisitos.
- [x] Captura final del dashboard.
- [x] CSV/HTML y comparación de una vs. dos réplicas.
- [x] Timeline y resultado del HPA.
- [x] Evidencia de containerd dentro de `grafana-vm`.

## 7. Controles manuales de GitHub

- [ ] Repositorio configurado como privado.
- [ ] Colaborador `CamiloSincal` agregado.
- [ ] Cambios agregados con `git add`.
- [ ] Commit final creado.
- [ ] Commit enviado al remoto con `git push`.
- [ ] `git status` limpio después del push.

Estos puntos no pueden demostrarse únicamente con manifiestos del clúster.

## 8. Orden recomendado para la evaluación

1. Mostrar nodos, Pods, Deployments, Services y HPA.
2. Mostrar Gateway y HTTPRoute.
3. Abrir logs en terminales separadas.
4. Enviar una predicción pública.
5. Mostrar publicación, consumo y cola en cero.
6. Mostrar las dos VMI y las unidades `ctr run` documentadas.
7. Abrir Grafana por el puerto 13000.
8. Mostrar evidencia HPA y comparación de réplicas.
9. Mostrar catálogo Zot y documentación.

## 9. Apagado posterior

Después de evaluar, detener los node pools conforme al procedimiento usado en GCP para evitar
costos. No eliminar PVC, IP estática de Zot ni recursos que deban conservar evidencia hasta que
la nota esté confirmada.
