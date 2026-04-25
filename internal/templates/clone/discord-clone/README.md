# discord-clone

Community chat platform starter built with nSelf.

## What you get

- `np_servers`, `np_channels`, `np_messages`, `np_members`, `np_roles` tables with RLS
- Hasura permissions and relationships
- Flutter starter with server list, channel list, and message view
- Required plugins: `chat`, `realtime`, `auth`, `moderation`

## Getting started

```bash
nself plugin install chat realtime auth moderation
nself start
nself db migrate
nself hasura metadata apply --file hasura/metadata.json
cd flutter && flutter run
```
