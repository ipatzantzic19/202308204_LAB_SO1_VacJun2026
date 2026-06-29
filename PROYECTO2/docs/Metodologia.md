# Metodología

El proyecto se desarrolló incrementalmente en cinco fases: infraestructura, mensajería,
persistencia, observabilidad y entrega. Cada fase se consideró completa únicamente después de
compilar, publicar sus imágenes en Zot, desplegar en GKE y obtener una prueba funcional.

Se aplicaron estos criterios:

1. **Contrato único:** `prediction.proto` es la fuente compartida entre cliente y servidor.
2. **Fallo visible:** cada capa propaga errores; una dependencia caída no produce HTTP 200.
3. **Entrega confiable:** RabbitMQ usa mensajes persistentes y el consumer confirma después de
   persistir en Valkey.
4. **Infraestructura declarativa:** Kubernetes, Gateway API, HPA y VMs se describen en YAML y
   se ensamblan con Kustomize.
5. **Secretos fuera de Git:** las credenciales se generan o sincronizan durante el despliegue.
6. **Medición reproducible:** scripts automatizan pruebas unitarias, carga 1 vs. 2 réplicas y
   el ciclo del HPA.
7. **Evidencia:** CSV, reportes HTML y resúmenes Markdown quedan en `evidence/`.

La comparación de rendimiento cambió únicamente la cantidad de réplicas de Go D2; Rust se
mantuvo fijo para evitar mezclar variables. La prueba del HPA se ejecutó por separado. Esta
separación permite atribuir los resultados observados al componente evaluado.
