#!/bin/bash
# ============================================================
#  setup_entorno.sh — Inicializa el entorno Docker completo
#
#  Uso: bash setup_entorno.sh
#
#  Lo que hace:
#    1. Verifica que Docker esté instalado y corriendo
#    2. Crea la red Docker "monitoring" (si no existe)
#    3. Descarga las imágenes de los contenedores de prueba
#    4. Levanta el stack: Valkey + redis_exporter + Prometheus + Grafana
#    5. Espera a que todos los servicios estén saludables
#    6. Muestra el estado final
# ============================================================

# ── Colores para la salida ────────────────────────────────────
GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

# ── Directorio del script ─────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo ""
echo -e "${CYAN}${BOLD}=============================================${NC}"
echo -e "${CYAN}${BOLD}  Setup del Entorno Docker — SOPES1 P1      ${NC}"
echo -e "${CYAN}${BOLD}=============================================${NC}"
echo ""

# ── PASO 1: Verificar Docker ──────────────────────────────────
echo -e "${YELLOW}[1/6] Verificando Docker...${NC}"

if ! command -v docker &>/dev/null; then
    echo -e "${RED}✗ Docker no está instalado.${NC}"
    echo "  Instala Docker con: sudo apt install docker.io"
    exit 1
fi

if ! docker info &>/dev/null; then
    echo -e "${RED}✗ Docker no está corriendo. Iniciando...${NC}"
    sudo systemctl start docker
    sleep 3
    if ! docker info &>/dev/null; then
        echo -e "${RED}✗ No se pudo iniciar Docker.${NC}"
        exit 1
    fi
fi

if ! command -v docker compose &>/dev/null && ! docker compose version &>/dev/null 2>&1; then
    echo -e "${RED}✗ Docker Compose no está disponible.${NC}"
    echo "  Instala con: sudo apt install docker-compose-plugin"
    exit 1
fi

echo -e "${GREEN}✓ Docker $(docker --version | cut -d' ' -f3 | tr -d ',') disponible.${NC}"
echo -e "${GREEN}✓ Docker Compose $(docker compose version --short 2>/dev/null) disponible.${NC}"
echo ""

# ── PASO 2: Crear la red "monitoring" ────────────────────────
echo -e "${YELLOW}[2/6] Configurando red Docker 'monitoring'...${NC}"

if docker network ls | grep -q "monitoring"; then
    echo -e "${GREEN}✓ Red 'monitoring' ya existe.${NC}"
else
    docker network create monitoring
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Red 'monitoring' creada exitosamente.${NC}"
    else
        echo -e "${RED}✗ Error creando la red 'monitoring'.${NC}"
        exit 1
    fi
fi
echo ""

# ── PASO 3: Descargar imágenes de contenedores de prueba ──────
echo -e "${YELLOW}[3/6] Descargando imágenes de contenedores de prueba...${NC}"
echo "  Esto puede tardar según tu velocidad de internet."
echo ""

echo -e "  → Descargando ${BOLD}roldyoran/go-client${NC} (alto consumo RAM)..."
docker pull roldyoran/go-client 2>&1 | tail -1
echo ""

echo -e "  → Descargando ${BOLD}alpine:latest${NC} (alto CPU y bajo consumo)..."
docker pull alpine:latest 2>&1 | tail -1
echo ""

echo -e "${GREEN}✓ Imágenes de prueba descargadas.${NC}"
echo ""

# ── PASO 4: Verificar que prometheus.yml existe ───────────────
echo -e "${YELLOW}[4/6] Verificando archivos de configuración...${NC}"

if [ ! -f "$SCRIPT_DIR/prometheus.yml" ]; then
    echo -e "${RED}✗ No se encontró prometheus.yml en $SCRIPT_DIR${NC}"
    exit 1
fi

if [ ! -f "$SCRIPT_DIR/docker-compose.yml" ]; then
    echo -e "${RED}✗ No se encontró docker-compose.yml en $SCRIPT_DIR${NC}"
    exit 1
fi

echo -e "${GREEN}✓ docker-compose.yml encontrado.${NC}"
echo -e "${GREEN}✓ prometheus.yml encontrado.${NC}"
echo ""

# ── PASO 5: Levantar el stack ─────────────────────────────────
echo -e "${YELLOW}[5/6] Levantando servicios (Valkey + redis_exporter + Prometheus + Grafana)...${NC}"

docker compose up -d 2>&1

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Error al levantar los servicios. Revisa con: docker compose logs${NC}"
    exit 1
fi

echo ""
echo -e "  Esperando que los servicios estén saludables (30s)..."
sleep 15

# Esperar específicamente a Grafana
MAX_WAIT=60
WAITED=0
echo -n "  Esperando Grafana"
while ! curl -s http://localhost:3000/api/health &>/dev/null; do
    echo -n "."
    sleep 3
    WAITED=$((WAITED + 3))
    if [ $WAITED -ge $MAX_WAIT ]; then
        echo ""
        echo -e "${YELLOW}⚠ Grafana tardó más de lo esperado. Continúa el setup de todas formas.${NC}"
        break
    fi
done
echo ""
echo ""

# ── PASO 6: Mostrar estado final ──────────────────────────────
echo -e "${YELLOW}[6/6] Estado del entorno:${NC}"
echo ""
docker compose ps
echo ""

echo -e "${CYAN}${BOLD}=============================================${NC}"
echo -e "${CYAN}${BOLD}  ✓ Entorno listo. Accesos:               ${NC}"
echo -e "${CYAN}=============================================${NC}"
echo ""
echo -e "  ${BOLD}Grafana:${NC}     http://localhost:3000"
echo -e "             Usuario: admin | Contraseña: admin"
echo ""
echo -e "  ${BOLD}Prometheus:${NC}  http://localhost:9090"
echo -e "             Targets: http://localhost:9090/targets"
echo ""
echo -e "  ${BOLD}Valkey:${NC}      localhost:6379"
echo -e "             CLI: docker exec -it valkey valkey-cli"
echo ""
echo -e "  ${BOLD}Verificar:${NC}   bash test_entorno.sh"
echo ""
echo -e "${CYAN}=============================================${NC}"
echo ""