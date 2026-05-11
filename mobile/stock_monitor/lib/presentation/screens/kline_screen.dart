import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/model/kline.dart';
import '../../domain/model/stock.dart';
import '../providers/api_providers.dart';
import '../widgets/kline_chart.dart';

class KlineScreen extends ConsumerStatefulWidget {
  const KlineScreen({super.key});
  @override
  ConsumerState<KlineScreen> createState() => _KlineScreenState();
}

class _KlineScreenState extends ConsumerState<KlineScreen> {
  String? _selectedSymbol;
  String _interval = '1d';
  List<WatchlistItem>? _watchlist;
  List<Bar> _bars = [];
  bool _loading = false;

  static const _intervals = {'分时': '1m', '5M': '5m', '15M': '15m', '30M': '30m', '1H': '1h', '日K': '1d', '周K': '1w', '月K': '1M'};

  @override
  void initState() {
    super.initState();
    _loadWatchlist();
  }

  Future<void> _loadWatchlist() async {
    final list = await ref.read(watchlistApiProvider).getAll();
    setState(() {
      _watchlist = list;
      if (list.isNotEmpty && _selectedSymbol == null) {
        _selectedSymbol = list.first.symbol;
        _loadKline();
      }
    });
  }

  Future<void> _loadKline() async {
    if (_selectedSymbol == null) return;
    setState(() => _loading = true);
    try {
      final data = await ref.read(quoteApiProvider).getKline(_selectedSymbol!, interval: _interval, count: 200);
      final allBars = <Bar>[];
      for (final item in data) {
        for (final k in item.k) {
          allBars.add(Bar(time: k.ts, open: k.o, high: k.h, low: k.l, close: k.cl, volume: k.v));
        }
      }
      allBars.sort((a, b) => a.time.compareTo(b.time));
      setState(() { _bars = allBars; _loading = false; });
    } catch (e) {
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('K线图')),
      body: Column(children: [
        if (_watchlist != null)
          Padding(
            padding: const EdgeInsets.all(8),
            child: DropdownButtonFormField<String>(
              value: _selectedSymbol,
              decoration: const InputDecoration(contentPadding: EdgeInsets.symmetric(horizontal: 12)),
              items: _watchlist!.map((w) => DropdownMenuItem(value: w.symbol, child: Text('${w.name} (${shortCode(w.symbol)})'))).toList(),
              onChanged: (v) { setState(() => _selectedSymbol = v); _loadKline(); },
            ),
          ),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          padding: const EdgeInsets.symmetric(horizontal: 8),
          child: Row(children: _intervals.entries.map((e) => Padding(
                padding: const EdgeInsets.symmetric(horizontal: 2),
                child: ChoiceChip(label: Text(e.key, style: const TextStyle(fontSize: 12)), selected: _interval == e.value, onSelected: (_) { setState(() => _interval = e.value); _loadKline(); }),
              )).toList()),
        ),
        const SizedBox(height: 8),
        Expanded(child: _loading ? const Center(child: CircularProgressIndicator()) : KlineChartWidget(bars: _bars)),
      ]),
    );
  }
}
