# Comparación Locust — Go D2

| Métrica | 1 réplica | 2 réplicas |
|---|---:|---:|
| Requests/s | 112.62478533359452 | 113.98610059924194 |
| Failures/s | 0.0 | 0.0 |
| Promedio (ms) | 127.24020255373868 | 123.21059002777952 |
| p50 (ms) | 120 | 120 |
| p95 (ms) | 150 | 150 |
| p99 (ms) | 230 | 200 |
| Solicitudes | 6653 | 6733 |
| Errores | 0 | 0 |

Conclusión: dos réplicas no cambiaron materialmente el throughput (+1.21%) al escalar Go D2 de una a dos réplicas. Rust se mantuvo en una réplica para aislar esta variable.

Pruebas ejecutadas contra `/grpc-202308204`; los CSV y reportes HTML del mismo directorio son la evidencia fuente.
