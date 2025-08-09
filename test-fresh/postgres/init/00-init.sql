-- Initialize database
CREATE DATABASE IF NOT EXISTS nhost;

-- Enable extensions if specified
\c nhost;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create schemas for hasura-storage and hasura-auth
CREATE SCHEMA IF NOT EXISTS storage;
CREATE SCHEMA IF NOT EXISTS auth;
