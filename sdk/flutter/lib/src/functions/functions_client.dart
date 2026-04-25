import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:http/http.dart' as http;
import '../client.dart';

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

/// The response from a serverless function invocation.
class FunctionResponse {
  final int statusCode;
  final Map<String, String> headers;
  final dynamic data;

  const FunctionResponse({
    required this.statusCode,
    required this.headers,
    this.data,
  });

  bool get ok => statusCode >= 200 && statusCode < 300;
}

/// An error from a functions invocation.
class FunctionsError implements Exception {
  final String message;
  final int? statusCode;

  const FunctionsError(this.message, {this.statusCode});

  @override
  String toString() => 'FunctionsError($statusCode): $message';
}

// ---------------------------------------------------------------------------
// FunctionsClient
// ---------------------------------------------------------------------------

/// Invokes nSelf serverless functions.
///
/// Access via [NselfClient.functions].
///
/// nSelf functions live at `{url}/functions/v1/{functionName}`.
class FunctionsClient {
  final NselfConfig _config;
  final http.Client _http;
  String? _accessToken;

  FunctionsClient({required NselfConfig config, required http.Client httpClient})
      : _config = config,
        _http = httpClient;

  /// Set (or clear) the Authorization token.
  void setAccessToken(String? token) => _accessToken = token;

  /// Invoke [functionName] with an optional JSON [body].
  ///
  /// ```dart
  /// final resp = await nself.functions.invoke('send-welcome-email', body: {
  ///   'user_id': userId,
  /// });
  /// ```
  Future<FunctionResponse> invoke(
    String functionName, {
    Map<String, dynamic>? body,
    Map<String, String>? headers,
    String method = 'POST',
  }) async {
    final uri =
        Uri.parse('${_config.url}/functions/v1/$functionName');
    try {
      final reqHeaders = {
        'Content-Type': 'application/json',
        ..._authHeader(),
        if (headers != null) ...headers,
      };
      final http.Response resp;
      if (method == 'GET') {
        resp = await _http.get(uri, headers: reqHeaders);
      } else {
        resp = await _http.post(
          uri,
          headers: reqHeaders,
          body: body != null ? jsonEncode(body) : null,
        );
      }

      dynamic decoded;
      final ct = resp.headers['content-type'] ?? '';
      if (ct.contains('application/json') && resp.body.isNotEmpty) {
        decoded = jsonDecode(resp.body);
      } else {
        decoded = resp.body;
      }

      if (resp.statusCode < 200 || resp.statusCode >= 300) {
        final msg = decoded is Map
            ? (decoded['error'] ?? decoded['message'] ?? 'Function error')
            : decoded;
        throw FunctionsError('$msg', statusCode: resp.statusCode);
      }

      return FunctionResponse(
        statusCode: resp.statusCode,
        headers: resp.headers,
        data: decoded,
      );
    } on SocketException catch (e) {
      throw FunctionsError('Network error: ${e.message}');
    }
  }

  /// Invoke [functionName] and return raw bytes (for binary-returning functions).
  Future<Uint8List> invokeRaw(
    String functionName, {
    Map<String, dynamic>? body,
  }) async {
    final uri =
        Uri.parse('${_config.url}/functions/v1/$functionName');
    final resp = await _http.post(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeader()},
      body: body != null ? jsonEncode(body) : null,
    );
    if (resp.statusCode < 200 || resp.statusCode >= 300) {
      throw FunctionsError('HTTP ${resp.statusCode}', statusCode: resp.statusCode);
    }
    return resp.bodyBytes;
  }

  Map<String, String> _authHeader() => {
        if (_accessToken != null) 'Authorization': 'Bearer $_accessToken',
        if (_accessToken == null) 'apikey': _config.anonKey,
      };
}
