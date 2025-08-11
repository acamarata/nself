const express = require('express');
const cors = require('cors');
const app = express();

app.use(cors());
app.use(express.json());

// Mock project configuration
const projectConfig = {
  project: {
    id: 'local-dev',
    name: process.env.PROJECT_NAME || 'nself-project',
    subdomain: 'local',
    region: 'local',
    createdAt: new Date().toISOString(),
    plan: 'local-dev',
    desiredState: 'running',
    nhostBaseFolder: '',
    repositoryProductionBranch: 'main',
    postgresVersion: '16',
    isProvisioned: true
  },
  config: {
    hasura: {
      version: process.env.HASURA_VERSION || 'v2.44.0',
      adminSecret: process.env.HASURA_GRAPHQL_ADMIN_SECRET || 'dev-secret',
      webhookSecret: null,
      jwtSecrets: [
        {
          type: 'HS256',
          key: process.env.JWT_KEY || 'default-jwt-key'
        }
      ],
      url: `https://api.${process.env.BASE_DOMAIN || 'local.nself.org'}`,
      consoleEnabled: true,
      cors: {
        domain: ['*']
      },
      environment: {},
      globalEnvironment: {}
    },
    functions: {
      url: `https://functions.${process.env.BASE_DOMAIN || 'local.nself.org'}`,
      enabled: true
    },
    auth: {
      version: process.env.AUTH_VERSION || '0.36.0',
      url: `https://auth.${process.env.BASE_DOMAIN || 'local.nself.org'}`,
      redirections: {
        clientUrl: 'https://localhost:3000',
        allowedUrls: process.env.AUTH_CLIENT_URL ? process.env.AUTH_CLIENT_URL.split(',') : []
      },
      signUp: {
        enabled: true
      },
      user: {
        roles: {
          default: 'user',
          allowed: ['user', 'admin']
        },
        locale: {
          default: 'en',
          allowed: ['en']
        }
      },
      session: {
        accessToken: {
          expiresIn: parseInt(process.env.AUTH_JWT_ACCESS_TOKEN_EXPIRES_IN || '900')
        },
        refreshToken: {
          expiresIn: parseInt(process.env.AUTH_JWT_REFRESH_TOKEN_EXPIRES_IN || '2592000')
        }
      },
      method: {
        emailPassword: {
          emailVerificationRequired: false,
          passwordMinLength: 8
        },
        emailOtp: { enabled: false },
        anonymous: { enabled: false },
        webauthn: { enabled: false },
        oauth: {
          apple: { enabled: false },
          google: { enabled: false },
          github: { enabled: false },
          facebook: { enabled: false },
          linkedin: { enabled: false }
        }
      }
    },
    postgres: {
      version: process.env.POSTGRES_VERSION || '16',
      database: process.env.POSTGRES_DB || 'nhost',
      host: process.env.POSTGRES_HOST || 'postgres',
      port: parseInt(process.env.POSTGRES_PORT || '5432')
    },
    provider: {
      smtp: {
        host: process.env.AUTH_SMTP_HOST || 'mailpit',
        port: parseInt(process.env.AUTH_SMTP_PORT || '1025'),
        secure: false,
        sender: process.env.AUTH_SMTP_SENDER || 'noreply@local.nself.org'
      }
    },
    storage: {
      version: process.env.STORAGE_VERSION || '0.6.1',
      url: `https://storage.${process.env.BASE_DOMAIN || 'local.nself.org'}`
    },
    observability: {
      grafana: {
        adminPassword: ''
      }
    }
  },
  systemConfig: {
    postgres: {
      connectionString: {
        admin: `postgres://postgres:${process.env.POSTGRES_PASSWORD}@postgres:5432/${process.env.POSTGRES_DB}`,
        user: `postgres://postgres:${process.env.POSTGRES_PASSWORD}@postgres:5432/${process.env.POSTGRES_DB}`
      },
      resources: {
        compute: {
          cpu: 500,
          memory: 1024
        }
      }
    },
    hasura: {
      resources: {
        compute: {
          cpu: 500,
          memory: 1024
        }
      }
    },
    auth: {
      resources: {
        compute: {
          cpu: 500,
          memory: 512
        }
      }
    },
    storage: {
      resources: {
        compute: {
          cpu: 500,
          memory: 512
        }
      }
    }
  },
  services: [
    {
      name: 'hasura',
      url: `https://api.${process.env.BASE_DOMAIN || 'local.nself.org'}`,
      type: 'hasura'
    },
    {
      name: 'auth',
      url: `https://auth.${process.env.BASE_DOMAIN || 'local.nself.org'}`,
      type: 'auth'
    },
    {
      name: 'storage',
      url: `https://storage.${process.env.BASE_DOMAIN || 'local.nself.org'}`,
      type: 'storage'
    },
    {
      name: 'functions',
      url: `https://functions.${process.env.BASE_DOMAIN || 'local.nself.org'}`,
      type: 'functions'
    }
  ]
};

// Health check
app.get('/healthz', (req, res) => {
  res.json({ status: 'healthy' });
});

// Main config endpoint
app.get('/v1/config', (req, res) => {
  res.json(projectConfig);
});

// Project info endpoint
app.get('/v1/project', (req, res) => {
  res.json(projectConfig.project);
});

// Services endpoint
app.get('/v1/services', (req, res) => {
  res.json(projectConfig.services);
});

// Auth config endpoint
app.get('/v1/auth/config', (req, res) => {
  res.json(projectConfig.config.auth);
});

// Hasura config endpoint
app.get('/v1/hasura/config', (req, res) => {
  res.json(projectConfig.config.hasura);
});

// Storage config endpoint
app.get('/v1/storage/config', (req, res) => {
  res.json(projectConfig.config.storage);
});

// Functions config endpoint
app.get('/v1/functions/config', (req, res) => {
  res.json(projectConfig.config.functions);
});

// Update config endpoints (for dashboard to save changes)
app.post('/v1/config', (req, res) => {
  console.log('Config update received:', req.body);
  // TODO: Update .env.local with new values
  res.json({ success: true, message: 'Config updated' });
});

app.patch('/v1/config', (req, res) => {
  console.log('Config patch received:', req.body);
  // TODO: Patch .env.local with changes
  res.json({ success: true, message: 'Config patched' });
});

// Handle all other routes
app.use('*', (req, res) => {
  console.log(`Unhandled route: ${req.method} ${req.originalUrl}`);
  res.status(404).json({ error: 'Not found', path: req.originalUrl });
});

const PORT = process.env.PORT || 4001;
app.listen(PORT, '0.0.0.0', () => {
  console.log(`Config server running on port ${PORT}`);
  console.log('Available endpoints:');
  console.log('  GET /v1/config - Main configuration');
  console.log('  GET /v1/project - Project info');
  console.log('  GET /v1/services - Services list');
  console.log('  GET /v1/auth/config - Auth configuration');
  console.log('  GET /v1/hasura/config - Hasura configuration');
  console.log('  GET /v1/storage/config - Storage configuration');
  console.log('  GET /v1/functions/config - Functions configuration');
});