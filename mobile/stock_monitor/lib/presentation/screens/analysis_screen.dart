import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/indicators.dart';
import '../../domain/model/kline.dart';
import '../../domain/model/stock.dart';
import '../providers/api_providers.dart';

enum _SortMode { score, name }

class AnalysisScreen extends ConsumerStatefulWidget {
  const AnalysisScreen({super.key});
  @override
  ConsumerState<AnalysisScreen> createState() => _AnalysisScreenState();
}

class _AnalysisScreenState extends ConsumerState<AnalysisScreen>
    with SingleTickerProviderStateMixin {
  List<_Result>? _sellResults;
  List<_Result>? _buyResults;
  bool _loading = true;
  late TabController _tabController;

  _SortMode _sortMode = _SortMode.score;
  String _exchangeFilter = 'ALL';

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
        final data =
            await quoteApi.getKline(stock.symbol, interval: '1d', count: 100);
        final bars = <Bar>[];
        for (final item in data) {
          for (final k in item.k) {
            bars.add(Bar(
                time: k.ts,
                open: k.o,
                high: k.h,
                low: k.l,
                close: k.cl,
                volume: k.v));
          }
        }
        bars.sort((a, b) => a.time.compareTo(b.time));
        sellResults.add(_Result(stock: stock, signal: evaluateSignals(bars)));
        buyResults
            .add(_Result(stock: stock, signal: evaluateBuySignals(bars)));
      } catch (_) {}
    }
    _sort(sellResults);
    _sort(buyResults);
    setState(() {
      _sellResults = sellResults;
      _buyResults = buyResults;
      _loading = false;
    });
  }

  void _sort(List<_Result> results) {
    switch (_sortMode) {
      case _SortMode.score:
        results.sort((a, b) => b.signal.score.compareTo(a.signal.score));
        break;
      case _SortMode.name:
        results.sort((a, b) => a.stock.name.compareTo(b.stock.name));
        break;
    }
  }

  List<_Result> _applyFilter(List<_Result> results) {
    if (_exchangeFilter == 'ALL') return results;
    return results
        .where((r) => r.stock.symbol.startsWith('$_exchangeFilter:'))
        .toList();
  }

  String _exchangePrefix(String symbol) {
    final idx = symbol.indexOf(':');
    return idx > 0 ? symbol.substring(0, idx) : '';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('技术分析'),
        actions: [
          IconButton(onPressed: _analyze, icon: const Icon(Icons.refresh)),
        ],
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
                _buildSignalView(false),
                _buildSignalView(true),
              ],
            ),
    );
  }

  Widget _buildFilterBar() {
    const exchanges = ['ALL', 'US', 'HK', 'SH', 'SZ'];
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Row(children: [
        _sortBtn(),
        const SizedBox(width: 8),
        Expanded(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: Row(
              children: exchanges.map((e) {
                final selected = _exchangeFilter == e;
                return Padding(
                  padding: const EdgeInsets.only(right: 6),
                  child: GestureDetector(
                    onTap: () => setState(() => _exchangeFilter = e),
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 10, vertical: 5),
                      decoration: BoxDecoration(
                        color: selected
                            ? AppTheme.accent
                            : AppTheme.surface,
                        borderRadius: BorderRadius.circular(14),
                        border: Border.all(
                            color: selected
                                ? AppTheme.accent
                                : AppTheme.border),
                      ),
                      child: Text(
                        e == 'ALL' ? '全部' : e,
                        style: TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.w500,
                          color: selected
                              ? Colors.white
                              : AppTheme.textSecondary,
                        ),
                      ),
                    ),
                  ),
                );
              }).toList(),
            ),
          ),
        ),
      ]),
    );
  }

  Widget _sortBtn() {
    return GestureDetector(
      onTap: () {
        setState(() {
          _sortMode =
              _sortMode == _SortMode.score ? _SortMode.name : _SortMode.score;
          if (_sellResults != null) _sort(_sellResults!);
          if (_buyResults != null) _sort(_buyResults!);
        });
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
        decoration: BoxDecoration(
          color: AppTheme.surface,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(color: AppTheme.border),
        ),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          Icon(
            _sortMode == _SortMode.score
                ? Icons.trending_down
                : Icons.sort_by_alpha,
            size: 14,
            color: AppTheme.textSecondary,
          ),
          const SizedBox(width: 4),
          Text(
            _sortMode == _SortMode.score ? '按信号' : '按名称',
            style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary),
          ),
        ]),
      ),
    );
  }

  Widget _buildSignalView(bool isBuy) {
    final raw = isBuy ? _buyResults : _sellResults;
    if (raw == null || raw.isEmpty) {
      return const Center(
          child:
              Text('暂无数据', style: TextStyle(color: AppTheme.textSecondary)));
    }
    final filtered = _applyFilter(raw);
    if (filtered.isEmpty) {
      return const Center(
          child: Text('该交易所暂无数据',
              style: TextStyle(color: AppTheme.textSecondary)));
    }
    final label = isBuy ? '买入' : '卖出';
    return Column(children: [
      _buildFilterBar(),
      Container(
          padding: const EdgeInsets.all(12),
          color: AppTheme.surface,
          child: _buildSummary(filtered, label)),
      Expanded(
        child: ListView.builder(
          itemCount: filtered.length,
          itemBuilder: (_, i) => _buildCard(filtered[i]),
        ),
      ),
    ]);
  }

  Widget _buildSummary(List<_Result> results, String label) {
    final avgScore = results
            .map((r) => r.signal.score / r.signal.maxScore)
            .reduce((a, b) => a + b) /
        results.length;
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
      Text('平均${label}分 ',
          style: const TextStyle(color: AppTheme.textSecondary)),
      Text('${(avgScore * 100).toStringAsFixed(1)}分',
          style: TextStyle(
              fontSize: 24, fontWeight: FontWeight.w800, color: color)),
      const SizedBox(width: 12),
      Container(
          padding:
              const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
          decoration: BoxDecoration(
              color: color.withAlpha(40),
              borderRadius: BorderRadius.circular(12)),
          child: Text(text,
              style: TextStyle(
                  color: color, fontWeight: FontWeight.w600))),
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
        title: Text('${r.stock.name} (${shortCode(r.stock.symbol)})',
            style: const TextStyle(fontWeight: FontWeight.w600)),
        subtitle: Text(r.signal.summary),
        trailing: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
          Text('${(pct * 100).toStringAsFixed(0)}分',
              style: TextStyle(
                  fontSize: 18, fontWeight: FontWeight.w800, color: color)),
          Text('${r.signal.count}个信号',
              style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary)),
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
            Text('${r.stock.name} (${r.stock.symbol})',
                style: const TextStyle(
                    fontSize: 20, fontWeight: FontWeight.w700)),
            const SizedBox(height: 4),
            Text(
                '评分: ${(pct * 100).toStringAsFixed(0)}分 — ${r.signal.summary}',
                style: TextStyle(color: color)),
            const SizedBox(height: 12),
            Text('${r.signal.count} / ${r.signal.total} 个信号触发',
                style: const TextStyle(color: AppTheme.textSecondary)),
            const SizedBox(height: 8),
            ...r.signal.signals.where((s) => s.triggered).map((s) => Padding(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  child: Row(children: [
                    Icon(
                        s.status == 'danger'
                            ? Icons.circle
                            : Icons.warning_amber,
                        size: 14,
                        color: s.status == 'danger'
                            ? (isBuy ? AppTheme.up : AppTheme.down)
                            : Colors.orange),
                    const SizedBox(width: 8),
                    Expanded(
                        child:
                            Text(s.name, style: const TextStyle(fontSize: 14))),
                    if (s.value != null)
                      Text(s.value!,
                          style: const TextStyle(
                              fontSize: 12, color: AppTheme.textSecondary)),
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
