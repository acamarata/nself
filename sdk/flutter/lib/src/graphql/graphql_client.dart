import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:http/http.dart' as http;
import 'package:web_socket_channel/web_socket_channel.dart';
import '../client.dart';

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

/// A single error returned inside a GraphQL response.
class GraphQLError {
  final String message;
  final List<Map<String, dynamic>>? locations;
  final List<dynamic>? path;
  final Map<String, dynamic>? extensions;

  const GraphQLError({
    required this.message,
    this.locations,
    this.path,
    this.extensions,
  });

  factory GraphQLError.fromJson(Map<String, dynamic> json) {
    return GraphQLError(
      message: json['message'] as String? ?? 'Unknown GraphQL error',
      locations: (json['locations'] as List?)
          ?.map((e) => Map<String, dynamic>.from(e as Map))
          .toList(),
      path: json['path'] as List?,
      extensions: json['extensions'] != null
          ? Map<String, dynamic>.from(json['extensions'] as Map)
          : null,
    );
  }

  @override
  String toString() => 'GraphQLError: $message';
}

/// The result of a GraphQL query or mutation.
class GraphQLResult {
  final Map<String, dynamic>? data;
  final List<GraphQLError>? errors;

  const GraphQLResult({this.data, this.errors});

  /// True when [errors] is non-null and non-empty.
  bool get hasErrors => errors != null && errors!.isNotEmpty;

  factory GraphQLResult.fromJson(Map<String, dynamic> json) {
    return GraphQLResult(
      data: json['data'] != null
          ? Map<String, dynamic>.from(json['data'] as Map)
          : null,
      errors: (json['errors'] as List?)
          ?.map((e) => GraphQLError.fromJson(Map<String, dynamic>.from(e as Map)))
          .toList(),
    );
  }
}

/// A handle for cancelling a running subscription.
class SubscriptionHandle {
  final String _id;
  final void Function(String id) _cancel;

  SubscriptionHandle(this._id, this._cancel);

  /// Cancel the subscription.
  void cancel() => _cancel(_id);
}

// ---------------------------------------------------------------------------
// GraphQLClient
// ---------------------------------------------------------------------------

/// Executes GraphQL queries, mutations, and subscriptions against Hasura.
///
/// Access via [NselfClient.graphql].
class GraphQLClient {
  final NselfConfig _config;
  final http.Client _http;

  WebSocketChannel? _ws;
  int _subIdCounter = 0;
  final Map<String, StreamController<GraphQLResult>> _subscriptions = {};
  String? _accessToken;

  GraphQLClient({required NselfConfig config, required http.Client httpClient})
      : _config = config,
        _http = httpClient;

  // --- Public API ---

  /// Set (or clear) the Authorization token used for all subsequent requests.
  void setAccessToken(String? token) => _accessToken = token;

  /// Execute a raw GraphQL query string.
  ///
  /// ```dart
  /// final result = await nself.graphql.rawQuery(
  ///   'query { np_users { id email } }',
  /// );
  /// ```
  Future<GraphQLResult> rawQuery(
    String query, {
    Map<String, dynamic>? variables,
    String? operationName,
  }) => _execute(query, variables: variables, operationName: operationName);

  /// Execute a raw GraphQL mutation string.
  Future<GraphQLResult> rawMutation(
    String mutation, {
    Map<String, dynamic>? variables,
  }) => _execute(mutation, variables: variables);

  /// Subscribe to a GraphQL subscription over WebSocket.
  ///
  /// Returns a [Stream] of [GraphQLResult] and a [SubscriptionHandle] to
  /// cancel the subscription when done.
  ///
  /// ```dart
  /// final (stream, handle) = nself.graphql.subscribe(
  ///   'subscription OnMessage { np_messages { id body } }',
  /// );
  /// stream.listen((result) => print(result.data));
  /// // Later:
  /// handle.cancel();
  /// ```
  (Stream<GraphQLResult>, SubscriptionHandle) subscribe(
    String query, {
    Map<String, dynamic>? variables,
  }) {
    _ensureWebSocket();
    final id = 'sub_${++_subIdCounter}';
    final controller = StreamController<GraphQLResult>.broadcast();
    _subscriptions[id] = controller;

    final payload = {
      'type': 'subscribe',
      'id': id,
      'payload': {
        'query': query,
        if (variables != null) 'variables': variables,
      },
    };
    _ws?.sink.add(jsonEncode(payload));

    final handle = SubscriptionHandle(id, (cancelId) {
      _ws?.sink.add(jsonEncode({'type': 'complete', 'id': cancelId}));
      _subscriptions[cancelId]?.close();
      _subscriptions.remove(cancelId);
    });

    return (controller.stream, handle);
  }

  // --- Internal ---

  Future<GraphQLResult> _execute(
    String query, {
    Map<String, dynamic>? variables,
    String? operationName,
  }) async {
    final uri = Uri.parse('${_config.url}/v1/graphql');
    final body = <String, dynamic>{'query': query};
    if (variables != null) body['variables'] = variables;
    if (operationName != null) body['operationName'] = operationName;

    try {
      final resp = await _http.post(
        uri,
        headers: {
          'Content-Type': 'application/json',
          'x-hasura-role': 'user',
          if (_accessToken != null) 'Authorization': 'Bearer $_accessToken',
          if (_accessToken == null) 'x-hasura-admin-secret': _config.anonKey,
        },
        body: jsonEncode(body),
      );

      if (resp.statusCode != 200) {
        return GraphQLResult(errors: [
          GraphQLError(message: 'HTTP ${resp.statusCode}: ${resp.body}'),
        ]);
      }

      final json = jsonDecode(resp.body) as Map<String, dynamic>;
      return GraphQLResult.fromJson(json);
    } on SocketException catch (e) {
      return GraphQLResult(
          errors: [GraphQLError(message: 'Network error: ${e.message}')]);
    }
  }

  void _ensureWebSocket() {
    if (_ws != null) return;
    final wsUrl = _config.url
        .replaceFirst('https://', 'wss://')
        .replaceFirst('http://', 'ws://');
    _ws = WebSocketChannel.connect(
      Uri.parse('$wsUrl/v1/graphql'),
      protocols: ['graphql-transport-ws'],
    );

    // Send connection_init with auth header.
    _ws!.sink.add(jsonEncode({
      'type': 'connection_init',
      'payload': {
        'headers': {
          if (_accessToken != null) 'Authorization': 'Bearer $_accessToken',
          if (_accessToken == null) 'x-hasura-admin-secret': _config.anonKey,
        },
      },
    }));

    _ws!.stream.listen(
      (data) {
        try {
          final msg = jsonDecode(data as String) as Map<String, dynamic>;
          final type = msg['type'] as String?;
          final id = msg['id'] as String?;
          if (type == 'next' && id != null && _subscriptions.containsKey(id)) {
            final payload = msg['payload'] as Map<String, dynamic>? ?? {};
            _subscriptions[id]!.add(GraphQLResult.fromJson(payload));
          } else if (type == 'error' && id != null) {
            final errors = (msg['payload'] as List?)
                ?.map((e) => GraphQLError.fromJson(
                    Map<String, dynamic>.from(e as Map)))
                .toList();
            _subscriptions[id]
                ?.add(GraphQLResult(errors: errors));
          } else if (type == 'complete' && id != null) {
            _subscriptions[id]?.close();
            _subscriptions.remove(id);
          }
        } catch (_) {}
      },
      onDone: () => _closeWebSocket(),
      onError: (_) => _closeWebSocket(),
    );
  }

  void _closeWebSocket() {
    for (final controller in _subscriptions.values) {
      controller.close();
    }
    _subscriptions.clear();
    _ws = null;
  }

  /// Dispose WebSocket connections. Called by [NselfClient.dispose].
  void dispose() => _closeWebSocket();
}
