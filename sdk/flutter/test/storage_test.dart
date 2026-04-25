import 'dart:convert';
import 'dart:typed_data';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:nself_sdk/nself_sdk.dart';

void main() {
  late NselfConfig config;

  setUp(() {
    config = const NselfConfig(
      url: 'https://api.test.nself.io',
      anonKey: 'test-anon-key',
    );
  });

  group('StorageBucket.getPublicUrl', () {
    test('builds correct URL without network call', () {
      final storage =
          StorageClient(config: config, httpClient: http.Client());
      final url = storage.from('avatars').getPublicUrl('uid/photo.jpg');

      expect(
        url,
        'https://api.test.nself.io/storage/v1/object/public/avatars/uid/photo.jpg',
      );
    });
  });

  group('StorageBucket.upload', () {
    test('calls POST /storage/v1/object on upload', () async {
      String? capturedPath;
      final mockClient = MockClient((req) async {
        capturedPath = req.url.path;
        return http.Response('{"Key": "avatars/uid/photo.jpg"}', 200);
      });

      final storage = StorageClient(config: config, httpClient: mockClient);
      await storage.from('avatars').upload(
            'uid/photo.jpg',
            Uint8List.fromList([1, 2, 3]),
          );

      expect(capturedPath, '/storage/v1/object/avatars/uid/photo.jpg');
    });

    test('throws on non-2xx status', () async {
      final mockClient = MockClient(
          (_) async => http.Response('Forbidden', 403));

      final storage = StorageClient(config: config, httpClient: mockClient);
      expect(
        () => storage.from('avatars').upload(
              'private/file.jpg',
              Uint8List.fromList([0]),
            ),
        throwsException,
      );
    });
  });

  group('StorageBucket.download', () {
    test('returns bytes on 200', () async {
      final expected = Uint8List.fromList([10, 20, 30]);
      final mockClient = MockClient(
          (_) async => http.Response.bytes(expected, 200));

      final storage = StorageClient(config: config, httpClient: mockClient);
      final bytes = await storage.from('avatars').download('uid/photo.jpg');

      expect(bytes, expected);
    });

    test('throws on 404', () async {
      final mockClient =
          MockClient((_) async => http.Response('Not Found', 404));

      final storage = StorageClient(config: config, httpClient: mockClient);
      expect(
        () => storage.from('avatars').download('missing.jpg'),
        throwsException,
      );
    });
  });

  group('StorageBucket.remove', () {
    test('calls DELETE endpoint', () async {
      String? method;
      final mockClient = MockClient((req) async {
        method = req.method;
        return http.Response('{}', 200);
      });

      final storage = StorageClient(config: config, httpClient: mockClient);
      await storage.from('avatars').remove('uid/old.jpg');

      expect(method, 'DELETE');
    });
  });

  group('StorageBucket.list', () {
    test('returns file names', () async {
      final mockClient = MockClient((_) async => http.Response(
            jsonEncode([
              {'name': 'file1.jpg'},
              {'name': 'file2.jpg'},
            ]),
            200,
            headers: {'content-type': 'application/json'},
          ));

      final storage = StorageClient(config: config, httpClient: mockClient);
      final files = await storage.from('avatars').list(prefix: 'uid/');

      expect(files, containsAll(['file1.jpg', 'file2.jpg']));
    });
  });

  group('StorageClient authorization', () {
    test('uses apikey header when no token set', () async {
      String? capturedApiKey;
      final mockClient = MockClient((req) async {
        capturedApiKey = req.headers['apikey'];
        return http.Response('{"Key":"ok"}', 200);
      });

      final storage = StorageClient(config: config, httpClient: mockClient);
      await storage.from('b').upload('f', Uint8List.fromList([0]));

      expect(capturedApiKey, 'test-anon-key');
    });

    test('uses Authorization header when token is set', () async {
      String? capturedAuth;
      final mockClient = MockClient((req) async {
        capturedAuth = req.headers['Authorization'];
        return http.Response('{"Key":"ok"}', 200);
      });

      final storage = StorageClient(config: config, httpClient: mockClient);
      storage.setAccessToken('user_jwt');
      await storage.from('b').upload('f', Uint8List.fromList([0]));

      expect(capturedAuth, 'Bearer user_jwt');
    });
  });
}
