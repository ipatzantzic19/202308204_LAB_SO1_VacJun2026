# Fase 3 — Consumer, KubeVirt y persistencia en Valkey

## Objetivo

Consumir `predictions`, procesar cada evento y persistir métricas en Valkey ejecutado en
un contenedor de containerd dentro de una VM administrada por KubeVirt.

## Referencias

- Clase 11: consumer AMQP, acknowledgements y health probes.
- Clase 13: KubeVirt en GKE N1, CR de KubeVirt, VirtualMachine y `virtctl`.

## Plan

1. Crear `go-consumer` con reconexión AMQP, `autoAck=false` y ACK después de guardar.
2. Publicar `go-consumer:v1` en Zot y desplegarlo inicialmente sin consumir hasta que
   Valkey esté disponible.
3. Instalar KubeVirt Operator y CR usando la versión estable; aplicar los ajustes de
   affinity/tolerations documentados para GKE.
4. Verificar `virt-handler`, `virt-controller` y `virt-api` en estado Running.
5. Construir una imagen de disco Alpine/Ubuntu compatible y publicarla como imagen OCI.
6. Crear VM independiente `valkey-vm` con PVC, 1 CPU y memoria suficiente.
7. Dentro de la VM, instalar containerd y ejecutar Valkey con volumen persistente.
8. Exponer Valkey mediante Service ClusterIP y configurar `VALKEY_ADDR` en el consumer.
9. Implementar claves para máximos, mínimos, modas, victorias, usuarios y serie temporal.
10. Validar AMQP ACK, reinicio del consumer y persistencia tras reiniciar la VM.

## Criterio de finalización

- KubeVirt está Available.
- `valkey-vm` está Running y usa nested virtualization.
- Valkey corre bajo `ctr`, no Docker.
- Consumer está Running en GKE y la cola vuelve a cero.
- Consultas a Valkey muestran información real del flujo público.

