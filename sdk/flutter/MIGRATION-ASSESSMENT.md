# Flutter SDK Migration Assessment

**Assessment date:** 2026-06-25
**Status:** Assessment only — no migration performed here. See ASI Policy 2.

---

## Current state

The Flutter SDK (`cli/sdk/flutter/`) provides plugin authoring helpers for nSelf:
authentication, GraphQL subscriptions, storage, realtime channels, functions invocation,
and push notifications. It is at v1.1.9 and tested against Flutter 3.19+ / Dart 3.0+.

---

## Consumers

| Repo | Usage |
|------|-------|
| `nclaw/` | Plugin auth + GraphQL client |
| `nchat/` | Realtime channels + push |
| `ntv/` | Auth token refresh |
| `clawde/` | Full SDK surface |
| `ntask/` | Auth + storage |

---

## Target per ASI Policy 2

ASI Policy 2 eliminates Flutter in favor of React Native + Expo for mobile/desktop.
The Flutter SDK would be replaced by `@nself/sdk` (TypeScript) + a thin React Native
bridge in `packages/native-bridge`.

---

## Migration complexity

| Area | Flutter SDK | RN equivalent | Notes |
|------|------------|---------------|-------|
| Auth | `NselfAuth.signIn()` | `@nself/auth-core` | Direct port, same JWT flow |
| GraphQL | `NselfGraphQL` | `@nself/graphql-client` | Both use ws:// subscriptions |
| Storage | `NselfStorage` | `@nself/sdk` StorageClient | Identical S3-compatible API |
| Realtime | `NselfRealtime` | `@nself/sdk` RealtimeClient | Port to WS-based JS client |
| Push | `NselfPush` | `@nself/push-client` | Already exists in packages/ |
| Native bridge | `flutter_secure_storage` | `packages/native-bridge` | Keychain/Keystore abstraction |

---

## Recommended approach (when approved)

1. Each consumer repo migrates in its own phase (nclaw P5, nchat P5, ntv separate).
2. Flutter SDK remains published at v1.1.9 for existing consumers during transition.
3. `packages/native-bridge` fills the secure-storage gap before migration begins.
4. Deprecation notice added to Flutter SDK README at migration start.

---

## Platform reach delta

Flutter SDK covers iOS, Android, macOS, Windows, Linux, Web.
RN + Expo covers iOS, Android. Tauri covers macOS, Windows, Linux.
Web is covered by Vite SPA. No platform reach is lost.

---

## Decision authority

This assessment must be reviewed by the project owner before any migration begins.
No code changes are made in this ticket. The assessment is the deliverable.
