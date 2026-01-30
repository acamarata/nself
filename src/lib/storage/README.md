# Storage Library

Comprehensive file upload pipeline for nself applications.

## Overview

This library provides enterprise-grade file upload capabilities with:

- Multipart upload handling
- Automatic thumbnail generation
- Virus scanning (ClamAV)
- Smart compression
- GraphQL integration
- Multiple storage backends

## Files

### upload-pipeline.sh

Core upload pipeline implementation.

**Features:**
- Multipart upload for large files (> 100MB)
- Thumbnail generation (AVIF, WebP, JPEG)
- Virus scanning with ClamAV
- Automatic compression for large files
- Progress tracking
- MinIO/S3 integration

**Functions:**
- `init_upload_pipeline()` - Initialize upload system
- `upload_file(file, dest, options)` - Upload with processing
- `generate_thumbnails(file, dest)` - Create responsive thumbnails
- `scan_file_for_viruses(file)` - ClamAV integration
- `compress_file(file)` - Smart compression
- `list_uploads(prefix)` - List uploaded files
- `delete_upload(path)` - Delete file
- `get_pipeline_status()` - Show system status

**Environment Variables:**
```bash
STORAGE_BACKEND=minio              # Storage backend
MINIO_ENDPOINT=http://minio:9000   # MinIO endpoint
MINIO_ACCESS_KEY=minioadmin        # Access key
MINIO_SECRET_KEY=minioadmin        # Secret key
MINIO_BUCKET=uploads               # Bucket name

UPLOAD_ENABLE_MULTIPART=true       # Enable multipart
UPLOAD_ENABLE_THUMBNAILS=false     # Generate thumbnails
UPLOAD_ENABLE_VIRUS_SCAN=false     # Scan viruses
UPLOAD_ENABLE_COMPRESSION=true     # Auto-compress

UPLOAD_THUMBNAIL_SIZES=150x150,300x300,600x600
UPLOAD_IMAGE_FORMATS=avif,webp,jpg
```

**Usage:**
```bash
source upload-pipeline.sh

init_upload_pipeline
upload_file "photo.jpg" "users/123/" "thumbnails,compression"
list_uploads "users/123/"
delete_upload "users/123/photo.jpg"
```

### graphql-integration.sh

Auto-generate GraphQL integration package.

**Features:**
- Database migration (files table)
- Hasura metadata (permissions, relationships)
- GraphQL operations (mutations, queries, subscriptions)
- TypeScript types
- React hooks

**Functions:**
- `generate_files_migration()` - SQL migration
- `generate_hasura_metadata()` - Hasura config
- `generate_typescript_types()` - TypeScript types
- `generate_react_hooks()` - React hooks
- `generate_graphql_package(output_dir)` - Complete package

**Generated Files:**
```
output_dir/
├── migrations/
│   └── YYYYMMDDHHMMSS_create_files_table.sql
├── metadata/
│   └── tables/public_files.yaml
├── graphql/
│   └── files.graphql
├── types/
│   └── files.ts
├── hooks/
│   └── useFiles.ts
└── README.md
```

**Usage:**
```bash
source graphql-integration.sh

generate_graphql_package ".backend/storage"

# Run migration
psql $DATABASE_URL < .backend/storage/migrations/*.sql

# Apply Hasura metadata
hasura metadata apply

# Copy to frontend
cp .backend/storage/types/files.ts src/types/
cp .backend/storage/hooks/useFiles.ts src/hooks/
```

## CLI Integration

The storage library is accessed via `nself storage` commands.

See: `src/cli/storage.sh`

### Commands

```bash
# Upload file
nself storage upload <file> [--thumbnails] [--virus-scan] [--compression]

# List files
nself storage list [prefix]

# Delete file
nself storage delete <path>

# Configuration
nself storage config
nself storage status
nself storage test

# Initialize
nself storage init

# GraphQL setup
nself storage graphql-setup [output_dir]
```

## GraphQL Schema

### Types

```graphql
type File {
  id: uuid!
  name: String!
  size: Int!
  mimeType: String!
  path: String!
  url: String!
  thumbnailUrl: String
  userId: uuid!
  createdAt: timestamptz!
  updatedAt: timestamptz!
  metadata: jsonb
  tags: [String!]
  isPublic: Boolean!
}
```

### Mutations

```graphql
mutation UploadFile($file: Upload!, $path: String, $isPublic: Boolean) {
  uploadFile(file: $file, path: $path, isPublic: $isPublic) {
    id
    name
    size
    url
    thumbnailUrl
  }
}

mutation UploadFiles($files: [Upload!]!, $path: String) {
  uploadFiles(files: $files, path: $path) {
    id
    name
    url
  }
}

mutation DeleteFile($id: uuid!) {
  delete_files_by_pk(id: $id) {
    id
  }
}
```

### Queries

```graphql
query GetFile($id: uuid!) {
  files_by_pk(id: $id) {
    id
    name
    url
    thumbnailUrl
    user {
      displayName
    }
  }
}

query ListUserFiles($userId: uuid!) {
  files(where: { userId: { _eq: $userId } }) {
    id
    name
    size
    url
    thumbnailUrl
    createdAt
  }
  files_aggregate(where: { userId: { _eq: $userId } }) {
    aggregate {
      count
      sum { size }
    }
  }
}
```

## React Integration

### Hooks

```typescript
import { useFileUpload, useUserFiles } from '@/hooks/useFiles';

// Upload file
const { upload, loading } = useFileUpload();
await upload(file, { path: 'users/123/', isPublic: false });

// List files
const { files, total, totalSize } = useUserFiles(userId);

// Delete file
const { remove } = useFileDelete();
await remove(fileId);
```

### Components

See `docs/guides/file-upload-examples.md` for complete examples:

- Avatar upload with cropping
- Multi-file upload with progress
- Drag & drop interface
- File manager with folders

## Security

### Best Practices

1. **File Type Validation**
   - Validate MIME type server-side
   - Check file signature (magic numbers)
   - Use allowlist, not blocklist

2. **File Size Limits**
   - Enforce on client and server
   - Database constraints
   - Storage quotas per user

3. **Virus Scanning**
   - Enable ClamAV in production
   - Daily virus definition updates
   - Quarantine infected files

4. **Access Control**
   - Row-level security (RLS)
   - Hasura permissions
   - Signed URLs for private files

5. **Storage Security**
   - Separate storage domain
   - Content-Disposition headers
   - Disable script execution
   - Content Security Policy

See `docs/security/file-upload-security.md` for complete guide.

## Performance

### Optimization Strategies

1. **CDN Integration**
   - CloudFlare, AWS CloudFront
   - Cache images for 1 year
   - Automatic image optimization

2. **Responsive Images**
   - Use thumbnails for different sizes
   - Modern formats (AVIF, WebP)
   - Lazy loading

3. **Compression**
   - Gzip for text files
   - Skip already compressed formats
   - Transparent decompression

4. **Multipart Upload**
   - Automatic for files > 100MB
   - Parallel chunk uploads
   - Resume capability

## Testing

### Unit Tests

```bash
# Test upload pipeline
bash test/storage/test-upload-pipeline.sh

# Test GraphQL integration
bash test/storage/test-graphql-integration.sh
```

### Integration Tests

```bash
# Initialize storage
nself storage init

# Test upload
nself storage test

# Upload real file
nself storage upload test.jpg --all-features

# Verify in MinIO
mc ls nself/uploads/

# Verify in database
psql $DATABASE_URL -c "SELECT * FROM files;"
```

## Troubleshooting

### Common Issues

**MinIO not running:**
```bash
nself status | grep minio
nself restart minio
```

**Thumbnails not generated:**
```bash
brew install imagemagick ffmpeg  # macOS
sudo apt-get install imagemagick ffmpeg  # Ubuntu
```

**Virus scan fails:**
```bash
sudo apt-get install clamav
sudo freshclam
```

**Upload fails:**
```bash
# Check logs
nself logs minio

# Test connection
mc alias set test http://minio:9000 minioadmin minioadmin

# List buckets
mc ls test/
```

## Resources

- [File Upload Pipeline Guide](../../docs/guides/file-upload-pipeline.md)
- [Security Best Practices](../../docs/security/file-upload-security.md)
- [Quick Start Tutorial](../../docs/tutorials/file-uploads-quickstart.md)
- [CLI Reference](../../docs/commands/storage.md)
- [Integration Examples](../../docs/guides/file-upload-examples.md)

## Contributing

When adding features:

1. Follow existing code style
2. Add comprehensive error handling
3. Use platform-compatible shell commands
4. Add tests for new functionality
5. Update documentation
6. Follow security best practices

## License

Part of nself - MIT License
