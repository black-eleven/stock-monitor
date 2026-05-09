import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../domain/model/stock.dart';
import '../providers/api_providers.dart';
import '../providers/quote_provider.dart';
import '../widgets/stock_card.dart';

class WatchlistScreen extends ConsumerStatefulWidget {
  const WatchlistScreen({super.key});
  @override
  ConsumerState<WatchlistScreen> createState() => _WatchlistScreenState();
}

class _WatchlistScreenState extends ConsumerState<WatchlistScreen> {
  List<WatchlistItem>? _watchlist;

  @override
  void initState() {
    super.initState();
    _load();
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
    final quote = ref.read(quoteProvider).getQuote(item.symbol);
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

  @override
  Widget build(BuildContext context) {
    if (_watchlist == null) return const Center(child: CircularProgressIndicator());
    final quotes = ref.watch(quoteProvider).quotes;

    return Scaffold(
      appBar: AppBar(title: const Text('自选股'), actions: [
        IconButton(onPressed: _add, icon: const Icon(Icons.add)),
      ]),
      body: _watchlist!.isEmpty
          ? const Center(child: Text('暂无自选股\n点击右上角 + 添加', textAlign: TextAlign.center, style: TextStyle(color: AppTheme.textSecondary)))
          : ListView.builder(
              itemCount: _watchlist!.length,
              itemBuilder: (_, i) {
                final item = _watchlist![i];
                return StockCard(item: item, quote: quotes[item.symbol], onTap: () => _showDetail(item), onDelete: () async {
                  await ref.read(watchlistApiProvider).remove(item.symbol);
                  _load();
                });
              },
            ),
    );
  }
}
