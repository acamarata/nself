# notion-clone

Collaborative workspace starter built with nSelf.

## What you get

- `np_workspaces`, `np_pages`, `np_blocks`, `np_page_members` tables with RLS
- Hasura permissions and relationships
- Flutter starter with sidebar tree and block editor
- Required plugins: `cms`, `auth`, `realtime`

## Getting started

```bash
nself plugin install cms auth realtime
nself start
nself db migrate
nself hasura metadata apply --file hasura/metadata.json
cd flutter && flutter run
```
