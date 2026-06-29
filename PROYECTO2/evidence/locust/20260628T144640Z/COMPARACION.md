# Comparación Locust — Rust API

| Métrica | 1 réplica | 2 réplicas |
|---|---:|---:|
| Requests/s | 113.61028400714244 | 112.98175634000545 |
| Failures/s | 0.0 | 0.0 |
| Promedio (ms) | 126.09706419895349 | 128.8317656750152 |
| p50 (ms) | 120 | 120 |
| p95 (ms) | 150 | 150 |
| p99 (ms) | 310 | 400 |
| Solicitudes | 6710 | 6671 |
| Errores | 0 | 0 |

Conclusión: dos réplicas no cambiaron materialmente el throughput (-0.55%). Esto indica si Rust es el cuello de botella bajo la carga ensayada; el resto del flujo conservó una réplica.

Pruebas ejecutadas contra `/grpc-202308204`; los CSV y reportes HTML del mismo directorio son la evidencia fuente.
