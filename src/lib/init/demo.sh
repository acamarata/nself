#!/usr/bin/env bash
# demo.sh - Demo setup functionality for nself init --demo
#
# Creates a complete demo environment with all services enabled,
# custom backend services, frontend apps, and remote schemas

# Source required utilities
DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$DEMO_DIR/../utils/display.sh" 2>/dev/null || true
source "$DEMO_DIR/../utils/env.sh" 2>/dev/null || true
source "$DEMO_DIR/gitignore.sh" 2>/dev/null || true

# Setup complete demo environment
# Inputs: $1 - script directory
# Outputs: Creates demo configuration files
# Returns: 0 on success, error code on failure
setup_demo() {
  local script_dir="${1:-$DEMO_DIR}"
  # Find the templates directory relative to the init module
  local templates_dir="$(cd "$DEMO_DIR" && cd ../../templates/demo && pwd)"
  local current_dir="$(pwd)"

  # Show demo header
  clear
  echo ""
  echo "════════════════════════════════════════════════════════════════"
  echo "                     🚀 nself Demo Setup"
  echo "════════════════════════════════════════════════════════════════"
  echo ""
  echo "This will create a complete demo application with:"
  echo ""
  echo "  ✨ All core services (PostgreSQL, Hasura, Auth, Nginx)"
  echo "  📦 All optional services enabled:"
  echo "     • Storage (MinIO)      • Redis & BullMQ"
  echo "     • Search (MeiliSearch) • Email (MailPit)"
  echo "     • MLflow              • Temporal"
  echo "     • Monitoring          • Functions"
  echo ""
  echo "  🔧 2 Custom backend services:"
  echo "     • api-service    - REST/GraphQL API (Node.js)"
  echo "     • worker-service - Background jobs (Python)"
  echo ""
  echo "  🎨 2 Frontend applications:"
  echo "     • main-app     - Customer app (Next.js)"
  echo "     • admin-portal - Admin portal (React+Vite)"
  echo ""
  echo "  🔗 Remote schemas configured for GraphQL federation"
  echo "  📊 Demo data and seed content"
  echo ""
  echo "════════════════════════════════════════════════════════════════"
  echo ""

  # Check if already initialized
  if [[ -f ".env" ]] || [[ -f ".env.dev" ]] || [[ -f "docker-compose.yml" ]]; then
    log_warning "Project appears to be already initialized"
    echo ""
    echo "Found existing files:"
    [[ -f ".env" ]] && echo "  • .env"
    [[ -f ".env.dev" ]] && echo "  • .env.dev"
    [[ -f "docker-compose.yml" ]] && echo "  • docker-compose.yml"
    echo ""
    echo -n "Do you want to continue and overwrite? (y/N): "
    read confirm
    if [[ "$confirm" != "y" ]] && [[ "$confirm" != "Y" ]]; then
      log_info "Demo setup cancelled"
      return 1
    fi
    echo ""
  fi

  # Check templates exist
  if [[ ! -f "$templates_dir/.env.demo" ]]; then
    log_error "Demo templates not found at $templates_dir"
    log_info "Please ensure nself is properly installed"
    return 1
  fi

  log_info "Creating demo configuration..."
  echo ""

  # Copy demo environment file
  cp "$templates_dir/.env.demo" .env.dev
  log_success "Created .env.dev with all services enabled"

  # Create local .env for overrides
  cat > .env << 'EOF'
# Local Configuration Overrides for Demo
# Add any personal overrides here

# Uncomment to change the default domain
# BASE_DOMAIN=localhost

# Uncomment to change default ports
# POSTGRES_PORT=5433
# REDIS_PORT=6380
EOF
  log_success "Created .env for local overrides"

  # Create .env.example from demo
  cp "$templates_dir/.env.demo" .env.example
  log_success "Created .env.example reference"

  # Ensure gitignore
  if [[ -f "$script_dir/gitignore.sh" ]]; then
    source "$script_dir/gitignore.sh"
    ensure_gitignore
    log_success "Created .gitignore with security rules"
  fi

  # Create placeholder directories for services
  mkdir -p services/api-service
  mkdir -p services/worker-service
  mkdir -p frontend/main-app
  mkdir -p frontend/admin-portal
  mkdir -p hasura/metadata
  mkdir -p hasura/migrations
  mkdir -p hasura/seeds

  # Create basic service files
  create_demo_service_files

  # Create basic frontend files
  create_demo_frontend_files

  # Create demo schema
  create_demo_schema

  log_success "Demo structure created"
  echo ""

  # Show completion message
  echo "════════════════════════════════════════════════════════════════"
  echo ""
  log_success "Demo Setup Complete!"
  echo ""
  echo "📁 Created Structure:"
  echo "   ."
  echo "   ├── .env.dev           # Demo configuration"
  echo "   ├── .env              # Local overrides"
  echo "   ├── services/"
  echo "   │   ├── api-service/  # Node.js API"
  echo "   │   └── worker-service/ # Python worker"
  echo "   ├── frontend/"
  echo "   │   ├── main-app/     # Next.js app"
  echo "   │   └── admin-portal/ # React admin"
  echo "   └── hasura/"
  echo "       ├── metadata/     # GraphQL config"
  echo "       ├── migrations/   # Database schema"
  echo "       └── seeds/       # Demo data"
  echo ""
  echo "════════════════════════════════════════════════════════════════"
  echo ""
  echo "🚀 Next Steps:"
  echo ""
  echo "   1. Build the demo:"
  echo "      $ nself build"
  echo ""
  echo "   2. Start all services:"
  echo "      $ nself start"
  echo ""
  echo "   3. Access your services:"
  echo ""
  echo "   Core Services:"
  echo "      ✅ Hasura API:         https://api.local.nself.org/graphql"
  echo "      ✅ Hasura Console:     https://hasura.local.nself.org"
  echo "      ✅ Admin Dashboard:     https://admin.local.nself.org"
  echo "      ✅ Auth Service:        https://auth.local.nself.org"
  echo ""
  echo "   Optional Services:"
  echo "      ✅ Email UI:           https://mailpit.local.nself.org"
  echo "      ✅ Search UI:          https://search.local.nself.org"
  echo "      ✅ Storage (MinIO):    https://storage.local.nself.org"
  echo "      ✅ Monitoring:         https://grafana.local.nself.org"
  echo "      ✅ BullMQ Dashboard:   https://bullmq.local.nself.org"
  echo ""
  echo "   Custom Services:"
  echo "      ✅ API Service:        https://api-service.local.nself.org"
  echo "      ✅ Worker Service:     https://worker-service.local.nself.org"
  echo ""
  echo "   Frontend Applications:"
  echo "      ✅ Main App:           https://app.local.nself.org"
  echo "         • API endpoint:     https://api.app.local.nself.org/graphql"
  echo "         • Auth endpoint:    https://auth.app.local.nself.org"
  echo ""
  echo "      ✅ Admin Portal:       https://portal.local.nself.org"
  echo "         • API endpoint:     https://api.portal.local.nself.org/graphql"
  echo "         • Auth endpoint:    https://auth.portal.local.nself.org"
  echo ""
  echo "   4. Demo Credentials:"
  echo "      • Database:            postgres / demo-password"
  echo "      • Hasura Admin:        demo-admin-secret"
  echo "      • Grafana:             admin / demo-grafana-password"
  echo "      • MinIO:               minioadmin / minioadmin"
  echo ""
  echo "════════════════════════════════════════════════════════════════"
  echo ""
  echo "📚 Documentation: https://docs.nself.org"
  echo "💬 Support: https://github.com/nself/nself/discussions"
  echo ""

  return 0
}

# Create demo service files
create_demo_service_files() {
  # API Service (Node.js)
  cat > services/api-service/package.json << 'EOF'
{
  "name": "api-service",
  "version": "1.0.0",
  "description": "Demo API service for nself",
  "main": "index.js",
  "scripts": {
    "start": "node index.js",
    "dev": "nodemon index.js"
  },
  "dependencies": {
    "express": "^4.18.0",
    "express-graphql": "^0.12.0",
    "graphql": "^16.0.0",
    "pg": "^8.0.0"
  },
  "devDependencies": {
    "nodemon": "^2.0.0"
  }
}
EOF

  cat > services/api-service/index.js << 'EOF'
const express = require('express');
const { graphqlHTTP } = require('express-graphql');
const { buildSchema } = require('graphql');

const app = express();

// Demo GraphQL schema
const schema = buildSchema(`
  type Query {
    hello: String
    apiStatus: String
    timestamp: String
  }
`);

const root = {
  hello: () => 'Hello from API Service!',
  apiStatus: () => 'Running',
  timestamp: () => new Date().toISOString()
};

app.use('/graphql', graphqlHTTP({
  schema: schema,
  rootValue: root,
  graphiql: true,
}));

app.get('/health', (req, res) => {
  res.json({ status: 'healthy', service: 'api-service' });
});

const PORT = process.env.SERVICE_1_PORT || 8001;
app.listen(PORT, () => {
  console.log(`API Service running on port ${PORT}`);
});
EOF

  cat > services/api-service/Dockerfile << 'EOF'
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 8001
CMD ["npm", "start"]
EOF

  # Worker Service (Python)
  cat > services/worker-service/requirements.txt << 'EOF'
fastapi==0.100.0
uvicorn==0.23.0
psycopg2-binary==2.9.0
redis==4.5.0
graphene==3.2.0
starlette-graphene3==0.6.0
EOF

  cat > services/worker-service/main.py << 'EOF'
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import graphene
from starlette_graphene3 import GraphQLApp

app = FastAPI(title="Worker Service")

# Enable CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

# Demo GraphQL schema
class Query(graphene.ObjectType):
    worker_status = graphene.String()
    job_count = graphene.Int()

    def resolve_worker_status(self, info):
        return "Active"

    def resolve_job_count(self, info):
        return 42

schema = graphene.Schema(query=Query)

# Add GraphQL endpoint
app.add_route("/graphql", GraphQLApp(schema=schema))

@app.get("/health")
def health_check():
    return {"status": "healthy", "service": "worker-service"}

@app.get("/")
def root():
    return {"message": "Worker Service Demo", "version": "1.0.0"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8002)
EOF

  cat > services/worker-service/Dockerfile << 'EOF'
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8002
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8002"]
EOF
}

# Create demo frontend files
create_demo_frontend_files() {
  # Main App (Next.js)
  cat > frontend/main-app/package.json << 'EOF'
{
  "name": "main-app",
  "version": "1.0.0",
  "description": "Demo main application",
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start"
  },
  "dependencies": {
    "next": "13.4.0",
    "react": "18.2.0",
    "react-dom": "18.2.0",
    "@apollo/client": "3.7.0",
    "graphql": "16.6.0"
  }
}
EOF

  cat > frontend/main-app/.env.local << 'EOF'
# Main App Configuration
NEXT_PUBLIC_GRAPHQL_ENDPOINT=https://api.app.local.nself.org/graphql
NEXT_PUBLIC_AUTH_ENDPOINT=https://auth.app.local.nself.org
NEXT_PUBLIC_APP_NAME=Main Application
NEXT_PUBLIC_APP_URL=https://app.local.nself.org
EOF

  cat > frontend/main-app/Dockerfile << 'EOF'
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 3000
CMD ["npm", "run", "dev"]
EOF

  # Admin Portal (React + Vite)
  cat > frontend/admin-portal/package.json << 'EOF'
{
  "name": "admin-portal",
  "version": "1.0.0",
  "description": "Demo admin portal",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "18.2.0",
    "react-dom": "18.2.0",
    "@apollo/client": "3.7.0",
    "graphql": "16.6.0"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "4.0.0",
    "vite": "4.3.0"
  }
}
EOF

  cat > frontend/admin-portal/.env.local << 'EOF'
# Admin Portal Configuration
VITE_GRAPHQL_ENDPOINT=https://api.portal.local.nself.org/graphql
VITE_AUTH_ENDPOINT=https://auth.portal.local.nself.org
VITE_APP_NAME=Admin Portal
VITE_APP_URL=https://portal.local.nself.org
EOF

  cat > frontend/admin-portal/Dockerfile << 'EOF'
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 3001
CMD ["npm", "run", "dev"]
EOF
}

# Create demo database schema
create_demo_schema() {
  cat > hasura/seeds/demo_data.sql << 'EOF'
-- Demo seed data for nself demo application

-- Create demo tables
CREATE TABLE IF NOT EXISTS users (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    total DECIMAL(10,2),
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert demo users
INSERT INTO users (email, name) VALUES
    ('demo@example.com', 'Demo User'),
    ('admin@example.com', 'Admin User'),
    ('test@example.com', 'Test User');

-- Insert demo products
INSERT INTO products (name, description, price) VALUES
    ('Product A', 'Description for Product A', 29.99),
    ('Product B', 'Description for Product B', 49.99),
    ('Product C', 'Description for Product C', 99.99);

-- Insert demo orders
INSERT INTO orders (user_id, total, status)
SELECT
    u.id,
    (RANDOM() * 200 + 50)::DECIMAL(10,2),
    CASE WHEN RANDOM() > 0.5 THEN 'completed' ELSE 'pending' END
FROM users u
CROSS JOIN generate_series(1, 3);
EOF
}

# Export the function
export -f setup_demo
export -f create_demo_service_files
export -f create_demo_frontend_files
export -f create_demo_schema