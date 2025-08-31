# nself Command Tree (v0.3.9-beta)

## Complete Command List (34 Commands)

```
nself
├── 🚀 Core Commands (8)
│   ├── init          Initialize a new project
│   ├── build         Build project structure and Docker images
│   ├── start         Start all services
│   ├── stop          Stop all services
│   ├── restart       Restart all or specific services
│   ├── status        Show service status and health
│   ├── logs          View service logs
│   └── clean         Clean up Docker resources
│
├── 📊 Database & Backup (2)
│   ├── db            Database operations (migrations, seeds, etc.)
│   └── backup        Comprehensive backup system
│       ├── create    Create backups
│       ├── restore   Restore from backup
│       ├── list      List available backups
│       ├── verify    Verify backup integrity
│       └── prune     Remove old backups
│
├── 🔧 Configuration (6)
│   ├── validate      Validate configuration files
│   ├── ssl           SSL certificate management
│   ├── trust         Install SSL certificates locally
│   ├── email         Email service configuration
│   ├── prod          Configure for production
│   └── urls          Show service URLs
│
├── 🎯 Admin & Monitoring (5)
│   ├── admin         Admin UI management
│   │   ├── enable    Enable admin dashboard
│   │   ├── disable   Disable admin dashboard
│   │   ├── status    Show admin configuration
│   │   ├── password  Set admin password
│   │   ├── reset     Reset admin to defaults
│   │   ├── logs      View admin service logs
│   │   └── open      Open admin UI in browser
│   ├── doctor        System diagnostics and fixes
│   ├── monitor       Real-time monitoring
│   ├── metrics       Metrics collection and export
│   └── mlflow        ML experiment tracking
│
├── 🚢 Deployment & Scaling (4)
│   ├── deploy        Deploy to remote servers
│   ├── scale         Scale services up/down
│   ├── rollback      Rollback to previous version
│   └── update        Update nself CLI
│
├── 🛠️ Development Tools (5)
│   ├── exec          Execute commands in containers
│   ├── diff          Show configuration changes
│   ├── reset         Reset project to clean state
│   ├── scaffold      Generate new service from template
│   └── search        Search service management
│
├── 📝 Utility Commands (4)
│   ├── version       Show version information
│   ├── help          Display help information
│   ├── up            Alias for 'start' (compatibility)
│   └── down          Alias for 'stop' (compatibility)
```

## Command Categories

### Production-Ready Commands ✅
All 34 commands are fully implemented and production-ready:
- Core infrastructure management
- Database operations with migrations
- Comprehensive backup system
- SSL certificate generation
- Email configuration with 16+ providers
- Admin UI with monitoring
- Deployment automation

### Commands with Subcommands

#### admin (7 subcommands)
- `nself admin enable` - Enable admin UI
- `nself admin disable` - Disable admin UI
- `nself admin status` - Show configuration
- `nself admin password <pass>` - Set password
- `nself admin reset` - Reset to defaults
- `nself admin logs` - View logs
- `nself admin open` - Open in browser

#### backup (10 subcommands)
- `nself backup create` - Create backup
- `nself backup restore <file>` - Restore backup
- `nself backup list` - List backups
- `nself backup verify` - Verify integrity
- `nself backup prune` - Remove old backups
- `nself backup schedule` - Schedule backups
- `nself backup export` - Export to cloud
- `nself backup import` - Import from cloud
- `nself backup snapshot` - Point-in-time snapshot
- `nself backup rollback` - Rollback to snapshot

#### db (Multiple operations)
- `nself db` - Interactive menu
- `nself db migrate` - Run migrations
- `nself db seed` - Apply seed data
- `nself db reset` - Reset database
- `nself db status` - Check status
- `nself db console` - Open SQL console

#### email (6 subcommands)
- `nself email setup` - Interactive wizard
- `nself email list` - List providers
- `nself email configure` - Configure provider
- `nself email validate` - Check config
- `nself email test` - Send test email
- `nself email docs` - Provider documentation

#### ssl (6 subcommands)
- `nself ssl bootstrap` - Generate certificates
- `nself ssl renew` - Renew certificates
- `nself ssl status` - Check status
- `nself ssl auto-renew` - Enable auto-renewal
- `nself ssl schedule` - Schedule renewal
- `nself ssl unschedule` - Remove schedule

## Quick Command Reference

### Starting a New Project
```bash
nself init          # Initialize project
nself build         # Build configuration
nself start         # Start services
nself status        # Check health
```

### Daily Operations
```bash
nself logs          # View logs
nself restart       # Restart services
nself exec postgres # Access container
nself doctor        # Run diagnostics
```

### Production Deployment
```bash
nself prod          # Production config
nself ssl bootstrap # Generate SSL
nself deploy        # Deploy to server
nself backup create # Create backup
```

### Maintenance
```bash
nself update        # Update CLI
nself clean         # Clean resources
nself reset         # Factory reset
nself rollback      # Rollback version
```

## Notes

- **Aliases**: `up` → `start`, `down` → `stop` (for Docker Compose familiarity)
- **Admin UI**: Available at localhost:3100 when enabled
- **SSL**: Automatic generation with mkcert
- **Backups**: Support for local and S3 storage
- **Email**: SMTP testing with swaks container