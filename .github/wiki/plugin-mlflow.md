# MLflow Plugin

> ML experiment tracking, model registry, and artifact storage. **Free, MIT licensed.**

## Install

```bash
nself plugin install mlflow
```

## What It Does

Runs an MLflow tracking server backed by your Postgres database. Track experiments, compare model runs, log parameters and metrics, and manage the model registry. Integrates with Python ML workflows via the MLflow client.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MLFLOW_PORT` | `5000` | MLflow server port |
| `MLFLOW_ARTIFACT_ROOT` | `/mlflow/artifacts` | Artifact storage path |

## Ports

| Port | Purpose |
|------|---------|
| 5000 | MLflow UI and REST API |

## Database Tables

0 plugin-managed tables, MLflow manages its own schema in Postgres directly via SQLAlchemy.

## Nginx Routes

| Route | Target |
|-------|--------|
| `mlflow.{base_domain}` | MLflow UI |

## Dependencies

Requires Postgres (always running in ɳSelf). Optionally uses MinIO for artifact storage, set `MLFLOW_ARTIFACT_ROOT=s3://...` and configure MinIO credentials.
