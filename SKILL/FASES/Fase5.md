# Fase 5 — Documentación, reproducibilidad y entrega

> **Estado:** implementada y documentada el 29 de junio de 2026. Solo quedan los controles
> manuales de privacidad, colaborador, commit y push en GitHub.

## Objetivo

Entregar un repositorio reproducible y evidencia verificable exclusivamente en Markdown.

## Plan

1. Crear `PROYECTO2/README.md` con requisitos, costos y arranque por orden.
2. Redactar `PROYECTO2/docs/ManualTecnico.md` con arquitectura y flujo completo.
3. Documentar REST, gRPC, RabbitMQ, Gateway, KubeVirt, containerd, Valkey y Grafana.
4. Publicar `prediction.proto` como OCI Artifact adicional en Zot usando ORAS y documentar su
   uso; el auxiliar dispensó este requisito, pero se conservó la implementación.
5. Incluir tabla de imágenes, tags, puertos, Services y variables de entorno.
6. Guardar evidencias de Pods, VMs, cola, Valkey, Grafana, HPA y Locust.
7. Documentar resultados de 1 vs 2 réplicas y conclusiones técnicas.
8. Ejecutar una prueba de reproducibilidad desde namespace limpio.
9. Revisar secretos, placeholders, archivos generados y estructura del repositorio.
10. Verificar repositorio privado y acceso de `CamiloSincal`.

## Criterio de finalización

- No existen placeholders como `<IP-ZOT>` o `<PROJECT-ID>` en manifiestos aplicables.
- Ningún secreto real está versionado.
- Todas las imágenes existen en Zot; el OCI Artifact adicional también fue verificado.
- El sistema se reconstruye siguiendo el README.
- El manual responde cada requisito del enunciado y contiene conclusiones sustentadas.
