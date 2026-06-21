---
name: enunciado_proyecto
description: Enunciado oficial del proyecto 1
---
# Proyecto 1 - SOPES 1


## Página 1

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
 
Universidad San Carlos de Guatemala 
Facultad de ingeniería. 
Ingeniería en ciencias y sistemas 
 
 
 
 
 
Proyecto 1: 
Sonda de Kernel en C y Daemon en 
Go para Telemetría de contenedores 
 
. 
 
PONDERACIÓN:  40 pts


## Página 2

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
Resumen Ejecutivo 
Este proyecto desarrollará un sistema integral compuesto por un módulo de kernel para Linux 
y una aplicación en GO que será un Daemon que se cargará al sistema, diseñado para 
monitorizar, analizar y gestionar de manera autónoma contenedores en tiempo real. El 
módulo de kernel interactúa directamente con el núcleo del sistema operativo para recopilar 
datos detallados sobre los procesos de los contenedores (PID, estado, nombre, consumo de 
memoria, CPU, E/S y otros recursos críticos), exponiendo esta información a través del 
sistema de archivos /proc para su procesamiento en el espacio de usuario. 
Enunciado del Proyecto 
Este proyecto tiene como objetivo principal diseñar e implementar un sistema integral para 
la monitorización proactiva, análisis automatizado y gestión inteligente de contenedores en 
entornos Linux. 
 
Descripción del problema o necesidad a resolver 
En la administración de sistemas y el desarrollo de aplicaciones contenerizadas, obtener 
información detallada sobre los procesos en ejecución y tomar acciones automatizadas es un 
desafío crítico. Aunque herramientas como ps o docker stats ofrecen datos básicos, carecen 
de acceso directo a las estructuras del kernel y no permiten una gestión proactiva de 
contenedores. Este proyecto propone una solución integral: desarrollar un módulo de kernel 
en C que exponga métricas avanzadas de procesos y contenedores (CPU, memoria, E/S) a 
través de /proc, junto con un Daemon en GO que no solo presente estos datos de forma 
legible, sino que también automatice decisiones (como terminar contenedores que excedan 
umbrales de recursos) a su vez que guarda datos importantes en Valkey para su posterior 
uso en Grafana. Para validar el sistema, se implementará cronjobs que generen contenedores 
de prueba cada minuto, simulando condiciones de carga y permitiendo evaluar la eficacia de 
las acciones correctivas. Así, el proyecto combina aprendizaje en programación a bajo nivel 
(kernel) y alto nivel (GO), mientras resuelve un problema real en entornos contenerizados: la 
monitorización y estabilización autónoma del sistema. A su vez que de dichos datos se 
mostraran de manera visual por medio de un dashboard en Grafana. 
1. Un módulo de kernel desarrollado en C que actuará como sensor de bajo nivel, 
accediendo directamente a las estructuras internas del kernel para capturar métricas 
detalladas tanto de los procesos asociados a contenedores como de los procesos 
generales del sistema, incluyendo consumo de recursos como CPU, memoria y E/S. 
2. Una Daemon en GO que funcionará como cerebro del sistema, procesando los datos 
del kernel en tiempo real para: 
○ Tomar decisiones autónomas detener y eliminar contenedores basadas en 
umbrales dinámicos y patrones de comportamiento establecidos previamente. 
○ Simular y validar la eficacia del sistema bajo condiciones de uso prolongado. 
○ Ejecutar scripts de automatización durante la ejecución.


## Página 3

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
3. Un Cronjob encargado de ejecutar el script que generara los contenedores de Docker 
cada minuto. 
 
4. Un Dashboard en Grafana utilizado para mostrar la información que recolecta el 
servicio de GO. 
MÓDULO DE KERNEL 
Deberá crear un módulo que capture las métricas necesarias para el análisis de la memoria 
y los contenedores activos en el sistema. 
La información debe ser capturada y guardada en la carpeta /proc: 
1. Capturar en MB o KB (a discreción del estudiante): 
● Total de memoria RAM 
● Memoria RAM libre 
● Memoria RAM en uso 
2. Todos los procesos relacionados a los contenedores generados por el script así 
como los procesos generales del sistema deberán contar con: 
● PID 
● Nombre 
● Línea de comando que se ejecutó o ID del contenedor 
● VSZ (Tamaño de la memoria virtual en KB) 
● RSS (Tamaño de memoria física en KB) 
● Porcentaje de Memoria utilizada 
● Porcentaje de CPU utilizado 
Sugerencias: 
● Utilizar la estructura task_struct (del kernel de Linux) para filtrar correctamente los 
procesos relacionados con los contenedores y extraer la información necesaria. 
● En dado caso el porcentaje de CPU sea un número extremadamente grande se 
permite mantenerlo así debido a los cálculos diferenciales que retorna el kernel. 
 
3. Los datos deberán ser guardados en sus respectivos archivos en /proc 
● Módulo de Procesos de Contenedores: /proc/continfo_pr1_so1_#CARNET


## Página 4

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
CRONJOB 
Se implementará un cronjob con ejecución cada 2 minutos para simular una carga de trabajo 
variable. Su lógica consistirá en: 
● Despliegue Aleatorio: Orquestar la creación de 5 contenedores, seleccionando 
aleatoriamente entre las imágenes de prueba desarrolladas. 
 
IMÁGENES DE DOCKER (Carga de Trabajo) 
Se desarrollarán 3 imágenes personalizadas para emular distintos perfiles de consumo: 
 
Categoría 
Imagen 
Requerida 
Descripción 
Funcional 
Imagen Recomendada 
de Docker Hub 
Comando de Ejemplo 
Alto 
Consumo 
go-client 
Consumo 
significativo de 
memoria RAM. 
roldyoran/go-client 
docker run -d 
roldyoran/go-client   
Alto 
Consumo 
alpine 
Alta carga 
computacional 
en la(s) CPU(s). 
alpine 
docker run -d alpine 
sh -c "while true; do 
echo '2^20' | bc > 
/dev/null; sleep 2; 
done" 
Bajo 
Consumo 
alpine 
Consumo ínfimo 
de RAM y CPU. 
alpine 
docker run -d alpine 
sleep 240


## Página 5

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
 
GRAFANA 
Como estudiante deberá realizar un dashboard en Grafana que muestre la siguiente 
información. 
 
 
Métrica / Visualización 
Descripción y Detalles 
Tipo de Visualización 
Sugerido 
Visión General del Sistema 
Total de RAM 
[Dato en Texto] 
Tarjeta (Card) o Indicador 
RAM Usada 
[Dato en Texto] 
Tarjeta (Card) o Indicador 
Memoria Libre 
[Dato en Texto] 
Tarjeta (Card) o Indicador 
Evolución Temporal 
Uso de RAM a lo largo 
del tiempo 
Monitorización del consumo 
de RAM con línea de 
evolución. Eje X: 
timestamp. 
Gráfico de Líneas (Time 
Series) o Tarjeta (Card) o 
Indicador 
 
Contenedores Eliminados 
a lo largo del tiempo 
Número de contenedores 
eliminados en cada 
momento. Eje X: 
timestamp. 
Gráfico de Barras (Time 
Series) o Tarjeta (Card) o 
Indicador 
 
Top Rankings 
(Consumos Históricos) 
Nota: Incluye contenedores activos e 
inactivos/eliminados.


## Página 6

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
Top 5 Contenedores por 
Consumo de RAM 
Muestra el PID y ID del 
proceso/contenedor. 
Gráfico de Pastel (Pie Chart) 
Top 5 Contenedores por 
Consumo de CPU 
Muestra el PID y ID del 
proceso/contenedor. 
Gráfico de Pastel (Pie Chart) 
 
Dashboard a Realizar:  
 
 
Daemon de GO 
Descripción (corazón del proyecto): 
Se requiere desarrollar un gestor de contenedores en Go encargado del análisis, 
ejecución y comunicación entre los diferentes componentes del servicio. 
 
Debe garantizar manejo seguro de memoria y cumplir con las siguientes funcionalidades: 
1. Inicio del servicio 
● Crear un contenedor de Grafana al inicializar el código. 
● Grafana será el encargado de leer los logs generados por el servicio de Go después 
del análisis de los datos.


## Página 7

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
● Se recomienda utilizar un Docker Compose para que el contenedor se pueda 
comunicar con la base de datos Valkey 
2. Cronjob 
● El daemon de Go iniciará la implementación y ejecución del cronjob en el sistema 
operativo, lo que activará el proceso de creación de contenedores. 
 
3. Ejecución del script para cargar el módulo de kernel 
● El daemon de Go ejecutará un script encargado de cargar e inicializar el módulo de 
kernel. 
4. Loop principal (ejecución cada 20 a 60 segundos) 
● El daemon operará de manera infinita en segundo plano. 
● En cada iteración (cada 20 a 60 segundos), realiza lo siguiente: 
○ Lectura del archivo en /proc/continfo_pr1_so1_#CARNET 
○ Deserialización del contenido 
○ Análisis para la gestión de contenedores (alta/baja, ajuste de recursos) 
■ detener y eliminar los contenedores según cálculos explicados en la 
siguiente sección. 
○ Generación y almacenamiento de registros (logs) en una base de datos 
Valkey, diseñado para su posterior visualización en Grafana 
 
5. Finalización del servicio 
● Antes de finalizar, el servicio deberá eliminar el cronjob asociado a la creación de 
los contenedores para evitar una sobrecarga en el sistema.


## Página 8

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
 
RESTRICCIONES CLAVE 
Contenedores en la máquina 
● Siempre deben existir 3 contenedores de bajo consumo en todo momento. 
● Siempre deben existir 2 contenedores de alto consumo en todo momento. 
● No debe eliminarse el contenedor de Grafana. 
 
Análisis y ordenamiento de contenedores 
Durante la gestión de contenedores, aplicar ordenamientos basados en: 
● Uso de RAM 
● VSZ (Tamaño de memoria virtual en KB) 
● RSS (Memoria física residente en KB) 
● Uso de CPU 
 
Toma de decisiones 
El servicio debe decidir qué contenedores eliminar y cuáles mantener, considerando las 
instrucciones dadas anteriormente.


## Página 9

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
 
ARQUITECTURA


## Página 10

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
 
Alcance del proyecto 
 
 
Alcance obligatorio:  
● Módulo de Kernel en C: Desarrollo de un sensor de bajo nivel que acceda a la 
estructura task_struct para capturar métricas de memoria RAM (total, libre, en uso) y 
detalles de procesos (PID, nombre, VSZ, RSS, %CPU, %Mem). 
● Interfaz de Comunicación /proc: Creación y gestión de archivos en el sistema /proc 
para exponer los datos capturados hacia el espacio de usuario. 
● Daemon de Gestión en Go: Implementación de un servicio que automatice la carga 
del módulo de kernel, gestione la ejecución de cronjobs y realice el análisis de 
métricas cada 20 a 60 segundos. 
● Persistencia de Datos: Configuración de una base de datos Valkey para el 
almacenamiento de registros y logs generados por el servicio de Go. 
● Simulación de Carga: Configuración de un Cronjob que despliegue aleatoriamente 5 
contenedores cada 2 minutos utilizando imágenes de alto y bajo consumo. 
● Gestión Autónoma de Recursos: Lógica para detener y eliminar contenedores 
según umbrales dinámicos, garantizando siempre la permanencia de 3 contenedores 
de bajo consumo y 2 de alto consumo. 
● Visualización en Tiempo Real: Diseño de un dashboard en Grafana que muestre la 
evolución temporal del uso de RAM, contenedores eliminados y rankings de consumo. 
Recursos y herramientas para utilizar 
 
Tipo ( Obligatorio / 
opcional) 
Categoría ( Software / hardware / 
Plataforma / Etc ) 
Descripción 
Obligatorio 
Software 
Lenguaje C: Utilizado para 
el desarrollo del módulo de 
kernel y la captura de 
métricas de bajo nive 
Obligatorio 
Software 
Lenguaje Go: Empleado 
para la creación del 
Daemon que funciona como 
el "cerebro" del sistema y 
gestiona la lógica de 
contenedores. 
Obligatorio 
Software 
Docker / Docker Compose: 
Plataforma de contenedores 
para ejecutar las imágenes 
de prueba, Grafana y 
Valkey. 
Obligatorio 
 
 
 
Software 
Valkey: Base de datos 
utilizada para el


## Página 11

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
 
 
almacenamiento persistente 
de los registros y métricas 
recolectados por el 
Daemon. 
Obligatorio 
Plataforma 
Grafana: Herramienta de 
visualización para la 
creación del dashboard de 
monitoreo de RAM y 
contenedores. 
Obligatorio 
Software 
Linux Kernel API: Uso de 
estructuras internas como 
task_struct y el sistema de 
archivos /proc para la 
telemetría. 
 
Entregables 
A continuación se detalla cada uno de los elementos que deberá presentar el estudiante 
Tipo 
Descripción 
Código Fuente 
Repositorio privado en GitHub nombrado como 
Carnet#_LAB_SO1_VacJun2026 o 
Carnet#_LAB_P1_SO1_VacJun2026. Incluye los archivos fuente 
.c, el Makefile del módulo de kernel y el código del Daemon en Go. 
Colaboradores 
Se debe añadir obligatoriamente a los auxiliares del curso 
(CamiloSincal) como colaboradores del repositorio. 
Documentación 
Técnica 
Manual técnico y guía de instalación detallada. El informe debe ser 
claro, conciso y completo, incluyendo también el manual de 
usuario. 
Evidencia Funcional 
Capturas de pantalla de las pruebas realizadas que sirvan como 
evidencia del funcionamiento del sistema. Debe demostrarse la 
lectura correcta del archivo en /proc con el listado de procesos 
(PID, nombre, memoria y CPU).


## Página 12

Sistemas Operativos 1 
Proyecto - vigente para el Segundo Semestre 2026  
 
 
 
Material de apoyo 
 
Categoría (Manual / 
Documentación oficial/ 
Ejemplo / Etc) 
Link 
Descripción 
Link Repositorio del curso 
 
Link del repositorio del curso 
con diversos ejemplos 
prácticos.  
 
Metodología 
A continuación se presenta una metodología de la lógica que puedes seguir para elaborar el 
presente proyecto: 
 
Pasos para seguir: 
 
● Paso 1: Desarrollo del Módulo de Kernel e Interfaz /proc Crear el módulo en C y 
su respectivo Makefile para exponer las métricas de memoria RAM y contenedores a 
través del sistema de archivos /proc. Es fundamental asegurar que el archivo sea 
legible y se cree correctamente al cargar el módulo. 
● Paso 2: Extracción de métricas con task_struct Implementar la iteración por los 
procesos del sistema utilizando la estructura task_struct para filtrar y capturar 
específicamente el PID, nombre, memoria (VSZ/RSS) y el porcentaje de CPU. 
● Paso 3: Configuración del entorno y contenedores de prueba Preparar las 3 
imágenes personalizadas en Docker para emular cargas de alto y bajo consumo, y 
configurar el entorno mediante Docker Compose para la comunicación entre Valkey y 
Grafana. 
● Paso 4: Construcción del Daemon en Go Desarrollar el servicio en Go encargado 
de cargar el módulo de kernel, gestionar la ejecución de los cronjobs y realizar la 
lectura y deserialización de los datos de /proc cada 20 a 60 segundos. 
● Paso 5: Implementación de la lógica de toma de decisiones Programar la lógica 
autónoma para que el Daemon analice las métricas y decida qué contenedores 
detener o eliminar, asegurando siempre la permanencia de 3 contenedores de bajo 
consumo y 2 de alto consumo. 
● Paso 6: Persistencia en Valkey y Visualización en Grafana Configurar el 
almacenamiento de los logs procesados en la base de datos Valkey y diseñar el 
dashboard en Grafana para monitorear en tiempo real el uso de RAM, rankings de 
consumo y contenedores eliminados. 
● Paso 7: Validación, Pruebas y Documentación Final Verificar el flujo completo del 
sistema, desde la creación automática de contenedores vía cronjob hasta su gestión 
por el Daemon, y elaborar el manual técnico y la guía de instalación para la entrega 
final