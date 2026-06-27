import random
import time
from locust import HttpUser, task, between

# Equipos válidos del proyecto
TEAMS = ["GTM", "MEX", "BRA", "ARG", "ESP"]

class QuinielaUser(HttpUser):
    # Tiempo de espera entre predicciones (simula usuario real)
    wait_time = between(0.1, 0.5)

    @task
    def enviar_prediccion(self):
        # Elegir equipo local y visitante (distintos)
        home_team = random.choice(TEAMS)
        away_team = random.choice([t for t in TEAMS if t != home_team])

        payload = {
            "home_team":  home_team,
            "away_team":  away_team,
            "home_goals": random.randint(0, 5),
            "away_goals": random.randint(0, 5),
            "username":   f"user_{random.randint(1, 1000)}",
            "timestamp":  time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        }

        # Ruta principal del Gateway API (carnet 202308204)
        self.client.post(
            "/grpc-202308204",
            json=payload,
            headers={"Content-Type": "application/json"}
        )
