FROM rust:1.78 AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
# Truco para cachear dependencias antes de copiar el código
RUN mkdir src && echo 'fn main() {}' > src/main.rs
RUN cargo build --release
RUN rm src/main.rs
# Ahora copiar el código real y compilar
COPY src ./src
RUN touch src/main.rs && cargo build --release

# Etapa 2: imagen final mínima (sin el compilador de Rust)
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/rust-api /usr/local/bin/rust-api
EXPOSE 8080
CMD ["rust-api"]