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

  group('GraphQLClient.rawQuery', () {
    test('returns data on success', () async {
      final mockClient = MockClient((_) async => http.Response(
            jsonEncode({
              'data': {
                'np_users': [
                  {'id': 'u1', 'email': 'a@test.com'},
                  {'id': 'u2', 'email': 'b@test.com'},
                ],
              },
            }),
            200,
            headers: {'content-type': 'application/json'},
          ));

      final graphql = GraphQLClient(config: config, httpClient: mockClient);
      final result = await graphql.rawQuery('query { np_users { id email } }');

      expect(result.hasErrors, isFalse);
      expect(result.data, isNotNull);
      final users = result.data!['np_users'] as List;
      expect(users.length, 2);
      expect(users.first['email'], 'a@test.com');
    });

    test('returns errors on GraphQL error response', () async {
      final mockClient = MockClient((_) async => http.Response(
            jsonEncode({
              'errors': [
                {'message': 'field "np_missing" not found'},
              ],
            }),
            200,
            headers: {'content-type': 'application/json'},
          ));

      final graphql = GraphQLClient(config: config, httpClient: mockClient);
      final result =
          await graphql.rawQuery('query { np_missing { id } }');

      expect(result.hasErrors, isTrue);
      expect(result.errors!.first.message, contains('np_missing'));
    });

    test('returns error result on HTTP 500', () async {
      final mockClient = MockClient(
          (_) async => http.Response('Internal server error', 500));

      final graphql = GraphQLClient(config: config, httpClient: mockClient);
      final result = await graphql.rawQuery('query { np_users { id } }');

      expect(result.hasErrors, isTrue);
      expect(result.errors!.first.message, contains('500'));
    });

    test('includes Authorization header when token is set', () async {
      String? capturedAuth;
      final mockClient = MockClient((req) async {
        capturedAuth = req.headers['Authorization'];
        return http.Response(
          jsonEncode({'data': {}}),
          200,
          headers: {'content-type': 'application/json'},
        );
      });

      final graphql = GraphQLClient(config: config, httpClient: mockClient);
      graphql.setAccessToken('user_jwt_token');
      await graphql.rawQuery('query { np_users { id } }');

      expect(capturedAuth, 'Bearer user_jwt_token');
    });
  });

  group('GraphQLClient.rawMutation', () {
    test('executes mutation and returns result', () async {
      final mockClient = MockClient((_) async => http.Response(
            jsonEncode({
              'data': {
                'insert_np_messages_one': {'id': 'msg-1'},
              },
            }),
            200,
            headers: {'content-type': 'application/json'},
          ));

      final graphql = GraphQLClient(config: config, httpClient: mockClient);
      final result = await graphql.rawMutation(
        r'mutation InsertMessage($body: String!) { insert_np_messages_one(object: {body: $body}) { id } }',
        variables: {'body': 'Hello world'},
      );

      expect(result.hasErrors, isFalse);
      expect(result.data!['insert_np_messages_one']['id'], 'msg-1');
    });
  });

  group('GraphQLResult', () {
    test('hasErrors is false when errors is null', () {
      const result = GraphQLResult(data: {});
      expect(result.hasErrors, isFalse);
    });

    test('hasErrors is false when errors list is empty', () {
      const result = GraphQLResult(errors: []);
      expect(result.hasErrors, isFalse);
    });

    test('hasErrors is true when errors present', () {
      final result = GraphQLResult(
          errors: [const GraphQLError(message: 'oops')]);
      expect(result.hasErrors, isTrue);
    });
  });
}
