#!/bin/bash
# ============================================================
#  setup_grafana.sh — Configura Grafana automáticamente
#
#  Uso: bash setup_grafana.sh
#
#  Lo que hace:
#    1. Espera a que Grafana esté listo
#    2. Crea el datasource de Prometheus
#    3. Importa el dashboard del proyecto
#
#  Requiere: curl, el daemon corriendo (:9200), Grafana (:3000)
# ============================================================

GRAFANA_URL="http://localhost:3000"
GRAFANA_USER="admin"
GRAFANA_PASS="admin"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DASHBOARD_FILE="$SCRIPT_DIR/dashboard.json"

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

echo ""
echo -e "${CYAN}${BOLD}================================================${NC}"
echo -e "${CYAN}${BOLD}  Setup de Grafana — SOPES1 P1                 ${NC}"
echo -e "${CYAN}${BOLD}================================================${NC}"
echo ""

# ── PASO 1: Esperar a que Grafana esté listo ──────────────────
echo -e "${YELLOW}[1/3] Esperando a que Grafana esté disponible...${NC}"
MAX=60; WAITED=0
while ! curl -s "$GRAFANA_URL/api/health" | grep -q '"database":"ok"'; do
    echo -n "."
    sleep 3; WAITED=$((WAITED+3))
    [ $WAITED -ge $MAX ] && echo -e "\n${RED}✗ Grafana no respondió en ${MAX}s.${NC}" && exit 1
done
echo -e "\n${GREEN}✓ Grafana está listo.${NC}"
echo ""

# ── PASO 2: Crear datasource de Prometheus ────────────────────
echo -e "${YELLOW}[2/3] Configurando datasource Prometheus...${NC}"

# Verificar si el datasource ya existe
EXISTING=$(curl -s -u "$GRAFANA_USER:$GRAFANA_PASS" \
    "$GRAFANA_URL/api/datasources/name/Prometheus" 2>/dev/null | \
    python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('id',''))" 2>/dev/null)

if [ -n "$EXISTING" ] && [ "$EXISTING" != "None" ]; then
    echo -e "${GREEN}✓ Datasource Prometheus ya existe (ID: $EXISTING).${NC}"
else
    # Crear el datasource vía API de Grafana
    DS_RESPONSE=$(curl -s -X POST \
        -u "$GRAFANA_USER:$GRAFANA_PASS" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Prometheus",
            "type": "prometheus",
            "url": "http://prometheus:9090",
            "access": "proxy",
            "isDefault": true,
            "jsonData": {
                "httpMethod": "POST",
                "prometheusType": "Prometheus"
            }
        }' \
        "$GRAFANA_URL/api/datasources")

    DS_ID=$(echo "$DS_RESPONSE" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('datasource',{}).get('id','error'))" 2>/dev/null)

    if [ "$DS_ID" != "error" ] && [ -n "$DS_ID" ]; then
        echo -e "${GREEN}✓ Datasource Prometheus creado (ID: $DS_ID).${NC}"
    else
        echo -e "${RED}✗ Error creando datasource:${NC}"
        echo "$DS_RESPONSE"
        echo ""
        echo -e "${YELLOW}→ Crea el datasource manualmente:${NC}"
        echo "  1. Abrir http://localhost:3000"
        echo "  2. Connections → Data sources → Add new"
        echo "  3. Tipo: Prometheus"
        echo "  4. URL: http://prometheus:9090"
        echo "  5. Save & test"
    fi
fi
echo ""

# ── PASO 3: Importar el dashboard ────────────────────────────
echo -e "${YELLOW}[3/3] Importando dashboard SOPES1...${NC}"

if [ ! -f "$DASHBOARD_FILE" ]; then
    echo -e "${RED}✗ No se encontró $DASHBOARD_FILE${NC}"
    exit 1
fi

# Obtener el UID del datasource para inyectarlo en el dashboard
DS_UID=$(curl -s -u "$GRAFANA_USER:$GRAFANA_PASS" \
    "$GRAFANA_URL/api/datasources/name/Prometheus" 2>/dev/null | \
    python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('uid',''))" 2>/dev/null)

# Preparar el payload de importación inyectando el UID real del datasource
IMPORT_PAYLOAD=$(python3 << PYEOF
import json, sys

with open("$DASHBOARD_FILE") as f:
    dashboard = json.load(f)

# Inyectar el UID real del datasource en cada panel
ds_uid = "$DS_UID"
for panel in dashboard.get("panels", []):
    for target in panel.get("targets", []):
        if "datasource" in target:
            target["datasource"]["uid"] = ds_uid

# Payload para la API de importación de Grafana
payload = {
    "dashboard": dashboard,
    "overwrite": True,
    "folderId": 0,
    "inputs": [{"name": "DS_PROMETHEUS", "type": "datasource", "pluginId": "prometheus", "value": ds_uid}]
}
print(json.dumps(payload))
PYEOF
)

IMPORT_RESPONSE=$(echo "$IMPORT_PAYLOAD" | curl -s -X POST \
    -u "$GRAFANA_USER:$GRAFANA_PASS" \
    -H "Content-Type: application/json" \
    -d @- \
    "$GRAFANA_URL/api/dashboards/import")

IMPORT_STATUS=$(echo "$IMPORT_RESPONSE" | python3 -c "
import json,sys
d=json.load(sys.stdin)
print(d.get('status','error'), d.get('url',''))
" 2>/dev/null)

if echo "$IMPORT_STATUS" | grep -q "success\|imported"; then
    DASH_URL=$(echo "$IMPORT_RESPONSE" | python3 -c "
import json,sys; d=json.load(sys.stdin); print(d.get('importedUrl',''))
" 2>/dev/null)
    echo -e "${GREEN}✓ Dashboard importado exitosamente.${NC}"
    echo -e "${GREEN}✓ URL: $GRAFANA_URL$DASH_URL${NC}"
else
    echo -e "${RED}✗ Error importando dashboard:${NC}"
    echo "$IMPORT_RESPONSE"
    echo ""
    echo -e "${YELLOW}→ Importa el dashboard manualmente:${NC}"
    echo "  1. Abrir http://localhost:3000"
    echo "  2. Dashboards → New → Import"
    echo "  3. Subir el archivo: $DASHBOARD_FILE"
    echo "  4. Seleccionar datasource: Prometheus"
    echo "  5. Import"
fi

echo ""
echo -e "${CYAN}${BOLD}================================================${NC}"
echo -e "  ${BOLD}Accesos:${NC}"
echo -e "  Dashboard: ${CYAN}$GRAFANA_URL/d/sopes1_p1_202308204${NC}"
echo -e "  Explorar:  ${CYAN}$GRAFANA_URL/explore${NC}"
echo -e "  Targets:   ${CYAN}http://localhost:9090/targets${NC}"
echo -e "${CYAN}${BOLD}================================================${NC}"
echo ""