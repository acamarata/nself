FROM ghcr.io/mlflow/mlflow:latest

# Install PostgreSQL adapter
RUN pip install --no-cache-dir psycopg2-binary

# Ensure artifacts directory exists
RUN mkdir -p /mlflow/artifacts

# Set working directory
WORKDIR /mlflow

# Default command handled by docker-compose