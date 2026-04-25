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

  group('AuthClient.signInWithEmail', () {
    test('returns a Session on 200', () async {
      final mockClient = MockClient((req) async {
        expect(req.url.path, contains('/auth/v1/token'));
        final body = jsonDecode(req.body) as Map<String, dynamic>;
        expect(body['email'], 'user@test.com');
        expect(body['password'], 'password123');

        return http.Response(
          jsonEncode({
            'access_token': 'at_test',
            'refresh_token': 'rt_test',
            'user': {
              'id': 'user-uuid',
              'email': 'user@test.com',
              'display_name': 'Test User',
            },
          }),
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final auth = AuthClient(config: config, httpClient: mockClient);
      final session = await auth.signInWithEmail('user@test.com', 'password123');

      expect(session.accessToken, 'at_test');
      expect(session.refreshToken, 'rt_test');
      expect(session.user.id, 'user-uuid');
      expect(session.user.email, 'user@test.com');
      expect(auth.currentSession, isNotNull);
      expect(auth.currentUser?.email, 'user@test.com');
    });

    test('throws AuthError on 401', () async {
      final mockClient = MockClient((_) async => http.Response(
            jsonEncode({'error_description': 'Invalid credentials'}),
            401,
            headers: {'content-type': 'application/json'},
          ));

      final auth = AuthClient(config: config, httpClient: mockClient);
      expect(
        () => auth.signInWithEmail('bad@test.com', 'wrong'),
        throwsA(isA<AuthError>().having((e) => e.statusCode, 'statusCode', 401)),
      );
    });
  });

  group('AuthClient.signUp', () {
    test('creates user and returns session on 200', () async {
      final mockClient = MockClient((req) async {
        return http.Response(
          jsonEncode({
            'access_token': 'at_signup',
            'refresh_token': 'rt_signup',
            'user': {
              'id': 'new-user-uuid',
              'email': 'new@test.com',
            },
          }),
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final auth = AuthClient(config: config, httpClient: mockClient);
      final session = await auth.signUp('new@test.com', 'secure123',
          displayName: 'New User');

      expect(session.user.id, 'new-user-uuid');
    });
  });

  group('AuthClient.refreshSession', () {
    test('returns refreshed session', () async {
      final mockClient = MockClient((_) async => http.Response(
            jsonEncode({
              'access_token': 'at_refreshed',
              'refresh_token': 'rt_refreshed',
              'user': {'id': 'uid', 'email': 'u@t.com'},
            }),
            200,
            headers: {'content-type': 'application/json'},
          ));

      final auth = AuthClient(config: config, httpClient: mockClient);
      final session = await auth.refreshSession('old_rt');

      expect(session.accessToken, 'at_refreshed');
    });
  });

  group('AuthClient.signOut', () {
    test('clears session and calls logout endpoint', () async {
      bool logoutCalled = false;
      final mockClient = MockClient((req) async {
        if (req.url.path.contains('/auth/v1/token')) {
          return http.Response(
            jsonEncode({
              'access_token': 'at_test',
              'refresh_token': 'rt_test',
              'user': {'id': 'uid', 'email': 'u@t.com'},
            }),
            200,
            headers: {'content-type': 'application/json'},
          );
        }
        if (req.url.path.contains('/auth/v1/logout')) {
          logoutCalled = true;
          return http.Response('{}', 200);
        }
        return http.Response('{}', 200);
      });

      final auth = AuthClient(config: config, httpClient: mockClient);
      await auth.signInWithEmail('u@t.com', 'pw');
      expect(auth.currentSession, isNotNull);

      await auth.signOut();
      expect(auth.currentSession, isNull);
      expect(logoutCalled, isTrue);
    });

    test('clears session even when logout endpoint fails', () async {
      final mockClient = MockClient((req) async {
        if (req.url.path.contains('/auth/v1/token')) {
          return http.Response(
            jsonEncode({
              'access_token': 'at_test',
              'refresh_token': 'rt_test',
              'user': {'id': 'uid', 'email': 'u@t.com'},
            }),
            200,
            headers: {'content-type': 'application/json'},
          );
        }
        return http.Response('Server error', 500);
      });

      final auth = AuthClient(config: config, httpClient: mockClient);
      await auth.signInWithEmail('u@t.com', 'pw');
      await auth.signOut();
      expect(auth.currentSession, isNull);
    });
  });

  group('NselfUser', () {
    test('parses optional fields gracefully', () {
      final user = NselfUser.fromJson({'id': 'uid', 'email': 'u@t.com'});
      expect(user.displayName, isNull);
      expect(user.accessTokenExpiry, isNull);
    });

    test('round-trips through toJson/fromJson', () {
      final original = NselfUser(
        id: 'test-id',
        email: 'user@example.com',
        displayName: 'Test',
      );
      final copy = NselfUser.fromJson(original.toJson());
      expect(copy.id, original.id);
      expect(copy.email, original.email);
      expect(copy.displayName, original.displayName);
    });
  });
}
