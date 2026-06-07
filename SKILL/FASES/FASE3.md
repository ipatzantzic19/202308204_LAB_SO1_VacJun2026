# Fase 3 — Script del Cronjob
**Proyecto 1 · Sistemas Operativos 1 · USAC Vacaciones Junio 2026**

---

## Archivos de esta fase

```
scripts/
├── spawn_containers.sh   ← Script principal (lo que entrega el proyecto)
└── test_spawn.sh         ← Suite de pruebas automáticas
```

---

## ¿Qué hace este script?

`spawn_containers.sh` es ejecutado automáticamente **cada 2 minutos** por el cronjob que registra el Daemon de Go. Cada vez que corre:

1. Itera 5 veces.
2. En cada iteración, elige **aleatoriamente** un tipo de contenedor:
   - **ALTO_RAM** → `roldyoran/go-client`
   - **ALTO_CPU** → `alpine` con bucle de cálculos matemáticos
   - **BAJO**     → `alpine sleep 240`
3. Crea el contenedor con un nombre único basado en timestamp.
4. Registra todo en `/tmp/spawn_containers.log`.

---

## Instalación y prueba

```bash
cd ~/Documentos/Github/202308204_LAB_SO1_VacJun2026/scripts/

# Dar permisos de ejecución
chmod +x spawn_containers.sh

# Prueba manual completa (crea 5 contenedores y los verifica)
bash test_spawn.sh

# Prueba directa del script (sin limpieza automática)
bash spawn_containers.sh
docker ps --filter "name=sopes1_"
```

---

## ¿Por qué el Daemon registra el cronjob y no lo hacemos manualmente?

El enunciado del proyecto dice explícitamente:

> *"El daemon de Go iniciará la implementación y ejecución del cronjob en el sistema operativo"*

Y el ejemplo del curso (`Clase 4/cronjob/main.go`) muestra exactamente cómo hacerlo en Go:

```go
// Patrón del curso para registrar un cronjob desde Go
func agregarCronJob(rutaScript string) {
    expresionCron := "*/2 * * * *"   // Cada 2 minutos (según enunciado)
    comandoCron := fmt.Sprintf("%s %s >> %s.log 2>&1", expresionCron, rutaScript, rutaScript)
    cmd := exec.Command("bash", "-c",
        fmt.Sprintf("(crontab -l 2>/dev/null; echo \"%s\") | crontab -", comandoCron))
    cmd.Run()
}
```

El Daemon también lo **elimina** al apagarse:

```go
// Al recibir SIGTERM/SIGINT el daemon limpia el cronjob
func eliminarCronJob(rutaScript string) {
    cmd := exec.Command("bash", "-c",
        fmt.Sprintf("crontab -l 2>/dev/null | grep -v '%s' | crontab -", rutaScript))
    cmd.Run()
}
```

---

## Nomenclatura de los contenedores

Todos los contenedores creados por el script siguen el patrón:

```
sopes1_{timestamp_nanosegundos}_{índice}
```

Ejemplos:
```
sopes1_1749240000123456789_1
sopes1_1749240000456789012_2
sopes1_1749240000789012345_3
```

El Daemon de Go filtra los contenedores que pertenecen al proyecto usando el prefijo `sopes1_` en `docker ps`.

---

## Ver el log del cronjob

```bash
# Ver el log completo
cat /tmp/spawn_containers.log

# Seguir el log en tiempo real (cuando el cronjob está activo)
tail -f /tmp/spawn_containers.log

# Ver solo los últimos eventos
tail -20 /tmp/spawn_containers.log
```

---

## Conexión con las otras fases

```
Fase 4 (Daemon Go)
  └── al iniciar  → registra este script en crontab (cada 2 min)
  └── en el loop  → lee contenedores activos → gestiona cuáles eliminar
  └── al apagar   → elimina el cronjob del crontab

Este script
  └── crea 5 contenedores aleatorios cada 2 minutos
  └── el Daemon los detecta leyendo /proc y docker ps
  └── el Daemon decide cuáles matar según las reglas:
        - mantener 3 bajos activos
        - mantener 2 altos activos
        - eliminar el resto
```