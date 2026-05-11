import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'core/theme.dart';
import 'presentation/providers/auth_provider.dart';
import 'presentation/screens/login_screen.dart';
import 'presentation/screens/watchlist_screen.dart';
import 'presentation/screens/kline_screen.dart';
import 'presentation/screens/holdings_screen.dart';
import 'presentation/screens/alerts_screen.dart';
import 'presentation/screens/analysis_screen.dart';

final _rootNavigatorKey = GlobalKey<NavigatorState>();
final _shellNavigatorKey = GlobalKey<NavigatorState>();

final routerProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authProvider);

  return GoRouter(
    navigatorKey: _rootNavigatorKey,
    initialLocation: '/watchlist',
    redirect: (context, state) {
      final isLoggedIn = authState.isLoggedIn;
      final isLoginRoute = state.uri.path == '/login';

      if (!isLoggedIn && !isLoginRoute) return '/login';
      if (isLoggedIn && isLoginRoute) return '/watchlist';
      return null;
    },
    routes: [
      GoRoute(path: '/login', builder: (_, __) => const LoginScreen()),
      ShellRoute(
        navigatorKey: _shellNavigatorKey,
        builder: (context, state, child) => AppShell(child: child),
        routes: [
          GoRoute(path: '/watchlist', builder: (_, __) => const WatchlistScreen()),
          GoRoute(path: '/kline', builder: (_, __) => const KlineScreen()),
          GoRoute(path: '/holdings', builder: (_, __) => const HoldingsScreen()),
          GoRoute(path: '/alerts', builder: (_, __) => const AlertsScreen()),
          GoRoute(path: '/analysis', builder: (_, __) => const AnalysisScreen()),
        ],
      ),
    ],
  );
});

class AppShell extends StatelessWidget {
  final Widget child;
  const AppShell({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: _calculateSelectedIndex(context),
        onTap: (i) => _onTap(context, i),
        type: BottomNavigationBarType.fixed,
        items: const [
          BottomNavigationBarItem(icon: Icon(Icons.home), label: '自选'),
          BottomNavigationBarItem(icon: Icon(Icons.show_chart), label: 'K线'),
          BottomNavigationBarItem(icon: Icon(Icons.account_balance_wallet), label: '持仓'),
          BottomNavigationBarItem(icon: Icon(Icons.notifications), label: '提醒'),
          BottomNavigationBarItem(icon: Icon(Icons.analytics), label: '分析'),
        ],
      ),
    );
  }

  int _calculateSelectedIndex(BuildContext context) {
    final loc = GoRouterState.of(context).uri.path;
    if (loc.startsWith('/kline')) return 1;
    if (loc.startsWith('/holdings')) return 2;
    if (loc.startsWith('/alerts')) return 3;
    if (loc.startsWith('/analysis')) return 4;
    return 0;
  }

  void _onTap(BuildContext context, int i) {
    final routes = ['/watchlist', '/kline', '/holdings', '/alerts', '/analysis'];
    context.go(routes[i]);
  }
}

class StockMonitorApp extends ConsumerWidget {
  const StockMonitorApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    return MaterialApp.router(
      title: 'Stock Monitor',
      theme: AppTheme.darkTheme,
      routerConfig: router,
      debugShowCheckedModeBanner: false,
    );
  }
}
