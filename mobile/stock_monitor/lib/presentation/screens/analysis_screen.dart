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

  List<String> _strategyKeys = [];
  List<String> _strategyNames = [];
  String _currentStrategy = 'comprehensive';

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _analyze();
    _loadStrategies();
  }

  Future<void> _loadStrategies() async {
    try {
      final api = ref.read(strategyApiProvider);
      final keys = await api.getStrategyKeys();
      final names = await api.getStrategies();
      if (mounted) setState(() { _strategyKeys = keys; _strategyNames = names; });
    } catch (_) {}
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
    final signalApi = ref.read(signalApiProvider);
    final sellResults = <_Result>[];
    final buyResults = <_Result>[];
    for (final stock in watchlist) {
      try {
        final data =
            await quoteApi.getKline(stock.symbol, interval: '1d', count: 100);
        final bars = <Bar>[];
        final rawBars = <KlineBar>[];
        for (final item in data) {
          for (final k in item.k) {
            bars.add(Bar(
                time: k.ts,
                open: k.o,
                high: k.h,
                low: k.l,
                close: k.cl,
                volume: k.v));
            rawBars.add(k);
          }
        }
        bars.sort((a, b) => a.time.compareTo(b.time));
        final sell = evaluateSignals(bars);
        final buy = evaluateBuySignals(bars);
        sellResults.add(_Result(
            stock: stock, signal: sell, bars: rawBars));
        buyResults.add(_Result(
            stock: stock, signal: buy, bars: rawBars));

        // Record signals to backend
        try {
          await signalApi.record(
            symbol: stock.symbol,
            buyScore: double.parse((buy.score).toStringAsFixed(2)),
            buyPct: ((buy.score / buy.maxScore) * 100).round(),
            sellScore: double.parse((sell.score).toStringAsFixed(2)),
            sellPct: ((sell.score / sell.maxScore) * 100).round(),
            buyCount: buy.count,
            sellCount: sell.count,
          );
        } catch (_) {}
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
          child: Text('暂无数据', style: TextStyle(color: AppTheme.textSecondary)));
    }
    final filtered = _applyFilter(raw);
    if (filtered.isEmpty) {
      return const Center(
          child: Text('该交易所暂无数据', style: TextStyle(color: AppTheme.textSecondary)));
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
            .reduce((a, b) => a + b) / results.length;
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
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
          decoration:
              BoxDecoration(color: color.withAlpha(40), borderRadius: BorderRadius.circular(12)),
          child: Text(text,
              style: TextStyle(color: color, fontWeight: FontWeight.w600))),
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

    // Auto-run comprehensive strategy analysis
    String? _analysis;
    bool _analyzing = true;
    _runStrategy('comprehensive', r.stock.symbol, r.bars).then((a) {
      if (mounted) {
        _analysis = a;
        _analyzing = false;
        (context as Element).markNeedsBuild();
      }
    });

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setSheetState) => Container(
          padding: const EdgeInsets.all(20),
          height: MediaQuery.of(ctx).size.height * 0.85,
          child: SingleChildScrollView(
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
                        Expanded(child: Text(s.name,
                            style: const TextStyle(fontSize: 14))),
                        if (s.value != null)
                          Text(s.value!,
                              style: const TextStyle(
                                  fontSize: 12, color: AppTheme.textSecondary)),
                      ]),
                    )),
                const Divider(height: 24),
                // Strategy selector
                Row(children: [
                  const Text('AI策略分析',
                      style: TextStyle(
                          fontWeight: FontWeight.w600,
                          color: AppTheme.accent)),
                  const Spacer(),
                  SizedBox(
                    width: 150,
                    child: DropdownButton<String>(
                      value: _currentStrategy,
                      isDense: true,
                      dropdownColor: AppTheme.surface,
                      style: const TextStyle(
                          color: AppTheme.textPrimary, fontSize: 13),
                      underline: Container(height: 0),
                      items: _strategyKeys.isNotEmpty
                          ? List.generate(_strategyKeys.length, (i) {
                              return DropdownMenuItem(
                                value: _strategyKeys[i],
                                child: Text(_strategyNames[i],
                                    style: const TextStyle(fontSize: 12)),
                              );
                            })
                          : [
                              const DropdownMenuItem(
                                value: 'comprehensive',
                                child: Text('综合分析'),
                              )
                            ],
                      onChanged: (v) {
                        if (v == null) return;
                        _currentStrategy = v;
                        setSheetState(() {
                          _analyzing = true;
                          _analysis = null;
                        });
                        _runStrategy(v, r.stock.symbol, r.bars).then((a) {
                          setSheetState(() {
                            _analysis = a;
                            _analyzing = false;
                          });
                        });
                      },
                    ),
                  ),
                ]),
                const SizedBox(height: 8),
                if (_analyzing)
                  const Center(
                      child: Padding(
                    padding: EdgeInsets.all(20),
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )),
                if (_analysis != null)
                  _buildMarkdown(_analysis!),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<String> _runStrategy(
      String strategy, String symbol, List<KlineBar> bars) async {
    try {
      final api = ref.read(strategyApiProvider);
      return await api.analyze(strategy, symbol, bars);
    } catch (e) {
      return '分析失败: $e';
    }
  }

  Widget _buildMarkdown(String text) {
    final spans = <InlineSpan>[];
    final lines = text.split('\n');
    for (final line in lines) {
      if (line.startsWith('### ')) {
        spans.add(TextSpan(
          text: '${line.substring(4)}\n',
          style: const TextStyle(
            color: AppTheme.accent,
            fontWeight: FontWeight.w600,
            fontSize: 14,
          ),
        ));
      } else if (line.startsWith('## ')) {
        spans.add(TextSpan(
          text: '${line.substring(3)}\n',
          style: const TextStyle(
            color: Color(0xFFFFD700),
            fontWeight: FontWeight.w700,
            fontSize: 15,
          ),
        ));
      } else if (line.startsWith('- ')) {
        spans.add(TextSpan(
          text: '• ${line.substring(2)}\n',
          style: const TextStyle(color: AppTheme.textPrimary, fontSize: 13),
        ));
      } else {
        // Handle **bold** and timestamps inline
        final parts = line.split(RegExp(r'(\*\*.+?\*\*|\b1[7-9]\d{8}\b)'));
        for (final part in parts) {
          if (part.startsWith('**') && part.endsWith('**')) {
            spans.add(TextSpan(
              text: part.substring(2, part.length - 2),
              style: const TextStyle(
                color: Color(0xFFFFD700),
                fontWeight: FontWeight.w700,
              ),
            ));
          } else if (RegExp(r'^1[7-9]\d{8}$').hasMatch(part)) {
            final ts = int.tryParse(part) ?? 0;
            final d = DateTime.fromMillisecondsSinceEpoch(
                ts * 1000 + 8 * 3600 * 1000);
            final pad = (int n) => n.toString().padLeft(2, '0');
            spans.add(TextSpan(
              text: '${d.year}-${pad(d.month)}-${pad(d.day)} ${pad(d.hour)}:${pad(d.minute)}',
              style: const TextStyle(
                color: Color(0xFFFFD700),
                fontSize: 12,
              ),
            ));
          } else {
            spans.add(TextSpan(
              text: part,
              style: const TextStyle(color: AppTheme.textPrimary, fontSize: 13),
            ));
          }
        }
        spans.add(const TextSpan(text: '\n'));
      }
    }
    return RichText(text: TextSpan(children: spans));
  }
}

class _Result {
  final WatchlistItem stock;
  final SignalResult signal;
  final List<KlineBar> bars;
  _Result({required this.stock, required this.signal, required this.bars});
}
