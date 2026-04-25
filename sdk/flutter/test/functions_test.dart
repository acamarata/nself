import 'dart:convert';
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

  group('FunctionsClient.invoke', () {
    test('posts to /functions/v1/{name} and returns decoded JSON', () async {
      String? capturedPath;
      final mockClient = MockClient((req) async {
        capturedPath = req.url.path;
        return http.Response(
          jsonEncode({'sent': true}),
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final functions =
          FunctionsClient(config: config, httpClient: mockClient);
      final resp = await functions.invoke('send-welcome-email', body: {
        'user_id': 'u1',
      });

      expect(capturedPath, '/functions/v1/send-welcome-email');
      expect(resp.ok, isTrue);
      expect(resp.data['sent'], isTrue);
    });

    test('throws FunctionsError on non-2xx', () async {
      final mockClient = MockClient((_) async => http.Response(
            jsonEncode({'error': 'Function not found'}),
            404,
            headers: {'content-type': 'application/json'},
          ));

      final functions =
          FunctionsClient(config: config, httpClient: mockClient);

      expect(
        () => functions.invoke('missing-fn'),
        throwsA(isA<FunctionsError>()
            .having((e) => e.statusCode, 'statusCode', 404)),
      );
    });

    test('sends Authorization header when token is set', () async {
      String? capturedAuth;
      final mockClient = MockClient((req) async {
        capturedAuth = req.headers['Authorization'];
        return http.Response(jsonEncode({}), 200,
            headers: {'content-type': 'application/json'});
      });

      final functions =
          FunctionsClient(config: config, httpClient: mockClient);
      functions.setAccessToken('bearer_xyz');
      await functions.invoke('my-fn');

      expect(capturedAuth, 'Bearer bearer_xyz');
    });

    test('sends GET request when method is GET', () async {
      String? capturedMethod;
      final mockClient = MockClient((req) async {
        capturedMethod = req.method;
        return http.Response(jsonEncode({'ok': true}), 200,
            headers: {'content-type': 'application/json'});
      });

      final functions =
          FunctionsClient(config: config, httpClient: mockClient);
      await functions.invoke('health-check', method: 'GET');

      expect(capturedMethod, 'GET');
    });

    test('FunctionResponse.ok is false when statusCode is 4xx', () {
      const resp = FunctionResponse(
        statusCode: 422,
        headers: {},
        data: null,
      );
      expect(resp.ok, isFalse);
    });
  });

  group('FunctionsClient.invokeRaw', () {
    test('returns raw bytes', () async {
      final expected = [0x89, 0x50, 0x4E, 0x47]; // PNG magic bytes
      final mockClient = MockClient((_) async => http.Response.bytes(
            expected,
            200,
          ));

      final functions =
          FunctionsClient(config: config, httpClient: mockClient);
      final bytes = await functions.invokeRaw('generate-image');

      expect(bytes, expected);
    });
  });
}
