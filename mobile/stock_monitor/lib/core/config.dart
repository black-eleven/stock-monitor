class AppConfig {
  static const String host = '10.0.2.2'; // Android emulator → host machine
  static const int port = 3000;
  static String get baseUrl => 'http://$host:$port/api';
  static String get wsUrl => 'ws://$host:$port/ws';
}
