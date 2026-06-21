Sistemas Operativos 1

Universidad San Carlos de Guatemala
Facultad de ingeniería.
Ingeniería en ciencias y sistemas

Proyecto #2
Quiniela de Mundial 2026 en la Nube con
Kubernetes
(Q.M.2026.K8s)

PONDERACIÓN: 60pts

Sistemas Operativos 1

1.  Resumen Ejecutivo

El  proyecto  “Q.M.2026.K8s”  tiene  como  propósito  aplicar  los  conocimientos  adquiridos  en
las unidades 1 y 2 del laboratorio, enfocándose en la implementación de una arquitectura de
sistema  distribuido  y  escalable  en  Google  Cloud  Platform  (GCP)  utilizando  Google
Kubernetes Engine (GKE).

Se  construirá  un  sistema  que  simula  la  recepción  y  procesamiento  de  predicciones  de
quiniela  del  Mundial  2026  enviadas  en  tiempo  real  por  distintos  usuarios.  Cada  predicción
representará  el  resultado  estimado  de  un  partido,  incluyendo  el  equipo  local,  equipo
visitante, goles predichos para cada equipo, el usuario que envía la predicción y una marca
de tiempo.

Este  sistema  involucrará  la  generación  de  tráfico  con  Locust,  una  API  REST  en  Rust,
servicios  en  Go  para  procesamiento  y  comunicación  gRPC,  RabbitMQ  como  sistema  de
mensajería,  consumidores  para  procesar  los  mensajes,  y  bases  de  datos/servicios  de
visualización  desplegados  sobre  máquinas  virtuales administradas  con KubeVirt  dentro  del
clúster.  En  particular,  Valkey  y  Grafana  deberán  ejecutarse  de  contenedores  gestionados
por  containerd  dentro  de  máquinas  virtuales  independientes  sobre  KubeVirt.  Todas  las
imágenes  Docker  de  los  componentes  deberán  publicarse  y  consumirse  desde  Zot,
desplegado en una máquina virtual de GCP fuera del clúster.

El proyecto busca demostrar la comprensión y aplicación de tecnologías de contenedores,
orquestación,  virtualización  dentro  de  Kubernetes,  comunicación  entre  microservicios,
manejo  de  concurrencia,  visualización  de  métricas  y  pruebas  de  carga,  así  como  la
integración opcional de Dapr como mecanismo alternativo para el envío de mensajes.

2.  Enunciado del Proyecto

En  la  actualidad,  los  sistemas  distribuidos  deben  ser  capaces  de  procesar  grandes
volúmenes  de  eventos  en  tiempo  real,  garantizar  escalabilidad,  tolerancia  a  fallos  y
observabilidad,  y  además  integrarse  con  distintos  mecanismos  de  comunicación  y
almacenamiento.  En  este  proyecto  se  simulará  una  plataforma  de  quiniela  donde
continuamente se reciben predicciones de partidos del Mundial 2026 enviadas por distintos
usuarios.

Cada evento contendrá información resumida sobre las predicciones de quiniela del Mundial
2026,  por  ejemplo:  equipo  local,  equipo  visitante,  goles  predichos  para  cada  equipo,  el
usuario  que  envía  la  predicción  y  una  marca  de  tiempo.  El  desafío  consiste  en  diseñar  e
implementar una arquitectura distribuida, escalable y resiliente que permita recibir, procesar,
enrutar,  consumir,  almacenar y  visualizar  dichos  datos  utilizando  tecnologías modernas  de
nube, contenedores y virtualización.

La solución deberá ejecutarse en GCP (Google Cloud Platform) utilizando instancias N1, un
clúster de GKE, Kubernetes Gateway API para exponer el sistema, una API REST en Rust,
servicios gRPC en Go, RabbitMQ como broker principal de mensajería, Valkey desplegado
dentro  de  un  contenedor  gestionado  por  contanerd  en  una  máquina  virtual  en  KubeVirt,  y
Grafana  desplegado  en  otra  máquina  virtual  independiente  dentro  de  un  contenedor
gestionado  por  containerd  también  en  KubeVirt.  Adicionalmente,  el  estudiante  podrá
implementar  un  flujo  alternativo  de  mensajería  utilizando  Dapr  como  parte  extra  del
proyecto.

3 Alcance del proyecto

Grafica en HD: https://drive.google.com/file/d/10sK4Ls_5_rrsNNl_MXLWENfS46DZDIQt/view?usp=sharing
 (se recomienda descargar el archivo y abrirlo en https://draw.io)

●  Componentes clave incluyen: Locust (generador de carga), Kubernetes Gateway API, API REST (Rust), servicios gRPC (Go),
RabbitMQ, consumidores (Go), Valkey en contenedor de containerd en una máquina virtual sobre KubeVirt, Grafana en contenedor
de containerd en máquina virtual sobre KubeVirt y Zot como Container Registry externo al clúster.

4.3 Alcance del proyecto

●  Alcance obligatorio:
o  Locust: Generación de tráfico con la estructura JSON especificada hacia el

endpoint público expuesto mediante Kubernetes Gateway API.

o  Los datos enviados deberán simular predicciones de la

quiniela con valores aleatorios dentro de rangos definidos.

o  Gateway API: Exposición del sistema utilizando Kubernetes Gateway API en

sustitución del uso de Ingress Controller. El sistema deberá contar con las rutas
para:

▪  /grpc-#carnet
▪  /dapr-#carnet (parte opcional con Dapr)

o  Deployments de Rust: API REST que recibe peticiones de Locust, envía a un
Deployment de Go, soporta alta carga y escala con HPA (1-3 réplicas, CPU >
30%).

o  Deployments de Go:

▪  Deployment 1 (API REST y gRPC Client): Recibe de Rust, actúa como
cliente gRPC, invoca funciones para publicar en RabbitMQ. Es decir un
deployment con 2 containers.

▪  Deployments 2 y 3 (gRPC Server y Writer RabbitMQ): recibe solicitudes
gRPC y publica mensajes en RabbitMQ. Deben realizarse pruebas con 1 y 2
réplicas en los componentes que la cátedra defina para análisis de
rendimiento. Es decir un deployment con 2 containers.

o  RabbitMQ: Broker principal de mensajería del proyecto. Será el único

sistema de colas obligatorio utilizado para el flujo principal.

o  RabbitMQ Client (Consumer) (Deployment): Deployment encargado de consumir
los mensajes de RabbitMQ, procesarlos y almacenar la información resultante en
Valkey.

o  Valkey en contenedor de containerd sobre KubeVirt: Valkey deberá

ejecutarse dentro de un contenedor gestionado por containerd en una máquina
virtual administrada por KubeVirt, desplegada dentro del clúster de Kubernetes.
Esta VM deberá ser independiente y dedicada al almacenamiento de datos
procesados. Se debe asegurar persistencia y conectividad entre los consumers y
la VM.

o  Grafana en contenedor de containerd sobre KubeVirt: Grafana deberá

ejecutarse dentro de un contenedor gestionado por containerd en una máquina
virtual distinta, también administrada por KubeVirt dentro del clúster. Esta VM
deberá conectarse a la fuente de datos correspondiente para construir y mostrar
los dashboards requeridos.

o  Zot: Implementado en una VM de GCP fuera del clúster K8s. Todas las imágenes
Docker de los componentes se publican y se descargan desde Zot. Usar HTTPS.
o  OCI Artifact: Descarga de archivo de entrada desde el registry como un OCI Artifact

o

(se debe especificar qué archivo y cómo se usa en la documentación)
Infraestructura en GCP: Todo el proyecto debe desplegarse en Google Cloud
Platform, utilizando instancias N1 para soportar KubeVirt.

o  Documentación: El manual técnico deberá explicar:

▪  arquitectura general,
▪

flujo completo de datos,

▪  configuración de Gateway API,
▪  comunicación REST y gRPC,
▪  uso de RabbitMQ,
▪  despliegue de Valkey y Grafana en contenedores gestionados por containerd sobre

máquinas virtuales de KubeVirt,

▪  configuración de HPA,
▪  publicación/consumo de imágenes desde Zot,
▪  pruebas realizadas y conclusiones.

o  El manual técnico deberá ser entregado exclusivamente en

formato Markdown.

o  Sugerencias Generales: Uso de namespaces, Gateway API, creación propia

de imágenes Docker.

o  Requisitos Mínimos para tener derecho a calificación:

▪  Clúster de Kubernetes en GCP
▪  Uso obligatorio de Locust
▪  Uso obligatorio de GKE
▪  Uso obligatorio de RabbitMQ
▪  Uso obligatorio de KubeVirt y Containerd para Valkey y Grafana

▪  Restricciones: Proyecto individual, uso obligatorio de Locust y GKE
▪

NO HABRÁ PRÓRROGA.

o  Github: Repositorio privado con nombre <CARNET>_LAB_P2_SO1_VacJun2026 o

<CARNET>_LAB_SO1_VacJun2026 con una carpeta que divida al proyecto 2 del 1. NO
OLVIDAR AGREGARME AL REPOSITORIO: CamiloSincal

●  Alcance opcional / punteo extra en Clase Magistral

o

Implementación con Dapr (10pts extra sobre 100): Se deberá
implementar un flujo adicional de envío de mensajes utilizando Dapr,
expuesto por medio de la ruta:

▪

/dapr-#carnet

Esta implementación podrá coexistir con el flujo base /grpc-#carnet, con
el objetivo de comparar ambos enfoques a nivel de arquitectura, integración y
comportamiento.

Para la implementación con Dapr, deberá integrarse el SDK de Dapr en los
servicios que actualmente utilizan gRPC, con el fin de habilitar:
-  El envío de información mediante pub/sub (publicador)
-  La recepción de información mediante suscripción a eventos
-  La comunicación desacoplada entre servicios a través de RabbitMQ

Esta implementación deberá estar debidamente documentada en el manual
técnico y será presentada al finalizar la evaluación.

o  Pruebas locales con k3s (5pts extra sobre 100): Se deberá
utilizar k3s para realizar y documentar pruebas locales de
kubernetes.

●  Recomendaciones:

o  Optimización del rendimiento: Uso de requests y limits en los

deployments para mejorar estabilidad del sistema.

o  Expiración de datos: Configuración de TTL o políticas de expiración en

Valkey para evitar saturación de memoria.

4.3.1 Estructura de los mensajes:

La estructura de los mensajes deberá representar prediccións de quiniela simulados.
Un ejemplo de JSON válido es el siguiente:
{

"home_team": "GTM",

"away_team": "MEX",

""home_goals": 2,

"away_goals": 1,

"username": "user_42",

"timestamp": "2026-06-15T18:00:00Z"

}

Reglas de generación

home_team: código del equipo local (3 letras, p.ej. GTM, MEX, BRA, ARG, ESP).

away_team: código del equipo visitante (3 letras, distinto al local).

home_goals: goles predichos para el equipo local, entero aleatorio.

away_goals: goles predichos para el equipo visitante, entero aleatorio.

username: identificador del usuario que envía la predicción.

timestamp: fecha y hora del evento.

Rangos sugeridos

●  home_goals: entre 0 y 5

●  away_goals: entre 0 y 5

●  username: cadena alfanumérica aleatoria, p.ej. "user_N" con N entre 1 y 1000

Propuesta de estructura gRPC

syntax = "proto3";

package worldcup2026;

option go_package = "./proto";

// Mensaje que se enviará

message MatchPredictionRequest {

Teams home_team = 1;

Teams away_team = 2;

int32 home_goals  = 3;

int32 away_goals  = 4;

string username = 5;

string timestamp  = 6;

}

// Equipos aceptados por el proyecto

enum Teams {

TEAMS_UNKNOWN = 0;

GTM = 1;

MEX = 2;

BRA = 3;

ARG = 4;

ESP = 5;

}

//  Respuesta  del  servidor

message MatchPredictionResponse {

string status = 1;

}

// Servicio gRPC

service MatchPredictionService {

rpc SendPrediction (MatchPredictionRequest) returns

(MatchPredictionResponse);

}

4.3.2 Estructura del Dashboard Requerido

Asignación de país en base al último dígito de su carnet

0,1 = GTM

2,3 = MEX

4,5 = BRA

6,7 = ARG

8,9 = ESP

Visualizaciones requeridas
▪  Mayor cantidad de goles predichos en un partido (local)
▪  Menor cantidad de goles predichos en un partido (local)
▪  Mayor cantidad de goles predichos en un partido (visitante)
▪  Menor cantidad de goles predichos en un partido (visitante)
▪  Top de equipos con más victorias predichas
▪  Top de usuarios más activos (más predicciones enviadas)
▪  Moda de goles predichos (local)
▪  Moda de goles predichos (visitante)
▪  Serie temporal del equipo asignado, mostrando la evolución de:

o  goles predichos como local
o  goles predichos como visitante

▪  Nombre del equipo asignado
▪  Cantidad total de predicciones recibidas para el equipo asignado

A continuación, se presenta un ejemplo ilustrativo del dashboard que se
espera desarrollar en Grafana, mostrando su estructura y visualizaciones
esperadas

Gráfica en HD: https://drive.google.com/file/d/1xd_uKJ_6Q9pJ66fJwinSK-5CKvBwni_q/view?usp=sharing
(se recomienda descargar el archivo y abrirlo en https://draw.io)

3.3 Entregables

1.  Código fuente en repositorio privado GitHub nombre y añadir al auxiliar: @CamiloSincal
2.  Manual técnico y guía de reproducibilidad/ejecución.
3.  Capturas de prueba y evidencia funcional
4.  Metodología
5.  Fase 1: Configuración e Infraestructura Base
5.1.  Configurar cuenta de GCP y crear el clúster de GKE utilizando instancias N1.
5.2.  Verificar los requisitos necesarios para soportar KubeVirt y virtualización anidada dentro del entorno.
5.3.  Instalar y configurar Zot en una VM de GCP fuera del clúster.
5.4.  Desarrollar y contenerizar la API REST en Rust. Publicar la imagen en Zot.
5.5.  Desarrollar y contenerizar el Go Deployment 1 encargado de la comunicación inicial y del rol de gRPC Client   Publicar la imagen en Zot.
5.6.  Configurar Locust para generar tráfico básico con la estructura JSON definida para predicciones de quiniela.
5.7.  Desplegar y configurar Kubernetes Gateway API para exponer los serviciosdel sistema.
5.8.  Realizar pruebas iniciales del flujo: Locust → Gateway API → API Rust → Go Deployment 1
6. Fase 2: Comunicación Interna y Publicación de Mensajes
6.1.  Desplegar RabbitMQ en GKE como broker principal de mensajería.
6.2.  Desarrollar y contenerizar el Go Deployment 2, encargado del rol de gRPC Server y publicación de mensajes en RabbitMQ. Publicar la imagen en Zot. 
6.3.  Integrar el Go Deployment 1 con el servicio publicador mediante gRPC.
6.4.  Configurar la ruta principal del sistema para el flujo estándar: /grpc-#carnet
6.5.  Realizar pruebas de publicación de mensajes hacia RabbitMQ.
6.6. Validar el flujo completo de recepción, transformación y envío de mensajes.
7.  Fase 3: Consumo de Mensajes y Persistencia en Máquina Virtual
7.1.  Desarrollar y contenerizar el Consumer en Go para procesar mensajes desde RabbitMQ. Publicar la imagen en Zot.
7.2.  Desplegar KubeVirt dentro del clúster de Kubernetes.
7.3.  Crear y configurar un contenedor con containerd en VM para Valkey dentro de KubeVirt, asegurando conectividad y persistencia.
7.4.Instalar y configurar Valkey dentro del contenedor de containerd de dicha máquina virtual.
7.5. Integrar el Consumer para almacenar los datos procesados en Valkey.
7.6.  Realizar pruebas de consumo y almacenamiento.
7.7. Validar la persistencia y consulta de los datos generados por el sistema.
Fase 4: Visualización, Ruta Alternativa y Pruebas de Carga
8.1.  Crear y configurar un contenedor con containerd en una VM independiente para Grafana dentro de KubeVirt.
8.2   Instalar y configurar Grafana en este contenedor de containerd De la máquina virtual.
8.3.  Configurar los dashboards requeridos para visualizar la información almacenada y procesada.
8.4.  Implementar, como parte de punteo extra, la ruta alternativa: /dapr-#carnet
8.5   Integrar y probar el envío de mensajes utilizando Dapr.
8.6.  Realizar pruebas de carga completas con Locust.
8.7.  Implementar y probar HPA para el deployment de Rust.
8.8.  Analizar el rendimiento del sistema con 1 y 2 réplicas en los deployments de Go definidos para evaluación.
8.9.  Comparar el comportamiento general del flujo estándar y del flujo con Dapr, en caso de haberse implementado.
9. Fase 5: Documentación, Validación y Entrega
9.1  Redactar el manual técnico, incluyendo respuestas a las preguntas y análisis
de rendimiento.
9.2  Verificar que todos los componentes estén correctamente configurados y
documentados en el repositorio.
9.3  Validar que las imágenes publicadas en Zot puedan ser consumidas
correctamente por los componentes del sistema.
9.4  Preparar la entrega final conforme a los requisitos establecidos.
6. Recursos y herramientas a utilizar
Listado de materiales que los estudiantes deberán usar o investigar:
●  Dapr sidecar: https://docs.dapr.io/concepts/dapr-services/sidecar/
●  Kubervirt: https://kubevirt.io/

