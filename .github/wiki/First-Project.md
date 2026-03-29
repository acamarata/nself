# First Project — Build a Todo API

This guide builds a simple Todo API end-to-end using nSelf. No frontend — pure backend API with GraphQL and JWT auth.

## 1. Initialize the Project

```bash
mkdir todo-api && cd todo-api
nself init --name todo-api --domain localhost
```

Accept all defaults. This generates a `.env` file with secure random secrets.

## 2. Start the Stack

```bash
nself build
nself start
```

Wait for all services to show healthy. Then confirm URLs:

```bash
nself urls
```

## 3. Open the Hasura Console

```bash
nself db hasura console
```

This opens `https://localhost/hasura/console` in your browser. The admin secret is in your `.env` as `HASURA_GRAPHQL_ADMIN_SECRET`.

## 4. Create the `todos` Table

In the Hasura Console:

1. Click **Data** → **Create Table**
2. Table name: `todos`
3. Add columns:
   - `id` — UUID, primary key, default: `gen_random_uuid()`
   - `title` — Text, not null
   - `done` — Boolean, not null, default: `false`
   - `created_at` — Timestamptz, default: `now()`
4. Click **Add Table**

## 5. Track the Table

Hasura will prompt you to track the new table. Click **Track** to expose it in the GraphQL API.

## 6. Insert and Query Data

In the GraphiQL explorer, insert a todo:

```graphql
mutation {
  insert_todos_one(object: { title: "Learn nSelf" }) {
    id
    title
    done
    created_at
  }
}
```

Then query all todos:

```graphql
query {
  todos {
    id
    title
    done
    created_at
  }
}
```

## 7. Test JWT Auth

The Auth service is running at `https://localhost/auth/`. Register a user:

```bash
curl -X POST https://localhost/auth/signup/email-password \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@example.com","password":"password123"}'
```

The response includes a JWT token. Use it in subsequent GraphQL requests:

```bash
curl -X POST https://localhost/v1/graphql \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{"query":"{ todos { id title } }"}'
```

Set up row-level security in Hasura permissions to restrict rows by user ID.

## 8. Stop When Done

```bash
nself stop
```

---

Next: [[Architecture]] · [[Plugin-Overview]]
