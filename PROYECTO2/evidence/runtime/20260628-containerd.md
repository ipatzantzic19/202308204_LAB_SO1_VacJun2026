# Evidencia de runtime en `grafana-vm`

Validación realizada el 28 de junio de 2026 mediante SSH por el subrecurso de KubeVirt. La
clave privada se generó en `PROYECTO2/.bin/` y está excluida de Git.

```text
cloud-init status: running (sin errores)

grafana-container.service: active (running)
ExecStart: /usr/bin/ctr run --rm --net-host ...
  zot.35-226-224-23.sslip.io/library/grafana-oss:11.5.2 grafana /run.sh

prometheus-container.service: active (running)
ExecStart: /usr/bin/ctr run --rm --net-host ...
  zot.35-226-224-23.sslip.io/library/prometheus:v2.55.1 prometheus ...

containerd.service: active (running)
containerd-shim-runc-v2: id=prometheus
containerd-shim-runc-v2: id=grafana
```

La contraseña aleatoria de Grafana fue redactada. Esta evidencia demuestra que los procesos
corren dentro de la VM bajo containerd y no como contenedores adicionales del Pod launcher.
