# ============================================================
#  Makefile - Módulo de Kernel SOPES1 Proyecto 1
#
#  Uso:
#    make          → compila el módulo (genera sysinfo_module.ko)
#    make clean    → elimina los archivos generados
#    make load     → compila + carga el módulo
#    make unload   → descarga el módulo
#    make reload   → descarga, recompila y vuelve a cargar
#    make test     → muestra el contenido del archivo /proc
#    make log      → muestra los últimos mensajes del kernel (dmesg)
#
# ============================================================

# obj-m indica que construimos un Módulo (no integrado al kernel)
obj-m += sys_info_module.o

# KDIR apunta al directorio de headers del kernel actualmente en ejecución
KDIR := /lib/modules/$(shell uname -r)/build

# PWD es el directorio donde está este Makefile
PWD := $(shell pwd)

# Nombre del archivo en /proc (debe coincidir con el #define en el .c)
# *** CAMBIA TUCARNET POR TU CARNET ***
PROC_NAME := continfo_pr1_so1_202308204

# ── Compilar ──────────────────────────────────────────────────────────────────
all:
	@echo "[MAKE] Compilando módulo de kernel..."
	$(MAKE) -C $(KDIR) M=$(PWD) modules
	@echo "[MAKE] ✓ Compilación exitosa. Archivo generado: sys_info_module.ko"

# ── Limpiar archivos generados ────────────────────────────────────────────────
clean:
	@echo "[MAKE] Limpiando archivos de compilación..."
	$(MAKE) -C $(KDIR) M=$(PWD) clean
	@echo "[MAKE] ✓ Limpieza completada."

# ── Cargar el módulo ──────────────────────────────────────────────────────────
load: all
	@echo "[MAKE] Cargando módulo..."
	@if lsmod | grep -q sys_info_module; then \
		echo "[MAKE] El módulo ya está cargado. Descargando primero..."; \
		sudo rmmod sys_info_module; \
	fi
	sudo insmod sys_info_module.ko
	@echo "[MAKE] ✓ Módulo cargado."
	@echo "[MAKE] Verificando /proc/$(PROC_NAME)..."
	@ls -la /proc/$(PROC_NAME) 2>/dev/null && echo "[MAKE] ✓ Archivo /proc creado correctamente." || echo "[MAKE] ✗ Error: no se creó el archivo /proc."

# ── Descargar el módulo ───────────────────────────────────────────────────────
unload:
	@echo "[MAKE] Descargando módulo..."
	@if lsmod | grep -q sys_info_module; then \
		sudo rmmod sys_info_module; \
		echo "[MAKE] ✓ Módulo descargado."; \
	else \
		echo "[MAKE] El módulo no estaba cargado."; \
	fi

# ── Recargar (unload + compile + load) ───────────────────────────────────────
reload: unload all load
	@echo "[MAKE] ✓ Módulo recargado exitosamente."

# ── Probar: mostrar el JSON del /proc ─────────────────────────────────────────
test:
	@echo "[MAKE] Leyendo /proc/$(PROC_NAME)..."
	@if [ -f /proc/$(PROC_NAME) ]; then \
		cat /proc/$(PROC_NAME) | python3 -m json.tool 2>/dev/null || cat /proc/$(PROC_NAME); \
	else \
		echo "[MAKE] ✗ /proc/$(PROC_NAME) no existe. ¿Está cargado el módulo? Usa: make load"; \
	fi

# ── Ver logs del kernel ───────────────────────────────────────────────────────
log:
	@sudo dmesg | grep -i "SOPES1" | tail -20

# ── Información del estado actual ─────────────────────────────────────────────
status:
	@echo "=== Estado del módulo ==="
	@lsmod | grep sys_info_module && echo "Estado: CARGADO" || echo "Estado: DESCARGADO"
	@echo ""
	@echo "=== Archivo /proc ==="
	@ls -la /proc/$(PROC_NAME) 2>/dev/null || echo "No existe."

.PHONY: all clean load unload reload test log status