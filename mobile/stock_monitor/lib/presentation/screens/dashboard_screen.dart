import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../providers/api_providers.dart';
import '../providers/quote_provider.dart';

class DashboardScreen extends ConsumerStatefulWidget {
  const DashboardScreen({super.key});
  @override
  ConsumerState<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends ConsumerState<DashboardScreen> {
  List<Map<String, dynamic>> _indices = [];
  List<Map<String, dynamic>> _topGainers = [];
  List<Map<String, dynamic>> _topLosers = [];
  List<Map<String, dynamic>> _topSignals = [];
  List<Map<String, dynamic>> _recentAlerts = [];

  @override
  void initState() {
    super.initState();
    _fetch();
  }

  Future<void> _fetch() async {
    try {
      final data = await ref.read(dashboardApiProvider).get();
      setState(() {
        _indices = List<Map<String, dynamic>>.from(data['indices'] ?? []);
        _topGainers = List<Map<String, dynamic>>.from(data['topGainers'] ?? []);
        _topLosers = List<Map<String, dynamic>>.from(data['topLosers'] ?? []);
        _topSignals = List<Map<String, dynamic>>.from(data['topSignals'] ?? []);
        _recentAlerts = List<Map<String, dynamic>>.from(data['recentAlerts'] ?? []);
      });
    } catch (_) {}
  }

  Future<void> _refresh() async {
    await _fetch();
    setState(() {}); // trigger rebuild for quote updates
  }

  void _navigateToStock(String symbol) {
    Navigator.of(context).pushNamed('/stock-detail', arguments: {'symbol': symbol});
  }

  @override
  Widget build(BuildContext context) {
    final quotes = ref.watch(quoteProvider).quotes;

    // Update index prices from live quotes
    for (final idx in _indices) {
      final q = quotes[idx['code']];
      if (q != null) {
        idx['price'] = q.price;
        if (q.yp > 0) idx['changePct'] = ((q.price - q.yp) / q.yp) * 100;
      }
    }

    return Scaffold(
      appBar: AppBar(title: const Text('仪表盘'), actions: [
        IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh)),
      ]),
      body: RefreshIndicator(
        onRefresh: _fetch,
        child: ListView(
          padding: const EdgeInsets.all(12),
          children: [
            // Index cards
            SizedBox(
              height: 80,
              child: ListView.separated(
                scrollDirection: Axis.horizontal,
                itemCount: _indices.length,
                separatorBuilder: (_, __) => const SizedBox(width: 8),
                itemBuilder: (_, i) {
                  final idx = _indices[i];
                  final pct = (idx['changePct'] as num?)?.toDouble() ?? 0;
                  final dir = pct >= 0;
                  return Container(
                    width: 140,
                    padding: const EdgeInsets.all(10),
                    decoration: BoxDecoration(
                      color: AppTheme.surface,
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: AppTheme.border),
                    ),
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Text('${idx['name'] ?? ''}',
                            style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
                        Text(formatPrice((idx['price'] as num?)?.toDouble() ?? 0),
                            style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700, color: dir ? AppTheme.up : AppTheme.down)),
                        Text('${dir ? '+' : ''}${pct.toStringAsFixed(2)}%',
                            style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: dir ? AppTheme.up : AppTheme.down)),
                      ],
                    ),
                  );
                },
              ),
            ),

            const SizedBox(height: 16),
            // Top gainers / losers
            Row(children: [
              Expanded(child: _moverCard('今日涨幅 Top 3', _topGainers)),
              const SizedBox(width: 12),
              Expanded(child: _moverCard('今日跌幅 Top 3', _topLosers)),
            ]),

            const SizedBox(height: 12),
            // Top buy signals
            _sectionCard('最强买入信号', _topSignals.map((s) => _signalItem(s)).toList()),

            const SizedBox(height: 12),
            // Recent alerts
            _sectionCard('最近预警', _recentAlerts.map((a) => _alertItem(a)).toList()),
          ],
        ),
      ),
    );
  }

  Widget _moverCard(String title, List<Map<String, dynamic>> items) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppTheme.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: AppTheme.textSecondary)),
        const SizedBox(height: 8),
        if (items.isEmpty)
          const Text('暂无数据', style: TextStyle(fontSize: 12, color: AppTheme.textSecondary))
        else
          ...items.map((item) {
            final symbol = item['symbol'] as String? ?? '';
            final pct = (item['changePct'] as num?)?.toDouble() ?? 0;
            final dir = pct >= 0;
            return GestureDetector(
              onTap: () => _navigateToStock(symbol),
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 3),
                child: Row(children: [
                  Expanded(child: Text(item['name'] ?? shortCode(symbol), style: const TextStyle(fontSize: 13), overflow: TextOverflow.ellipsis)),
                  Text('${dir ? '+' : ''}${pct.toStringAsFixed(2)}%',
                      style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: dir ? AppTheme.up : AppTheme.down)),
                ]),
              ),
            );
          }),
      ]),
    );
  }

  Widget _sectionCard(String title, List<Widget> children) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppTheme.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: AppTheme.textSecondary)),
        const SizedBox(height: 8),
        if (children.isEmpty)
          const Text('暂无数据', style: TextStyle(fontSize: 12, color: AppTheme.textSecondary))
        else
          ...children,
      ]),
    );
  }

  Widget _signalItem(Map<String, dynamic> s) {
    final pct = ((s['buyPct'] as num?)?.toDouble() ?? 0).round();
    final symbol = s['symbol'] as String? ?? '';
    Color color;
    if (pct >= 50) { color = AppTheme.up; }
    else if (pct >= 25) { color = Colors.orange; }
    else { color = AppTheme.textSecondary; }
    return GestureDetector(
      onTap: () => _navigateToStock(symbol),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 3),
        child: Row(children: [
          Expanded(child: Text(s['name'] ?? shortCode(symbol), style: const TextStyle(fontSize: 13), overflow: TextOverflow.ellipsis)),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
            decoration: BoxDecoration(color: color.withAlpha(30), borderRadius: BorderRadius.circular(8)),
            child: Text('买入 $pct%', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: color)),
          ),
        ]),
      ),
    );
  }

  Widget _alertItem(Map<String, dynamic> a) {
    final symbol = a['symbol'] as String? ?? '';
    return GestureDetector(
      onTap: () => _navigateToStock(symbol),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 3),
        child: Row(children: [
          Expanded(child: Text(a['message'] ?? '', style: const TextStyle(fontSize: 13), overflow: TextOverflow.ellipsis)),
          Text(shortCode(symbol), style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary)),
        ]),
      ),
    );
  }
}
