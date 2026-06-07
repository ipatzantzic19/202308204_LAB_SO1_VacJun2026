#!/bin/bash
# ============================================================
#  test_spawn.sh — Verifica que spawn_containers.sh funciona
#
#  Uso: bash test_spawn.sh
#
#  Pruebas:
#    1. El script tiene permisos de ejecución
#    2. Docker está disponible
#    3. Las imágenes necesarias están en local
#    4. El script crea exactamente 5 contenedores
#    5. Los nombres siguen el patrón "sopes1_*"
#    6. Los contenedores siguen vivos después de crearlos
#    7. Limpieza: elimina los contenedores de prueba
# ============================================================

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SPAWN_SCRIPT="$SCRIPT_DIR/spawn_containers.sh"
PASS=0; FAIL=0

echo ""
echo -e "${CYAN}${BOLD}================================================${NC}"
echo -e "${CYAN}${BOLD}  Test del Script de Cronjob — SOPES1 P1       ${NC}"
echo -e "${CYAN}${BOLD}================================================${NC}"
echo ""

# ── Función de verificación ───────────────────────────────────
check() {
    local desc="$1"; local ok="$2"; local detail="${3:-}"
    if [ "$ok" -eq 0 ]; then
        echo -e "  ${GREEN}✓ PASS${NC}  $desc"
        [ -n "$detail" ] && echo -e "           ${CYAN}→ $detail${NC}"
        PASS=$((PASS+1))
    else
        echo -e "  ${RED}✗ FAIL${NC}  $desc"
        [ -n "$detail" ] && echo -e "           ${RED}→ $detail${NC}"
        FAIL=$((FAIL+1))
    fi
}

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 1/7] Verificando el script...${NC}"

[ -f "$SPAWN_SCRIPT" ]
check "spawn_containers.sh existe en $SCRIPT_DIR" $?

chmod +x "$SPAWN_SCRIPT"
[ -x "$SPAWN_SCRIPT" ]
check "spawn_containers.sh tiene permisos de ejecución" $?
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 2/7] Docker disponible...${NC}"

docker info &>/dev/null
check "Docker daemon está activo" $? "$(docker --version)"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 3/7] Imágenes disponibles localmente...${NC}"

docker images | grep -q "roldyoran/go-client"
check "Imagen roldyoran/go-client en local" $? "$(docker images roldyoran/go-client --format '{{.Size}}' 2>/dev/null)"

docker images | grep -q "alpine"
check "Imagen alpine en local" $? "$(docker images alpine --format '{{.Size}}' 2>/dev/null)"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 4/7] Ejecutando spawn_containers.sh...${NC}"
echo -e "  (Creará 5 contenedores, puede tardar ~15 segundos)"
echo ""

# Guardar lista de contenedores sopes1 ANTES de ejecutar
ANTES=$(docker ps --filter "name=sopes1_" --format "{{.Names}}" | wc -l)

# Ejecutar el script
bash "$SPAWN_SCRIPT"
EXIT_CODE=$?

check "spawn_containers.sh terminó sin errores (exit code: $EXIT_CODE)" $EXIT_CODE
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 5/7] Verificando cantidad de contenedores creados...${NC}"

sleep 2  # Dar tiempo a Docker para registrar los contenedores

# Contar contenedores sopes1 DESPUÉS
DESPUES=$(docker ps --filter "name=sopes1_" --format "{{.Names}}" | wc -l)
NUEVOS=$((DESPUES - ANTES))

[ "$NUEVOS" -eq 5 ]
check "Se crearon exactamente 5 contenedores nuevos" $? "Antes: $ANTES | Después: $DESPUES | Nuevos: $NUEVOS"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 6/7] Verificando nombres y estado de los contenedores...${NC}"

CONTENEDORES_SOPES1=$(docker ps --filter "name=sopes1_" --format "{{.Names}}")
TOTAL_ACTIVOS=$(echo "$CONTENEDORES_SOPES1" | grep -c "sopes1_" || true)

echo -e "  Contenedores sopes1 activos:"
docker ps --filter "name=sopes1_" \
    --format "  {{.Names}}\t{{.Image}}\t{{.Status}}" | \
    while read line; do echo -e "  ${CYAN}$line${NC}"; done

echo ""

[ "$TOTAL_ACTIVOS" -ge 5 ]
check "Hay al menos 5 contenedores sopes1 activos" $? "Total: $TOTAL_ACTIVOS"

# Verificar que los nombres siguen el patrón correcto
PATRON_OK=$(docker ps --filter "name=sopes1_" --format "{{.Names}}" | grep -c "^sopes1_" || true)
[ "$PATRON_OK" -ge 5 ]
check "Los nombres siguen el patrón 'sopes1_*'" $? "$PATRON_OK contenedores con nombre correcto"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 7/7] Verificando log generado...${NC}"

[ -f "/tmp/spawn_containers.log" ]
check "Archivo de log creado en /tmp/spawn_containers.log" $?

LOG_LINES=$(wc -l < /tmp/spawn_containers.log 2>/dev/null || echo 0)
[ "$LOG_LINES" -gt 5 ]
check "Log tiene contenido ($LOG_LINES líneas)" $? "Últimas 5 líneas del log:"
echo ""
echo -e "  ${CYAN}--- /tmp/spawn_containers.log (últimas 5 líneas) ---${NC}"
tail -5 /tmp/spawn_containers.log | while read line; do
    echo -e "  ${CYAN}$line${NC}"
done
echo ""

# ─────────────────────────────────────────────────────────────
# LIMPIEZA: Eliminar los contenedores de prueba
# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}Limpiando contenedores de prueba...${NC}"
ELIMINADOS=0
for nombre in $(docker ps --filter "name=sopes1_" --format "{{.Names}}"); do
    docker stop "$nombre" &>/dev/null
    docker rm "$nombre" &>/dev/null
    ELIMINADOS=$((ELIMINADOS + 1))
done
echo -e "  ${GREEN}✓ $ELIMINADOS contenedores eliminados.${NC}"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${CYAN}${BOLD}================================================${NC}"
echo -e "  Resultado: ${GREEN}$PASS PASS${NC}  |  ${RED}$FAIL FAIL${NC}"

if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}${BOLD}✓ Fase 3 completada exitosamente.${NC}"
    echo ""
    echo -e "  El script está listo para ser registrado"
    echo -e "  como cronjob por el Daemon de Go (Fase 4)."
else
    echo -e "  ${RED}✗ Hay $FAIL prueba(s) fallidas.${NC}"
fi
echo -e "${CYAN}${BOLD}================================================${NC}"
echo ""