# {{SERVICE_NAME}} - Socket.IO TypeScript Service

Real-time bidirectional event-based communication service built with Socket.IO and TypeScript.

## Features

- ✅ WebSocket support with fallback to long-polling
- ✅ Type-safe event handling
- ✅ Room and namespace support
- ✅ Broadcasting capabilities
- ✅ Redis adapter ready for multi-instance deployment
- ✅ Health check endpoint
- ✅ Graceful shutdown handling
- ✅ Hot reload in development

## Quick Start

### Development Mode

```bash
# Install dependencies
npm install

# Start with hot reload
npm run dev
```

The server will start on port {{PORT}} with hot reload enabled.

### Production Mode

```bash
# Build TypeScript
npm run build

# Start production server
npm start
```

## Redis Adapter (Multi-Instance Support)

To enable horizontal scaling with multiple Socket.IO instances:

### 1. Install Redis Adapter

```bash
npm install @socket.io/redis-adapter ioredis
```

### 2. Update server.ts

```typescript
import { createAdapter } from '@socket.io/redis-adapter';
import { createClient } from 'redis';

const pubClient = createClient({ url: process.env.REDIS_URL || 'redis://localhost:6379' });
const subClient = pubClient.duplicate();

Promise.all([pubClient.connect(), subClient.connect()]).then(() => {
  io.adapter(createAdapter(pubClient, subClient));
  console.log('✓ Redis adapter connected');
});
```

### 3. Enable Redis in nself

```bash
# In .env
REDIS_ENABLED=true
```

### 4. Configure Service in .env

```bash
CS_N={{SERVICE_NAME}}:socketio-ts:{{PORT}}:ws
CS_N_REDIS_PREFIX=ws:
CS_N_REPLICAS=2  # Run 2 instances
```

## Environment Variables

```bash
PORT={{PORT}}                    # Server port
NODE_ENV=development             # Environment (development|production)
CORS_ORIGIN=*                    # CORS allowed origins
REDIS_URL=redis://localhost:6379 # Redis connection URL (for adapter)

# nself Integration
PROJECT_NAME={{PROJECT_NAME}}
BASE_DOMAIN={{BASE_DOMAIN}}
HASURA_GRAPHQL_ENDPOINT={{HASURA_GRAPHQL_ENDPOINT}}
```

## API Endpoints

### HTTP Endpoints

- `GET /` - Service info
- `GET /health` - Health check (includes connection count)
- `GET /api/info` - Detailed service information

### Socket.IO Events

#### Client → Server

- `message` - Send a message
- `join_room` - Join a room
- `leave_room` - Leave a room

#### Server → Client

- `welcome` - Welcome message on connection
- `user_count` - Current connected users count
- `message_response` - Echo of sent message
- `broadcast_message` - Broadcast from other users
- `joined_room` - Confirmation of room join
- `left_room` - Confirmation of room leave
- `user_joined` - Notification when user joins room
- `user_left` - Notification when user leaves room

## Usage Examples

### JavaScript Client

```javascript
import { io } from 'socket.io-client';

const socket = io('http://localhost:{{PORT}}');

// Listen for welcome
socket.on('welcome', (data) => {
  console.log('Welcome:', data);
});

// Send message
socket.emit('message', { text: 'Hello World!' });

// Listen for response
socket.on('message_response', (data) => {
  console.log('Response:', data);
});

// Join room
socket.emit('join_room', 'general');

// Listen for room messages
socket.on('broadcast_message', (data) => {
  console.log('Broadcast:', data);
});
```

### TypeScript Client (Type-Safe)

```typescript
import { io, Socket } from 'socket.io-client';

interface ServerToClientEvents {
  welcome: (data: { message: string; socketId: string; timestamp: string }) => void;
  user_count: (count: number) => void;
  message_response: (data: any) => void;
  broadcast_message: (data: any) => void;
  joined_room: (data: { room: string; timestamp: string }) => void;
  user_joined: (data: { socketId: string; room: string; timestamp: string }) => void;
}

interface ClientToServerEvents {
  message: (data: { text: string; user?: string; room?: string }) => void;
  join_room: (room: string) => void;
  leave_room: (room: string) => void;
}

const socket: Socket<ServerToClientEvents, ClientToServerEvents> = io('http://localhost:{{PORT}}');

socket.on('welcome', (data) => {
  console.log(data.message); // TypeScript knows the structure!
});

socket.emit('message', { text: 'Hello!' }); // Type-checked!
```

## Customization

### Adding Custom Events

Edit `src/server.ts`:

```typescript
// Add new event handler
socket.on('custom_event', (data: CustomEventData) => {
  // Your logic here

  // Emit response
  socket.emit('custom_response', {
    success: true,
    data: processedData
  });
});
```

### Adding Authentication

```typescript
import { Server } from 'socket.io';

const io = new Server(server, {
  cors: { /* ... */ }
});

// Middleware for authentication
io.use((socket, next) => {
  const token = socket.handshake.auth.token;

  if (!token) {
    return next(new Error('Authentication error'));
  }

  // Verify token (use JWT, etc.)
  verifyToken(token).then(user => {
    socket.data.user = user;
    next();
  }).catch(err => {
    next(new Error('Authentication error'));
  });
});

io.on('connection', (socket) => {
  const user = socket.data.user;
  console.log(`User ${user.id} connected`);
});
```

### Adding Hasura Integration

```typescript
import fetch from 'node-fetch';

const HASURA_ENDPOINT = process.env.HASURA_GRAPHQL_ENDPOINT;
const HASURA_SECRET = process.env.HASURA_GRAPHQL_ADMIN_SECRET;

async function queryHasura(query: string, variables: any = {}) {
  const response = await fetch(HASURA_ENDPOINT, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-hasura-admin-secret': HASURA_SECRET
    },
    body: JSON.stringify({ query, variables })
  });

  return response.json();
}

// Use in event handlers
socket.on('get_messages', async (roomId: string) => {
  const result = await queryHasura(`
    query GetMessages($roomId: uuid!) {
      messages(where: { room_id: { _eq: $roomId } }) {
        id
        text
        user_id
        created_at
      }
    }
  `, { roomId });

  socket.emit('messages', result.data.messages);
});
```

## Docker

The service includes a production-ready Dockerfile with:
- Multi-stage build
- Security best practices (non-root user)
- Health checks
- Proper signal handling (dumb-init)

Build and run:

```bash
docker build -t {{SERVICE_NAME}} .
docker run -p {{PORT}}:{{PORT}} {{SERVICE_NAME}}
```

## Testing

Health check:

```bash
curl http://localhost:{{PORT}}/health
```

Expected response:

```json
{
  "status": "healthy",
  "service": "{{SERVICE_NAME}}",
  "timestamp": "2024-01-30T12:00:00.000Z",
  "connections": 0
}
```

## Monitoring

The service exposes metrics through the `/api/info` endpoint:

```bash
curl http://localhost:{{PORT}}/api/info
```

Response includes:
- Environment
- Uptime
- Memory usage
- Connection count

## Deployment

### Single Instance

```bash
# In .env
CS_1={{SERVICE_NAME}}:socketio-ts:{{PORT}}
```

### Multiple Instances (with Redis)

```bash
# In .env
REDIS_ENABLED=true
CS_1={{SERVICE_NAME}}:socketio-ts:{{PORT}}
CS_1_REPLICAS=3
CS_1_REDIS_PREFIX=socket:
```

Then run:

```bash
nself build && nself start
```

## Troubleshooting

### Port already in use

```bash
# Check what's using the port
lsof -i :{{PORT}}

# Change port in .env
CS_N={{SERVICE_NAME}}:socketio-ts:3101  # Use different port
```

### CORS errors

Update CORS configuration in `src/server.ts`:

```typescript
app.use(cors({
  origin: 'https://yourdomain.com',  // Specific domain
  credentials: true
}));
```

### Redis connection issues

```bash
# Check Redis is running
docker ps | grep redis

# Check connection
redis-cli ping
```

## Resources

- [Socket.IO Documentation](https://socket.io/docs/v4/)
- [Socket.IO Redis Adapter](https://socket.io/docs/v4/redis-adapter/)
- [TypeScript Socket.IO](https://socket.io/docs/v4/typescript/)
- [nself Documentation](https://nself.org/docs)
