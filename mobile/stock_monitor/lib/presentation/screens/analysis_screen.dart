import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/indicators.dart';
import '../../domain/model/kline.dart';
import '../../domain/model/stock.dart';
import '../providers/api_providers.dart';

class AnalysisScreen extends ConsumerStatefulWidget {
  const AnalysisScreen({super.key});
  @override
  ConsumerState<AnalysisScreen> createState() => _AnalysisScreenState();
}

class _AnalysisScreenState extends ConsumerState<AnalysisScreen> with SingleTickerProviderStateMixin {
  List<_Result>? _sellResults;
  List<_Result>? _buyResults;
  bool _loading = true;
  late TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _analyze();
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _analyze() async {
    setState(() => _loading = true);
    final watchlist = await ref.read(watchlistApiProvider).getAll();
    final quoteApi = ref.read(quoteApiProvider);
    final sellResults = <_Result>[];
    final buyResults = <_Result>[];
    for (final stock in watchlist) {
      try {
        final data = await quoteApi.getKline(stock.symbol, interval: '1d', count: 100);
        final bars = <Bar>[];
        for (final item in data) {
          for (final k in item.k) {
            bars.add(Bar(time: k.ts, open: k.o, high: k.h, low: k.l, close: k.cl, volume: k.v));
          }
        }
        bars.sort((a, b) => a.time.compareTo(b.time));
        sellResults.add(_Result(stock: stock, signal: evaluateSignals(bars)));
        buyResults.add(_Result(stock: stock, signal: evaluateBuySignals(bars)));
      } catch (_) {}
    }
    sellResults.sort((a, b) => b.signal.score.compareTo(a.signal.score));
    buyResults.sort((a, b) => b.signal.score.compareTo(a.signal.score));
    setState(() { _sellResults = sellResults; _buyResults = buyResults; _loading = false; });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('技术分析'),
        actions: [IconButton(onPressed: _analyze, icon: const Icon(Icons.refresh))],
        bottom: TabBar(
          controller: _tabController,
          indicatorColor: AppTheme.accent,
          labelColor: AppTheme.accent,
          unselectedLabelColor: AppTheme.textSecondary,
          tabs: const [
            Tab(text: '卖出分析'),
            Tab(text: '买入推荐'),
          ],
        ),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : TabBarView(
              controller: _tabController,
              children: [
                _buildSellView(),
                _buildBuyView(),
              ],
            ),
    );
  }

  Widget _buildSellView() {
    if (_sellResults == null || _sellResults!.isEmpty) {
      return const Center(child: Text('暂无数据', style: TextStyle(color: AppTheme.textSecondary)));
    }
    return Column(children: [
      Container(padding: const EdgeInsets.all(12), color: AppTheme.surface, child: _buildSummary(_sellResults!, '卖出')),
      Expanded(
        child: ListView.builder(
          itemCount: _sellResults!.length,
          itemBuilder: (_, i) => _buildCard(_sellResults![i]),
        ),
      ),
    ]);
  }

  Widget _buildBuyView() {
    if (_buyResults == null || _buyResults!.isEmpty) {
      return const Center(child: Text('暂无数据', style: TextStyle(color: AppTheme.textSecondary)));
    }
    return Column(children: [
      Container(padding: const EdgeInsets.all(12), color: AppTheme.surface, child: _buildSummary(_buyResults!, '买入')),
      Expanded(
        child: ListView.builder(
          itemCount: _buyResults!.length,
          itemBuilder: (_, i) => _buildCard(_buyResults![i]),
        ),
      ),
    ]);
  }

  Widget _buildSummary(List<_Result> results, String label) {
    final avgScore = results.map((r) => r.signal.score / r.signal.maxScore).reduce((a, b) => a + b) / results.length;
    final color = avgScore >= 0.5
        ? (label == '买入' ? AppTheme.up : AppTheme.down)
        : avgScore >= 0.25
            ? Colors.orange
            : AppTheme.textSecondary;
    final text = avgScore >= 0.5
        ? (label == '买入' ? '强烈偏多' : '强烈偏空')
        : avgScore >= 0.25
            ? (label == '买入' ? '偏多' : '偏弱')
            : '正常';
    return Row(mainAxisAlignment: MainAxisAlignment.center, children: [
      Text('平均${label}分 ', style: const TextStyle(color: AppTheme.textSecondary)),
      Text('${(avgScore * 100).toStringAsFixed(1)}分', style: TextStyle(fontSize: 24, fontWeight: FontWeight.w800, color: color)),
      const SizedBox(width: 12),
      Container(padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4), decoration: BoxDecoration(color: color.withAlpha(40), borderRadius: BorderRadius.circular(12)), child: Text(text, style: TextStyle(color: color, fontWeight: FontWeight.w600))),
    ]);
  }

  Widget _buildCard(_Result r) {
    final pct = r.signal.score / r.signal.maxScore;
    final isBuy = r.signal.total == 10;
    final color = pct >= 0.5
        ? (isBuy ? AppTheme.up : AppTheme.down)
        : pct > 0
            ? Colors.orange
            : AppTheme.textSecondary;
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: ListTile(
        title: Text('${r.stock.name} (${shortCode(r.stock.symbol)})', style: const TextStyle(fontWeight: FontWeight.w600)),
        subtitle: Text(r.signal.summary),
        trailing: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
          Text('${(pct * 100).toStringAsFixed(0)}分', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800, color: color)),
          Text('${r.signal.count}个信号', style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary)),
        ]),
        onTap: () => _showDetail(r),
      ),
    );
  }

  void _showDetail(_Result r) {
    final pct = r.signal.score / r.signal.maxScore;
    final isBuy = r.signal.total == 10;
    final color = pct >= 0.5
        ? (isBuy ? AppTheme.up : AppTheme.down)
        : pct > 0
            ? Colors.orange
            : AppTheme.textSecondary;
    showModalBottomSheet(
      context: context,
      builder: (_) => Container(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('${r.stock.name} (${r.stock.symbol})', style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w700)),
            const SizedBox(height: 4),
            Text('评分: ${(pct * 100).toStringAsFixed(0)}分 — ${r.signal.summary}', style: TextStyle(color: color)),
            const SizedBox(height: 12),
            Text('${r.signal.count} / ${r.signal.total} 个信号触发', style: const TextStyle(color: AppTheme.textSecondary)),
            const SizedBox(height: 8),
            ...r.signal.signals.where((s) => s.triggered).map((s) => Padding(
              padding: const EdgeInsets.symmetric(vertical: 4),
              child: Row(children: [
                Icon(s.status == 'danger' ? Icons.circle : Icons.warning_amber, size: 14, color: s.status == 'danger' ? (isBuy ? AppTheme.up : AppTheme.down) : Colors.orange),
                const SizedBox(width: 8),
                Expanded(child: Text(s.name, style: const TextStyle(fontSize: 14))),
                if (s.value != null) Text(s.value!, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
              ]),
            )),
          ],
        ),
      ),
    );
  }
}

class _Result {
  final WatchlistItem stock;
  final SignalResult signal;
  _Result({required this.stock, required this.signal});
}
