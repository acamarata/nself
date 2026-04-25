import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:nself_sdk/nself_sdk.dart';

/// Example app demonstrating all nself_sdk surfaces.
///
/// Configure your nSelf backend URL and anon key via --dart-define:
///   flutter run \
///     --dart-define=NSELF_URL=https://api.myapp.com \
///     --dart-define=NSELF_ANON_KEY=your_anon_key
void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  await NselfClient.initialize(
    url: const String.fromEnvironment(
      'NSELF_URL',
      defaultValue: 'https://api.nself.org',
    ),
    anonKey: const String.fromEnvironment(
      'NSELF_ANON_KEY',
      defaultValue: 'changeme',
    ),
  );

  runApp(const NselfExampleApp());
}

class NselfExampleApp extends StatelessWidget {
  const NselfExampleApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'nself_sdk Example',
      theme: ThemeData(colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue)),
      home: const ExampleHome(),
    );
  }
}

// ---------------------------------------------------------------------------
// Home screen — tab layout, one tab per SDK surface
// ---------------------------------------------------------------------------

class ExampleHome extends StatelessWidget {
  const ExampleHome({super.key});

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 5,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('nself_sdk Demo'),
          bottom: const TabBar(
            isScrollable: true,
            tabs: [
              Tab(text: 'Auth'),
              Tab(text: 'GraphQL'),
              Tab(text: 'Storage'),
              Tab(text: 'Realtime'),
              Tab(text: 'Functions'),
            ],
          ),
        ),
        body: const TabBarView(
          children: [
            AuthDemo(),
            GraphQLDemo(),
            StorageDemo(),
            RealtimeDemo(),
            FunctionsDemo(),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Auth demo
// ---------------------------------------------------------------------------

class AuthDemo extends StatefulWidget {
  const AuthDemo({super.key});

  @override
  State<AuthDemo> createState() => _AuthDemoState();
}

class _AuthDemoState extends State<AuthDemo> {
  final _emailCtrl = TextEditingController(text: 'demo@nself.org');
  final _passCtrl = TextEditingController(text: 'password123');
  String _status = 'Not signed in';

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextField(
              controller: _emailCtrl,
              decoration: const InputDecoration(labelText: 'Email')),
          TextField(
              controller: _passCtrl,
              decoration: const InputDecoration(labelText: 'Password'),
              obscureText: true),
          const SizedBox(height: 12),
          Row(children: [
            FilledButton(
              onPressed: _signIn,
              child: const Text('Sign In'),
            ),
            const SizedBox(width: 8),
            OutlinedButton(
              onPressed: _signOut,
              child: const Text('Sign Out'),
            ),
          ]),
          const SizedBox(height: 12),
          Text(_status, style: const TextStyle(fontFamily: 'monospace')),
        ],
      ),
    );
  }

  Future<void> _signIn() async {
    try {
      final session = await NselfClient.instance.auth.signInWithEmail(
        _emailCtrl.text.trim(),
        _passCtrl.text,
      );
      setState(() => _status = 'Signed in as ${session.user.email}');
    } on AuthError catch (e) {
      setState(() => _status = 'Error: $e');
    }
  }

  Future<void> _signOut() async {
    await NselfClient.instance.auth.signOut();
    setState(() => _status = 'Signed out');
  }
}

// ---------------------------------------------------------------------------
// GraphQL demo
// ---------------------------------------------------------------------------

class GraphQLDemo extends StatefulWidget {
  const GraphQLDemo({super.key});

  @override
  State<GraphQLDemo> createState() => _GraphQLDemoState();
}

class _GraphQLDemoState extends State<GraphQLDemo> {
  String _result = 'Press "Query" to run a query.';

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          FilledButton(onPressed: _runQuery, child: const Text('Query')),
          const SizedBox(height: 12),
          Expanded(
            child: SingleChildScrollView(
              child: Text(_result,
                  style: const TextStyle(fontFamily: 'monospace', fontSize: 12)),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _runQuery() async {
    final gql = NselfClient.instance.graphql;
    // Set the auth token from the current session if signed in.
    final token = NselfClient.instance.auth.currentSession?.accessToken;
    gql.setAccessToken(token);

    final result = await gql.rawQuery(r'''
      query {
        np_users(limit: 5) {
          id
          email
        }
      }
    ''');

    setState(() {
      if (result.hasErrors) {
        _result = 'Errors:\n${result.errors!.map((e) => e.message).join('\n')}';
      } else {
        _result = 'Data:\n${result.data}';
      }
    });
  }
}

// ---------------------------------------------------------------------------
// Storage demo
// ---------------------------------------------------------------------------

class StorageDemo extends StatefulWidget {
  const StorageDemo({super.key});

  @override
  State<StorageDemo> createState() => _StorageDemoState();
}

class _StorageDemoState extends State<StorageDemo> {
  String _status = 'Ready.';

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          FilledButton(onPressed: _upload, child: const Text('Upload test file')),
          const SizedBox(height: 8),
          FilledButton(onPressed: _getUrl, child: const Text('Get public URL')),
          const SizedBox(height: 12),
          Text(_status, style: const TextStyle(fontFamily: 'monospace')),
        ],
      ),
    );
  }

  Future<void> _upload() async {
    try {
      final storage = NselfClient.instance.storage;
      storage.setAccessToken(
          NselfClient.instance.auth.currentSession?.accessToken);
      await storage
          .from('example')
          .upload('test.txt', Uint8List.fromList('Hello nSelf!'.codeUnits));
      setState(() => _status = 'Upload complete.');
    } catch (e) {
      setState(() => _status = 'Upload error: $e');
    }
  }

  void _getUrl() {
    final url = NselfClient.instance.storage
        .from('example')
        .getPublicUrl('test.txt');
    setState(() => _status = 'URL: $url');
  }
}

// ---------------------------------------------------------------------------
// Realtime demo
// ---------------------------------------------------------------------------

class RealtimeDemo extends StatefulWidget {
  const RealtimeDemo({super.key});

  @override
  State<RealtimeDemo> createState() => _RealtimeDemoState();
}

class _RealtimeDemoState extends State<RealtimeDemo> {
  final _events = <String>[];
  SubscriptionHandle? _handle;
  RealtimeChannel? _channel;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            FilledButton(onPressed: _subscribe, child: const Text('Subscribe')),
            const SizedBox(width: 8),
            OutlinedButton(onPressed: _unsubscribe, child: const Text('Unsubscribe')),
          ]),
          const SizedBox(height: 12),
          Expanded(
            child: ListView.builder(
              itemCount: _events.length,
              itemBuilder: (ctx, i) => Text(
                _events[i],
                style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
              ),
            ),
          ),
        ],
      ),
    );
  }

  void _subscribe() {
    final realtime = NselfClient.instance.realtime;
    realtime.setAccessToken(
        NselfClient.instance.auth.currentSession?.accessToken);

    _channel = realtime.channel('demo');
    _channel!
        .onPostgresChanges(
          table: 'np_messages',
          callback: (e) {
            setState(() => _events.add(
                '${e.type.name.toUpperCase()}: ${e.record ?? e.oldRecord}'));
          },
        )
        .subscribe();
  }

  void _unsubscribe() {
    _channel?.unsubscribe();
    setState(() => _events.clear());
  }

  @override
  void dispose() {
    _channel?.unsubscribe();
    super.dispose();
  }
}

// ---------------------------------------------------------------------------
// Functions demo
// ---------------------------------------------------------------------------

class FunctionsDemo extends StatefulWidget {
  const FunctionsDemo({super.key});

  @override
  State<FunctionsDemo> createState() => _FunctionsDemoState();
}

class _FunctionsDemoState extends State<FunctionsDemo> {
  final _fnCtrl = TextEditingController(text: 'hello');
  String _result = 'Press "Invoke" to call a function.';

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextField(
              controller: _fnCtrl,
              decoration: const InputDecoration(labelText: 'Function name')),
          const SizedBox(height: 8),
          FilledButton(onPressed: _invoke, child: const Text('Invoke')),
          const SizedBox(height: 12),
          Text(_result, style: const TextStyle(fontFamily: 'monospace')),
        ],
      ),
    );
  }

  Future<void> _invoke() async {
    try {
      final functions = NselfClient.instance.functions;
      functions.setAccessToken(
          NselfClient.instance.auth.currentSession?.accessToken);
      final resp = await functions.invoke(
        _fnCtrl.text.trim(),
        body: {'hello': 'nself'},
      );
      setState(() => _result = 'Status: ${resp.statusCode}\n${resp.data}');
    } on FunctionsError catch (e) {
      setState(() => _result = 'Error: $e');
    }
  }
}
