# airbnb-clone

Rental marketplace starter built with nSelf.

## What you get

- `np_listings`, `np_bookings`, `np_reviews` Postgres tables with RLS
- Hasura permissions and relationships
- Flutter starter with listing browse, detail, and host dashboard screens
- Required plugins: `auth`, `notify`, `photos`

## Getting started

```bash
nself start
nself db migrate
nself hasura metadata apply --file hasura/metadata.json
cd flutter && flutter run
```

## Required plugins

Install these via `nself plugin install <name>` (free tier):

- `auth` — user authentication
- `notify` — booking confirmation emails
- `photos` — listing image storage
