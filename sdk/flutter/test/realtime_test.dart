import 'package:flutter_test/flutter_test.dart';
import 'package:nself_sdk/nself_sdk.dart';

void main() {
  late NselfConfig config;

  setUp(() {
    config = const NselfConfig(
      url: 'https://api.test.nself.io',
      anonKey: 'test-anon-key',
    );
  });

  group('RealtimeClient.channel', () {
    test('returns the same channel instance for the same name', () {
      final realtime = RealtimeClient(config: config);
      final ch1 = realtime.channel('room:42');
      final ch2 = realtime.channel('room:42');

      expect(identical(ch1, ch2), isTrue);
      realtime.dispose();
    });

    test('returns different instances for different names', () {
      final realtime = RealtimeClient(config: config);
      final ch1 = realtime.channel('room:1');
      final ch2 = realtime.channel('room:2');

      expect(identical(ch1, ch2), isFalse);
      realtime.dispose();
    });
  });

  group('RealtimeChannel builder', () {
    test('onPostgresChanges returns the channel for chaining', () {
      final realtime = RealtimeClient(config: config);
      final ch = realtime.channel('test');

      final returned = ch.onPostgresChanges(
        table: 'messages',
        callback: (_) {},
      );

      expect(returned, same(ch));
      realtime.dispose();
    });

    test('onBroadcast returns the channel for chaining', () {
      final realtime = RealtimeClient(config: config);
      final ch = realtime.channel('test');

      final returned = ch.onBroadcast(
        event: 'cursor_move',
        callback: (_) {},
      );

      expect(returned, same(ch));
      realtime.dispose();
    });
  });

  group('PostgresChangesEvent', () {
    test('parses INSERT event', () {
      final event = PostgresChangesEvent.fromJson({
        'type': 'INSERT',
        'schema': 'public',
        'table': 'np_messages',
        'record': {'id': 'msg-1', 'body': 'Hello'},
        'old_record': null,
      });

      expect(event.type, PostgresChangeType.insert);
      expect(event.table, 'np_messages');
      expect(event.record?['body'], 'Hello');
      expect(event.oldRecord, isNull);
    });

    test('parses UPDATE event with old_record', () {
      final event = PostgresChangesEvent.fromJson({
        'type': 'UPDATE',
        'schema': 'public',
        'table': 'np_messages',
        'record': {'id': 'msg-1', 'body': 'Updated'},
        'old_record': {'id': 'msg-1', 'body': 'Original'},
      });

      expect(event.type, PostgresChangeType.update);
      expect(event.oldRecord?['body'], 'Original');
    });

    test('parses DELETE event', () {
      final event = PostgresChangesEvent.fromJson({
        'type': 'DELETE',
        'schema': 'public',
        'table': 'np_messages',
        'old_record': {'id': 'msg-1'},
      });

      expect(event.type, PostgresChangeType.delete);
    });

    test('unknown type defaults to truncate', () {
      final event = PostgresChangesEvent.fromJson({
        'type': 'TRUNCATE',
        'schema': 'public',
        'table': 'np_messages',
      });

      expect(event.type, PostgresChangeType.truncate);
    });
  });

  group('RealtimeClient.dispose', () {
    test('can be called multiple times safely', () {
      final realtime = RealtimeClient(config: config);
      realtime.dispose();
      realtime.dispose(); // should not throw
    });
  });
}
