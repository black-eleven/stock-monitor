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

class _AnalysisScreenState extends ConsumerState<AnalysisScreen> {
  List<_Result>? _results;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _analyze();
  }

  Future<void> _analyze() async {
    setState(() => _loading = true);
    final watchlist = await ref.read(watchlistApiProvider).getAll();
    final quoteApi = ref.read(quoteApiProvider);
    final results = <_Result>[];
    for (final stock in watchlist) {
      try {
        final data = await quoteApi.getKline(stock.symbol, interval: '1d', count: 100);
        final bars = <Bar>[];
        for (final item in data) {
          for (final k in item.k) {
            bars.add(Bar(time: k.ts, open: k.o, high: k.h, low: k.l, close: k.cl));
          }
        }
        bars.sort((a, b) => a.time.compareTo(b.time));
        results.add(_Result(stock: stock, signal: evaluateSignals(bars)));
      } catch (_) {}
    }
    results.sort((a, b) => b.signal.score.compareTo(a.signal.score));
    setState(() { _results = results; _loading = false; });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('卖出分析'), actions: [IconButton(onPressed: _analyze, icon: const Icon(Icons.refresh))]),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _results == null || _results!.isEmpty
              ? const Center(child: Text('暂无数据', style: TextStyle(color: AppTheme.textSecondary)))
              : Column(children: [
                  Container(padding: const EdgeInsets.all(12), color: AppTheme.surface, child: _buildSummary()),
                  Expanded(
                    child: ListView.builder(
                      itemCount: _results!.length,
                      itemBuilder: (_, i) {
                        final r = _results![i];
                        final pct = r.signal.score / r.signal.maxScore;
                        final color = pct >= 0.5 ? AppTheme.down : pct > 0 ? Colors.orange : AppTheme.up;
                        return Card(
                          margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                          child: ListTile(
                            title: Text('${r.stock.name} (${shortCode(r.stock.symbol)})', style: const TextStyle(fontWeight: FontWeight.w600)),
                            subtitle: Text(r.signal.summary),
                            trailing: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                              Text('${(pct * 100).toStringAsFixed(0)}分', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800, color: color)),
                              Text('${r.signal.count}个信号', style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary)),
                            ]),
                          ),
                        );
                      },
                    ),
                  ),
                ]),
    );
  }

  Widget _buildSummary() {
    if (_results!.isEmpty) return const SizedBox.shrink();
    final avgScore = _results!.map((r) => r.signal.score / r.signal.maxScore).reduce((a, b) => a + b) / _results!.length;
    final color = avgScore >= 0.5 ? AppTheme.down : avgScore >= 0.25 ? Colors.orange : AppTheme.up;
    final label = avgScore >= 0.5 ? '强烈偏空' : avgScore >= 0.25 ? '偏弱' : '正常';
    return Row(mainAxisAlignment: MainAxisAlignment.center, children: [
      const Text('平均卖出分 ', style: TextStyle(color: AppTheme.textSecondary)),
      Text('${(avgScore * 100).toStringAsFixed(1)}分', style: TextStyle(fontSize: 24, fontWeight: FontWeight.w800, color: color)),
      const SizedBox(width: 12),
      Container(padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4), decoration: BoxDecoration(color: color.withAlpha(40), borderRadius: BorderRadius.circular(12)), child: Text(label, style: TextStyle(color: color, fontWeight: FontWeight.w600))),
    ]);
  }
}

class _Result {
  final WatchlistItem stock;
  final SignalResult signal;
  _Result({required this.stock, required this.signal});
}
