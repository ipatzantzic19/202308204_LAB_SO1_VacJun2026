# Fase 4 — Grafana, observabilidad y pruebas de carga

## Objetivo

Visualizar datos reales, habilitar HPA y medir el sistema con una y dos réplicas.

## Plan

1. Crear una segunda VM KubeVirt independiente: `grafana-vm`.
2. Instalar containerd dentro de la VM y ejecutar Grafana con `ctr`.
3. Configurar datasource Redis/Valkey y persistencia del directorio de Grafana.
4. Construir los paneles obligatorios; el carnet `202308204` asigna el equipo **BRA**.
5. Exponer Grafana de forma controlada mediante Service/Gateway o port-forward para
   evaluación.
6. Aplicar HPA a Rust: mínimo 1, máximo 3, objetivo CPU 30%.
7. Ejecutar Locust contra `/grpc-202308204` y registrar RPS, percentiles y errores.
8. Repetir las pruebas con 1 y 2 réplicas del Deployment indicado por la cátedra.
9. Implementar Dapr solo como flujo opcional `/dapr-202308204`, sin reemplazar RabbitMQ.

## Dashboard obligatorio BRA

- Máximo y mínimo de goles local/visitante.
- Top de equipos con victorias predichas.
- Top de usuarios activos.
- Moda de goles local/visitante.
- Serie temporal BRA como local y visitante.
- Nombre BRA y total de predicciones relacionadas.

## Criterio de finalización

- Grafana corre en containerd dentro de su propia VM.
- Todos los paneles muestran datos reales.
- HPA escala y regresa a una réplica.
- Existen resultados reproducibles de 1 vs 2 réplicas.
- El flujo obligatorio mantiene una tasa de error aceptable.

