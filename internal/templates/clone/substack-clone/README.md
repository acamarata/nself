# substack-clone

Newsletter publishing starter built with nSelf.

## What you get

- `np_newsletters`, `np_posts`, `np_subscribers`, `np_tiers` tables with RLS
- Hasura permissions and relationships
- Flutter starter with post list, editor, and subscriber management
- Required plugins: `cms`, `notify`, `auth`

## Getting started

```bash
nself plugin install cms notify auth
nself start
nself db migrate
nself hasura metadata apply --file hasura/metadata.json
cd flutter && flutter run
```
