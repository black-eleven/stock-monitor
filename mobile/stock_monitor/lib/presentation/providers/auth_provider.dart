import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

class AuthState {
  final String? token;
  final String? username;
  final String? role;
  final bool isLoggedIn;

  const AuthState({this.token, this.username, this.role, this.isLoggedIn = false});
}

class AuthNotifier extends StateNotifier<AuthState> {
  AuthNotifier() : super(const AuthState()) {
    _loadFromStorage();
  }

  Future<void> _loadFromStorage() async {
    final prefs = await SharedPreferences.getInstance();
    final token = prefs.getString('token');
    final username = prefs.getString('username');
    final role = prefs.getString('role');
    if (token != null) {
      state = AuthState(token: token, username: username, role: role, isLoggedIn: true);
    }
  }

  Future<void> login(String token, String username, String role) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('token', token);
    await prefs.setString('username', username);
    await prefs.setString('role', role);
    state = AuthState(token: token, username: username, role: role, isLoggedIn: true);
  }

  Future<void> logout() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove('token');
    await prefs.remove('username');
    await prefs.remove('role');
    state = const AuthState();
  }
}

final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier();
});
