import 'dart:convert';
import 'dart:io';
import 'package:http/http.dart' as http;
import '../client.dart';

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

/// A signed-in user.
class NselfUser {
  final String id;
  final String email;
  final String? displayName;
  final DateTime? accessTokenExpiry;

  const NselfUser({
    required this.id,
    required this.email,
    this.displayName,
    this.accessTokenExpiry,
  });

  factory NselfUser.fromJson(Map<String, dynamic> json) {
    return NselfUser(
      id: json['id'] as String,
      email: json['email'] as String,
      displayName: json['display_name'] as String?,
      accessTokenExpiry: json['access_token_expiry'] != null
          ? DateTime.tryParse(json['access_token_expiry'] as String)
          : null,
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'email': email,
        if (displayName != null) 'display_name': displayName,
        if (accessTokenExpiry != null)
          'access_token_expiry': accessTokenExpiry!.toIso8601String(),
      };
}

/// An active session returned by sign-in or sign-up.
class Session {
  final String accessToken;
  final String refreshToken;
  final NselfUser user;

  const Session({
    required this.accessToken,
    required this.refreshToken,
    required this.user,
  });

  factory Session.fromJson(Map<String, dynamic> json) {
    return Session(
      accessToken: json['access_token'] as String,
      refreshToken: json['refresh_token'] as String,
      user: NselfUser.fromJson(json['user'] as Map<String, dynamic>),
    );
  }
}

/// An error from the auth service.
class AuthError implements Exception {
  final String message;
  final int? statusCode;

  const AuthError(this.message, {this.statusCode});

  @override
  String toString() => 'AuthError($statusCode): $message';
}

// ---------------------------------------------------------------------------
// AuthClient
// ---------------------------------------------------------------------------

/// Handles all authentication operations against the nSelf auth endpoints.
///
/// Access via [NselfClient.auth].
class AuthClient {
  final NselfConfig _config;
  final http.Client _http;

  Session? _currentSession;

  AuthClient({required NselfConfig config, required http.Client httpClient})
      : _config = config,
        _http = httpClient;

  // --- Public API ---

  /// The active session, or null when signed out.
  Session? get currentSession => _currentSession;

  /// The currently signed-in user, or null.
  NselfUser? get currentUser => _currentSession?.user;

  /// Sign in with email and password.
  ///
  /// Returns the new [Session] on success.
  /// Throws [AuthError] on failure.
  Future<Session> signInWithEmail(String email, String password) async {
    final body = await _post('/auth/v1/token?grant_type=password', {
      'email': email,
      'password': password,
    });
    final session = Session.fromJson(body);
    _currentSession = session;
    return session;
  }

  /// Create a new account.
  ///
  /// Returns the new [Session] on success.
  Future<Session> signUp(
    String email,
    String password, {
    String? displayName,
  }) async {
    final body = await _post('/auth/v1/signup', {
      'email': email,
      'password': password,
      if (displayName != null) 'data': {'display_name': displayName},
    });
    final session = Session.fromJson(body);
    _currentSession = session;
    return session;
  }

  /// Refresh the session using the stored refresh token.
  ///
  /// Call this when the access token has expired.
  Future<Session> refreshSession(String refreshToken) async {
    final body = await _post('/auth/v1/token?grant_type=refresh_token', {
      'refresh_token': refreshToken,
    });
    final session = Session.fromJson(body);
    _currentSession = session;
    return session;
  }

  /// Sign out the current user.
  ///
  /// Revokes the access token server-side (best-effort).
  Future<void> signOut() async {
    final token = _currentSession?.accessToken;
    _currentSession = null;
    if (token != null) {
      try {
        await _post('/auth/v1/logout', {}, bearerToken: token);
      } catch (_) {
        // Best-effort; session is cleared locally regardless.
      }
    }
  }

  // --- Internal HTTP helpers ---

  Future<Map<String, dynamic>> _post(
    String path,
    Map<String, dynamic> body, {
    String? bearerToken,
  }) async {
    final uri = Uri.parse('${_config.url}$path');
    try {
      final resp = await _http.post(
        uri,
        headers: _headers(bearerToken),
        body: jsonEncode(body),
      );
      return _handleResponse(resp);
    } on SocketException catch (e) {
      throw AuthError('Network error: ${e.message}');
    }
  }

  Map<String, String> _headers(String? bearerToken) => {
        'Content-Type': 'application/json',
        'apikey': _config.anonKey,
        if (bearerToken != null) 'Authorization': 'Bearer $bearerToken',
      };

  Map<String, dynamic> _handleResponse(http.Response resp) {
    if (resp.statusCode >= 200 && resp.statusCode < 300) {
      if (resp.body.isEmpty) return {};
      return jsonDecode(resp.body) as Map<String, dynamic>;
    }
    String? msg;
    try {
      final decoded = jsonDecode(resp.body) as Map<String, dynamic>;
      msg = decoded['error_description'] as String? ??
          decoded['message'] as String? ??
          decoded['error'] as String?;
    } catch (_) {}
    throw AuthError(msg ?? 'HTTP ${resp.statusCode}', statusCode: resp.statusCode);
  }
}
