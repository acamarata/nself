# nself_sdk

Official Flutter SDK for [nSelf](https://nself.org). Wraps nSelf's Hasura GraphQL,
Auth, Storage, Realtime, and Functions endpoints into idiomatic Dart APIs.

Works with nclaw, nchat, ntv, nfamily, and any custom nSelf-powered app.

## Quick start

```dart
import 'package:nself_sdk/nself_sdk.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  await NselfClient.initialize(
    url: const String.fromEnvironment('NSELF_URL'),
    anonKey: const String.fromEnvironment('NSELF_ANON_KEY'),
  );

  runApp(const MyApp());
}
```

## Auth

```dart
final nself = NselfClient.instance;

// Sign in
final session = await nself.auth.signInWithEmail('user@example.com', 'password');

// Sign up
final session = await nself.auth.signUp('new@example.com', 'password',
    displayName: 'Alice');

// Refresh
final refreshed = await nself.auth.refreshSession(session.refreshToken);

// Sign out
await nself.auth.signOut();

// Access the current user
final user = nself.auth.currentUser; // NselfUser? 
```

## GraphQL

```dart
// Pass the access token to authorize requests
nself.graphql.setAccessToken(session.accessToken);

// Query
final result = await nself.graphql.rawQuery(r'''
  query GetMessages {
    np_messages(order_by: {created_at: desc}, limit: 20) {
      id
      body
      created_at
    }
  }
''');

if (!result.hasErrors) {
  final messages = result.data!['np_messages'] as List;
  // ...
}

// Mutation
await nself.graphql.rawMutation(r'''
  mutation InsertMessage($body: String!) {
    insert_np_messages_one(object: {body: $body}) { id }
  }
''', variables: {'body': 'Hello world'});

// Subscription
final (stream, handle) = nself.graphql.subscribe(r'''
  subscription OnNewMessage {
    np_messages(limit: 1, order_by: {created_at: desc}) {
      id body created_at
    }
  }
''');

stream.listen((result) {
  if (!result.hasErrors) print(result.data);
});

// Cancel when done
handle.cancel();
```

## Storage

```dart
nself.storage.setAccessToken(session.accessToken);

// Upload
await nself.storage.from('avatars').upload(
  'user/$userId/photo.jpg',
  imageBytes,
  options: const UploadOptions(contentType: 'image/jpeg'),
);

// Get public URL (no network call)
final url = nself.storage.from('avatars').getPublicUrl('user/$userId/photo.jpg');

// Download
final bytes = await nself.storage.from('avatars').download('user/$userId/photo.jpg');

// List
final files = await nself.storage.from('avatars').list(prefix: 'user/$userId/');

// Delete
await nself.storage.from('avatars').remove('user/$userId/old.jpg');
```

## Realtime

```dart
nself.realtime.setAccessToken(session.accessToken);

// Listen for Postgres change-data-capture events
nself.realtime
  .channel('messages')
  .onPostgresChanges(
    schema: 'public',
    table: 'np_messages',
    callback: (event) {
      switch (event.type) {
        case PostgresChangeType.insert:
          print('New message: ${event.record}');
        case PostgresChangeType.update:
          print('Updated: ${event.record}');
        case PostgresChangeType.delete:
          print('Deleted: ${event.oldRecord}');
        case PostgresChangeType.truncate:
          break;
      }
    },
  )
  .subscribe();

// Broadcast events
nself.realtime
  .channel('presence')
  .onBroadcast(event: 'cursor_move', callback: (e) => print(e.payload))
  .subscribe();

// Unsubscribe
nself.realtime.channel('messages').unsubscribe();
```

## Functions

```dart
nself.functions.setAccessToken(session.accessToken);

// Invoke a function with a JSON body
final resp = await nself.functions.invoke('send-welcome-email', body: {
  'user_id': userId,
});
print(resp.data); // decoded JSON

// Invoke and receive raw bytes (e.g. image generation)
final bytes = await nself.functions.invokeRaw('generate-avatar');
```

## Running the example app

```bash
cd example
flutter run \
  --dart-define=NSELF_URL=https://api.yourapp.com \
  --dart-define=NSELF_ANON_KEY=your_anon_key
```

## Running tests

```bash
flutter test
```

## Requirements

- Flutter 3.19 or later (stable channel)
- nSelf CLI v1.0.9 or later

## License

MIT. See [LICENSE](LICENSE).
