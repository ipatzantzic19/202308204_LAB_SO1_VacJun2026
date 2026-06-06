---
name: guia_aprendizaje
description: Guía de aprendizaje para el proyecto 1
---
# 📚 Guía de Aprendizaje — Proyecto 1 SOPES 1
**Lo que aprenderás al completar este proyecto explicado desde cero.**

---

## 🗺️ El Flujo Completo del Sistema (Visión General)

Antes de entrar en detalles, entender el flujo completo te ayudará a ver cómo todo encaja:

```
┌─────────────────────────────────────────────────────────────────┐
│                    TU MÁQUINA LINUX                             │
│                                                                 │
│  [KERNEL SPACE]                    [USER SPACE]                 │
│  ┌─────────────┐    /proc/         ┌─────────────────┐         │
│  │  Módulo C   │──────────────────>│   Daemon Go     │         │
│  │ (task_struct│  continfo_pr1...  │  (lee + analiza)│         │
│  │  /proc)     │                   └────────┬────────┘         │
│  └─────────────┘                            │                  │
│                                             │ docker stop/rm   │
│  [DOCKER CONTAINERS]◄────────────────────────┘                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐    │ guardar logs     │
│  │go-client │ │alpine CPU│ │alpine low│    ▼                  │
│  └──────────┘ └──────────┘ └──────────┘ ┌────────┐            │
│                                          │Valkey  │            │
│  [CRONJOB] cada 2min                     │(BD)    │            │
│  spawn_containers.sh ─────────────────>  └───┬────┘            │
│  (crea 5 contenedores random)                │                 │
│                                              ▼                 │
│                                          ┌────────┐            │
│                                          │Grafana │            │
│                                          │(viz)   │            │
│                                          └────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🧩 BLOQUE 1 — El Kernel de Linux y los Módulos

### ¿Qué es el Kernel?

El **kernel** es el núcleo del sistema operativo. Es la capa de software más baja que existe entre tu hardware y tus programas. Cuando escribes `ls` en la terminal, tu shell le pide al kernel que acceda al disco y liste archivos.

El kernel tiene acceso a **todo**: memoria, CPU, dispositivos, red. Los programas normales (como tu navegador) corren en **"modo usuario"** con permisos limitados. El kernel corre en **"modo kernel"** con acceso total.

### ¿Qué es un Módulo de Kernel?

Un **módulo de kernel** (también llamado LKM - Loadable Kernel Module) es código en C que puedes insertar dentro del kernel mientras el sistema está corriendo, **sin necesidad de reiniciarlo**. Es como un "plugin" para el kernel.

```bash
# Cargar un módulo
sudo insmod mi_modulo.ko

# Ver módulos cargados
lsmod

# Descargar un módulo
sudo rmmod mi_modulo
```

### Estructura mínima de un módulo

Todo módulo de kernel tiene obligatoriamente dos funciones:

```c
#include <linux/module.h>  // Base para cualquier módulo
#include <linux/kernel.h>  // printk() (el "printf del kernel")
#include <linux/init.h>    // Macros __init y __exit

// Se ejecuta cuando haces: sudo insmod modulo.ko
static int __init mi_modulo_init(void) {
    printk(KERN_INFO "¡Hola desde el kernel!\n");
    return 0;  // 0 = éxito
}

// Se ejecuta cuando haces: sudo rmmod modulo
static void __exit mi_modulo_exit(void) {
    printk(KERN_INFO "¡Adiós desde el kernel!\n");
}

// Registrar las funciones
module_init(mi_modulo_init);
module_exit(mi_modulo_exit);

MODULE_LICENSE("GPL");  // Obligatorio para acceder a símbolos del kernel
```

**¿Por qué `printk` y no `printf`?** Porque en el kernel no existe la biblioteca estándar de C (libc). `printk` escribe al log del kernel, que puedes ver con `dmesg`.

### El Makefile del módulo

El sistema de construcción del kernel usa Makefiles especiales:

```makefile
obj-m += sysinfo_module.o   # "obj-m" = objeto módulo (no integrado al kernel)

KDIR := /lib/modules/$(shell uname -r)/build  # Headers del kernel actual
PWD  := $(shell pwd)

all:
    $(MAKE) -C $(KDIR) M=$(PWD) modules
    # -C $(KDIR) = "entra al directorio del kernel"
    # M=$(PWD)   = "pero compila los módulos de este directorio"

clean:
    $(MAKE) -C $(KDIR) M=$(PWD) clean
```

---

## 🗂️ BLOQUE 2 — El Sistema de Archivos /proc

### ¿Qué es /proc?

`/proc` es un **sistema de archivos virtual** (no existe en disco). El kernel lo genera en memoria y lo expone como si fueran archivos normales. Es el canal de comunicación estándar entre el kernel y el espacio de usuario.

```bash
# Ejemplos de archivos /proc que ya existen en tu sistema
cat /proc/cpuinfo       # Info de tu CPU
cat /proc/meminfo       # Info de memoria RAM
cat /proc/uptime        # Tiempo encendido
cat /proc/1/status      # Información del proceso con PID 1
```

### ¿Cómo creamos nuestro propio archivo /proc?

En nuestro módulo usamos `proc_create()` y `seq_file`:

```c
#include <linux/proc_fs.h>   // proc_create()
#include <linux/seq_file.h>  // seq_printf(), single_open()

// Esta función es llamada cuando alguien hace "cat /proc/mi_archivo"
static int mostrar_datos(struct seq_file *m, void *v) {
    seq_printf(m, "Hola desde /proc!\n");
    seq_printf(m, "CPU cores: %d\n", num_online_cpus());
    return 0;
}

// "Puerta de entrada" al archivo
static int mi_open(struct inode *inode, struct file *file) {
    return single_open(file, mostrar_datos, NULL);
    //     ^ single_open maneja toda la complejidad de seq_file
}

// Tabla de operaciones del archivo (qué hacer en open, read, etc.)
static const struct proc_ops mi_fops = {
    .proc_open    = mi_open,
    .proc_read    = seq_read,
    .proc_lseek   = seq_lseek,
    .proc_release = single_release,
};

// En init del módulo:
proc_create("mi_archivo", 0444, NULL, &mi_fops);
// "mi_archivo" = nombre, 0444 = permisos (lectura pública), NULL = directorio raíz de /proc

// En exit del módulo:
remove_proc_entry("mi_archivo", NULL);
```

---

## 🔬 BLOQUE 3 — La estructura `task_struct`

### ¿Qué es task_struct?

`task_struct` es **la estructura más importante del kernel Linux**. Cada proceso que corre en tu sistema tiene exactamente una instancia de `task_struct` que almacena toda la información de ese proceso.

```c
// task_struct tiene CIENTOS de campos. Los más relevantes para nosotros:
struct task_struct {
    pid_t   pid;           // ID del proceso
    pid_t   tgid;          // ID del grupo de hilos (PID del proceso principal)
    char    comm[16];      // Nombre del proceso (primeros 15 chars + '\0')
    
    struct mm_struct *mm;  // Información de memoria (NULL si es kernel thread)
    
    u64     utime;         // Tiempo en modo usuario (en nanosegundos)
    u64     stime;         // Tiempo en modo kernel
    
    // ... muchos más campos
};
```

### ¿Cómo iteramos todos los procesos?

El kernel mantiene una lista enlazada circular de todos los procesos. Para iterar de forma segura:

```c
#include <linux/sched.h>        // task_struct, for_each_process
#include <linux/sched/signal.h> // for_each_process macro

struct task_struct *task;

// SIEMPRE usar rcu_read_lock/unlock para acceso seguro
rcu_read_lock();
for_each_process(task) {
    printk(KERN_INFO "PID: %d, Nombre: %s\n", task->pid, task->comm);
}
rcu_read_unlock();
```

**¿Por qué RCU (Read-Copy-Update)?** Es un mecanismo de sincronización del kernel. La lista de procesos puede cambiar mientras la estamos leyendo (procesos que nacen o mueren). RCU garantiza que podemos leer de forma segura sin bloquear al resto del sistema.

### ¿Cómo obtenemos la memoria de un proceso?

```c
#include <linux/mm.h>

if (task->mm) {  // task->mm == NULL significa que es un thread del kernel
    
    // VSZ: Virtual Size (tamaño total del espacio virtual en páginas)
    unsigned long vsz_pages = task->mm->total_vm;
    unsigned long vsz_kb = vsz_pages << (PAGE_SHIFT - 10);
    // PAGE_SHIFT = 12 en x86-64 (páginas de 4096 bytes = 2^12)
    // << (12 - 10) = << 2 = multiplicar por 4 (convierte páginas a KB)

    // RSS: Resident Set Size (páginas físicas actualmente en RAM)
    unsigned long rss_pages = get_mm_rss(task->mm);
    unsigned long rss_kb = rss_pages << (PAGE_SHIFT - 10);
}
```

### ¿Cómo obtenemos la memoria RAM total del sistema?

```c
#include <linux/mm.h>

struct sysinfo si;
si_meminfo(&si);

// si.totalram = páginas totales de RAM
// si.freeram  = páginas libres de RAM
// si.mem_unit = tamaño de unidad (generalmente 1 o PAGE_SIZE)

unsigned long total_mb = (si.totalram * si.mem_unit) >> 20;
// >> 20 convierte bytes a megabytes (2^20 = 1,048,576)
```

---

## 🐹 BLOQUE 4 — El Daemon en Go

### ¿Qué es un Daemon?

Un **daemon** (también escrito "demonio") es un programa que corre en **segundo plano**, sin interfaz de usuario, de manera continua. Por ejemplo, `sshd` (servidor SSH), `cron` (programador de tareas) son daemons.

Nuestro daemon en Go es el cerebro del proyecto: lee datos, toma decisiones, y reporta resultados.

### ¿Por qué Go para el daemon?

- **Concurrencia nativa**: Go tiene goroutines, muy útiles para tareas paralelas.
- **Compilación estática**: Genera un binario único sin dependencias.
- **Manejo explícito de errores**: Obliga a manejar errores, haciendo el código más robusto.
- **Velocidad**: Casi tan rápido como C para operaciones del sistema.

### Conceptos clave de Go que usamos

#### Goroutines y channels
```go
// Una goroutine es como un hilo de ejecución muy liviano
go func() {
    // Este código corre concurrentemente
    fmt.Println("Corriendo en paralelo")
}()

// Un channel es un canal de comunicación entre goroutines
ch := make(chan int)
go func() { ch <- 42 }()     // enviar
valor := <-ch                  // recibir
```

#### Manejo de señales del SO
```go
// Para que el daemon pueda "apagarse limpiamente" al recibir Ctrl+C
import (
    "os/signal"
    "syscall"
)

sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

select {
case sig := <-sigChan:
    fmt.Printf("Señal recibida: %v\n", sig)
    // Hacer limpieza antes de salir
}
```

#### Ejecutar comandos del sistema
```go
import "os/exec"

// Equivalente a correr en la terminal: docker stop abc123
out, err := exec.Command("docker", "stop", "abc123").CombinedOutput()
if err != nil {
    log.Printf("Error: %v\nSalida: %s", err, out)
}
```

#### Parsear JSON (deserialización)
```go
import "encoding/json"

// Si el módulo retorna este JSON:
// {"memoria": {"total_mb": 8192}, "procesos": [...]}

type MemInfo struct {
    TotalMB uint64 `json:"total_mb"`
}
type Snapshot struct {
    Memoria MemInfo `json:"memoria"`
}

data, _ := os.ReadFile("/proc/mi_archivo")
var snapshot Snapshot
json.Unmarshal(data, &snapshot)
// Ahora snapshot.Memoria.TotalMB tiene el valor
```

### El Loop Principal del Daemon

```go
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        // Esto se ejecuta cada 30 segundos
        ejecutarCiclo()
    case <-sigChan:
        // Esto se ejecuta cuando el usuario hace Ctrl+C
        limpiarYSalir()
        return
    }
}
```

---

## 🐳 BLOQUE 5 — Docker y Contenedores

### ¿Qué es un contenedor Docker?

Un **contenedor** es un proceso aislado que cree que es la única cosa corriendo en su propio sistema. Tiene su propio filesystem, red, y vista de procesos. Sin embargo, comparte el kernel del host.

Es como un apartamento en un edificio: cada familia (contenedor) tiene su propio espacio privado, pero comparten la misma estructura (kernel) y servicios básicos (red del host).

### Diferencia entre imagen y contenedor

- **Imagen**: La plantilla (receta). Ej: `alpine:latest`, `ubuntu:22.04`. Es de solo lectura.
- **Contenedor**: Una instancia corriendo de una imagen. Puede haber múltiples contenedores del mismo image.

```bash
docker run -d --name mi_contenedor alpine sleep 300
#           ^ en background  ^ nombre   ^ imagen ^ comando
```

### Docker Compose

`docker-compose.yml` define múltiples servicios y sus relaciones:

```yaml
services:
  base_de_datos:
    image: valkey/valkey:latest
    ports:
      - "6379:6379"        # puerto_host:puerto_contenedor
    networks:
      - mi_red

  aplicacion:
    image: grafana/grafana
    ports:
      - "3000:3000"
    depends_on:
      - base_de_datos      # Arranca después de base_de_datos
    networks:
      - mi_red             # Comparte la misma red → pueden comunicarse

networks:
  mi_red:
    driver: bridge
```

Con Docker Compose, los contenedores de la misma red pueden comunicarse usando el nombre del servicio como hostname. Por ejemplo, Grafana puede conectarse a Valkey usando `valkey:6379`.

---

## 🔴 BLOQUE 6 — Valkey (Base de Datos Clave-Valor)

### ¿Qué es Valkey?

**Valkey** es un fork open-source de Redis mantenido por la Linux Foundation. Es una base de datos **en memoria** del tipo **clave-valor** (key-value store). Extremadamente rápida porque todo vive en RAM.

Nuestro daemon la usa para guardar métricas y logs que Grafana leerá.

### Conceptos básicos de Valkey/Redis

```bash
# Conectarse al CLI
docker exec -it valkey_sopes1 valkey-cli

# Tipos de datos fundamentales:

# STRING: guarda cualquier texto
SET nombre "Carlos"
GET nombre   # → "Carlos"
SET contador 0
INCR contador   # → 1 (incremento atómico)

# LISTA: como un array, perfecto para historial
LPUSH historial "evento1"   # insertar al inicio
LPUSH historial "evento2"
LRANGE historial 0 -1       # obtener todos: ["evento2", "evento1"]
LLEN historial               # → 2

# SET ORDENADO (Sorted Set): como ranking
ZADD ranking:ram 8192 "contenedor_abc"  # score=8192, member="contenedor_abc"
ZADD ranking:ram 4096 "contenedor_xyz"
ZREVRANGE ranking:ram 0 4               # Top 5 (mayor a menor)
ZREVRANGE ranking:ram 0 4 WITHSCORES    # Con los scores

# Expiración automática
SET memoria:actual "{...}" EX 3600      # Expira en 1 hora
```

### Usando Valkey desde Go

```go
import "github.com/redis/go-redis/v9"

// Conectar
rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})
ctx := context.Background()

// Guardar string
rdb.Set(ctx, "clave", "valor", time.Hour)

// Leer string
val, err := rdb.Get(ctx, "clave").Result()

// Insertar en lista
rdb.LPush(ctx, "mi_lista", "nuevo_elemento")
rdb.LTrim(ctx, "mi_lista", 0, 99)  // Mantener máximo 100 elementos

// Agregar a sorted set
rdb.ZAdd(ctx, "ranking", redis.Z{Score: 100.5, Member: "id_contenedor"})
```

---

## 📊 BLOQUE 7 — Grafana (Visualización)

### ¿Qué es Grafana?

**Grafana** es una plataforma de visualización de métricas y logs. Se conecta a fuentes de datos (Valkey, InfluxDB, Prometheus, etc.) y crea dashboards con gráficas en tiempo real.

### Conceptos clave de Grafana

**Dashboard**: Página con múltiples paneles (gráficas, indicadores, etc.).

**Panel**: Una visualización individual. Puede ser:
- **Stat/Card**: Un número grande (útil para RAM total, contenedores activos).
- **Time Series**: Gráfica de línea en el tiempo (evolución de RAM).
- **Bar Chart**: Gráfica de barras.
- **Pie Chart**: Gráfica de pastel (top contenedores por consumo).

**Query**: La consulta que le haces a tu fuente de datos para obtener los datos del panel.

### Flujo de datos en Grafana

```
Valkey ──────────→ Grafana DataSource Plugin ──────→ Panel
(almacena métricas)  (le pregunta a Valkey)      (muestra visual)
```

---

## 🔄 BLOQUE 8 — Cron (Programador de Tareas)

### ¿Qué es Cron?

`cron` es el programador de tareas del sistema Linux. Permite ejecutar comandos o scripts automáticamente en horarios definidos.

### Sintaxis de crontab

```
# ┌───────────── minutos (0-59)
# │ ┌───────────── horas (0-23)
# │ │ ┌───────────── día del mes (1-31)
# │ │ │ ┌───────────── mes (1-12)
# │ │ │ │ ┌───────────── día de la semana (0-7, 0 y 7 = domingo)
# │ │ │ │ │
# * * * * * comando a ejecutar

*/2 * * * * /ruta/script.sh    # Cada 2 minutos
0 */1 * * * /script.sh         # Cada hora
0 8 * * 1-5 /script.sh         # Lunes a viernes a las 8am
```

### Gestionar crontab

```bash
# Ver el crontab actual del usuario
crontab -l

# Editar el crontab
crontab -e

# El daemon Go lo gestiona programáticamente:
# 1. Obtiene el crontab actual: crontab -l
# 2. Agrega la nueva línea
# 3. Escribe de vuelta: crontab -  (lee desde stdin)
```

---

## 🎓 BLOQUE 9 — Conceptos de Sistemas Operativos que aprenderás

Al completar este proyecto, habrás aprendido en la práctica:

### Espacio de Kernel vs Espacio de Usuario

| Kernel Space | User Space |
|---|---|
| Código que corre con privilegios totales | Código con acceso limitado |
| Acceso directo al hardware | Debe pedir al kernel via syscalls |
| Nuestro módulo C | Nuestro daemon Go |
| Si falla → kernel panic (BSOD de Linux) | Si falla → el programa muere, el OS sobrevive |

### Syscalls (Llamadas al Sistema)

Cuando un programa en Go hace `os.ReadFile(...)`, internamente hace una **syscall** llamada `read()`. El kernel ejecuta el código real de leer el archivo y devuelve los datos al programa. Es el único mecanismo legítimo para que programas de usuario pidan servicios al kernel.

### Estructuras de Procesos

Ahora sabes que cada proceso en Linux tiene:
- **PID**: Identificador único.
- **task_struct**: La estructura que el kernel mantiene con toda la info del proceso.
- **mm_struct**: La información de memoria virtual del proceso.
- **Espacio de direcciones virtuales**: Cada proceso cree que tiene toda la memoria para él solo. El kernel traduce estas direcciones virtuales a físicas.

### Memoria Virtual vs Física

- **VSZ (Virtual Size)**: Cuánta memoria virtual tiene mapeada el proceso (puede ser mucho más de lo que realmente usa).
- **RSS (Resident Set Size)**: Las páginas que realmente están en RAM física ahora mismo.
- Un proceso puede tener VSZ = 1 GB pero RSS = 50 MB (el resto está en disco o no se ha accedido aún).

### Telemetría de Sistema

Has aprendido el patrón completo de **observabilidad**:
1. **Recolección** (módulo kernel → /proc).
2. **Procesamiento** (daemon Go).
3. **Almacenamiento** (Valkey).
4. **Visualización** (Grafana).

Este es exactamente el patrón que usan herramientas profesionales como Prometheus + Grafana o el stack ELK (Elasticsearch, Logstash, Kibana).

---

## ❓ Preguntas Frecuentes (FAQ)

### "¿Por qué el módulo tarda en compilar?"
El sistema de build del kernel es complejo. Es normal que tome 1-2 minutos la primera vez.

### "dmesg muestra 'tainted kernel', ¿es malo?"
No para este proyecto. Significa que el kernel tiene un módulo de terceros cargado. Es esperado.

### "El porcentaje de CPU da números enormes (>100%)"
El enunciado lo permite explícitamente. Los valores de `utime` y `stime` son contadores acumulados en nanosegundos, no porcentajes directos. Para un porcentaje real se necesitan dos mediciones con diferencia de tiempo.

### "¿Cómo sé si un proceso pertenece a un contenedor Docker?"
Los contenedores Docker crean sus procesos directamente en el sistema. Puedes identificarlos buscando procesos cuyo nombre sea el binario que corriste. En el módulo, podrías filtrar por cgroup o por el cmdline que incluye el ID del contenedor. Para este proyecto, filtrar por los nombres de imagen es suficiente.

### "¿Es Valkey compatible con comandos Redis?"
Sí, Valkey es binariamente compatible con Redis. Los clientes Go de Redis funcionan perfectamente con Valkey.

### "¿Por qué usar /proc y no sockets o pipes?"
`/proc` es el mecanismo estándar y más simple para exponer datos del kernel al espacio de usuario de forma de solo lectura. No requiere manejar conexiones ni protocolos complejos.

---

## 📖 Recursos para Profundizar

- **Módulos de kernel**: `Linux Device Drivers, 3rd Edition` (gratuito en línea).
- **Estructura task_struct**: `https://elixir.bootlin.com/linux/latest/source/include/linux/sched.h`
- **Go concurrencia**: Tour oficial de Go: `tour.golang.org`.
- **Valkey docs**: `valkey.io/docs`.
- **Grafana docs**: `grafana.com/docs/grafana`.
- **Repositorio del curso**: `github.com/CamiloSincal/EJEMPLOS_SOPES1_VACJUN2026`.

---

## 🏁 ¿Qué deberías saber al terminar?

Al entregar este proyecto deberías poder explicar:

1. **¿Qué hace `module_init` y `module_exit`?** → Funciones que el kernel llama al cargar/descargar el módulo.
2. **¿Qué es `task_struct`?** → La estructura que el kernel mantiene por cada proceso, con su PID, nombre, memoria, etc.
3. **¿Por qué usamos `rcu_read_lock`?** → Para leer de forma segura la lista de procesos que puede cambiar concurrentemente.
4. **¿Cómo funciona `/proc`?** → Sistema de archivos virtual que el kernel expone para comunicarse con el espacio de usuario.
5. **¿Qué hace el daemon cada 30 segundos?** → Lee /proc, parsea JSON, gestiona contenedores, guarda en Valkey.
6. **¿Por qué 3 bajos + 2 altos?** → Es la política de recursos definida: siempre debe haber ese mínimo de cada tipo.
7. **¿Cómo Grafana obtiene los datos?** → Se conecta a Valkey como datasource y hace queries a las claves donde el daemon guardó las métricas.