# nself Admin UI

nself Admin is the central web-based management interface for your entire nself deployment. It provides a unified dashboard to monitor, configure, and control all aspects of your Backend-as-a-Service infrastructure.

## Overview

nself Admin is a powerful, extensible administration panel that acts as your command center for:
- Real-time service monitoring and health checks
- Configuration management across all services
- Database administration (PostgreSQL, Redis)
- Log viewing and analysis
- Performance metrics and resource usage
- Service orchestration and scaling
- User and permission management

## Features

### Current Capabilities
- **Service Dashboard** - Real-time status of all running services
- **Configuration Editor** - Modify environment variables and settings
- **Health Monitoring** - Service health checks and uptime tracking
- **Log Viewer** - Centralized log access from all containers
- **Quick Actions** - Start/stop/restart services with one click
- **Resource Metrics** - CPU, memory, disk usage per service

### Planned Features
- **Database Management** - Built-in query editor and table browser (replacing pgAdmin)
- **User Management** - Create and manage authentication users
- **API Explorer** - Test GraphQL and REST endpoints
- **Backup Management** - Schedule and manage database backups
- **Migration Tools** - Database migration and seed management
- **Performance Profiler** - Identify bottlenecks and optimize queries
- **Alert Configuration** - Set up monitoring alerts and notifications
- **Template Manager** - Install and manage custom service templates
- **Security Auditor** - Security scanning and compliance checks

## Configuration

Enable nself Admin in your `.env` file:

```bash
# nself Admin Configuration
NSELF_ADMIN_ENABLED=true
NSELF_ADMIN_PORT=3100
NSELF_ADMIN_ROUTE=admin.${BASE_DOMAIN}
ADMIN_PORT=3005  # Internal port

# Optional: Authentication
NSELF_ADMIN_AUTH_ENABLED=true
NSELF_ADMIN_USERNAME=admin
NSELF_ADMIN_PASSWORD=secure-password-here

# Optional: Advanced Settings
NSELF_ADMIN_THEME=dark
NSELF_ADMIN_LANGUAGE=en
NSELF_ADMIN_TIMEZONE=UTC
NSELF_ADMIN_SESSION_TIMEOUT=3600
```

## Access

After enabling and starting nself Admin:

### Local Development
- URL: `https://admin.local.nself.org`
- Default credentials: `admin` / `admin` (change immediately)

### Production
- URL: `https://admin.<your-domain>`
- Requires authentication setup

## Architecture

nself Admin is built with:
- **Frontend**: React with TypeScript, Material-UI
- **Backend**: Node.js with Express
- **Real-time**: WebSocket connections for live updates
- **Data Source**: Direct Docker API and service APIs

### Integration Points

nself Admin integrates with:
- **Docker API** - Container management and stats
- **PostgreSQL** - Direct database access
- **Hasura** - GraphQL schema introspection
- **Prometheus** - Metrics collection
- **Loki** - Log aggregation
- **Redis** - Cache and session inspection

## Use Cases

### 1. Development Environment Management
- Quickly reset databases
- View real-time logs during debugging
- Modify environment variables without restarting
- Test API endpoints

### 2. Production Monitoring
- Monitor service health and uptime
- Track resource usage trends
- Set up alerts for critical issues
- View aggregated logs

### 3. Database Administration
- Run SQL queries
- View and modify data
- Manage database users and permissions
- Export/import data

### 4. Service Orchestration
- Scale services up/down
- Rolling updates and deployments
- Service dependency management
- Load balancing configuration

## Security

nself Admin implements multiple security layers:
- **Authentication**: JWT-based authentication
- **Authorization**: Role-based access control (RBAC)
- **Encryption**: All traffic over HTTPS
- **Audit Logging**: All actions logged with user attribution
- **Session Management**: Automatic timeout and refresh
- **IP Whitelisting**: Optional IP-based access control

### Security Best Practices
1. Always change default credentials
2. Use strong passwords (minimum 16 characters)
3. Enable 2FA when available
4. Restrict access by IP in production
5. Regular security updates
6. Monitor audit logs

## Comparison with Other Admin Tools

### vs pgAdmin
- **Integrated**: Part of nself ecosystem, not standalone
- **Multi-service**: Manages all services, not just PostgreSQL
- **Lighter**: Lower resource usage
- **Unified Auth**: Single sign-on with nself Auth

### vs Portainer
- **Specialized**: Tailored for nself deployments
- **Simpler**: Focused UI without Docker complexity
- **Integrated Monitoring**: Built-in Prometheus/Grafana integration
- **nself-aware**: Understands nself service relationships

### vs Adminer
- **Modern UI**: React-based responsive interface
- **Multi-database**: PostgreSQL, Redis, and more
- **API Testing**: Includes GraphQL/REST testing
- **Real-time Updates**: WebSocket-based live data

## Customization

### Themes
nself Admin supports custom themes:
```javascript
// Custom theme example
{
  "primary": "#1976d2",
  "secondary": "#dc004e",
  "background": "#f5f5f5",
  "dark": true
}
```

### Plugins
Extend functionality with plugins:
```javascript
// Plugin structure
{
  "name": "custom-monitor",
  "version": "1.0.0",
  "hooks": {
    "dashboard": "renderCustomWidget",
    "menu": "addCustomMenuItem"
  }
}
```

### Custom Widgets
Add dashboard widgets for specific needs:
- Custom metrics displays
- Third-party service integration
- Business-specific KPIs
- Custom action buttons

## API

nself Admin exposes its own API for automation:

```bash
# Get service status
GET /api/services/status

# Restart a service
POST /api/services/{name}/restart

# Run database query
POST /api/database/query
{
  "query": "SELECT * FROM users LIMIT 10"
}

# Get logs
GET /api/logs/{service}?lines=100
```

## Troubleshooting

### Admin UI Not Loading
- Check `NSELF_ADMIN_ENABLED=true` in .env
- Verify port 3100 is not in use
- Check nginx routing configuration
- Ensure Docker socket is accessible

### Cannot Connect to Services
- Verify Docker network configuration
- Check service health endpoints
- Ensure proper environment variables
- Review firewall rules

### Authentication Issues
- Reset admin password via CLI: `nself admin reset-password`
- Check JWT secret configuration
- Verify session timeout settings
- Clear browser cache and cookies

## CLI Integration

Manage nself Admin from the command line:

```bash
# Enable admin UI
nself admin enable

# Disable admin UI
nself admin disable

# Reset admin password
nself admin reset-password

# View admin logs
nself admin logs

# Check admin status
nself admin status
```

## Resource Requirements

- **CPU**: 0.25 cores minimum
- **Memory**: 256MB minimum, 512MB recommended
- **Storage**: 100MB for application, 1GB for logs/metrics
- **Network**: Low bandwidth, increases with monitoring

## Future Roadmap

### Q1 2025
- Database query builder UI
- Advanced user management
- API documentation generator

### Q2 2025
- ML model management interface
- Kubernetes deployment support
- Multi-tenant administration

### Q3 2025
- Mobile app for monitoring
- AI-powered insights
- Automated optimization

## Related Documentation

- [Services Overview](SERVICES)
- [Optional Services](SERVICES_OPTIONAL)
- [Monitoring Bundle](MONITORING_BUNDLE)
- [Environment Configuration](ENVIRONMENT-VARIABLES)
- [Troubleshooting](TROUBLESHOOTING)