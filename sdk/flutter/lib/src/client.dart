import 'dart:async';
import 'package:http/http.dart' as http;
import 'auth/auth_client.dart';
import 'graphql/graphql_client.dart';
import 'storage/storage_client.dart';
import 'realtime/realtime_client.dart';
import 'functions/functions_client.dart';

/// Configuration for [NselfClient].
class NselfConfig {
  /// Base URL of the nSelf backend (no trailing slash).
  /// Example: `https://api.myapp.com`
  final String url;

  /// Public anonymous key (anon role JWT or shared secret).
  final String anonKey;

  /// Optional custom HTTP client — useful in tests.
  final http.Client? httpClient;

  const NselfConfig({
    required this.url,
    required this.anonKey,
    this.httpClient,
  });
}

/// Root entry point for all nSelf SDK surfaces.
///
/// Initialise once with [NselfClient.initialize], then access the singleton
/// via [NselfClient.instance].
///
/// ```dart
/// await NselfClient.initialize(
///   url: 'https://api.myapp.com',
///   anonKey: const String.fromEnvironment('NSELF_ANON_KEY'),
/// );
/// final nself = NselfClient.instance;
/// ```
class NselfClient {
  NselfClient._({
    required NselfConfig config,
    http.Client? httpClient,
  }) : _config = config {
    final client = httpClient ?? http.Client();
    auth = AuthClient(config: config, httpClient: client);
    graphql = GraphQLClient(config: config, httpClient: client);
    storage = StorageClient(config: config, httpClient: client);
    realtime = RealtimeClient(config: config);
    functions = FunctionsClient(config: config, httpClient: client);
  }

  // ---------------------------------------------------------------------------
  // Singleton
  // ---------------------------------------------------------------------------

  static NselfClient? _instance;

  /// The current singleton instance. Throws if [initialize] has not been called.
  static NselfClient get instance {
    assert(
      _instance != null,
      'NselfClient.initialize() must be called before accessing NselfClient.instance.',
    );
    return _instance!;
  }

  /// Initialize the SDK. Call this once from your app's `main()`.
  ///
  /// Subsequent calls replace the existing singleton (useful in tests).
  static Future<NselfClient> initialize({
    required String url,
    required String anonKey,
    http.Client? httpClient,
  }) async {
    _instance?.realtime.dispose();
    _instance = NselfClient._(
      config: NselfConfig(url: url, anonKey: anonKey),
      httpClient: httpClient,
    );
    return _instance!;
  }

  /// Dispose all resources. Primarily for testing teardown.
  static void dispose() {
    _instance?.realtime.dispose();
    _instance = null;
  }

  // ---------------------------------------------------------------------------
  // Surfaces
  // ---------------------------------------------------------------------------

  final NselfConfig _config;

  /// Authentication — sign in, sign up, sign out, session management.
  late final AuthClient auth;

  /// GraphQL — queries, mutations, and WebSocket subscriptions via Hasura.
  late final GraphQLClient graphql;

  /// Storage — file upload, download, and URL generation via MinIO.
  late final StorageClient storage;

  /// Realtime — Postgres CDC and broadcast channels.
  late final RealtimeClient realtime;

  /// Functions — invoke nSelf serverless functions.
  late final FunctionsClient functions;

  NselfConfig get config => _config;
}
