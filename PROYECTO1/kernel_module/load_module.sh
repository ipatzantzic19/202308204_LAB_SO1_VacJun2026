#!/bin/bash
# ============================================================
#  load_module.sh — Carga el módulo de kernel de forma segura
#
#  Uso: sudo bash load_module.sh
#       (o desde el daemon Go que lo llama automáticamente)
# ============================================================

# ── Configuración ─────────────────────────────────────────────
# Ruta absoluta al directorio donde está el módulo compilado
# Ajusta esta ruta si tu proyecto está en otro lugar
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_PATH="$SCRIPT_DIR/sysinfo_module.ko"
MODULE_NAME="sysinfo_module"

# ─────────────────────────────────────────────────────────────
echo "[KERNEL] ============================================="
echo "[KERNEL] Cargando módulo de kernel SOPES1..."
echo "[KERNEL] Módulo: $MODULE_PATH"

# ── Verificar que el archivo .ko existe ───────────────────────
if [ ! -f "$MODULE_PATH" ]; then
    echo "[KERNEL] ✗ Error: No se encontró '$MODULE_PATH'"
    echo "[KERNEL]   Compila primero con: cd $(dirname $MODULE_PATH) && make"
    exit 1
fi

# ── Verificar si ya está cargado ──────────────────────────────
if lsmod | grep -q "^$MODULE_NAME"; then
    echo "[KERNEL] El módulo ya está cargado. No se hace nada."
    echo "[KERNEL] ============================================="
    exit 0
fi

# ── Cargar el módulo ──────────────────────────────────────────
echo "[KERNEL] Ejecutando: sudo insmod $MODULE_PATH"
sudo insmod "$MODULE_PATH"
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo "[KERNEL] ✓ Módulo cargado exitosamente."
    echo "[KERNEL] Últimos mensajes del kernel:"
    sudo dmesg | grep "SOPES1" | tail -3
else
    echo "[KERNEL] ✗ Error al cargar el módulo (código: $EXIT_CODE)"
    echo "[KERNEL] Revisa los logs con: sudo dmesg | tail -20"
    exit $EXIT_CODE
fi

echo "[KERNEL] ============================================="
exit 0