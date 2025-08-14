#!/usr/bin/env bash

# dockerfile-generator.sh - Auto-generate missing Dockerfiles for any service

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../utils/display.sh" 2>/dev/null || true

# Generate appropriate Dockerfile based on service name and context
generate_dockerfile_for_service() {
    local service_name="$1"
    local service_path="${2:-./$service_name}"
    
    log_info "Auto-generating Dockerfile for service: $service_name"
    
    # Create directory if it doesn't exist
    mkdir -p "$service_path"
    
    # Determine service type based on name and generate appropriate files
    case "$service_name" in
        functions)
            generate_functions_service "$service_path"
            ;;
        config-server)
            generate_config_server "$service_path"
            ;;
        dashboard)
            generate_dashboard_service "$service_path"
            ;;
        auth)
            generate_auth_service "$service_path"
            ;;
        storage)
            generate_storage_service "$service_path"
            ;;
        hasura)
            generate_hasura_service "$service_path"
            ;;
        *)
            # Default to a basic Node.js service
            generate_generic_node_service "$service_name" "$service_path"
            ;;
    esac
    
    return 0
}

# Generate functions service
generate_functions_service() {
    local path="$1"
    
    cat > "$path/Dockerfile" << 'EOF'
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY . .
EXPOSE 3000
CMD ["node", "index.js"]
EOF

    cat > "$path/package.json" << 'EOF'
{
  "name": "functions",
  "version": "1.0.0",
  "description": "Serverless functions",
  "scripts": {
    "start": "node index.js",
    "dev": "nodemon index.js"
  },
  "dependencies": {
    "express": "^4.18.0",
    "cors": "^2.8.5"
  },
  "devDependencies": {
    "nodemon": "^3.0.0"
  }
}
EOF

    cat > "$path/index.js" << 'EOF'
const express = require('express');
const cors = require('cors');
const app = express();
const port = process.env.FUNCTIONS_PORT || 3000;

app.use(cors());
app.use(express.json());

app.get('/', (req, res) => {
  res.json({ message: 'Functions service ready', version: '1.0.0' });
});

app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'functions' });
});

// Add your serverless functions here
app.post('/functions/:name', async (req, res) => {
  const { name } = req.params;
  res.json({ 
    function: name,
    result: 'Function executed successfully',
    timestamp: new Date()
  });
});

app.listen(port, () => {
  console.log(`Functions service listening on port ${port}`);
});
EOF
    
    log_success "Generated functions service at: $path"
}

# Generate config-server service
generate_config_server() {
    local path="$1"
    
    cat > "$path/Dockerfile" << 'EOF'
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY . .
EXPOSE 4001
CMD ["node", "index.js"]
EOF

    cat > "$path/package.json" << 'EOF'
{
  "name": "config-server",
  "version": "1.0.0",
  "description": "Configuration server",
  "scripts": {
    "start": "node index.js"
  },
  "dependencies": {
    "express": "^4.18.0",
    "dotenv": "^16.0.0"
  }
}
EOF

    cat > "$path/index.js" << 'EOF'
const express = require('express');
const app = express();
const port = process.env.CONFIG_SERVER_PORT || 4001;

app.use(express.json());

// Configuration endpoint
app.get('/config', (req, res) => {
  res.json({
    environment: process.env.NODE_ENV || 'development',
    services: {
      hasura: process.env.HASURA_ENDPOINT,
      auth: process.env.AUTH_ENDPOINT,
      storage: process.env.STORAGE_ENDPOINT
    }
  });
});

app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'config-server' });
});

// Also support /healthz endpoint (common k8s convention)
app.get('/healthz', (req, res) => {
  res.json({ status: 'ok', service: 'config-server' });
});

app.listen(port, () => {
  console.log(`Config server listening on port ${port}`);
});
EOF
    
    log_success "Generated config-server at: $path"
}

# Generate dashboard service
generate_dashboard_service() {
    local path="$1"
    
    cat > "$path/Dockerfile" << 'EOF'
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
EOF

    cat > "$path/package.json" << 'EOF'
{
  "name": "dashboard",
  "version": "1.0.0",
  "description": "Admin dashboard",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "vite": "^4.0.0",
    "@vitejs/plugin-react": "^3.0.0"
  }
}
EOF

    cat > "$path/nginx.conf" << 'EOF'
server {
    listen 80;
    location / {
        root /usr/share/nginx/html;
        index index.html;
        try_files $uri $uri/ /index.html;
    }
}
EOF

    # Create a simple index.html
    mkdir -p "$path/dist"
    cat > "$path/dist/index.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Dashboard</title>
</head>
<body>
    <h1>Dashboard</h1>
    <p>Dashboard service is running</p>
</body>
</html>
EOF
    
    log_success "Generated dashboard service at: $path"
}

# Generate auth service placeholder
generate_auth_service() {
    local path="$1"
    
    # Auth is usually hasura-auth, just create a marker
    cat > "$path/Dockerfile" << 'EOF'
# Auth service is provided by hasura-auth image
# This is a placeholder for docker-compose compatibility
FROM busybox:latest
CMD ["echo", "Auth service uses hasura-auth image"]
EOF
    
    log_success "Generated auth service placeholder at: $path"
}

# Generate storage service placeholder
generate_storage_service() {
    local path="$1"
    
    # Storage is usually hasura-storage, just create a marker
    cat > "$path/Dockerfile" << 'EOF'
# Storage service is provided by hasura-storage image
# This is a placeholder for docker-compose compatibility
FROM busybox:latest
CMD ["echo", "Storage service uses hasura-storage image"]
EOF
    
    log_success "Generated storage service placeholder at: $path"
}

# Generate hasura service placeholder
generate_hasura_service() {
    local path="$1"
    
    # Hasura uses official image, just create a marker
    cat > "$path/Dockerfile" << 'EOF'
# Hasura service is provided by hasura/graphql-engine image
# This is a placeholder for docker-compose compatibility
FROM busybox:latest
CMD ["echo", "Hasura service uses official hasura image"]
EOF
    
    log_success "Generated hasura service placeholder at: $path"
}

# Generate generic Node.js service
generate_generic_node_service() {
    local service_name="$1"
    local path="$2"
    
    cat > "$path/Dockerfile" << 'EOF'
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY . .
EXPOSE 3000
CMD ["node", "index.js"]
EOF

    cat > "$path/package.json" << EOF
{
  "name": "$service_name",
  "version": "1.0.0",
  "description": "$service_name service",
  "scripts": {
    "start": "node index.js"
  },
  "dependencies": {
    "express": "^4.18.0"
  }
}
EOF

    cat > "$path/index.js" << EOF
const express = require('express');
const app = express();
const port = process.env.PORT || 3000;

app.get('/', (req, res) => {
  res.json({ message: '${service_name} service ready' });
});

app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: '${service_name}' });
});

app.listen(port, () => {
  console.log('${service_name} service listening on port ' + port);
});
EOF
    
    log_success "Generated generic service '$service_name' at: $path"
}

# Export functions
export -f generate_dockerfile_for_service