#!/bin/bash
# ============================================================
#  unload_module.sh — Descarga el módulo de kernel de forma segura
#
#  Uso: sudo bash unload_module.sh
# ============================================================

MODULE_NAME="sysinfo_module"

echo "[KERNEL] ============================================="
echo "[KERNEL] Descargando módulo de kernel: $MODULE_NAME"

if lsmod | grep -q "^$MODULE_NAME"; then
    sudo rmmod "$MODULE_NAME"
    if [ $? -eq 0 ]; then
        echo "[KERNEL] ✓ Módulo descargado exitosamente."
        sudo dmesg | grep "SOPES1" | tail -2
    else
        echo "[KERNEL] ✗ Error al descargar el módulo."
        exit 1
    fi
else
    echo "[KERNEL] El módulo no estaba cargado."
fi

echo "[KERNEL] ============================================="
exit 0