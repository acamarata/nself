FROM rust:1.75 AS builder

WORKDIR /app

# Copy manifest files
COPY Cargo.toml Cargo.lock ./

# Create src directory with dummy main
RUN mkdir src && echo "fn main() {}" > src/main.rs

# Build dependencies
RUN cargo build --release

# Copy actual source
COPY src ./src

# Touch main.rs to trigger rebuild with actual source
RUN touch src/main.rs && cargo build --release

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/target/release/${SERVICE_NAME} /usr/local/bin/app

# Create non-root user
RUN useradd -m -u 1001 rust

USER rust

EXPOSE ${PORT:-8080}

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:${PORT:-8080}/health || exit 1

CMD ["/usr/local/bin/app"]