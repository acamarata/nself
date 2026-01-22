# MLflow Command

Manage MLflow for ML experiment tracking and model registry.

## Quick Start

```bash
# Enable MLflow
nself mlflow enable

# Rebuild and start
nself build && nself start

# Open the UI
nself mlflow open

# Test connection
nself mlflow test
```

## Commands

| Command | Description |
|---------|-------------|
| `nself mlflow` | Show status |
| `nself mlflow enable` | Enable MLflow service |
| `nself mlflow disable` | Disable MLflow service |
| `nself mlflow open` | Open MLflow UI in browser |
| `nself mlflow configure <setting> <value>` | Configure settings |
| `nself mlflow experiments` | List experiments |
| `nself mlflow experiments create <name>` | Create experiment |
| `nself mlflow experiments delete <id>` | Delete experiment |
| `nself mlflow runs [exp_id]` | List runs |
| `nself mlflow test` | Test connection |
| `nself mlflow logs [-f]` | View logs |

## Prerequisites

MLflow requires:
- **PostgreSQL** - For backend store (already included)
- **MinIO** - For artifact storage (enable with `MINIO_ENABLED=true`)

## Configuration

```bash
# Set username
nself mlflow configure username admin

# Set/generate password
nself mlflow configure password
nself mlflow configure password mypassword

# Change port
nself mlflow configure port 5001
```

## Experiment Management

### List Experiments

```bash
nself mlflow experiments
```

Output:
```
  ID       NAME                           STATE      ARTIFACT LOCATION
  --------  ------------------------------ ---------- ----------
  0        Default                        active     s3://mlflow/0
  1        my-model                       active     s3://mlflow/1
```

### Create Experiment

```bash
nself mlflow experiments create my-model
```

### Delete Experiment

```bash
nself mlflow experiments delete 1
```

### List Runs

```bash
# All runs
nself mlflow runs

# Runs for specific experiment
nself mlflow runs 1
```

## Python Integration

### Basic Usage

```python
import mlflow

# Connect to MLflow
mlflow.set_tracking_uri('http://localhost:5000')

# Start an experiment
mlflow.set_experiment('my-model')

with mlflow.start_run():
    # Log parameters
    mlflow.log_param('learning_rate', 0.01)
    mlflow.log_param('epochs', 100)

    # Log metrics
    mlflow.log_metric('accuracy', 0.95)
    mlflow.log_metric('loss', 0.05)

    # Log model
    mlflow.sklearn.log_model(model, 'model')
```

### With Authentication

```python
import mlflow
import os

mlflow.set_tracking_uri('http://localhost:5000')

# Set authentication
os.environ['MLFLOW_TRACKING_USERNAME'] = 'admin'
os.environ['MLFLOW_TRACKING_PASSWORD'] = 'your-password'
```

### Model Registry

```python
# Register a model
mlflow.register_model(
    'runs:/abc123/model',
    'production-model'
)

# Load a model
model = mlflow.pyfunc.load_model('models:/production-model/Production')
```

## R Integration

```r
library(mlflow)

# Connect
mlflow_set_tracking_uri('http://localhost:5000')

# Log run
with(mlflow_start_run(), {
    mlflow_log_param('alpha', 0.5)
    mlflow_log_metric('rmse', 2.5)
    mlflow_log_artifact('model.rds')
})
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MLFLOW_ENABLED` | Enable MLflow | `false` |
| `MLFLOW_PORT` | MLflow port | `5000` |
| `MLFLOW_USERNAME` | Admin username | `admin` |
| `MLFLOW_PASSWORD` | Admin password | auto-generated |

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│   Your Code     │────▶│    MLflow       │
│  (Python/R)     │     │   Tracking      │
└─────────────────┘     │   Server        │
                        └────────┬────────┘
                                 │
                    ┌────────────┼────────────┐
                    ▼            ▼            ▼
              ┌──────────┐ ┌──────────┐ ┌──────────┐
              │PostgreSQL│ │  MinIO   │ │ Artifacts│
              │ Backend  │ │ Storage  │ │  (S3)    │
              └──────────┘ └──────────┘ └──────────┘
```

## UI Features

Access at `https://mlflow.<your-domain>` or `http://localhost:5000`:

- **Experiments**: Compare runs, view metrics
- **Models**: Model registry, versioning
- **Artifacts**: Browse logged files, models
- **Metrics**: Interactive charts, comparisons

## Credentials

Default credentials are generated on first enable:

```bash
# View credentials
cat .env | grep MLFLOW

# Or check status
nself mlflow status
```

## Best Practices

1. **Use experiments** - Group related runs
2. **Log everything** - Parameters, metrics, artifacts
3. **Tag runs** - Add metadata for filtering
4. **Version models** - Use model registry
5. **Set up MinIO** - For reliable artifact storage

## Troubleshooting

### Cannot Connect

```bash
# Check if running
nself status mlflow

# Test connection
nself mlflow test

# View logs
nself mlflow logs
```

### Authentication Issues

```bash
# Verify credentials
nself mlflow status

# Reset password
nself mlflow configure password
```

### Missing Artifacts

Ensure MinIO is enabled:

```bash
# In .env
MINIO_ENABLED=true

# Rebuild
nself build && nself restart
```
