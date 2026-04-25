# slack-clone

Team messaging starter built with nSelf.

## What you get

- `np_workspaces`, `np_channels`, `np_messages`, `np_threads`, `np_reactions` tables with RLS
- Hasura permissions and relationships
- Flutter starter with workspace picker, channel view, and message view
- Required plugins: `chat`, `livekit`, `realtime`, `auth`, `notify`

## Getting started

```bash
nself plugin install chat livekit realtime auth notify
nself start
nself db migrate
nself hasura metadata apply --file hasura/metadata.json
cd flutter && flutter run
```
