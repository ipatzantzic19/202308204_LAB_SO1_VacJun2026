#!/bin/bash
# ============================================================
#  test_module.sh — Verifica que el módulo funciona correctamente
#
#  Uso: bash test_module.sh
#
#  Realiza 5 pruebas:
#    1. El módulo está cargado (lsmod)
#    2. El archivo /proc existe
#    3. El archivo /proc tiene contenido
#    4. El JSON es válido (parseable)
#    5. El JSON contiene los campos requeridos por el proyecto
# ============================================================

PROC_NAME="continfo_pr1_so1_202308204"
PROC_FILE="/proc/$PROC_NAME"
MODULE_NAME="sys_info_module"

# Colores para la salida
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # Sin color

PASS=0
FAIL=0

echo ""
echo -e "${CYAN}=============================================${NC}"
echo -e "${CYAN}  Test del Módulo de Kernel - SOPES1 P1     ${NC}"
echo -e "${CYAN}  Archivo: /proc/$PROC_NAME${NC}"
echo -e "${CYAN}=============================================${NC}"
echo ""

# ── Función auxiliar para reportar resultados ─────────────────
check() {
    local desc="$1"
    local result="$2"  # 0 = pass, otro = fail
    local detail="$3"

    if [ "$result" -eq 0 ]; then
        echo -e "  ${GREEN}✓ PASS${NC}  $desc"
        [ -n "$detail" ] && echo -e "           ${CYAN}→ $detail${NC}"
        PASS=$((PASS + 1))
    else
        echo -e "  ${RED}✗ FAIL${NC}  $desc"
        [ -n "$detail" ] && echo -e "           ${RED}→ $detail${NC}"
        FAIL=$((FAIL + 1))
    fi
}

# ── TEST 1: Módulo cargado ────────────────────────────────────
echo -e "${YELLOW}[TEST 1/5] Verificando que el módulo está cargado...${NC}"
if lsmod | grep -q "^$MODULE_NAME"; then
    check "Módulo '$MODULE_NAME' en lsmod" 0 "$(lsmod | grep "^$MODULE_NAME")"
else
    check "Módulo '$MODULE_NAME' en lsmod" 1 "No aparece en lsmod. Carga con: sudo insmod sysinfo_module.ko"
fi
echo ""

# ── TEST 2: Archivo /proc existe ─────────────────────────────
echo -e "${YELLOW}[TEST 2/5] Verificando que /proc/$PROC_NAME existe...${NC}"
if [ -f "$PROC_FILE" ]; then
    PERMS=$(ls -la "$PROC_FILE" 2>/dev/null)
    check "Archivo $PROC_FILE existe" 0 "$PERMS"
else
    check "Archivo $PROC_FILE existe" 1 "El archivo no existe en /proc"
fi
echo ""

# ── TEST 3: Archivo tiene contenido ──────────────────────────
echo -e "${YELLOW}[TEST 3/5] Verificando que el archivo tiene contenido...${NC}"
if [ -f "$PROC_FILE" ]; then
    CONTENT=$(cat "$PROC_FILE" 2>/dev/null)
    LEN=${#CONTENT}
    if [ "$LEN" -gt 10 ]; then
        check "Archivo tiene contenido ($LEN bytes)" 0 "Primeros 100 chars: ${CONTENT:0:100}..."
    else
        check "Archivo tiene contenido" 1 "El archivo está vacío o tiene muy poco contenido"
    fi
else
    check "Archivo tiene contenido" 1 "No se puede leer (archivo no existe)"
fi
echo ""

# ── TEST 4: JSON válido ───────────────────────────────────────
echo -e "${YELLOW}[TEST 4/5] Validando que el contenido es JSON válido...${NC}"
if [ -f "$PROC_FILE" ] && command -v python3 &>/dev/null; then
    PARSE_RESULT=$(cat "$PROC_FILE" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    print('OK')
    print(f'Campos raiz: {list(data.keys())}')
    print(f'Procesos: {len(data.get(\"Processes\", []))}')
except Exception as e:
    print(f'ERROR: {e}')
" 2>&1)

    if echo "$PARSE_RESULT" | grep -q "^OK"; then
        check "JSON es válido" 0 "$PARSE_RESULT"
    else
        check "JSON es válido" 1 "$PARSE_RESULT"
    fi
else
    if ! command -v python3 &>/dev/null; then
        echo -e "  ${YELLOW}⚠ SKIP${NC}  JSON válido (python3 no disponible para validar)"
    else
        check "JSON es válido" 1 "No se puede leer el archivo"
    fi
fi
echo ""

# ── TEST 5: Campos requeridos por el proyecto ─────────────────
echo -e "${YELLOW}[TEST 5/5] Verificando campos requeridos por el enunciado...${NC}"
if [ -f "$PROC_FILE" ] && command -v python3 &>/dev/null; then
    python3 << 'PYEOF'
import json, sys

proc_file = "/proc/PROC_NAME_PLACEHOLDER"

try:
    with open(proc_file) as f:
        data = json.load(f)
except Exception as e:
    print(f"  ERROR al leer/parsear: {e}")
    sys.exit(1)

CYAN  = "\033[0;36m"
GREEN = "\033[0;32m"
RED   = "\033[0;31m"
NC    = "\033[0m"

def field_check(label, condition, detail=""):
    if condition:
        print(f"  {GREEN}✓{NC}  {label}" + (f" → {CYAN}{detail}{NC}" if detail else ""))
    else:
        print(f"  {RED}✗{NC}  {label}" + (f" → {RED}FALTA{NC}" if not detail else f" → {RED}{detail}{NC}"))

# Campos de memoria
print(f"\n  Campos de Memoria:")
field_check("Totalram (total de RAM)", "Totalram" in data, str(data.get("Totalram", "FALTA")) + " KB")
field_check("Freeram  (RAM libre)",    "Freeram"  in data, str(data.get("Freeram",  "FALTA")) + " KB")
field_check("Usedram  (RAM usada)",    "Usedram"  in data, str(data.get("Usedram",  "FALTA")) + " KB")
field_check("Procs    (nro procesos)", "Procs"    in data, str(data.get("Procs",    "FALTA")))

# Campos de procesos
print(f"\n  Array de Procesos:")
procs = data.get("Processes", None)
field_check("Processes (array de procesos)", procs is not None, f"{len(procs)} procesos" if procs else "FALTA")

if procs and len(procs) > 0:
    p = procs[0]
    print(f"\n  Campos por proceso (verificando el primero: PID={p.get('PID','?')}):")
    field_check("PID",          "PID"          in p, str(p.get("PID", "FALTA")))
    field_check("Name",         "Name"         in p, str(p.get("Name", "FALTA")))
    field_check("Cmdline",      "Cmdline"      in p, str(p.get("Cmdline", "FALTA"))[:50])
    field_check("vsz (KB)",     "vsz"          in p, str(p.get("vsz", "FALTA")) + " KB")
    field_check("rss (KB)",     "rss"          in p, str(p.get("rss", "FALTA")) + " KB")
    field_check("Memory_Usage", "Memory_Usage" in p, str(p.get("Memory_Usage", "FALTA")) + "%")
    field_check("CPU_Usage",    "CPU_Usage"    in p, str(p.get("CPU_Usage", "FALTA")) + "%")

PYEOF
    # Sustituir el placeholder del PROC_NAME en el script de Python
    python3 << PYEOF2
import json, sys

proc_file = "$PROC_FILE"

try:
    with open(proc_file) as f:
        data = json.load(f)
except Exception as e:
    print(f"  ERROR al leer/parsear: {e}")
    sys.exit(1)

CYAN  = "\033[0;36m"
GREEN = "\033[0;32m"
RED   = "\033[0;31m"
NC    = "\033[0m"

def field_check(label, condition, detail=""):
    if condition:
        print(f"  {GREEN}✓{NC}  {label}" + (f" → {CYAN}{detail}{NC}" if detail else ""))
    else:
        print(f"  {RED}✗{NC}  {label}" + (f" → {RED}FALTA{NC}" if not detail else f" → {RED}{detail}{NC}"))

print(f"\n  Campos de Memoria:")
field_check("Totalram (total de RAM)", "Totalram" in data, str(data.get("Totalram", "FALTA")) + " KB")
field_check("Freeram  (RAM libre)",    "Freeram"  in data, str(data.get("Freeram",  "FALTA")) + " KB")
field_check("Usedram  (RAM usada)",    "Usedram"  in data, str(data.get("Usedram",  "FALTA")) + " KB")
field_check("Procs    (nro procesos)", "Procs"    in data, str(data.get("Procs",    "FALTA")))

print(f"\n  Array de Procesos:")
procs = data.get("Processes", None)
field_check("Processes (array de procesos)", procs is not None, f"{len(procs)} procesos" if procs else "FALTA")

if procs and len(procs) > 0:
    p = procs[0]
    print(f"\n  Campos por proceso (verificando el primero: PID={p.get('PID','?')}):")
    field_check("PID",          "PID"          in p, str(p.get("PID", "FALTA")))
    field_check("Name",         "Name"         in p, str(p.get("Name", "FALTA")))
    field_check("Cmdline",      "Cmdline"      in p, str(p.get("Cmdline", "FALTA"))[:50])
    field_check("vsz (KB)",     "vsz"          in p, str(p.get("vsz", "FALTA")) + " KB")
    field_check("rss (KB)",     "rss"          in p, str(p.get("rss", "FALTA")) + " KB")
    field_check("Memory_Usage", "Memory_Usage" in p, str(p.get("Memory_Usage", "FALTA")) + "%")
    field_check("CPU_Usage",    "CPU_Usage"    in p, str(p.get("CPU_Usage", "FALTA")) + "%")
PYEOF2

else
    echo -e "  ${YELLOW}⚠ SKIP${NC}  (python3 no disponible o archivo no existe)"
fi
echo ""

# ── Resumen final ─────────────────────────────────────────────
echo -e "${CYAN}=============================================${NC}"
echo -e "  Resultado: ${GREEN}$PASS PASS${NC}  |  ${RED}$FAIL FAIL${NC}"
if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}✓ Fase 1 completada exitosamente.${NC}"
else
    echo -e "  ${RED}✗ Hay $FAIL prueba(s) fallidas. Revisa los errores arriba.${NC}"
fi
echo -e "${CYAN}=============================================${NC}"
echo ""