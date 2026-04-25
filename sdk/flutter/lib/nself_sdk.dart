/// nself_sdk — Official Flutter SDK for nSelf.
///
/// Wraps nSelf's Hasura GraphQL, Auth, Storage, Realtime, and Functions
/// endpoints into idiomatic Dart APIs. Replaces hand-rolled HTTP clients
/// across nclaw, nchat, ntv, and nfamily.
///
/// ## Quick start
///
/// ```dart
/// import 'package:nself_sdk/nself_sdk.dart';
///
/// await NselfClient.initialize(
///   url: 'https://api.myapp.com',
///   anonKey: const String.fromEnvironment('NSELF_ANON_KEY'),
/// );
///
/// final nself = NselfClient.instance;
///
/// // Auth
/// final session = await nself.auth.signInWithEmail('user@example.com', 'pw');
///
/// // GraphQL
/// final result = await nself.graphql.rawQuery(
///   'query { np_users { id email } }',
/// );
///
/// // Storage
/// await nself.storage.from('avatars').upload('uid/photo.jpg', bytes);
/// final url = nself.storage.from('avatars').getPublicUrl('uid/photo.jpg');
///
/// // Realtime
/// nself.realtime
///   .channel('room:42')
///   .onPostgresChanges(schema: 'public', table: 'messages', callback: (e) => print(e))
///   .subscribe();
/// ```
library nself_sdk;

export 'src/client.dart' show NselfClient, NselfConfig;
export 'src/auth/auth_client.dart'
    show AuthClient, Session, NselfUser, AuthError;
export 'src/graphql/graphql_client.dart'
    show GraphQLClient, GraphQLResult, GraphQLError, SubscriptionHandle;
export 'src/storage/storage_client.dart'
    show StorageClient, StorageBucket, UploadOptions;
export 'src/realtime/realtime_client.dart'
    show RealtimeClient, RealtimeChannel, PostgresChangesEvent, RealtimeEvent,
        PostgresChangeType;
export 'src/functions/functions_client.dart'
    show FunctionsClient, FunctionResponse, FunctionsError;
