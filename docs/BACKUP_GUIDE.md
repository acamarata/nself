# nself Backup & Restore Guide

Complete guide to backing up and restoring your nself applications.

## Table of Contents

- [Quick Start](#quick-start)
- [Backup Types](#backup-types)
- [Cloud Storage Setup](#cloud-storage-setup)
- [Retention Policies](#retention-policies)
- [Automated Backups](#automated-backups)
- [Disaster Recovery](#disaster-recovery)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Create Your First Backup

```bash
# Create a full backup (recommended)
nself backup create

# Create with custom name
nself backup create full before-update-v2
```

### List Available Backups

```bash
nself backup list

# Output:
# Local Backups:
# Name                                     Size       Created
# ────                                     ────       ───────
# nself_backup_full_20240115_143022.tar.gz  245MB    2024-01-15 14:30
# nself_backup_database_20240114_090000.tar.gz  12MB     2024-01-14 09:00
```

### Restore from Backup

```bash
# Restore everything
nself backup restore nself_backup_full_20240115_143022.tar.gz

# Restore only database
nself backup restore nself_backup_full_20240115_143022.tar.gz database
```

## Backup Types

### Full Backup (Default)

Includes everything needed to restore your application:

- PostgreSQL databases (all schemas and data)
- Environment files (.env, .env.local, .env.production)
- Docker-compose configurations
- Docker volumes (all persistent data)
- SSL certificates
- Hasura metadata
- Nginx configurations

```bash
nself backup create full
```

### Database Backup

Database and related metadata only:

- PostgreSQL full dump
- Hasura metadata
- Database schemas and migrations

```bash
nself backup create database
```

### Configuration Backup

Settings and configuration files only:

- Environment files
- Docker-compose files
- Nginx configurations
- No data or volumes

```bash
nself backup create config
```

## Cloud Storage Setup

### Interactive Setup Wizard

The easiest way to configure cloud backups:

```bash
nself backup cloud setup

# Select from:
# 1) Amazon S3
# 2) Dropbox
# 3) Google Drive
# 4) OneDrive
# 5) rclone (40+ providers)
# 6) None (disable)
```

### Amazon S3 / MinIO

```bash
# Configure S3
export S3_BUCKET=my-backups
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=yyy

# For MinIO or S3-compatible
export S3_ENDPOINT=https://minio.mycompany.com

# Test connection
nself backup cloud test
```

### Dropbox

1. Get access token from https://www.dropbox.com/developers/apps
2. Configure:

```bash
export BACKUP_CLOUD_PROVIDER=dropbox
export DROPBOX_TOKEN=your-token-here
export DROPBOX_FOLDER=/nself-backups
```

### Google Drive

Install gdrive CLI first:

```bash
# Download from https://github.com/prasmussen/gdrive
# Or use rclone instead (recommended)

export BACKUP_CLOUD_PROVIDER=gdrive
export GDRIVE_FOLDER_ID=folder-id-optional
```

### OneDrive

Requires rclone:

```bash
# Install rclone
curl https://rclone.org/install.sh | sudo bash

# Configure
rclone config create onedrive onedrive

export BACKUP_CLOUD_PROVIDER=onedrive
export ONEDRIVE_FOLDER=nself-backups
```

### Universal Solution with rclone

Supports 40+ cloud providers:

```bash
# Install rclone
curl https://rclone.org/install.sh | sudo bash

# Configure your provider
rclone config

# Set environment
export BACKUP_CLOUD_PROVIDER=rclone
export RCLONE_REMOTE=myremote
export RCLONE_PATH=nself-backups
```

## Retention Policies

### Simple Age-Based

Remove backups older than X days:

```bash
# Default 30 days
nself backup prune

# Custom retention
nself backup prune age 7  # Keep last 7 days
nself backup prune age 90 # Keep last 90 days

# Minimum backup protection
export BACKUP_RETENTION_MIN=3  # Always keep at least 3 backups
```

### Grandfather-Father-Son (GFS)

Enterprise retention strategy:

```bash
nself backup prune gfs

# Keeps:
# - Last 7 daily backups
# - Last 4 weekly backups (Sundays)
# - Last 12 monthly backups (1st of month)
```

### Smart Retention

Intelligent retention based on backup age:

```bash
nself backup prune smart

# Automatically keeps:
# - All backups from last 24 hours
# - Daily backups for last week
# - Weekly backups for last month
# - Monthly backups for last year
# - Yearly backups forever
```

### Cloud Pruning

Clean up cloud storage:

```bash
# Prune cloud backups older than 30 days
nself backup prune cloud 30

# Works with S3 and rclone providers
```

## Automated Backups

### Schedule with Cron

```bash
# Schedule options
nself backup schedule hourly   # Every hour
nself backup schedule daily    # Every day at 3 AM (recommended)
nself backup schedule weekly   # Every Sunday at 3 AM
nself backup schedule monthly  # 1st of month at 3 AM

# View current schedule
crontab -l

# Remove schedule
crontab -e  # Delete nself-backup line
```

### Custom Automation

Create your own backup script:

```bash
#!/bin/bash
# backup-production.sh

# Load environment
source /path/to/project/.env.local

# Create backup with timestamp
nself backup create full production-$(date +%Y%m%d)

# Apply smart retention
nself backup prune smart

# Notify on success
curl -X POST https://hooks.slack.com/xxx \
  -d '{"text":"Backup completed successfully"}'
```

### Docker-based Scheduling

Using Docker Swarm or Kubernetes CronJob:

```yaml
# docker-compose.backup.yml
services:
  backup:
    image: nself/cli:latest
    volumes:
      - ./:/project
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - S3_BUCKET=${S3_BUCKET}
    command: backup create full
    deploy:
      mode: replicated
      replicas: 0
      labels:
        - "swarm.cronjob.enable=true"
        - "swarm.cronjob.schedule=0 3 * * *"
```

## Disaster Recovery

### Complete System Recovery

When everything is lost:

```bash
# 1. Reinstall nself
curl -fsSL https://raw.githubusercontent.com/nself/nself/main/install.sh | bash

# 2. Create new project directory
mkdir recovered-app && cd recovered-app

# 3. Download backup from cloud
aws s3 cp s3://my-backups/nself-backups/latest.tar.gz ./

# 4. Restore everything
nself backup restore latest.tar.gz

# 5. Start services
nself build && nself start
```

### Point-in-Time Recovery

Restore to specific moment:

```bash
# List all backups with dates
nself backup list

# Find backup before issue occurred
nself backup restore nself_backup_full_20240114_090000.tar.gz

# Verify data integrity
nself doctor
```

### Partial Recovery

Restore only what's needed:

```bash
# Database corruption - restore DB only
nself backup restore backup.tar.gz database

# Lost configuration - restore config only
nself backup restore backup.tar.gz config

# Manual extraction for specific files
tar -xzf backup.tar.gz -C /tmp/restore
cp /tmp/restore/config/.env.production ./
```

## Best Practices

### Production Recommendations

1. **Backup Frequency**
   - Production: Daily full backups minimum
   - Staging: Weekly full backups
   - Development: Before major changes

2. **Retention Strategy**
   ```bash
   # Production setup
   export BACKUP_RETENTION_DAYS=30
   export BACKUP_RETENTION_MIN=7
   nself backup schedule daily
   nself backup prune smart  # Run weekly
   ```

3. **3-2-1 Rule**
   - 3 copies of data (production + 2 backups)
   - 2 different storage types (local + cloud)
   - 1 offsite backup (cloud)

4. **Testing Backups**
   ```bash
   # Monthly restore test
   nself backup restore latest.tar.gz --dry-run
   nself backup verify latest.tar.gz
   ```

### Security Considerations

1. **Encrypt Cloud Backups**
   ```bash
   # Encrypt before upload
   openssl enc -aes-256-cbc -in backup.tar.gz -out backup.tar.gz.enc
   ```

2. **Secure Credentials**
   ```bash
   # Use environment files
   echo "S3_BUCKET=backups" >> .env.local
   echo "AWS_SECRET_ACCESS_KEY=xxx" >> .env.secrets
   chmod 600 .env.secrets
   ```

3. **Access Control**
   - Use IAM roles for S3
   - Rotate access tokens regularly
   - Audit backup access logs

### Monitoring

Set up alerts for backup failures:

```bash
# backup-monitor.sh
#!/bin/bash

if ! nself backup create full; then
  # Send alert
  curl -X POST https://api.pagerduty.com/incidents \
    -H "Authorization: Token token=xxx" \
    -d '{"incident":{"type":"incident","title":"Backup failed"}}'
fi
```

## Troubleshooting

### Common Issues

**Backup fails with "permission denied"**
```bash
# Fix Docker socket permissions
sudo chmod 666 /var/run/docker.sock
```

**"No space left on device"**
```bash
# Clean old backups
nself backup prune age 7

# Clean Docker resources
docker system prune -a
```

**Cloud upload fails**
```bash
# Test cloud connection
nself backup cloud test

# Check credentials
nself backup cloud status

# Reconfigure
nself backup cloud setup
```

**Restore fails with "file exists"**
```bash
# Force restore (careful!)
nself stop
rm -rf ./volumes/*
nself backup restore backup.tar.gz
```

### Verify Backup Integrity

```bash
# Check backup file
tar -tzf backup.tar.gz | head -20

# Test restore in separate location
mkdir /tmp/test-restore
cd /tmp/test-restore
nself backup restore /path/to/backup.tar.gz --dry-run
```

### Performance Optimization

For large databases:

```bash
# Compress with higher ratio (slower)
export BACKUP_COMPRESSION=9

# Use parallel compression
export BACKUP_THREADS=4

# Exclude unnecessary volumes
export BACKUP_EXCLUDE="cache,tmp"
```

## Advanced Usage

### Custom Backup Scripts

Extend backup functionality:

```bash
#!/bin/bash
# advanced-backup.sh

# Pre-backup hook
nself exec postgres pg_isready || exit 1

# Create backup with metadata
BACKUP_NAME="backup_$(git rev-parse --short HEAD)_$(date +%Y%m%d)"
nself backup create full "$BACKUP_NAME"

# Add metadata
echo "{
  'version': '$(cat VERSION)',
  'commit': '$(git rev-parse HEAD)',
  'date': '$(date -Iseconds)',
  'size': '$(du -h backups/$BACKUP_NAME.tar.gz | cut -f1)'
}" > "backups/$BACKUP_NAME.json"

# Upload with metadata
aws s3 cp "backups/$BACKUP_NAME.tar.gz" "s3://backups/"
aws s3 cp "backups/$BACKUP_NAME.json" "s3://backups/"
```

### Integration with CI/CD

GitLab CI example:

```yaml
backup:
  stage: backup
  script:
    - nself backup create full
    - nself backup prune smart
  only:
    - schedules
  artifacts:
    paths:
      - backups/
    expire_in: 30 days
```

### Multi-Region Backups

```bash
# Primary region
export S3_BUCKET=backups-us-east-1
nself backup create

# Replicate to secondary region
aws s3 sync s3://backups-us-east-1 s3://backups-eu-west-1 --source-region us-east-1 --region eu-west-1
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/nself/nself/issues
- Documentation: https://docs.nself.org
- Community: https://discord.gg/nself