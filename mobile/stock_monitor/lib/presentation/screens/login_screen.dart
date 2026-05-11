import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:dio/dio.dart';
import '../../core/config.dart';
import '../../core/theme.dart';
import '../providers/auth_provider.dart';

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  bool _isRegister = false;
  final _usernameCtrl = TextEditingController();
  final _passwordCtrl = TextEditingController();
  final _confirmCtrl = TextEditingController();
  final _inviteCtrl = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _usernameCtrl.dispose();
    _passwordCtrl.dispose();
    _confirmCtrl.dispose();
    _inviteCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final username = _usernameCtrl.text.trim();
    final password = _passwordCtrl.text;
    if (username.isEmpty || password.isEmpty) {
      setState(() => _error = '请填写所有字段');
      return;
    }

    final dio = Dio(BaseOptions(
      baseUrl: AppConfig.baseUrl,
      headers: {'Content-Type': 'application/json'},
    ));

    try {
      if (_isRegister) {
        if (password != _confirmCtrl.text) {
          setState(() => _error = '两次输入的密码不一致');
          return;
        }
        final res = await dio.post('/auth/register', data: {
          'username': username,
          'password': password,
          'inviteCode': _inviteCtrl.text.trim(),
        });
        final data = res.data;
        await ref.read(authProvider.notifier).login(
          data['token'], data['user']['username'], data['user']['role'],
        );
      } else {
        final res = await dio.post('/auth/login', data: {
          'username': username,
          'password': password,
        });
        final data = res.data;
        await ref.read(authProvider.notifier).login(
          data['token'], data['user']['username'], data['user']['role'],
        );
      }
    } on DioException catch (e) {
      final msg = e.response?.data?['error'] ?? '请求失败';
      setState(() => _error = msg);
    } catch (e) {
      setState(() => _error = e.toString());
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppTheme.bg,
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(32),
          child: Container(
            padding: const EdgeInsets.all(24),
            decoration: BoxDecoration(
              color: AppTheme.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: AppTheme.border),
            ),
            constraints: const BoxConstraints(maxWidth: 360),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Text('Stock Monitor', style: TextStyle(fontSize: 20, fontWeight: FontWeight.w700)),
                const SizedBox(height: 24),
                TextField(
                  controller: _usernameCtrl,
                  decoration: const InputDecoration(labelText: '用户名'),
                  style: const TextStyle(color: AppTheme.textPrimary),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _passwordCtrl,
                  decoration: const InputDecoration(labelText: '密码'),
                  obscureText: true,
                  style: const TextStyle(color: AppTheme.textPrimary),
                ),
                if (_isRegister) ...[
                  const SizedBox(height: 12),
                  TextField(
                    controller: _confirmCtrl,
                    decoration: const InputDecoration(labelText: '确认密码'),
                    obscureText: true,
                    style: const TextStyle(color: AppTheme.textPrimary),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _inviteCtrl,
                    decoration: const InputDecoration(labelText: '邀请码'),
                    style: const TextStyle(color: AppTheme.textPrimary),
                  ),
                ],
                if (_error != null) ...[
                  const SizedBox(height: 12),
                  Text(_error!, style: const TextStyle(color: AppTheme.down, fontSize: 13)),
                ],
                const SizedBox(height: 16),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(onPressed: _submit, child: Text(_isRegister ? '注册' : '登录')),
                ),
                const SizedBox(height: 12),
                TextButton(
                  onPressed: () => setState(() { _isRegister = !_isRegister; _error = null; }),
                  child: Text(_isRegister ? '已有账号？登录' : '没有账号？注册'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
