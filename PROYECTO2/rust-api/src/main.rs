// Rust REST API — recibe predicciones de Locust y las reenvía al Go D1
// Concepto Clase 6: funciones, Result, match, estructuras

use axum::{
    extract::State,
    http::StatusCode,
    routing::post,
    Json, Router,
};
use serde::{Deserialize, Serialize};
use std::{env, sync::Arc};
use tracing::info;

// Estructura del mensaje JSON (Clase 6: tipos de datos)
// #[derive(Deserialize)] → axum deserializa el JSON automáticamente
#[derive(Debug, Deserialize, Serialize, Clone)]
struct Prediction {
    home_team:  String,
    away_team:  String,
    home_goals: i32,
    away_goals: i32,
    username:   String,
    timestamp:  String,
}

// Respuesta que devuelve nuestra API
#[derive(Serialize)]
struct ApiResponse {
    status: String,
}

// Estado compartido entre los handlers (dirección del Go D1)
struct AppState {
    go_d1_url:     String,
    http_client:   reqwest::Client,
}

// Handler del endpoint POST /
// Concepto Clase 6: funciones con Result + Clase 8 REST
async fn receive_prediction(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<Prediction>,
) -> (StatusCode, Json<ApiResponse>) {
    info!("Predicción recibida: {} vs {} ({} - {})",
        payload.home_team, payload.away_team,
        payload.home_goals, payload.away_goals
    );

    // Reenviar al Go D1 (Clase 6: manejo de Result con match)
    let resultado = state.http_client
        .post(&state.go_d1_url)
        .json(&payload)
        .send()
        .await;

    match resultado {
        Ok(resp) => {
            info!("Go D1 respondió con status: {}", resp.status());
            (StatusCode::OK, Json(ApiResponse { status: "ok".to_string() }))
        }
        Err(e) => {
            // Clase 6: Err es un valor, no una excepción
            tracing::error!("Error reenviando al Go D1: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR,
             Json(ApiResponse { status: "error".to_string() }))
        }
    }
}

#[tokio::main]
async fn main() {
    // Inicializar logs
    tracing_subscriber::fmt::init();

    // Leer la URL del Go D1 desde variable de entorno (Clase 9: ConfigMap/env vars)
    let go_d1_url = env::var("GO_D1_URL")
        .unwrap_or_else(|_| "http://go-d1-service:8080".to_string());

    info!("Rust API iniciando. Go D1 URL: {}", go_d1_url);

    let state = Arc::new(AppState {
        go_d1_url,
        http_client: reqwest::Client::new(),
    });

    // Definir rutas (Clase 8: REST endpoint)
    let app = Router::new()
        .route("/", post(receive_prediction))
        .with_state(state);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080")
        .await
        .unwrap();

    info!("Escuchando en 0.0.0.0:8080");
    axum::serve(listener, app).await.unwrap();
}