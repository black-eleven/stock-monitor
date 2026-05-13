import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/model/stock.dart';
import '../../domain/model/recommendation.dart';
import '../providers/api_providers.dart';
import '../providers/quote_provider.dart';
import '../widgets/stock_card.dart';

class WatchlistScreen extends ConsumerStatefulWidget {
  const WatchlistScreen({super.key});
  @override
  ConsumerState<WatchlistScreen> createState() => _WatchlistScreenState();
}

class _WatchlistScreenState extends ConsumerState<WatchlistScreen>
    with SingleTickerProviderStateMixin {
  List<WatchlistItem>? _watchlist;
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

  void _showDetail(WatchlistItem item) {
    final quote = ref.read(quoteProvider).quotes[item.symbol];
    if (quote == null) return;
    showModalBottomSheet(
      context: context,
      builder: (_) => StockDetailSheet(
        item: item,
        quote: quote,
        onDelete: () async {
          await ref.read(watchlistApiProvider).remove(item.symbol);
          _load();
        },
      ),
    );
  }

  void _openKline(String symbol) {
    Navigator.of(context).pushNamed('/kline', arguments: {'symbol': symbol});
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
          onTap: () => _showDetail(item),
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
      onOpenKline: _openKline,
    );
  }
}

class _RecommendTab extends StatefulWidget {
  final Future<void> Function(String symbol, String name) onAddToWatchlist;
  final void Function(String symbol) onOpenKline;

  const _RecommendTab({required this.onAddToWatchlist, required this.onOpenKline});

  @override
  State<_RecommendTab> createState() => _RecommendTabState();
}

class _RecommendTabState extends State<_RecommendTab> {
  final _controller = TextEditingController();
  List<Recommendation>? _recs;
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
    });

    try {
      final container = ProviderScope.containerOf(context);
      final api = container.read(recommendApiProvider);
      final recs = await api.recommend(industry);
      setState(() {
        _recs = recs;
        _loading = false;
        if (recs.isEmpty) {
          _error = '未找到相关推荐';
        }
      });
    } catch (e) {
      setState(() {
        _error = '获取推荐失败: $e';
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          Row(
            children: [
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
            ],
          ),
          const SizedBox(height: 16),
          Expanded(child: _buildResults()),
        ],
      ),
    );
  }

  Widget _buildResults() {
    if (_error != null) {
      return Center(
        child: Text(_error!, style: const TextStyle(color: AppTheme.textSecondary)),
      );
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
          onAdd: () => widget.onAddToWatchlist(r.symbol, r.name),
          onTap: () => widget.onOpenKline(r.symbol),
        );
      },
    );
  }
}

class _RecommendCard extends StatelessWidget {
  final Recommendation rec;
  final VoidCallback onAdd;
  final VoidCallback onTap;

  const _RecommendCard({required this.rec, required this.onAdd, required this.onTap});

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
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(
                      color: AppTheme.up.withAlpha(25),
                      borderRadius: BorderRadius.circular(4),
                    ),
                    child: Text('#${rec.rank}', style: const TextStyle(fontSize: 12, color: AppTheme.up, fontWeight: FontWeight.w700)),
                  ),
                  const SizedBox(width: 8),
                  Text(rec.symbol, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 16, color: AppTheme.textPrimary)),
                  const Spacer(),
                  if (rec.price > 0)
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Text(formatPrice(rec.price), style: TextStyle(fontWeight: FontWeight.w700, color: changeColor)),
                        Text('${rec.changePercent >= 0 ? '+' : ''}${rec.changePercent.toStringAsFixed(2)}%', style: TextStyle(fontSize: 12, color: changeColor)),
                      ],
                    ),
                  const SizedBox(width: 8),
                  IconButton(
                    icon: const Icon(Icons.add_circle_outline, color: AppTheme.up),
                    onPressed: onAdd,
                    tooltip: '加入自选',
                  ),
                ],
              ),
              if (rec.highlights.isNotEmpty) ...[
                const SizedBox(height: 8),
                Wrap(
                  spacing: 6,
                  runSpacing: 4,
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
              const SizedBox(height: 4),
              Row(
                children: [
                  Icon(Icons.article_outlined, size: 14, color: AppTheme.textSecondary.withAlpha(150)),
                  const SizedBox(width: 4),
                  Text('${rec.newsCount} 篇相关新闻', style: TextStyle(fontSize: 12, color: AppTheme.textSecondary.withAlpha(150))),
                  const SizedBox(width: 12),
                  Icon(Icons.auto_awesome, size: 14, color: AppTheme.textSecondary.withAlpha(150)),
                  const SizedBox(width: 4),
                  Text('综合评分 ${(rec.score * 100).toStringAsFixed(0)}', style: TextStyle(fontSize: 12, color: AppTheme.textSecondary.withAlpha(150))),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
