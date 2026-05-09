import 'package:flutter/material.dart';

class AppTheme {
  static const Color bg = Color(0xFF0d1117);
  static const Color surface = Color(0xFF161b22);
  static const Color border = Color(0xFF30363d);
  static const Color textPrimary = Color(0xFFe6edf3);
  static const Color textSecondary = Color(0xFF8b949e);
  static const Color up = Color(0xFF3fb950);
  static const Color down = Color(0xFFf85149);
  static const Color accent = Color(0xFF1f6feb);

  static ThemeData get darkTheme => ThemeData(
        brightness: Brightness.dark,
        scaffoldBackgroundColor: bg,
        appBarTheme: const AppBarTheme(
          backgroundColor: surface,
          foregroundColor: textPrimary,
        ),
        bottomNavigationBarTheme: const BottomNavigationBarThemeData(
          backgroundColor: surface,
          selectedItemColor: accent,
          unselectedItemColor: textSecondary,
        ),
        cardColor: surface,
        dividerColor: border,
        colorScheme: const ColorScheme.dark(
          primary: accent,
          surface: surface,
        ),
      );
}
