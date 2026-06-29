#!/usr/bin/env python3
import csv
import pathlib
import sys


def aggregate(path: pathlib.Path) -> dict[str, str]:
    with path.open(newline="") as handle:
        rows = list(csv.DictReader(handle))
    for row in rows:
        if row.get("Name") == "Aggregated":
            return row
    raise SystemExit(f"no se encontró la fila Aggregated en {path}")


if len(sys.argv) != 3:
    raise SystemExit("uso: compare-locust.py <go-d2-1_stats.csv> <go-d2-2_stats.csv>")

one = aggregate(pathlib.Path(sys.argv[1]))
two = aggregate(pathlib.Path(sys.argv[2]))
columns = [
    ("Requests/s", "Requests/s"),
    ("Failures/s", "Failures/s"),
    ("Average Response Time", "Promedio (ms)"),
    ("50%", "p50 (ms)"),
    ("95%", "p95 (ms)"),
    ("99%", "p99 (ms)"),
    ("Request Count", "Solicitudes"),
    ("Failure Count", "Errores"),
]

print("# Comparación Locust — Go D2")
print()
print("| Métrica | 1 réplica | 2 réplicas |")
print("|---|---:|---:|")
for key, label in columns:
    print(f"| {label} | {one.get(key, 'N/D')} | {two.get(key, 'N/D')} |")
print()
one_rps = float(one["Requests/s"])
two_rps = float(two["Requests/s"])
variation = ((two_rps / one_rps) - 1) * 100 if one_rps else 0
if variation > 2:
    conclusion = "dos réplicas mejoraron"
elif variation < -2:
    conclusion = "dos réplicas redujeron"
else:
    conclusion = "dos réplicas no cambiaron materialmente"
print(f"Conclusión: {conclusion} el throughput ({variation:+.2f}%) al escalar Go D2 de una a dos réplicas. Rust se mantuvo en una réplica para aislar esta variable.")
print()
print("Pruebas ejecutadas contra `/grpc-202308204`; los CSV y reportes HTML del mismo directorio son la evidencia fuente.")
