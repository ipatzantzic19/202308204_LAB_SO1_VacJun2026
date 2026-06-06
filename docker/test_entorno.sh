#!/bin/bash
# ============================================================
#  test_entorno.sh — Verifica que el stack Docker está correcto
#
#  Uso: bash test_entorno.sh
#
#  Pruebas:
#    1. Red "monitoring" existe
#    2. Valkey corre y responde PING
#    3. redis_exporter expone métricas
#    4. Prometheus está UP y los targets están saludables
#    5. Grafana responde en el puerto 3000
#    6. Las imágenes de contenedores están disponibles
# ============================================================

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

PASS=0; FAIL=0

echo ""
echo -e "${CYAN}${BOLD}================================================${NC}"
echo -e "${CYAN}${BOLD}  Test del Entorno Docker — SOPES1 P1          ${NC}"
echo -e "${CYAN}${BOLD}================================================${NC}"
echo ""

# ── Función auxiliar ──────────────────────────────────────────
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
echo -e "${YELLOW}[TEST 1/6] Red Docker 'monitoring'...${NC}"
docker network ls | grep -q "monitoring"
check "Red 'monitoring' existe" $? "$(docker network inspect monitoring --format '{{.Driver}} — {{.Scope}}' 2>/dev/null)"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 2/6] Valkey (puerto 6379)...${NC}"

# Verificar que el contenedor corre
docker ps | grep -q "valkey"
check "Contenedor 'valkey' está corriendo" $? "$(docker inspect valkey --format '{{.State.Status}}' 2>/dev/null)"

# Verificar que responde al PING de valkey-cli
PING_RESULT=$(docker exec valkey valkey-cli PING 2>/dev/null)
[ "$PING_RESULT" = "PONG" ]
check "Valkey responde PING → PONG" $? "$PING_RESULT"

# Verificar escritura y lectura
docker exec valkey valkey-cli SET test_key "sopes1_ok" EX 60 &>/dev/null
READ_VAL=$(docker exec valkey valkey-cli GET test_key 2>/dev/null)
[ "$READ_VAL" = "sopes1_ok" ]
check "Valkey acepta SET/GET" $? "SET 'test_key' → GET '$READ_VAL'"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 3/6] redis_exporter (puerto 9121)...${NC}"

docker ps | grep -q "redis_exporter"
check "Contenedor 'redis_exporter' corre" $? "$(docker inspect redis_exporter --format '{{.State.Status}}' 2>/dev/null)"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:9121/metrics 2>/dev/null)
[ "$HTTP_CODE" = "200" ]
check "redis_exporter expone /metrics (HTTP $HTTP_CODE)" $? "http://localhost:9121/metrics"

# Verificar que hay métricas de Valkey
METRICS_CHECK=$(curl -s http://localhost:9121/metrics 2>/dev/null | grep -c "redis_up")
[ "$METRICS_CHECK" -gt 0 ]
check "Métrica 'redis_up' presente en el exporter" $? "Encontradas: $METRICS_CHECK ocurrencias"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 4/6] Prometheus (puerto 9090)...${NC}"

docker ps | grep -q "prometheus"
check "Contenedor 'prometheus' corre" $? "$(docker inspect prometheus --format '{{.State.Status}}' 2>/dev/null)"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/-/healthy 2>/dev/null)
[ "$HTTP_CODE" = "200" ]
check "Prometheus responde /-/healthy (HTTP $HTTP_CODE)" $? "http://localhost:9090/-/healthy"

# Verificar targets (puede tardar en estar todos UP)
TARGETS_UP=$(curl -s "http://localhost:9090/api/v1/targets" 2>/dev/null | \
    python3 -c "
import json,sys
d=json.load(sys.stdin)
ups=[t for t in d['data']['activeTargets'] if t['health']=='up']
print(f'{len(ups)} UP de {len(d[\"data\"][\"activeTargets\"])} targets')
" 2>/dev/null)
check "Targets en Prometheus" 0 "$TARGETS_UP"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 5/6] Grafana (puerto 3000)...${NC}"

docker ps | grep -q "grafana"
check "Contenedor 'grafana' corre" $? "$(docker inspect grafana --format '{{.State.Status}}' 2>/dev/null)"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/api/health 2>/dev/null)
[ "$HTTP_CODE" = "200" ]
check "Grafana responde /api/health (HTTP $HTTP_CODE)" $? "http://localhost:3000"

GRAFANA_HEALTH=$(curl -s http://localhost:3000/api/health 2>/dev/null | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('database','?'))" 2>/dev/null)
[ "$GRAFANA_HEALTH" = "ok" ]
check "Base de datos interna de Grafana" $? "Estado: $GRAFANA_HEALTH"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${YELLOW}[TEST 6/6] Imágenes de contenedores de prueba...${NC}"

docker images | grep -q "roldyoran/go-client"
check "Imagen 'roldyoran/go-client' disponible localmente" $? "$(docker images roldyoran/go-client --format '{{.Repository}}:{{.Tag}} ({{.Size}})' 2>/dev/null)"

docker images | grep -q "alpine"
check "Imagen 'alpine' disponible localmente" $? "$(docker images alpine --format '{{.Repository}}:{{.Tag}} ({{.Size}})' 2>/dev/null)"
echo ""

# ─────────────────────────────────────────────────────────────
echo -e "${CYAN}${BOLD}================================================${NC}"
echo -e "  Resultado: ${GREEN}$PASS PASS${NC}  |  ${RED}$FAIL FAIL${NC}"

if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}${BOLD}✓ Fase 2 completada exitosamente.${NC}"
    echo ""
    echo -e "  Próximos pasos:"
    echo -e "    → Configura Prometheus como datasource en Grafana:"
    echo -e "      http://localhost:3000 → Connections → Data sources"
    echo -e "    → Prueba las imágenes: ${BOLD}bash test_imagenes.sh${NC}"
    echo -e "    → Continúa con la Fase 3 (script del Cronjob)"
else
    echo -e "  ${RED}✗ Hay $FAIL prueba(s) fallidas.${NC}"
    echo -e "  Revisa con: ${BOLD}docker compose logs${NC}"
fi
echo -e "${CYAN}${BOLD}================================================${NC}"
echo ""