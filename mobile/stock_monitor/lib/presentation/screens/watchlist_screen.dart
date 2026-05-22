import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/indicators.dart';
import '../../domain/model/kline.dart';
import '../../domain/model/stock.dart';
import '../../domain/model/recommendation.dart';
import '../providers/api_providers.dart';
import '../providers/quote_provider.dart';
import '../widgets/recommend_detail_sheet.dart';
import '../widgets/stock_card.dart';

class WatchlistScreen extends ConsumerStatefulWidget {
  const WatchlistScreen({super.key});
  @override
  ConsumerState<WatchlistScreen> createState() => _WatchlistScreenState();
}

class _WatchlistScreenState extends ConsumerState<WatchlistScreen>
    with SingleTickerProviderStateMixin {
  List<WatchlistItem>? _watchlist;
  Map<String, SignalResult>? _buySignals;
  Map<String, SignalResult>? _sellSignals;
  late TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _load();
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final api = ref.read(watchlistApiProvider);
    final list = await api.getAll();
    setState(() => _watchlist = list);
    if (list.isNotEmpty) {
      await _fetchWatchlistSignals(list);
    }
  }

  Future<void> _fetchWatchlistSignals(List<WatchlistItem> items) async {
    final quoteApi = ref.read(quoteApiProvider);
    final buySignals = <String, SignalResult>{};
    final sellSignals = <String, SignalResult>{};

    final futures = items.map((item) async {
      try {
        final data = await quoteApi.getKline(item.symbol, interval: '1d', count: 100);
        final bars = <Bar>[];
        for (final item in data) {
          for (final k in item.k) {
            bars.add(Bar(time: k.ts, open: k.o, high: k.h, low: k.l, close: k.cl, volume: k.v));
          }
        }
        bars.sort((a, b) => a.time.compareTo(b.time));
        buySignals[item.symbol] = evaluateBuySignals(bars);
        sellSignals[item.symbol] = evaluateSignals(bars);
      } catch (_) {}
    });

    await Future.wait(futures);
    if (mounted) {
      setState(() {
        _buySignals = buySignals;
        _sellSignals = sellSignals;
      });
    }
  }

  Future<void> _add() async {
    final symbolCtrl = TextEditingController();
    final nameCtrl = TextEditingController();

    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('添加自选'),
        content: Column(mainAxisSize: MainAxisSize.min, children: [
          TextField(controller: symbolCtrl, decoration: const InputDecoration(hintText: '代码 (如 HK:700)')),
          const SizedBox(height: 12),
          TextField(controller: nameCtrl, decoration: const InputDecoration(hintText: '名称 (如 腾讯控股)')),
        ]),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('添加')),
        ],
      ),
    );

    if (ok == true && symbolCtrl.text.isNotEmpty && nameCtrl.text.isNotEmpty) {
      try {
        await ref.read(watchlistApiProvider).add(symbolCtrl.text.toUpperCase(), nameCtrl.text);
        await _load();
      } catch (e) {
        if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('添加失败: $e')));
      }
    }
  }

  void _openStockDetail(String symbol) {
    Navigator.of(context).pushNamed('/stock-detail', arguments: {'symbol': symbol});
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('自选股'),
        bottom: TabBar(
          controller: _tabController,
          tabs: const [
            Tab(text: '我的自选'),
            Tab(text: '推荐发现'),
          ],
        ),
        actions: [
          IconButton(onPressed: _add, icon: const Icon(Icons.add)),
        ],
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          _buildWatchlistTab(),
          _buildRecommendTab(),
        ],
      ),
    );
  }

  Widget _buildWatchlistTab() {
    if (_watchlist == null) return const Center(child: CircularProgressIndicator());
    final quotes = ref.watch(quoteProvider).quotes;

    if (_watchlist!.isEmpty) {
      return const Center(
        child: Text('暂无自选股\n点击右上角 + 添加',
            textAlign: TextAlign.center, style: TextStyle(color: AppTheme.textSecondary)),
      );
    }

    return ListView.builder(
      itemCount: _watchlist!.length,
      itemBuilder: (_, i) {
        final item = _watchlist![i];
        return StockCard(
          item: item,
          quote: quotes[item.symbol],
          buySignal: _buySignals?[item.symbol],
          sellSignal: _sellSignals?[item.symbol],
          onTap: () => _openStockDetail(item.symbol),
          onDelete: () async {
            await ref.read(watchlistApiProvider).remove(item.symbol);
            _load();
          },
        );
      },
    );
  }

  Widget _buildRecommendTab() {
    return _RecommendTab(
      onAddToWatchlist: (symbol, name) async {
        try {
          await ref.read(watchlistApiProvider).add(symbol, name);
          await _load();
          _tabController.animateTo(0);
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text('已添加 $name 到自选股')),
            );
          }
        } catch (e) {
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text('添加失败: $e')),
            );
          }
        }
      },
      onOpenDetail: _openStockDetail,
    );
  }
}

class _RecommendTab extends StatefulWidget {
  final Future<void> Function(String symbol, String name) onAddToWatchlist;
  final void Function(String symbol) onOpenDetail;

  const _RecommendTab({required this.onAddToWatchlist, required this.onOpenDetail});

  @override
  State<_RecommendTab> createState() => _RecommendTabState();
}

class _RecommendTabState extends State<_RecommendTab> {
  final _controller = TextEditingController();
  List<Recommendation>? _recs;
  Map<String, SignalResult>? _buySignals;
  Map<String, SignalResult>? _sellSignals;
  String? _error;
  bool _loading = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _search() async {
    final industry = _controller.text.trim();
    if (industry.isEmpty) return;

    setState(() {
      _loading = true;
      _error = null;
      _recs = null;
      _buySignals = null;
      _sellSignals = null;
    });

    try {
      final container = ProviderScope.containerOf(context);
      final api = container.read(recommendApiProvider);
      final recs = await api.recommend(industry);
      setState(() {
        _recs = recs;
        if (recs.isEmpty) _error = '未找到相关推荐';
      });
      if (recs.isNotEmpty) await _fetchSignals(recs);
      setState(() => _loading = false);
    } catch (e) {
      setState(() { _error = '获取推荐失败: $e'; _loading = false; });
    }
  }

  Future<void> _fetchSignals(List<Recommendation> recs) async {
    final container = ProviderScope.containerOf(context);
    final quoteApi = container.read(quoteApiProvider);
    final buySignals = <String, SignalResult>{};
    final sellSignals = <String, SignalResult>{};

    final futures = recs.map((rec) async {
      try {
        final data = await quoteApi.getKline(rec.symbol, interval: '1d', count: 100);
        final bars = <Bar>[];
        for (final item in data) {
          for (final k in item.k) {
            bars.add(Bar(time: k.ts, open: k.o, high: k.h, low: k.l, close: k.cl, volume: k.v));
          }
        }
        bars.sort((a, b) => a.time.compareTo(b.time));
        buySignals[rec.symbol] = evaluateBuySignals(bars);
        sellSignals[rec.symbol] = evaluateSignals(bars);
      } catch (_) {}
    });

    await Future.wait(futures);
    if (mounted) setState(() { _buySignals = buySignals; _sellSignals = sellSignals; });
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(children: [
        Row(children: [
          Expanded(
            child: TextField(
              controller: _controller,
              decoration: const InputDecoration(
                hintText: '输入行业关键词 (如 AI, 新能源, 半导体)',
                border: OutlineInputBorder(),
                contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 10),
              ),
              onSubmitted: (_) => _search(),
            ),
          ),
          const SizedBox(width: 8),
          FilledButton(
            onPressed: _loading ? null : _search,
            child: _loading
                ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                : const Text('搜索'),
          ),
        ]),
        const SizedBox(height: 16),
        Expanded(child: _buildResults()),
      ]),
    );
  }

  Widget _buildResults() {
    if (_error != null) {
      return Center(child: Text(_error!, style: const TextStyle(color: AppTheme.textSecondary)));
    }
    if (_recs == null) {
      return const Center(
        child: Text('输入行业关键词搜索推荐股票', style: TextStyle(color: AppTheme.textSecondary)),
      );
    }
    return ListView.builder(
      itemCount: _recs!.length,
      itemBuilder: (_, i) {
        final r = _recs![i];
        return _RecommendCard(
          rec: r,
          buySignal: _buySignals?[r.symbol],
          sellSignal: _sellSignals?[r.symbol],
          signalsLoading: _buySignals == null,
          onAdd: () => widget.onAddToWatchlist(r.symbol, r.name),
          onTap: () => widget.onOpenDetail(r.symbol),
          onDetailTap: () => _showRecommendDetail(r),
        );
      },
    );
  }

  void _showRecommendDetail(Recommendation rec) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (_) => RecommendDetailSheet(
        rec: rec,
        buySignal: _buySignals?[rec.symbol],
        sellSignal: _sellSignals?[rec.symbol],
        onAdd: () => widget.onAddToWatchlist(rec.symbol, rec.name),
      ),
    );
  }
}

class _RecommendCard extends StatelessWidget {
  final Recommendation rec;
  final SignalResult? buySignal;
  final SignalResult? sellSignal;
  final bool signalsLoading;
  final VoidCallback onAdd;
  final VoidCallback onTap;
  final VoidCallback onDetailTap;

  const _RecommendCard({
    required this.rec,
    this.buySignal,
    this.sellSignal,
    required this.signalsLoading,
    required this.onAdd,
    required this.onTap,
    required this.onDetailTap,
  });

  @override
  Widget build(BuildContext context) {
    final changeDir = rec.changePercent >= 0 ? 'up' : 'down';
    final changeColor = changeDir == 'up' ? AppTheme.up : AppTheme.down;

    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Row(children: [
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                  color: AppTheme.up.withAlpha(25),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text('#${rec.rank}', style: const TextStyle(fontSize: 12, color: AppTheme.up, fontWeight: FontWeight.w700)),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text(rec.symbol, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 16, color: AppTheme.textPrimary)),
                  if (rec.name.isNotEmpty && rec.name != rec.symbol)
                    Text(rec.name, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
                ]),
              ),
              if (rec.price > 0)
                Column(crossAxisAlignment: CrossAxisAlignment.end, children: [
                  Text(formatPrice(rec.price), style: TextStyle(fontWeight: FontWeight.w700, color: changeColor)),
                  Text('${rec.changePercent >= 0 ? '+' : ''}${rec.changePercent.toStringAsFixed(2)}%',
                      style: TextStyle(fontSize: 12, color: changeColor)),
                ]),
              const SizedBox(width: 8),
              IconButton(
                icon: const Icon(Icons.add_circle_outline, color: AppTheme.up),
                onPressed: onAdd, tooltip: '加入自选',
              ),
            ]),
            if (rec.highlights.isNotEmpty) ...[
              const SizedBox(height: 8),
              Wrap(spacing: 6, runSpacing: 4,
                children: rec.highlights.take(2).map((h) => Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: AppTheme.textSecondary.withAlpha(20),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(h, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
                )).toList(),
              ),
            ],
            if (buySignal != null || sellSignal != null || signalsLoading) ...[
              const SizedBox(height: 6),
              _buildSignalRow(),
            ],
          ]),
        ),
      ),
    );
  }

  Widget _buildSignalRow() {
    if (signalsLoading) {
      return Row(children: [
        Icon(Icons.show_chart, size: 14, color: AppTheme.textSecondary.withAlpha(150)),
        const SizedBox(width: 4),
        Text('加载技术信号...', style: TextStyle(fontSize: 12, color: AppTheme.textSecondary.withAlpha(150))),
      ]);
    }

    final buyPct = buySignal != null ? (buySignal!.score / buySignal!.maxScore * 100) : 0.0;
    final sellPct = sellSignal != null ? (sellSignal!.score / sellSignal!.maxScore * 100) : 0.0;
    final hasBuy = buySignal != null && buyPct >= 25;
    final hasSell = sellSignal != null && sellPct >= 25;

    Color signalColor;
    String text;
    if (hasBuy && buyPct >= 50) {
      signalColor = AppTheme.up;
      text = '强烈买入 ${buyPct.toStringAsFixed(0)}% · ${buySignal!.count}信号';
    } else if (hasSell && sellPct >= 50) {
      signalColor = AppTheme.down;
      text = '强烈卖出 ${sellPct.toStringAsFixed(0)}% · ${sellSignal!.count}信号';
    } else if (hasBuy) {
      signalColor = Colors.orange;
      text = '值得关注 ${buyPct.toStringAsFixed(0)}% · ${buySignal!.count}信号';
    } else if (hasSell) {
      signalColor = Colors.orange;
      text = '偏弱 ${sellPct.toStringAsFixed(0)}% · ${sellSignal!.count}信号';
    } else {
      signalColor = AppTheme.textSecondary;
      text = '暂无明确信号';
    }

    return GestureDetector(
      onTap: onDetailTap,
      child: Row(children: [
        Icon(Icons.show_chart, size: 14, color: signalColor),
        const SizedBox(width: 4),
        Text(text, style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: signalColor)),
        const Spacer(),
        Icon(Icons.chevron_right, size: 16, color: signalColor),
      ]),
    );
  }
}
