# Simple Blog - Complete Tutorial

This step-by-step tutorial will guide you through building a complete blog application with nself.

**Time Required:** 30-45 minutes
**Difficulty:** Beginner
**Prerequisites:** Basic knowledge of SQL, GraphQL, and React

---

## Table of Contents

1. [Initial Setup](#initial-setup)
2. [Database Design](#database-design)
3. [Hasura Configuration](#hasura-configuration)
4. [Authentication Setup](#authentication-setup)
5. [Building the Frontend](#building-the-frontend)
6. [Testing Everything](#testing-everything)
7. [Next Steps](#next-steps)

---

## Initial Setup

### Step 1: Create Project Directory

```bash
# Create and enter project directory
mkdir simple-blog
cd simple-blog

# Copy example files
cp -r /path/to/nself/examples/01-simple-blog/* .
```

### Step 2: Configure Environment

```bash
# Copy environment template
cp .env.example .env

# Edit configuration
nano .env
```

**Minimal .env configuration:**

```bash
# Project Information
PROJECT_NAME=simple-blog
ENV=dev
BASE_DOMAIN=localhost

# Database Configuration
POSTGRES_DB=blog_db
POSTGRES_USER=postgres
POSTGRES_PASSWORD=ChangeMeInProduction123!
POSTGRES_PORT=5432

# Hasura Configuration
HASURA_GRAPHQL_ADMIN_SECRET=myadminsecretkey
HASURA_GRAPHQL_JWT_SECRET='{"type":"HS256","key":"a-very-long-secret-key-minimum-32-characters-required"}'

# Auth Configuration
AUTH_SERVER_URL=http://localhost:1337
AUTH_CLIENT_URL=http://localhost:3000

# Frontend App
FRONTEND_APP_1_NAME=blog-frontend
FRONTEND_APP_1_PORT=3000
FRONTEND_APP_1_ROUTE=/
```

**Important:** Change passwords and secrets before deploying to production!

### Step 3: Initialize nself

```bash
# Initialize project (creates necessary files)
nself init

# Build infrastructure (generates docker-compose.yml, nginx config, etc.)
nself build

# Start all services
nself start
```

**What just happened?**
- PostgreSQL database started on port 5432
- Hasura GraphQL engine started
- Auth service started on port 1337
- Nginx reverse proxy configured

**Verify services are running:**

```bash
# Check service status
nself status

# Expected output:
# ✓ postgres     (healthy)
# ✓ hasura       (healthy)
# ✓ auth         (healthy)
# ✓ nginx        (healthy)
```

---

## Database Design

### Step 4: Create Database Schema

Create `database/schema.sql`:

```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Posts table
CREATE TABLE posts (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  title TEXT NOT NULL,
  slug TEXT UNIQUE NOT NULL,
  content TEXT NOT NULL,
  excerpt TEXT,
  author_id UUID NOT NULL,
  published BOOLEAN DEFAULT false,
  published_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

  -- Foreign key to auth.users
  CONSTRAINT fk_author
    FOREIGN KEY (author_id)
    REFERENCES auth.users(id)
    ON DELETE CASCADE
);

-- Comments table
CREATE TABLE comments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  post_id UUID NOT NULL,
  author_id UUID NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

  -- Foreign keys
  CONSTRAINT fk_post
    FOREIGN KEY (post_id)
    REFERENCES posts(id)
    ON DELETE CASCADE,

  CONSTRAINT fk_comment_author
    FOREIGN KEY (author_id)
    REFERENCES auth.users(id)
    ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_published ON posts(published, published_at DESC);
CREATE INDEX idx_posts_slug ON posts(slug);
CREATE INDEX idx_comments_post ON comments(post_id);
CREATE INDEX idx_comments_author ON comments(author_id);

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to posts
CREATE TRIGGER update_posts_updated_at
  BEFORE UPDATE ON posts
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to comments
CREATE TRIGGER update_comments_updated_at
  BEFORE UPDATE ON comments
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();

-- Function to auto-generate slug
CREATE OR REPLACE FUNCTION generate_slug(title TEXT)
RETURNS TEXT AS $$
BEGIN
  RETURN lower(
    regexp_replace(
      regexp_replace(title, '[^a-zA-Z0-9\s-]', '', 'g'),
      '\s+', '-', 'g'
    )
  );
END;
$$ LANGUAGE plpgsql;

-- Auto-generate slug on insert
CREATE OR REPLACE FUNCTION auto_generate_slug()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.slug IS NULL OR NEW.slug = '' THEN
    NEW.slug := generate_slug(NEW.title);
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER posts_auto_slug
  BEFORE INSERT ON posts
  FOR EACH ROW
  EXECUTE FUNCTION auto_generate_slug();
```

### Step 5: Apply Schema

```bash
# Apply schema to database
nself db execute --file database/schema.sql

# Verify tables were created
nself db query "SELECT tablename FROM pg_tables WHERE schemaname = 'public';"

# Expected output:
# tablename
# -----------
# posts
# comments
```

### Step 6: Create Sample Data

Create `database/seeds/sample-data.sql`:

```sql
-- Insert sample posts
-- Note: Replace author_id with actual user ID after creating users

-- Sample Post 1
INSERT INTO posts (title, slug, content, excerpt, author_id, published, published_at)
VALUES (
  'Welcome to My Blog',
  'welcome-to-my-blog',
  'This is my first blog post! I''m excited to share my thoughts with you.',
  'An introduction to my blog',
  'INSERT_USER_ID_HERE',
  true,
  NOW()
);

-- Sample Post 2
INSERT INTO posts (title, slug, content, excerpt, author_id, published, published_at)
VALUES (
  'Getting Started with nself',
  'getting-started-with-nself',
  'nself makes it incredibly easy to build full-stack applications...',
  'Learn how to use nself',
  'INSERT_USER_ID_HERE',
  true,
  NOW()
);
```

**We'll apply this seed data after creating our first user in Step 9.**

---

## Hasura Configuration

### Step 7: Track Tables in Hasura

```bash
# Open Hasura Console
nself admin hasura

# Or open directly in browser:
# http://api.localhost
# Admin Secret: (value from .env HASURA_GRAPHQL_ADMIN_SECRET)
```

**In Hasura Console:**

1. Go to **Data** tab
2. Click **Track All** under "Untracked tables or views"
3. You should see `posts` and `comments` tables tracked

### Step 8: Setup Relationships

**In Hasura Console → Data → posts table:**

1. Click **Relationships** tab
2. Add **Object Relationship:**
   - Name: `author`
   - Reference: `auth.users` table
   - From: `author_id` → To: `id`

3. Add **Array Relationship:**
   - Name: `comments`
   - Reference: `comments` table
   - From: `id` → To: `post_id`

**In Hasura Console → Data → comments table:**

1. Add **Object Relationship:**
   - Name: `post`
   - Reference: `posts` table
   - From: `post_id` → To: `id`

2. Add **Object Relationship:**
   - Name: `author`
   - Reference: `auth.users` table
   - From: `author_id` → To: `id`

### Step 9: Configure Permissions

**For `posts` table:**

**Public (anonymous users):**
- **Select:** Allow reading published posts only
  ```json
  {
    "published": {
      "_eq": true
    }
  }
  ```
  Columns: `id`, `title`, `slug`, `excerpt`, `published_at`, `created_at`

**User role:**
- **Select:** All published posts
  ```json
  {
    "published": {
      "_eq": true
    }
  }
  ```
  Columns: All

- **Insert:** Users can create posts
  ```json
  {
    "author_id": {
      "_eq": "X-Hasura-User-Id"
    }
  }
  ```
  Column presets: `author_id` = `X-Hasura-User-Id`
  Columns: `title`, `content`, `excerpt`, `published`

- **Update:** Users can only update their own posts
  ```json
  {
    "author_id": {
      "_eq": "X-Hasura-User-Id"
    }
  }
  ```
  Columns: `title`, `content`, `excerpt`, `published`

- **Delete:** Users can only delete their own posts
  ```json
  {
    "author_id": {
      "_eq": "X-Hasura-User-Id"
    }
  }
  ```

**For `comments` table:**

**User role:**
- **Select:** Allow reading all comments
  ```json
  {}
  ```

- **Insert:** Users can create comments
  ```json
  {
    "author_id": {
      "_eq": "X-Hasura-User-Id"
    }
  }
  ```
  Column presets: `author_id` = `X-Hasura-User-Id`

- **Update:** Users can only update their own comments
  ```json
  {
    "author_id": {
      "_eq": "X-Hasura-User-Id"
    }
  }
  ```

- **Delete:** Users can only delete their own comments
  ```json
  {
    "author_id": {
      "_eq": "X-Hasura-User-Id"
    }
  }
  ```

---

## Authentication Setup

### Step 10: Create Test User

```bash
# Using Hasura Console API explorer (http://api.localhost)
# Or use curl:

curl -X POST http://auth.localhost/signup/email-password \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "SecurePassword123!",
    "options": {
      "displayName": "John Doe"
    }
  }'

# Response should include:
# {
#   "session": { ... },
#   "user": {
#     "id": "uuid-here",
#     "email": "john@example.com"
#   }
# }
```

**Save the user ID - you'll need it for seed data!**

### Step 11: Apply Seed Data

```bash
# Edit seed file with actual user ID
nano database/seeds/sample-data.sql

# Replace INSERT_USER_ID_HERE with the UUID from Step 10

# Apply seed data
nself db execute --file database/seeds/sample-data.sql
```

### Step 12: Test GraphQL API

**In Hasura Console → API tab:**

```graphql
query GetPosts {
  posts(
    where: { published: { _eq: true } }
    order_by: { published_at: desc }
  ) {
    id
    title
    slug
    excerpt
    published_at
    author {
      displayName
      avatarUrl
    }
    comments {
      id
      content
      author {
        displayName
      }
    }
  }
}
```

**You should see your sample posts!**

---

## Building the Frontend

### Step 13: Setup React Project

```bash
# Create frontend directory
mkdir -p frontend
cd frontend

# Initialize Vite project
npm create vite@latest . -- --template react-ts

# Install dependencies
npm install

# Install additional libraries
npm install @nhost/react @nhost/nhost-js
npm install @apollo/client graphql
npm install react-router-dom
npm install tailwindcss postcss autoprefixer
npm install @headlessui/react @heroicons/react
```

### Step 14: Configure TailwindCSS

```bash
# Initialize Tailwind
npx tailwindcss init -p
```

Edit `tailwind.config.js`:

```javascript
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
```

Edit `src/index.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

### Step 15: Configure nHost Client

Create `src/lib/nhost.ts`:

```typescript
import { NhostClient } from '@nhost/nhost-js'

export const nhost = new NhostClient({
  subdomain: 'localhost',
  region: 'localhost',
  authUrl: 'http://auth.localhost',
  graphqlUrl: 'http://api.localhost/v1/graphql',
  storageUrl: 'http://storage.localhost',
})
```

### Step 16: Create Main App Structure

Edit `src/App.tsx`:

```typescript
import { NhostProvider } from '@nhost/react'
import { nhost } from './lib/nhost'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { HomePage } from './pages/HomePage'
import { PostPage } from './pages/PostPage'
import { LoginPage } from './pages/LoginPage'
import { SignupPage } from './pages/SignupPage'
import { NewPostPage } from './pages/NewPostPage'
import { Layout } from './components/Layout'

function App() {
  return (
    <NhostProvider nhost={nhost}>
      <BrowserRouter>
        <Layout>
          <Routes>
            <Route path="/" element={<HomePage />} />
            <Route path="/post/:slug" element={<PostPage />} />
            <Route path="/login" element={<LoginPage />} />
            <Route path="/signup" element={<SignupPage />} />
            <Route path="/new-post" element={<NewPostPage />} />
          </Routes>
        </Layout>
      </BrowserRouter>
    </NhostProvider>
  )
}

export default App
```

### Step 17: Create Layout Component

Create `src/components/Layout.tsx`:

```typescript
import { useAuthenticationStatus, useSignOut } from '@nhost/react'
import { Link } from 'react-router-dom'

export function Layout({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuthenticationStatus()
  const { signOut } = useSignOut()

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <Link to="/" className="text-xl font-bold text-gray-900">
                My Blog
              </Link>
            </div>

            <div className="flex items-center space-x-4">
              {!isLoading && (
                <>
                  {isAuthenticated ? (
                    <>
                      <Link
                        to="/new-post"
                        className="text-gray-700 hover:text-gray-900"
                      >
                        New Post
                      </Link>
                      <button
                        onClick={signOut}
                        className="text-gray-700 hover:text-gray-900"
                      >
                        Sign Out
                      </button>
                    </>
                  ) : (
                    <>
                      <Link
                        to="/login"
                        className="text-gray-700 hover:text-gray-900"
                      >
                        Login
                      </Link>
                      <Link
                        to="/signup"
                        className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
                      >
                        Sign Up
                      </Link>
                    </>
                  )}
                </>
              )}
            </div>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {children}
      </main>
    </div>
  )
}
```

### Step 18: Create Home Page

Create `src/pages/HomePage.tsx`:

```typescript
import { useQuery, gql } from '@apollo/client'
import { Link } from 'react-router-dom'

const GET_POSTS = gql`
  query GetPosts {
    posts(
      where: { published: { _eq: true } }
      order_by: { published_at: desc }
    ) {
      id
      title
      slug
      excerpt
      published_at
      author {
        displayName
      }
    }
  }
`

export function HomePage() {
  const { data, loading, error } = useQuery(GET_POSTS)

  if (loading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>

  return (
    <div>
      <h1 className="text-3xl font-bold mb-8">Recent Posts</h1>

      <div className="space-y-6">
        {data?.posts.map((post: any) => (
          <article
            key={post.id}
            className="bg-white p-6 rounded-lg shadow-sm"
          >
            <Link to={`/post/${post.slug}`}>
              <h2 className="text-2xl font-semibold text-gray-900 hover:text-blue-600">
                {post.title}
              </h2>
            </Link>

            <p className="mt-2 text-gray-600">{post.excerpt}</p>

            <div className="mt-4 text-sm text-gray-500">
              By {post.author.displayName} •{' '}
              {new Date(post.published_at).toLocaleDateString()}
            </div>
          </article>
        ))}
      </div>
    </div>
  )
}
```

### Step 19: Create Login Page

Create `src/pages/LoginPage.tsx`:

```typescript
import { useState } from 'react'
import { useSignInEmailPassword } from '@nhost/react'
import { useNavigate } from 'react-router-dom'

export function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const { signInEmailPassword, isLoading, error } = useSignInEmailPassword()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    const result = await signInEmailPassword(email, password)

    if (result.isSuccess) {
      navigate('/')
    }
  }

  return (
    <div className="max-w-md mx-auto">
      <h1 className="text-3xl font-bold mb-8">Login</h1>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700">
            Email
          </label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">
            Password
          </label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
            required
          />
        </div>

        {error && (
          <div className="text-red-600 text-sm">
            {error.message}
          </div>
        )}

        <button
          type="submit"
          disabled={isLoading}
          className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 disabled:opacity-50"
        >
          {isLoading ? 'Logging in...' : 'Login'}
        </button>
      </form>
    </div>
  )
}
```

### Step 20: Start Frontend

```bash
# From frontend/ directory
npm run dev

# Open browser to:
# http://localhost:3000
```

---

## Testing Everything

### Step 21: Test User Flow

1. **Sign Up:**
   - Go to http://localhost:3000/signup
   - Create a new account
   - You should be redirected to home page

2. **View Posts:**
   - You should see sample posts on home page
   - Click a post to view details

3. **Create Post:**
   - Click "New Post" in navigation
   - Fill in title and content
   - Submit
   - View your new post

4. **Add Comment:**
   - View a post
   - Add a comment
   - See it appear in real-time

### Step 22: Test GraphQL API

```bash
# Test creating a post via API
curl -X POST http://api.localhost/v1/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "query": "mutation CreatePost($title: String!, $content: String!) { insert_posts_one(object: {title: $title, content: $content, published: true}) { id title } }",
    "variables": {
      "title": "API Test Post",
      "content": "Created via API"
    }
  }'
```

---

## Next Steps

Congratulations! You've built a complete blog application with nself.

### Add More Features

1. **Categories and Tags:**
   - Add `categories` and `tags` tables
   - Many-to-many relationships
   - Filter posts by category

2. **Image Uploads:**
   - Enable MinIO (MINIO_ENABLED=true)
   - Upload featured images
   - Implement file storage

3. **Search:**
   - Enable MeiliSearch (MEILISEARCH_ENABLED=true)
   - Index posts
   - Add search UI

4. **Email Notifications:**
   - Enable MailPit (MAILPIT_ENABLED=true)
   - Send welcome emails
   - Comment notifications

5. **Rich Text Editor:**
   - Install TipTap or Draft.js
   - Format blog posts
   - Embed media

### Deploy to Production

Follow `DEPLOYMENT.md` to deploy your blog to a production server.

### Explore More Examples

- [SaaS Starter](../02-saas-starter/) - Multi-tenant architecture
- [E-commerce](../03-ecommerce/) - Payment processing
- [Real-time Chat](../04-realtime-chat/) - WebSocket features

---

## Resources

- **nself Documentation:** [nself Documentation](../../../README.MD)
- **Hasura Docs:** https://hasura.io/docs/
- **nHost Docs:** https://docs.nhost.io/
- **React Docs:** https://react.dev/

## Get Help

- **GitHub Issues:** https://github.com/acamarata/nself/issues
- **Discussions:** https://github.com/acamarata/nself/discussions

---

**Congratulations on completing the tutorial!**

You now have a solid foundation in nself. Happy building!
