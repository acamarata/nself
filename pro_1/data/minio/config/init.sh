#!/bin/bash

# Wait for MinIO to be ready
until curl -s http://localhost:9000/minio/health/live; do
  echo "Waiting for MinIO to be ready..."
  sleep 1
done

# Create buckets
mc alias set myminio http://localhost:9000 ${MINIO_ROOT_USER:-minioadmin} ${MINIO_ROOT_PASSWORD:-minioadmin}

# Create public bucket
mc mb myminio/public
mc policy set public myminio/public

# Create private bucket
mc mb myminio/private
mc policy set private myminio/private

echo "MinIO initialization completed"
