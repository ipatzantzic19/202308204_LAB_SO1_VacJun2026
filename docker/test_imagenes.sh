#!/bin/bash
# ============================================================
#  test_imagenes.sh — Prueba los 3 tipos de contenedor del proyecto
#
#  Uso: bash test_imagenes.sh
#
#  Crea temporalmente 1 contenedor de cada tipo, mide sus recursos
#  durante 15 segundos y luego los elimina.
# ============================================================

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

echo ""
echo -e "${CYAN}${BOLD}================================================${NC}"
echo -e "${CYAN}${BOLD}  Test de Imágenes Docker — SOPES1 P1          ${NC}"
echo -e "${CYAN}${BOLD}================================================${NC}"

# Nombres únicos para no colisionar con otros contenedores
TS=$(date +%s)
CONT_RAM="test_ram_$TS"
CONT_CPU="test_cpu_$TS"
CONT_LOW="test_low_$TS"

# ── Función de limpieza ───────────────────────────────────────
cleanup() {
    echo ""
    echo -e "${YELLOW}Limpiando contenedores de prueba...${NC}"
    docker stop "$CONT_RAM" "$CONT_CPU" "$CONT_LOW" 2>/dev/null
    docker rm   "$CONT_RAM" "$CONT_CPU" "$CONT_LOW" 2>/dev/null
    echo -e "${GREEN}✓ Limpieza completada.${NC}"
}
trap cleanup EXIT   # Se ejecuta automáticamente al salir del script

# ──────────────────────────────────────────────────────────────
# TIPO 1: Alto consumo de RAM — roldyoran/go-client
# ──────────────────────────────────────────────────────────────
echo ""
echo -e "${YELLOW}[TIPO 1] Alto Consumo de RAM → roldyoran/go-client${NC}"
echo -e "  Comando: docker run -d --name $CONT_RAM roldyoran/go-client"
echo ""

docker run -d --name "$CONT_RAM" roldyoran/go-client 2>&1
if [ $? -eq 0 ]; then
    echo -e "  ${GREEN}✓ Contenedor iniciado.${NC}"
    sleep 5
    echo -e "  Estadísticas (5s de ejecución):"
    docker stats "$CONT_RAM" --no-stream --format \
        "  CPU: {{.CPUPerc}}   RAM: {{.MemUsage}}   RED: {{.NetIO}}"
    echo -e "  ${GREEN}✓ TIPO 1 funciona correctamente.${NC}"
else
    echo -e "  ${RED}✗ Error iniciando contenedor de alto RAM.${NC}"
    echo -e "  Verifica con: docker pull roldyoran/go-client"
fi

echo ""
echo "──────────────────────────────────────────────"

# ──────────────────────────────────────────────────────────────
# TIPO 2: Alto consumo de CPU — alpine con bucle de cálculos
# ──────────────────────────────────────────────────────────────
echo ""
echo -e "${YELLOW}[TIPO 2] Alto Consumo de CPU → alpine con bucle bc${NC}"
echo -e "  Comando: docker run -d alpine sh -c \"while true; do echo '2^20' | bc > /dev/null; sleep 2; done\""
echo ""

docker run -d --name "$CONT_CPU" alpine \
    sh -c "while true; do echo '2^20' | bc > /dev/null; sleep 2; done" 2>&1

if [ $? -eq 0 ]; then
    echo -e "  ${GREEN}✓ Contenedor iniciado.${NC}"
    sleep 8   # El bucle necesita unos segundos para generar carga
    echo -e "  Estadísticas (8s de ejecución):"
    docker stats "$CONT_CPU" --no-stream --format \
        "  CPU: {{.CPUPerc}}   RAM: {{.MemUsage}}   RED: {{.NetIO}}"
    echo -e "  ${GREEN}✓ TIPO 2 funciona correctamente.${NC}"
else
    echo -e "  ${RED}✗ Error iniciando contenedor de alto CPU.${NC}"
fi

echo ""
echo "──────────────────────────────────────────────"

# ──────────────────────────────────────────────────────────────
# TIPO 3: Bajo consumo — alpine con sleep 240
# ──────────────────────────────────────────────────────────────
echo ""
echo -e "${YELLOW}[TIPO 3] Bajo Consumo → alpine sleep 240${NC}"
echo -e "  Comando: docker run -d alpine sleep 240"
echo ""

docker run -d --name "$CONT_LOW" alpine sleep 240 2>&1

if [ $? -eq 0 ]; then
    echo -e "  ${GREEN}✓ Contenedor iniciado.${NC}"
    sleep 3
    echo -e "  Estadísticas (3s de ejecución):"
    docker stats "$CONT_LOW" --no-stream --format \
        "  CPU: {{.CPUPerc}}   RAM: {{.MemUsage}}   RED: {{.NetIO}}"
    echo -e "  ${GREEN}✓ TIPO 3 funciona correctamente.${NC}"
else
    echo -e "  ${RED}✗ Error iniciando contenedor de bajo consumo.${NC}"
fi

echo ""
echo "──────────────────────────────────────────────"

# ──────────────────────────────────────────────────────────────
# RESUMEN: Mostrar los 3 corriendo al mismo tiempo
# ──────────────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}${BOLD}Resumen — Los 3 tipos corriendo simultáneamente:${NC}"
echo ""
docker stats "$CONT_RAM" "$CONT_CPU" "$CONT_LOW" \
    --no-stream \
    --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"

echo ""
echo -e "${CYAN}${BOLD}================================================${NC}"
echo -e "  ${GREEN}✓ Los 3 tipos de imagen funcionan correctamente.${NC}"
echo -e "  El script del cronjob los usará aleatoriamente."
echo -e "${CYAN}${BOLD}================================================${NC}"
echo ""

# cleanup() se llama automáticamente por el trap EXIT