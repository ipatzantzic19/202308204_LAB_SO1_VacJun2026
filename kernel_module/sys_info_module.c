/*
 * ============================================================
 *  Módulo de Kernel: Sonda de Telemetría de Contenedores
 *
 *  Expone en /proc/continfo_pr1_so1_202308204 un JSON con:
 *    - Métricas de RAM (total, libre, usada) en KB
 *    - Lista de todos los procesos del sistema con:
 *        PID, Name, Cmdline, VSZ, RSS, %Memoria, %CPU
 * ============================================================
 *
 *  Basado en el patrón enseñado en:
 *  - Clase 3/MetricasSO  → si_meminfo, seq_file, proc_fs
 *  - Clase 3/MetricasSO2 → task_struct, for_each_process,
 *                           get_process_cmdline, jiffies
 */
/* ============================================================
   HEADERS REQUERIDOS
   (igual que en el ejemplo MetricasSO2 del curso)
   ============================================================ */
#include <linux/module.h>       /* module_init, module_exit, MODULE_* */
#include <linux/kernel.h>       /* printk, KERN_INFO */
#include <linux/string.h>       /* memset */
#include <linux/init.h>         /* __init, __exit */
#include <linux/proc_fs.h>      /* proc_create, remove_proc_entry */
#include <linux/seq_file.h>     /* seq_file, seq_printf, single_open */
#include <linux/mm.h>           /* si_meminfo, get_mm_rss, PAGE_SHIFT */
#include <linux/sched.h>        /* task_struct, for_each_process */
#include <linux/timer.h>        /* (incluido para compatibilidad) */
#include <linux/jiffies.h>      /* jiffies, num_online_cpus */
#include <linux/uaccess.h>      /* access_process_vm */
#include <linux/tty.h>          /* (incluido para compatibilidad) */
#include <linux/sched/signal.h> /* for_each_process macro */
#include <linux/fs.h>           /* inode, file */
#include <linux/slab.h>         /* kmalloc, kfree */
#include <linux/sched/mm.h>     /* get_task_mm, mmput */
#include <linux/binfmts.h>      /* (incluido para compatibilidad) */
#include <linux/timekeeping.h>  /* (incluido para compatibilidad) */

/* ============================================================
   METADATOS DEL MÓDULO
   ============================================================ */
MODULE_LICENSE("GPL");
MODULE_AUTHOR("Isai Patzan - 202308204");
MODULE_DESCRIPTION("Modulo kernel para telemetria de contenedores - SOPES1 P1");
MODULE_VERSION("1.0");

#define PROC_NAME        "continfo_pr1_so1_202308204"
#define MAX_CMDLINE_LEN  256

/* ============================================================
   FUNCIÓN: Obtener línea de comandos de un proceso
   (Copiada directamente del ejemplo MetricasSO2 del curso)

   - task:     el proceso cuyo cmdline queremos leer
   - retorna:  string con el cmdline (debe liberarse con kfree)
               o NULL si no fue posible obtenerlo
   ============================================================ */
static char *get_process_cmdline(struct task_struct *task)
{
    struct mm_struct *mm;                           // Estructura que representa el espacio de memoria de un proceso
    char *cmdline;                                  // Buffer para almacenar el cmdline (se reserva dinámicamente)
    unsigned long arg_start = 0, arg_end = 0;       // Límites del cmdline en el espacio de usuario
    int len = 0, i;                                 // Longitud del cmdline y variable de iteración

    /* Reservar memoria en el heap del kernel */
    cmdline = kmalloc(MAX_CMDLINE_LEN, GFP_KERNEL);
    if (!cmdline)
        return NULL;

    /* get_task_mm obtiene el mm_struct del proceso de forma segura
       (incrementa el contador de referencias) */
    mm = get_task_mm(task);
    if (!mm) {
        /* Proceso de kernel (no tiene espacio de usuario) */
        kfree(cmdline);
        return NULL;
    }

    /* Leer arg_start y arg_end con el lock del mmap (kernel >= 6.8) */
    down_read(&mm->mmap_lock);
    arg_start = mm->arg_start;
    arg_end   = mm->arg_end;
    up_read(&mm->mmap_lock);

    /* Calcular longitud del cmdline */
    if (arg_end > arg_start)
        len = arg_end - arg_start;
    else
        len = 0;

    if (len > MAX_CMDLINE_LEN - 1)
        len = MAX_CMDLINE_LEN - 1;

    if (len > 0) {
        /* Leer la memoria del proceso desde el espacio de usuario */
        if (access_process_vm(task, arg_start, cmdline, len, 0) != len) {
            mmput(mm);
            kfree(cmdline);
            return NULL;
        }
    } else {
        cmdline[0] = '\0';
    }

    cmdline[len] = '\0';

    /* Los argumentos están separados por '\0', los convertimos a espacios */
    for (i = 0; i < len; i++)
        if (cmdline[i] == '\0')
            cmdline[i] = ' ';

    /* Eliminar espacios al final */
    while (len > 0 && cmdline[len - 1] == ' ')
        cmdline[--len] = '\0';

    /* Liberar la referencia al mm_struct */
    mmput(mm);
    return cmdline;
}

/* ============================================================
   FUNCIÓN: Sanitizar strings para JSON
   Reemplaza comillas dobles y backslashes que romperían el JSON
   ============================================================ */
static void sanitize_for_json(char *str, int maxlen)
{
    int i;
    for (i = 0; i < maxlen && str[i] != '\0'; i++) {
        if (str[i] == '"' || str[i] == '\\')
            str[i] = '\'';
        /* Eliminar caracteres de control que rompen JSON */
        if (str[i] < 0x20)
            str[i] = ' ';
    }
}

/* ============================================================
   FUNCIÓN PRINCIPAL: Generar el JSON en /proc
   (Patrón idéntico al MetricasSO2 del curso)

   Esta función se ejecuta cada vez que alguien hace:
       cat /proc/continfo_pr1_so1_202308204

   Genera un JSON con esta estructura:
   {
     "Totalram": 8192000,     <- en KB
     "Freeram":  4096000,     <- en KB
     "Usedram":  4096000,     <- en KB
     "Procs":    250,
     "Processes": [
       {
         "PID": 1,
         "Name": "systemd",
         "Cmdline": "/sbin/init",
         "vsz": 102400,
         "rss": 8192,
         "Memory_Usage": 0.1,
         "CPU_Usage": 0.00
       },
       ...
     ]
   }
   ============================================================ */
static int sysinfo_show(struct seq_file *m, void *v)
{
    struct sysinfo   si;                                    // Estructura para almacenar info de memoria del sistema
    struct task_struct *task;                               // Puntero para iterar sobre los procesos del sistema
    unsigned long    total_jiffies;                         // Contador de ticks del sistema desde el arranque (para calcular %CPU)
    unsigned long    totalram_kb, freeram_kb, usedram_kb;   // Métricas de memoria en KB
    int              first_process = 1;                     // Flag para controlar la coma entre objetos JSON
    int              process_count = 0;                     // Contador de procesos totales (para la métrica "Procs")

    /* ── PASO 1: Obtener info de memoria ─────────────────────────
       si_meminfo llena la estructura sysinfo con datos del kernel
       PAGE_SHIFT = 12 en x86-64, así:
         << (PAGE_SHIFT - 10) = << 2 convierte páginas → KB
    ────────────────────────────────────────────────────────────── */
    si_meminfo(&si);
    totalram_kb = si.totalram << (PAGE_SHIFT - 10);
    freeram_kb  = si.freeram  << (PAGE_SHIFT - 10);
    usedram_kb  = totalram_kb - freeram_kb;

    /* jiffies: contador de "ticks" del sistema desde que arrancó */
    total_jiffies = jiffies;

    /* ── PASO 2: Contar procesos (primer pasada) ──────────────── */
    for_each_process(task) {
        process_count++;
    }

    /* ── PASO 3: Escribir el JSON en el archivo /proc ──────────── */

    /* Apertura del objeto JSON raíz */
    seq_printf(m, "{\n");

    /* Métricas de memoria */
    seq_printf(m, "  \"Totalram\": %lu,\n", totalram_kb);
    seq_printf(m, "  \"Freeram\": %lu,\n",  freeram_kb);
    seq_printf(m, "  \"Usedram\": %lu,\n",  usedram_kb);
    seq_printf(m, "  \"Procs\": %d,\n",     process_count);

    /* Inicio del array de procesos */
    seq_printf(m, "  \"Processes\": [\n");

    /* ── PASO 4: Iterar todos los procesos con task_struct ───────
       rcu_read_lock/unlock protege el acceso concurrente a la
       lista de procesos (Read-Copy-Update, igual que en MetricasSO2)
    ────────────────────────────────────────────────────────────── */
    rcu_read_lock();
    for_each_process(task) {
        unsigned long vsz        = 0;
        unsigned long rss        = 0;
        unsigned long mem_usage  = 0; /* ×10 para tener 1 decimal */
        unsigned long cpu_usage  = 0; /* ×100 para tener 2 decimales */
        unsigned long total_time = 0;
        char comm_safe[TASK_COMM_LEN + 1];
        char *cmdline = NULL;

        /* ── Obtener VSZ y RSS ─────────────────────────────────
           task->mm == NULL → thread del kernel (sin user space)
        ──────────────────────────────────────────────────────── */
        if (task->mm) {
            /* VSZ: tamaño del espacio virtual en KB */
            vsz = task->mm->total_vm << (PAGE_SHIFT - 10);

            /* RSS: páginas físicas en uso en KB */
            rss = get_mm_rss(task->mm) << (PAGE_SHIFT - 10);

            /* %Memoria = (RSS / totalram) × 100
               Multiplicamos ×1000 para conservar 1 decimal:
               15 → "1.5%", 100 → "10.0%"                 */
            if (totalram_kb > 0)
                mem_usage = (rss * 1000) / totalram_kb;
        }

        /* ── Calcular %CPU (igual que MetricasSO2) ────────────
           utime + stime = tiempo acumulado del proceso en ticks
           Multiplicamos ×10000 para tener 2 decimales,
           luego dividimos por el número de CPUs online.

           NOTA: el valor puede ser grande en sistemas que llevan
           mucho tiempo corriendo. Esto es normal según el enunciado.
        ──────────────────────────────────────────────────────── */
        total_time = task->utime + task->stime;
        if (total_jiffies > 0) {
            cpu_usage = (total_time * 10000) / total_jiffies;
            cpu_usage = cpu_usage / num_online_cpus();
        }

        /* ── Obtener nombre del proceso de forma segura ────────
           task->comm puede cambiar concurrentemente; usamos
           una copia local.
        ──────────────────────────────────────────────────────── */
        memset(comm_safe, 0, sizeof(comm_safe));
        get_task_comm(comm_safe, task);
        sanitize_for_json(comm_safe, sizeof(comm_safe));

        /* ── Obtener cmdline ──────────────────────────────────
           get_process_cmdline puede retornar NULL para threads
           del kernel o procesos sin args accesibles.
        ──────────────────────────────────────────────────────── */
        cmdline = get_process_cmdline(task);
        if (cmdline)
            sanitize_for_json(cmdline, MAX_CMDLINE_LEN);

        /* ── Escribir el objeto proceso en el JSON ────────────
           Separador de coma entre objetos (no al final)
        ──────────────────────────────────────────────────────── */
        if (!first_process)
            seq_printf(m, ",\n");
        else
            first_process = 0;

        seq_printf(m, "    {\n");
        seq_printf(m, "      \"PID\": %d,\n",          task->pid);
        seq_printf(m, "      \"Name\": \"%s\",\n",     comm_safe);
        seq_printf(m, "      \"Cmdline\": \"%s\",\n",  cmdline ? cmdline : "N/A");
        seq_printf(m, "      \"vsz\": %lu,\n",         vsz);
        seq_printf(m, "      \"rss\": %lu,\n",         rss);

        /*
         * Memory_Usage: "1.5" viene de mem_usage=15 → 15/10=1, 15%10=5
         * CPU_Usage:    "0.25" viene de cpu_usage=25 → 25/100=0, 25%100=25
         * Formato idéntico al MetricasSO2 del curso.
         */
        seq_printf(m, "      \"Memory_Usage\": %lu.%lu,\n",
                   mem_usage / 10, mem_usage % 10);
        seq_printf(m, "      \"CPU_Usage\": %lu.%02lu\n",
                   cpu_usage / 100, cpu_usage % 100);

        seq_printf(m, "    }");

        /* Liberar la memoria del cmdline */
        if (cmdline)
            kfree(cmdline);
    }
    rcu_read_unlock();

    /* Cerrar el array y el objeto JSON raíz */
    seq_printf(m, "\n  ]\n}\n");
    return 0;
}

/* ============================================================
   APERTURA DEL ARCHIVO /proc
   (Patrón estándar del curso: single_open + sysinfo_show)
   ============================================================ */
static int sysinfo_open(struct inode *inode, struct file *file)
{
    return single_open(file, sysinfo_show, NULL);
}

/* ============================================================
   TABLA DE OPERACIONES DEL ARCHIVO /proc
   (Misma estructura que MetricasSO y MetricasSO2 del curso)
   ============================================================ */
static const struct proc_ops sysinfo_ops = {
    .proc_open    = sysinfo_open,
    .proc_read    = seq_read,
    .proc_lseek   = seq_lseek,
    .proc_release = single_release,
};

/* ============================================================
   INIT: Se ejecuta al hacer: sudo insmod sysinfo_module.ko
   ============================================================ */
static int __init sysinfo_init(void)
{
    struct proc_dir_entry *entry;

    entry = proc_create(PROC_NAME, 0444, NULL, &sysinfo_ops);
    if (!entry) {
        printk(KERN_ERR "[SOPES1] Error: no se pudo crear /proc/%s\n", PROC_NAME);
        return -ENOMEM;
    }

    printk(KERN_INFO "[SOPES1] Modulo cargado. Archivo disponible en: /proc/%s\n", PROC_NAME);
    return 0;
}

/* ============================================================
   EXIT: Se ejecuta al hacer: sudo rmmod sysinfo_module
   ============================================================ */
static void __exit sysinfo_exit(void)
{
    remove_proc_entry(PROC_NAME, NULL);
    printk(KERN_INFO "[SOPES1] Modulo descargado. /proc/%s eliminado.\n", PROC_NAME);
}

module_init(sysinfo_init);
module_exit(sysinfo_exit);