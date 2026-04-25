import 'dart:convert';
import 'dart:typed_data';
import 'package:http/http.dart' as http;
import '../client.dart';

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

/// Options for a storage upload operation.
class UploadOptions {
  final String? contentType;
  final bool upsert;

  const UploadOptions({this.contentType, this.upsert = false});
}

// ---------------------------------------------------------------------------
// StorageBucket
// ---------------------------------------------------------------------------

/// Scoped handle for a single MinIO/nSelf storage bucket.
///
/// Obtain via [StorageClient.from].
class StorageBucket {
  final String _bucketName;
  final StorageClient _client;

  StorageBucket(this._bucketName, this._client);

  /// Upload [data] to [path] in this bucket.
  ///
  /// ```dart
  /// await nself.storage.from('avatars').upload('uid/photo.jpg', bytes);
  /// ```
  Future<void> upload(
    String path,
    Uint8List data, {
    UploadOptions options = const UploadOptions(),
  }) =>
      _client._upload(_bucketName, path, data, options);

  /// Download the file at [path].
  Future<Uint8List> download(String path) =>
      _client._download(_bucketName, path);

  /// Return the public URL for [path] in this bucket.
  ///
  /// The URL is constructed locally — no network call required.
  String getPublicUrl(String path) =>
      '${_client._config.url}/storage/v1/object/public/$_bucketName/$path';

  /// Delete the file at [path].
  Future<void> remove(String path) => _client._remove(_bucketName, path);

  /// List files at the given [prefix] inside this bucket.
  Future<List<String>> list({String prefix = ''}) =>
      _client._list(_bucketName, prefix);
}

// ---------------------------------------------------------------------------
// StorageClient
// ---------------------------------------------------------------------------

/// Handles file storage operations against nSelf's MinIO-compatible API.
///
/// Access via [NselfClient.storage].
class StorageClient {
  final NselfConfig _config;
  final http.Client _http;

  String? _accessToken;

  StorageClient({required NselfConfig config, required http.Client httpClient})
      : _config = config,
        _http = httpClient;

  /// Set (or clear) the Authorization token.
  void setAccessToken(String? token) => _accessToken = token;

  /// Return a [StorageBucket] scoped to [bucketName].
  StorageBucket from(String bucketName) => StorageBucket(bucketName, this);

  // --- Internal operations ---

  Future<void> _upload(
    String bucket,
    String path,
    Uint8List data,
    UploadOptions options,
  ) async {
    final uri = Uri.parse(
        '${_config.url}/storage/v1/object/$bucket/$path');
    final method = options.upsert ? 'PUT' : 'POST';
    final request = http.Request(method, uri)
      ..headers.addAll({
        'Content-Type': options.contentType ?? 'application/octet-stream',
        ..._authHeader(),
      })
      ..bodyBytes = data;

    final streamed = await _http.send(request);
    if (streamed.statusCode < 200 || streamed.statusCode >= 300) {
      final body = await streamed.stream.bytesToString();
      throw Exception('Storage upload failed (${streamed.statusCode}): $body');
    }
  }

  Future<Uint8List> _download(String bucket, String path) async {
    final uri =
        Uri.parse('${_config.url}/storage/v1/object/$bucket/$path');
    final resp = await _http.get(uri, headers: _authHeader());
    if (resp.statusCode != 200) {
      throw Exception('Storage download failed (${resp.statusCode})');
    }
    return resp.bodyBytes;
  }

  Future<void> _remove(String bucket, String path) async {
    final uri =
        Uri.parse('${_config.url}/storage/v1/object/$bucket/$path');
    final resp = await _http.delete(uri, headers: _authHeader());
    if (resp.statusCode < 200 || resp.statusCode >= 300) {
      throw Exception('Storage delete failed (${resp.statusCode})');
    }
  }

  Future<List<String>> _list(String bucket, String prefix) async {
    final uri =
        Uri.parse('${_config.url}/storage/v1/object/list/$bucket');
    final resp = await _http.post(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeader()},
      body: jsonEncode({'prefix': prefix, 'limit': 100}),
    );
    if (resp.statusCode != 200) {
      throw Exception('Storage list failed (${resp.statusCode})');
    }
    final list = jsonDecode(resp.body) as List;
    return list.map((e) => (e as Map)['name'] as String).toList();
  }

  Map<String, String> _authHeader() => {
        if (_accessToken != null) 'Authorization': 'Bearer $_accessToken',
        if (_accessToken == null) 'apikey': _config.anonKey,
      };
}
