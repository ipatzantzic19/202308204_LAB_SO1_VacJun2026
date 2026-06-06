# Fase 1 — Módulo de Kernel
**Proyecto 1 · Sistemas Operativos 1 · USAC Vacaciones Junio 2026**

---

## Archivos de esta fase

```
kernel_module/
├── sysinfo_module.c   ← Código fuente del módulo (lo que entregas)
├── Makefile           ← Sistema de compilación (lo que entregas)
├── load_module.sh     ← Script para cargar el módulo
├── unload_module.sh   ← Script para descargar el módulo
└── test_module.sh     ← Script de pruebas automáticas
```

---

## ⚠️ Antes de hacer cualquier cosa: cambia tu carnet

Abre `sysinfo_module.c` y busca la línea:
```c
#define PROC_NAME  "continfo_pr1_so1_TUCARNET"
```
Reemplaza `TUCARNET` con tu número de carnet. Ejemplo:
```c
#define PROC_NAME  "continfo_pr1_so1_202312345"
```

Haz lo mismo en `Makefile` (línea `PROC_NAME :=`) y en `test_module.sh` (línea `PROC_NAME=`).

---

## Instalación de dependencias (una sola vez)

```bash
sudo apt update
sudo apt install -y linux-headers-$(uname -r) build-essential gcc make
```

Si falla por headers faltantes (error durante `make`), ejecuta esto:
```bash
cd /usr/src/linux-headers-$(uname -r)
sudo cp /boot/config-$(uname -r) .config
sudo make oldconfig
sudo make prepare
sudo make modules_prepare
```

---

## Compilar y cargar el módulo

```bash
# Opción A: Todo en un comando
make load

# Opción B: Paso a paso
make                         # compila → genera sysinfo_module.ko
sudo insmod sysinfo_module.ko  # carga en el kernel

# Ver que se cargó correctamente
lsmod | grep sysinfo_module
sudo dmesg | grep SOPES1
```

---

## Verificar que funciona

```bash
# Prueba directa (reemplaza TUCARNET)
cat /proc/continfo_pr1_so1_TUCARNET

# Prueba con formato bonito
cat /proc/continfo_pr1_so1_TUCARNET | python3 -m json.tool

# Suite de pruebas automáticas
bash test_module.sh
```

**Salida esperada (fragmento):**
```json
{
  "Totalram": 8192000,
  "Freeram": 4096000,
  "Usedram": 4096000,
  "Procs": 253,
  "Processes": [
    {
      "PID": 1,
      "Name": "systemd",
      "Cmdline": "/sbin/init splash",
      "vsz": 102400,
      "rss": 8192,
      "Memory_Usage": 0.1,
      "CPU_Usage": 0.00
    },
    ...
  ]
}
```

---

## Comandos del Makefile

| Comando | Qué hace |
|---|---|
| `make` | Compila el módulo |
| `make clean` | Elimina archivos compilados |
| `make load` | Compila + carga el módulo |
| `make unload` | Descarga el módulo |
| `make reload` | Descarga + recompila + carga |
| `make test` | Muestra el JSON formateado |
| `make log` | Muestra mensajes del kernel (dmesg) |
| `make status` | Estado actual del módulo |

---

## Errores comunes y soluciones

**`make: *** No rule to make target`**
→ Falta el nombre del objeto. Verifica que el Makefile diga `obj-m += sysinfo_module.o`.

**`insmod: ERROR: could not insert module`**
→ Ejecuta `sudo dmesg | tail -5` para ver el error exacto.
→ Si dice "Invalid module format": recompila con `make clean && make`.

**`cat: /proc/...: No such file or directory`**
→ El módulo no está cargado. Ejecuta `sudo insmod sysinfo_module.ko`.

**`Killed` al hacer `cat /proc/...`**
→ El módulo tiene un error que produce kernel panic. Revisa que `rcu_read_lock()` esté antes de `for_each_process`.

**JSON inválido (comillas rotas)**
→ La función `sanitize_for_json` maneja esto. Si sigue pasando, verifica que el módulo compilado sea el más reciente (`make reload`).

---

## Cómo se conecta con el resto del proyecto

```
sysinfo_module.ko
      │
      │ (cargado en el kernel)
      │
      ▼
/proc/continfo_pr1_so1_TUCARNET   ← archivo que el Daemon Go lee cada 30s
      │
      │ (cat + json.Unmarshal)
      ▼
   Daemon Go  →  Valkey  →  Grafana
```

El Daemon (Fase 4) leerá este archivo con `os.ReadFile()` y lo parseará como JSON para tomar decisiones sobre los contenedores Docker.