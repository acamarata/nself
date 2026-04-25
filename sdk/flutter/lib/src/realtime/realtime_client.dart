import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../client.dart';

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

/// Type of a Postgres CDC event.
enum PostgresChangeType { insert, update, delete, truncate }

/// A single Postgres change-data-capture event.
class PostgresChangesEvent {
  final PostgresChangeType type;
  final String schema;
  final String table;
  final Map<String, dynamic>? record;
  final Map<String, dynamic>? oldRecord;

  const PostgresChangesEvent({
    required this.type,
    required this.schema,
    required this.table,
    this.record,
    this.oldRecord,
  });

  factory PostgresChangesEvent.fromJson(Map<String, dynamic> json) {
    final typeStr = (json['type'] as String? ?? '').toLowerCase();
    final type = switch (typeStr) {
      'insert' => PostgresChangeType.insert,
      'update' => PostgresChangeType.update,
      'delete' => PostgresChangeType.delete,
      _ => PostgresChangeType.truncate,
    };
    return PostgresChangesEvent(
      type: type,
      schema: json['schema'] as String? ?? 'public',
      table: json['table'] as String? ?? '',
      record: json['record'] != null
          ? Map<String, dynamic>.from(json['record'] as Map)
          : null,
      oldRecord: json['old_record'] != null
          ? Map<String, dynamic>.from(json['old_record'] as Map)
          : null,
    );
  }
}

/// A generic realtime event (broadcast / presence / postgres_changes).
class RealtimeEvent {
  final String type;
  final Map<String, dynamic> payload;

  const RealtimeEvent({required this.type, required this.payload});
}

// ---------------------------------------------------------------------------
// RealtimeChannel
// ---------------------------------------------------------------------------

/// A named pub-sub channel.
///
/// Obtain via [RealtimeClient.channel].
class RealtimeChannel {
  final String _name;
  final RealtimeClient _client;

  final _postgresCallbacks =
      <void Function(PostgresChangesEvent)>[];
  final _broadcastCallbacks = <void Function(RealtimeEvent)>[];

  RealtimeChannel(this._name, this._client);

  /// Register a callback for Postgres CDC events on [table].
  ///
  /// ```dart
  /// nself.realtime
  ///   .channel('room:42')
  ///   .onPostgresChanges(
  ///     schema: 'public',
  ///     table: 'messages',
  ///     callback: (e) => print(e.record),
  ///   )
  ///   .subscribe();
  /// ```
  RealtimeChannel onPostgresChanges({
    String schema = 'public',
    required String table,
    PostgresChangeType? event,
    required void Function(PostgresChangesEvent) callback,
  }) {
    _postgresCallbacks.add(callback);
    return this;
  }

  /// Register a callback for broadcast messages on this channel.
  RealtimeChannel onBroadcast({
    required String event,
    required void Function(RealtimeEvent) callback,
  }) {
    _broadcastCallbacks.add(callback);
    return this;
  }

  /// Activate the channel subscription.
  void subscribe() => _client._subscribe(this);

  /// Unsubscribe this channel.
  void unsubscribe() => _client._unsubscribe(_name);

  void _dispatchPostgres(PostgresChangesEvent event) {
    for (final cb in _postgresCallbacks) {
      cb(event);
    }
  }

  void _dispatchBroadcast(RealtimeEvent event) {
    for (final cb in _broadcastCallbacks) {
      cb(event);
    }
  }
}

// ---------------------------------------------------------------------------
// RealtimeClient
// ---------------------------------------------------------------------------

/// Manages WebSocket connections for Postgres CDC and broadcast channels.
///
/// Access via [NselfClient.realtime].
class RealtimeClient {
  final NselfConfig _config;
  WebSocketChannel? _ws;
  final Map<String, RealtimeChannel> _channels = {};
  String? _accessToken;
  bool _connected = false;
  final _pendingSubscriptions = <RealtimeChannel>[];

  RealtimeClient({required NselfConfig config}) : _config = config;

  /// Set (or clear) the Authorization token.
  void setAccessToken(String? token) => _accessToken = token;

  /// Return a [RealtimeChannel] named [name].
  ///
  /// Channels are reused: calling [channel] twice with the same name returns
  /// the same object.
  RealtimeChannel channel(String name) {
    return _channels.putIfAbsent(name, () => RealtimeChannel(name, this));
  }

  // --- Internal ---

  void _subscribe(RealtimeChannel ch) {
    if (!_connected) {
      _pendingSubscriptions.add(ch);
      _connect();
      return;
    }
    _sendSubscribe(ch);
  }

  void _unsubscribe(String name) {
    _channels.remove(name);
    if (_ws != null) {
      _ws!.sink.add(jsonEncode({
        'event': 'phx_leave',
        'topic': 'realtime:$name',
        'payload': {},
        'ref': null,
      }));
    }
  }

  void _connect() {
    if (_ws != null) return;
    final wsUrl = _config.url
        .replaceFirst('https://', 'wss://')
        .replaceFirst('http://', 'ws://');
    final params = _accessToken != null
        ? '?apikey=${_config.anonKey}&vsn=1.0.0'
        : '?apikey=${_config.anonKey}&vsn=1.0.0';
    _ws = WebSocketChannel.connect(Uri.parse('$wsUrl/realtime/v1/websocket$params'));

    _ws!.stream.listen(
      (data) {
        try {
          final msg = jsonDecode(data as String) as Map<String, dynamic>;
          _handleMessage(msg);
        } catch (_) {}
      },
      onDone: () => _handleDisconnect(),
      onError: (_) => _handleDisconnect(),
    );

    // Phoenix heartbeat.
    Timer.periodic(const Duration(seconds: 30), (_) {
      if (_ws != null) {
        _ws!.sink.add(jsonEncode({
          'event': 'heartbeat',
          'topic': 'phoenix',
          'payload': {},
          'ref': null,
        }));
      }
    });

    _connected = true;
    for (final pending in List.of(_pendingSubscriptions)) {
      _sendSubscribe(pending);
    }
    _pendingSubscriptions.clear();
  }

  void _sendSubscribe(RealtimeChannel ch) {
    _ws?.sink.add(jsonEncode({
      'event': 'phx_join',
      'topic': 'realtime:${ch._name}',
      'payload': {
        'config': {
          'broadcast': {'self': false},
          'postgres_changes': [
            {'event': '*', 'schema': 'public'},
          ],
        },
        if (_accessToken != null) 'access_token': _accessToken,
      },
      'ref': null,
    }));
  }

  void _handleMessage(Map<String, dynamic> msg) {
    final event = msg['event'] as String?;
    final topic = msg['topic'] as String?;
    final payload = msg['payload'] as Map<String, dynamic>? ?? {};

    if (topic == null) return;
    final channelName = topic.startsWith('realtime:')
        ? topic.substring('realtime:'.length)
        : topic;
    final ch = _channels[channelName];
    if (ch == null) return;

    if (event == 'postgres_changes') {
      final data = payload['data'] as Map<String, dynamic>? ?? payload;
      ch._dispatchPostgres(PostgresChangesEvent.fromJson(data));
    } else if (event != null && event != 'phx_reply') {
      ch._dispatchBroadcast(RealtimeEvent(type: event, payload: payload));
    }
  }

  void _handleDisconnect() {
    _ws = null;
    _connected = false;
  }

  /// Dispose WebSocket connections.
  void dispose() {
    _ws?.sink.close();
    _ws = null;
    _channels.clear();
    _connected = false;
  }
}
