# Flutter SDK Migration Assessment

**Assessment date:** 2026-06-21
**SDK:** `nself_sdk` (Flutter) v1.1.9
**Policy reference:** ASI Policy 2 — Flutter Elimination Mandate
**Status:** Assessment only. No migration performed here.

---

## Summary

ASI Policy 2 eliminates Flutter in favour of React Native + Expo (mobile/TV) and Tauri 2 (desktop). This document assesses the scope of migrating the `nself_sdk` Flutter package to an equivalent React Native / TypeScript surface, and identifies which consumers depend on it today.

---

## Current Consumers

| Repo | Platform coverage | Flutter dep? |
|------|-------------------|-------------|
| `nclaw/` | macOS, iOS, Android (Flutter desktop+mobile) | Yes — wraps `nself_sdk` for auth + GraphQL |
| `nchat/` | iOS, Android (Flutter mobile) | Yes — wraps `nself_sdk` for auth + realtime |
| `ntv/` | Android TV, Fire TV (Flutter TV) | Yes — wraps `nself_sdk` for plugin/license calls |
| `clawde/` | macOS, Linux, Windows (Flutter desktop) | Yes — wraps `nself_sdk` for plugin management |
| `ntask/` | iOS, Android (Flutter mobile) | No — uses REST directly, not the Flutter SDK |

All four active consumers are themselves slated for Flutter→RN migration in their respective phase plans per ASI Policy 2.

---

## Flutter SDK Surface (what needs a TS/RN equivalent)

| Module | Flutter capability | RN+Expo equivalent |
|--------|-------------------|--------------------|
| `lib/src/auth.dart` | JWT auth, token refresh | `@nself/auth-core` (already exists in `packages/@nself/auth-core`) |
| `lib/src/graphql.dart` | GraphQL query/mutation | `@nself/graphql-client` (already exists in `packages/@nself/graphql-client`) |
| `lib/src/storage.dart` | Secure on-device storage | `expo-secure-store` + `@nself/native-bridge` |
| `lib/src/realtime.dart` | WebSocket subscriptions | Native WebSocket + `@nself/push-client` |
| `lib/src/functions.dart` | Plugin function invocation | `@nself/sdk` (new, in `sdk/ts-sdk/`) |
| `lib/src/push.dart` | Push notification receipt | `expo-notifications` + `@nself/push-client` |

---

## Migration Effort Estimate

| Consumer | Effort | Platform delta | Risk |
|----------|--------|----------------|------|
| `nchat/` | High — full rewrite in RN | No platform reach lost (iOS+Android covered) | Medium |
| `nclaw/` | High — full rewrite in RN+Tauri | No platform reach lost (desktop→Tauri, mobile→RN) | High (Rust core integration) |
| `ntv/` | Medium — rewrite in `react-native-tvos` | No platform reach lost (tvOS added, Android TV + Fire TV covered) | Medium |
| `clawde/` | High — full rewrite in Tauri | No platform reach lost (macOS+Linux+Windows covered by Tauri) | High (file-system, terminal access) |

Total across all consumers: **2-3 months** of dedicated effort per ASI assessment rules. Sequencing should follow phase planning per app (each app's next phase owns its own migration Epic).

---

## What Is NOT Lost

- iOS + Android coverage: React Native + Expo covers both.
- Desktop coverage: Tauri 2 covers macOS + Linux + Windows.
- TV coverage: `react-native-tvos` covers Apple TV + Android TV + Fire TV.
- Shared code: All `@nself/*` TS packages in `packages/` replace Flutter-specific SDK modules — zero duplication of business logic.

---

## What IS Lost (platform gaps to track)

- **Apple Watch / watchOS**: neither RN nor Tauri covers this. Not currently supported by any nSelf consumer, so no regression.
- **Fuchsia**: Flutter-only; not a target for any nSelf consumer.

---

## Recommendation

Do NOT migrate in this ticket. Each consumer app migration is a separate Epic owned by that app's phase plan. The Flutter SDK (`nself_sdk`) should remain published and maintained until the LAST consumer completes its migration. Deprecation notice should be added to `pubspec.yaml` and `README.md` once all four consumers are migrated.

Track migration progress via SPORT `F13-CROSS-REPO-DEPS.md` Flutter SDK consumer rows.

---

## Open Items (for consumer phase planning)

1. `nclaw/`: Rust core (`nclaw-core`) exposes a Dart FFI today. The RN equivalent will use `@nclaw/native-bridge` via JSI or a dedicated Rust→RN module. This is the highest-risk piece — needs its own ADR.
2. `ntv/`: `react-native-tvos` does not have parity with Flutter's `video_player` plugin for HLS/DASH streams. Needs evaluation in ntv's migration Epic.
3. `clawde/`: File-system and terminal access in Tauri 2 requires Rust Tauri commands. ClawDE's migration Epic should audit all Dart platform channel calls.
