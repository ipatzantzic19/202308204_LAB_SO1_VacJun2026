#!/bin/bash
# ============================================================
#  spawn_containers.sh — Crea 5 contenedores aleatorios
#
#  Ejecutado por el cronjob cada 2 minutos.
#  El Daemon de Go lo registra en cron al iniciar y lo
#  elimina cuando el daemon se detiene (Fase 4).
#
#  Tipos disponibles (según enunciado):
#    0 → Alto RAM  : roldyoran/go-client
#    1 → Alto CPU  : alpine + bucle bc
#    2 → Bajo      : alpine sleep 240
#
#  Uso manual para pruebas: bash spawn_containers.sh
# ============================================================

# ── Configuración ─────────────────────────────────────────────
# Usar el directorio del script para los logs (más controlable que /tmp)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="${SCRIPT_DIR}/spawn_containers_internal.log"
TOTAL_CONTENEDORES=5

# Timestamp legible para los logs
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

# ── Función de log ────────────────────────────────────────────
log() {
    echo "[$TIMESTAMP] $1" | tee -a "$LOG_FILE" 2>/dev/null || echo "[$TIMESTAMP] $1"
}

# ── Inicio ────────────────────────────────────────────────────
log "========================================"
log "Iniciando creación de $TOTAL_CONTENEDORES contenedores..."
log "========================================"

# ── Crear 5 contenedores aleatorios ──────────────────────────
CREADOS=0
FALLIDOS=0

for i in $(seq 1 $TOTAL_CONTENEDORES); do

    # Número aleatorio entre 0 y 2 (inclusive)
    TIPO=$((RANDOM % 3))

    # Nombre corto y descriptivo: sopes1_<TIPO_LETRA>_<TIMESTAMP_CORTO>
    # Ejemplo: sopes1_R_294841, sopes1_C_294841, sopes1_L_294841
    # Esto evita nombres gigantes en Grafana y mantiene unicidad
    TIMESTAMP_CORTO=$(date +%s | tail -c 7)
    
    case $TIPO in
        0) TIPO_LETRA="R" ;;  # RAM alto
        1) TIPO_LETRA="C" ;;  # CPU alto
        2) TIPO_LETRA="L" ;;  # Low consumo
    esac
    
    NOMBRE="sopes1_${TIPO_LETRA}_${TIMESTAMP_CORTO}"

    case $TIPO in

        # ── TIPO 0: Alto consumo de RAM ───────────────────────
        0)
            log "  [${i}/${TOTAL_CONTENEDORES}] Tipo: ALTO_RAM → roldyoran/go-client"
            docker run -d \
                --name "$NOMBRE" \
                roldyoran/go-client \
                >> "$LOG_FILE" 2>&1

            EXIT_CODE=$?
            ;;

        # ── TIPO 1: Alto consumo de CPU ───────────────────────
        1)
            log "  [${i}/${TOTAL_CONTENEDORES}] Tipo: ALTO_CPU → alpine + bucle bc"
            docker run -d \
                --name "$NOMBRE" \
                alpine \
                sh -c "while true; do echo '2^20' | bc > /dev/null; sleep 2; done" \
                >> "$LOG_FILE" 2>&1

            EXIT_CODE=$?
            ;;

        # ── TIPO 2: Bajo consumo ──────────────────────────────
        2)
            log "  [${i}/${TOTAL_CONTENEDORES}] Tipo: BAJO → alpine sleep 240"
            docker run -d \
                --name "$NOMBRE" \
                alpine \
                sleep 240 \
                >> "$LOG_FILE" 2>&1

            EXIT_CODE=$?
            ;;
    esac

    # Registrar resultado de cada contenedor
    if [ $EXIT_CODE -eq 0 ]; then
        log "  └─ ✓ Creado: $NOMBRE"
        CREADOS=$((CREADOS + 1))
    else
        log "  └─ ✗ Error creando: $NOMBRE (código: $EXIT_CODE)"
        FALLIDOS=$((FALLIDOS + 1))
    fi

    # Pequeña pausa entre contenedores para no saturar el daemon de Docker
    sleep 0.3
done

# ── Resumen ───────────────────────────────────────────────────
log "----------------------------------------"
log "Resumen: $CREADOS creados, $FALLIDOS fallidos"
log "Contenedores sopes1 activos ahora:"
docker ps --filter "name=sopes1_" --format "  {{.Names}} | {{.Image}} | {{.Status}}" \
    >> "$LOG_FILE" 2>&1
log "========================================"

exit 0