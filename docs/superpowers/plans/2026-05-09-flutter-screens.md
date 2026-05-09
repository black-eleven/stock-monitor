# Flutter Screens — Detailed Implementation

> Companion to `2026-05-09-go-flutter-migration.md`. Contains complete widget code for Tasks 23-28.

---

### Task 23: Watchlist screen + Stock Card widget

**Files:**
- Modify: `mobile/stock_monitor/lib/app.dart` (remove WatchlistScreen placeholder)
- Create: `mobile/stock_monitor/lib/presentation/screens/watchlist_screen.dart`
- Create: `mobile/stock_monitor/lib/presentation/widgets/stock_card.dart`

- [ ] **Step 1: Write stock_card.dart**

```dart
// lib/presentation/widgets/stock_card.dart
import 'package:flutter/material.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/model/stock.dart';

class StockCard extends StatelessWidget {
  final WatchlistItem item;
  final Quote? quote;
  final VoidCallback onTap;
  final VoidCallback onDelete;

  const StockCard({
    super.key,
    required this.item,
    this.quote,
    required this.onTap,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: ListTile(
        onTap: onTap,
        title: Text(item.name, style: const TextStyle(fontWeight: FontWeight.w600, color: AppTheme.textPrimary)),
        subtitle: Text(shortCode(item.symbol), style: const TextStyle(color: AppTheme.textSecondary)),
        trailing: quote != null
            ? Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(formatPrice(quote!.price),
                    style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700,
                      color: changeDir(quote!.price, quote!.yp) == 'up' ? AppTheme.up : AppTheme.down)),
                  Text(formatChange(quote!.price, quote!.yp),
                    style: TextStyle(fontSize: 13,
                      color: changeDir(quote!.price, quote!.yp) == 'up' ? AppTheme.up : AppTheme.down)),
                ],
              )
            : const Icon(Icons.chevron_right, color: AppTheme.textSecondary),
      ),
    );
  }
}

class StockDetailSheet extends StatelessWidget {
  final WatchlistItem item;
  final Quote quote;
  final VoidCallback onDelete;

  const StockDetailSheet({
    super.key,
    required this.item,
    required this.quote,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            Expanded(child: Text(item.name, style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w700))),
            IconButton(onPressed: () { Navigator.pop(context); onDelete(); }, icon: const Icon(Icons.delete_outline, color: AppTheme.down)),
          ]),
          const SizedBox(height: 8),
          Row(children: [
            Text(formatPrice(quote.price), style: TextStyle(fontSize: 32, fontWeight: FontWeight.w800,
              color: changeDir(quote.price, quote.yp) == 'up' ? AppTheme.up : AppTheme.down)),
            const SizedBox(width: 12),
            Text(formatChange(quote.price, quote.yp), style: TextStyle(fontSize: 18,
              color: changeDir(quote.price, quote.yp) == 'up' ? AppTheme.up : AppTheme.down)),
          ]),
          const SizedBox(height: 20),
          _infoRow('今开', formatPrice(quote.open)),
          _infoRow('最高', formatPrice(quote.high)),
          _infoRow('最低', formatPrice(quote.low)),
          _infoRow('昨收', formatPrice(quote.yp)),
          _infoRow('成交量', formatVolume(quote.volume)),
          _infoRow('成交额', formatVolume(quote.turnover)),
        ],
      ),
    );
  }

  Widget _infoRow(String label, String value) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 4),
    child: Row(children: [
      SizedBox(width: 80, child: Text(label, style: const TextStyle(color: AppTheme.textSecondary))),
      Text(value, style: const TextStyle(fontWeight: FontWeight.w500)),
    ]),
  );
}
```

- [ ] **Step 2: Write watchlist_screen.dart**

```dart
// lib/presentation/screens/watchlist_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
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
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(controller: symbolCtrl, decoration: const InputDecoration(hintText: '代码 (如 HK:700)')),
            const SizedBox(height: 12),
            TextField(controller: nameCtrl, decoration: const InputDecoration(hintText: '名称 (如 腾讯控股)')),
          ],
        ),
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
    if (_watchlist == null) {
      return const Center(child: CircularProgressIndicator());
    }

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
            ),
    );
  }
}
```

Note: Add `import '../../core/theme.dart';` at the top of the file.

- [ ] **Step 3: Update app.dart to import the real screen**

In `app.dart`, remove the `WatchlistScreen` placeholder class and add:
```dart
import 'package:stock_monitor/presentation/screens/watchlist_screen.dart';
```

- [ ] **Step 4: Verify compilation**

Run: `cd mobile/stock_monitor && flutter analyze lib/presentation/screens/watchlist_screen.dart 2>&1 | tail -3`
Expected: No issues found

- [ ] **Step 5: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/screens/ mobile/stock_monitor/lib/presentation/widgets/ mobile/stock_monitor/lib/app.dart
git commit -m "feat: implement watchlist screen with stock cards"
```

---

### Task 24: K-line screen + chart widget

**Files:**
- Create: `mobile/stock_monitor/lib/presentation/screens/kline_screen.dart`
- Create: `mobile/stock_monitor/lib/presentation/widgets/kline_chart.dart`

- [ ] **Step 1: Write kline_chart.dart**

```dart
// lib/presentation/widgets/kline_chart.dart
import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';
import '../../core/theme.dart';
import '../../domain/model/kline.dart';

class KlineChartWidget extends StatelessWidget {
  final List<Bar> bars;
  final List<Bar>? rawBars;

  const KlineChartWidget({super.key, required this.bars, this.rawBars});

  @override
  Widget build(BuildContext context) {
    if (bars.isEmpty) return const Center(child: Text('无数据'));

    final spots = bars.map((b) => CandlestickData(
      date: b.time,
      open: b.open,
      high: b.high,
      low: b.low,
      close: b.close,
      color: b.close >= b.open ? AppTheme.up : AppTheme.down,
    )).toList();

    final visibleSpots = spots.length > 100 ? spots.sublist(spots.length - 100) : spots;

    return Column(
      children: [
        Expanded(
          flex: 60,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4),
            child: CandlestickChart(
              CandlestickChartData(
                minY: visibleSpots.map((s) => s.low).reduce((a, b) => a < b ? a : b) * 0.995,
                maxY: visibleSpots.map((s) => s.high).reduce((a, b) => a > b ? a : b) * 1.005,
                lineColor: AppTheme.border,
                dataSets: [
                  CandlestickDataSet(data: visibleSpots),
                ],
              ),
            ),
          ),
        ),
      ],
    );
  }
}
```

- [ ] **Step 2: Write kline_screen.dart**

```dart
// lib/presentation/screens/kline_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../domain/model/kline.dart';
import '../../domain/model/stock.dart';
import '../providers/api_providers.dart';
import '../providers/kline_provider.dart';
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

  static const _intervals = {'分时': '1m', '5M': '5m', '15M': '15m', '30M': '30m', '1H': '1h', '日K': '1d', '周K': '1w', '月K': '1M'};

  @override
  void initState() {
    super.initState();
    _loadWatchlist();
  }

  Future<void> _loadWatchlist() async {
    final api = ref.read(watchlistApiProvider);
    final list = await api.getAll();
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
    final api = ref.read(quoteApiProvider);
    final data = await api.getKline(_selectedSymbol!, interval: _interval, count: 200);
    final allBars = <Bar>[];
    for (final item in data) {
      for (final k in item.k) {
        allBars.add(Bar(time: k.ts, open: k.o, high: k.h, low: k.l, close: k.cl));
      }
    }
    allBars.sort((a, b) => a.time.compareTo(b.time));
    setState(() => _bars = allBars);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('K线图')),
      body: Column(
        children: [
          // Symbol selector
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

          // Interval buttons
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: Row(
              children: _intervals.entries.map((e) => Padding(
                padding: const EdgeInsets.symmetric(horizontal: 2),
                child: ChoiceChip(
                  label: Text(e.key, style: const TextStyle(fontSize: 12)),
                  selected: _interval == e.value,
                  onSelected: (_) { setState(() => _interval = e.value); _loadKline(); },
                ),
              )).toList(),
            ),
          ),

          const SizedBox(height: 8),

          // Chart
          Expanded(child: _bars.isNotEmpty
            ? KlineChartWidget(bars: _bars)
            : const Center(child: CircularProgressIndicator())),
        ],
      ),
    );
  }
}
```

Note: Add `import '../../core/utils.dart';` for `shortCode`.

- [ ] **Step 3: Update app.dart** — import KlineScreen and remove placeholder

- [ ] **Step 4: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/screens/kline_screen.dart mobile/stock_monitor/lib/presentation/widgets/kline_chart.dart
git commit -m "feat: implement K-line screen with candlestick chart"
```

---

### Task 25: Technical indicators (Dart port of indicators.js)

**Files:**
- Create: `mobile/stock_monitor/lib/domain/indicators.dart`

- [ ] **Step 1: Port indicators from JS to Dart**

```dart
// lib/domain/indicators.dart
import 'model/kline.dart';

class MA {
  final int time;
  final double value;
  MA(this.time, this.value);
}

List<MA> calcMA(List<Bar> bars, int period) {
  if (bars.length < period) return [];
  final result = <MA>[];
  double sum = 0;
  for (int i = 0; i < bars.length; i++) {
    sum += bars[i].close;
    if (i >= period) sum -= bars[i - period].close;
    if (i >= period - 1) result.add(MA(bars[i].time, sum / period));
  }
  return result;
}

class RSIVal {
  final int time;
  final double value;
  RSIVal(this.time, this.value);
}

List<RSIVal> calcRSI(List<Bar> bars, int period) {
  if (bars.length <= period) return [];
  final result = <RSIVal>[];
  double avgGain = 0, avgLoss = 0;

  for (int i = 1; i <= period; i++) {
    final diff = bars[i].close - bars[i - 1].close;
    if (diff > 0) avgGain += diff; else avgLoss -= diff;
  }
  avgGain /= period;
  avgLoss /= period;

  for (int i = period; i < bars.length; i++) {
    if (avgLoss == 0) {
      result.add(RSIVal(bars[i].time, 100));
    } else {
      final rs = avgGain / avgLoss;
      result.add(RSIVal(bars[i].time, 100 - 100 / (1 + rs)));
    }
    final diff = bars[i].close - bars[i - 1].close;
    final gain = diff > 0 ? diff : 0.0;
    final loss = diff < 0 ? -diff : 0.0;
    avgGain = (avgGain * (period - 1) + gain) / period;
    avgLoss = (avgLoss * (period - 1) + loss) / period;
  }
  return result;
}

class MACDResult {
  final List<MA> dif;
  final List<MA> dea;
  final List<MA> macd;
  MACDResult(this.dif, this.dea, this.macd);
}

double _ema(List<double> values, int period, int i) {
  final k = 2.0 / (period + 1);
  double ema = values[0];
  for (int j = 1; j <= i; j++) {
    ema = values[j] * k + ema * (1 - k);
  }
  return ema;
}

MACDResult? calcMACD(List<Bar> bars, {int fast = 12, int slow = 26, int signal = 9}) {
  if (bars.length < slow) return null;
  final closes = bars.map((b) => b.close).toList();

  final dif = <MA>[];
  for (int i = slow - 1; i < bars.length; i++) {
    final fastEma = _ema(closes.sublist(0, i + 1), fast, i);
    final slowEma = _ema(closes.sublist(0, i + 1), slow, i);
    dif.add(MA(bars[i].time, fastEma - slowEma));
  }

  final dea = <MA>[];
  final macd = <MA>[];
  final difValues = dif.map((d) => d.value).toList();
  for (int i = signal - 1; i < dif.length; i++) {
    final deaVal = _ema(difValues.sublist(0, i + 1), signal, i);
    dea.add(MA(dif[i].time, deaVal));
    macd.add(MA(dif[i].time, (dif[i].value - deaVal) * 2));
  }

  return MACDResult(dif, dea, macd);
}

class SignalResult {
  final double score;
  final double maxScore;
  final int count;
  final String summary;
  SignalResult({required this.score, required this.maxScore, required this.count, required this.summary});
}

SignalResult evaluateSignals(List<Bar> bars) {
  final ma5 = calcMA(bars, 5);
  final ma20 = calcMA(bars, 20);
  final ma60 = calcMA(bars, 60);
  final rsi = calcRSI(bars, 14);
  final macd = calcMACD(bars);
  double score = 0;
  int count = 0;
  const maxScore = 13.0;

  if (ma5.length >= 2 && ma20.length >= 2) {
    final prevMA5 = ma5[ma5.length - 2].value;
    final prevMA20 = ma20[ma20.length - 2].value;
    final currMA5 = ma5.last.value;
    final currMA20 = ma20.last.value;
    if (prevMA5 >= prevMA20 && currMA5 < currMA20) { score += 2.0; count++; }
    if (bars.last.close < currMA20) { score += 1.0; count++; }
  }

  if (ma60.isNotEmpty && bars.last.close < ma60.last.value) { score += 0.5; count++; }

  if (rsi.isNotEmpty) {
    final rsiVal = rsi.last.value;
    if (rsiVal > 70) { score += 1.0; count++; }
    // Bearish divergence check (simplified): RSI decreasing while price makes higher high
    if (rsi.length >= 20 && bars.length >= 20) {
      final recentRSI = rsi.sublist(rsi.length - 20);
      final recentBars = bars.sublist(bars.length - 20);
      double maxPrice = 0, maxRsi = 0;
      int pricePeak = 0, rsiPeak = 0;
      for (int i = 0; i < 20; i++) {
        if (recentBars[i].high > maxPrice) { maxPrice = recentBars[i].high; pricePeak = i; }
        if (recentRSI[i].value > maxRsi) { maxRsi = recentRSI[i].value; rsiPeak = i; }
      }
      if (pricePeak > rsiPeak && recentBars.last.high >= maxPrice * 0.98) { score += 2.5; count++; }
    }
  }

  if (macd != null && macd.dea.length >= 2 && macd.dif.length >= 2) {
    final prevDIF = macd.dif[macd.dif.length - 2].value;
    final prevDEA = macd.dea[macd.dea.length - 2].value;
    final currDIF = macd.dif.last.value;
    final currDEA = macd.dea.last.value;
    if (prevDIF >= prevDEA && currDIF < currDEA) { score += 2.0; count++; }
  }

  String summary;
  final pct = score / maxScore;
  if (pct >= 0.5) summary = '强烈卖出信号';
  else if (pct >= 0.25) summary = '偏弱，注意风险';
  else if (pct > 0) summary = '短期偏弱';
  else summary = '正常';

  return SignalResult(score: score, maxScore: maxScore, count: count, summary: summary);
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd mobile/stock_monitor && flutter analyze lib/domain/indicators.dart 2>&1 | tail -3`
Expected: No issues found

- [ ] **Step 3: Commit**

```bash
git add mobile/stock_monitor/lib/domain/indicators.dart
git commit -m "feat: port technical indicators to Dart (MA, RSI, MACD, signals)"
```

---

### Task 26: Holdings screen

**Files:**
- Create: `mobile/stock_monitor/lib/presentation/screens/holdings_screen.dart`

- [ ] **Step 1: Write holdings_screen.dart**

```dart
// lib/presentation/screens/holdings_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/model/holding.dart';
import '../providers/api_providers.dart';
import '../providers/quote_provider.dart';

class HoldingsScreen extends ConsumerStatefulWidget {
  const HoldingsScreen({super.key});
  @override
  ConsumerState<HoldingsScreen> createState() => _HoldingsScreenState();
}

class _HoldingsScreenState extends ConsumerState<HoldingsScreen> {
  List<Holding>? _holdings;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final api = ref.read(holdingApiProvider);
    final list = await api.getAll();
    setState(() => _holdings = list);
  }

  Future<void> _add() async {
    final symbolCtrl = TextEditingController();
    final nameCtrl = TextEditingController();
    final sharesCtrl = TextEditingController();
    final costCtrl = TextEditingController();

    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('添加持仓'),
        content: SingleChildScrollView(
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            TextField(controller: symbolCtrl, decoration: const InputDecoration(hintText: '代码')),
            TextField(controller: nameCtrl, decoration: const InputDecoration(hintText: '名称')),
            TextField(controller: sharesCtrl, decoration: const InputDecoration(hintText: '股数'), keyboardType: TextInputType.number),
            TextField(controller: costCtrl, decoration: const InputDecoration(hintText: '成本价'), keyboardType: TextInputType.number),
          ]),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('添加')),
        ],
      ),
    );

    if (ok == true) {
      try {
        await ref.read(holdingApiProvider).add({
          'symbol': symbolCtrl.text.toUpperCase(),
          'name': nameCtrl.text,
          'shares': double.parse(sharesCtrl.text),
          'avgCost': double.parse(costCtrl.text),
        });
        _load();
      } catch (e) {
        if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('添加失败: $e')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_holdings == null) return const Center(child: CircularProgressIndicator());

    final quotes = ref.watch(quoteProvider).quotes;

    double totalCost = 0, totalValue = 0;
    for (final h in _holdings!) {
      totalCost += h.shares * h.avgCost;
      final q = quotes[h.symbol];
      if (q != null) totalValue += h.shares * q.price;
    }
    final totalPL = totalValue - totalCost;
    final totalPLPct = totalCost > 0 ? totalPL / totalCost * 100 : 0;

    return Scaffold(
      appBar: AppBar(title: const Text('持仓'), actions: [
        IconButton(onPressed: _add, icon: const Icon(Icons.add)),
      ]),
      body: _holdings!.isEmpty
          ? const Center(child: Text('暂无持仓', style: TextStyle(color: AppTheme.textSecondary)))
          : Column(
              children: [
                // Summary bar
                Container(
                  padding: const EdgeInsets.all(12),
                  color: AppTheme.surface,
                  child: Row(mainAxisAlignment: MainAxisAlignment.spaceAround, children: [
                    _summaryBox('总成本', formatPrice(totalCost)),
                    _summaryBox('总市值', formatPrice(totalValue)),
                    _summaryBox('总盈亏', formatPrice(totalPL), color: totalPL >= 0 ? AppTheme.up : AppTheme.down),
                    _summaryBox('盈亏率', '${totalPLPct.toStringAsFixed(2)}%', color: totalPLPct >= 0 ? AppTheme.up : AppTheme.down),
                  ]),
                ),
                // Holdings list
                Expanded(
                  child: ListView.builder(
                    itemCount: _holdings!.length,
                    itemBuilder: (_, i) {
                      final h = _holdings![i];
                      final q = quotes[h.symbol];
                      final mv = q != null ? h.shares * q.price : 0.0;
                      final pl = mv - h.shares * h.avgCost;
                      final plPct = h.avgCost > 0 ? (q?.price ?? 0 - h.avgCost) / h.avgCost * 100 : 0;

                      return Card(
                        margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                        child: ListTile(
                          title: Text('${h.name} (${shortCode(h.symbol)})', style: const TextStyle(fontWeight: FontWeight.w600)),
                          subtitle: Text('${h.shares.toStringAsFixed(0)}股 × ¥${formatPrice(h.avgCost)}'),
                          trailing: Column(mainAxisAlignment: MainAxisAlignment.center, crossAxisAlignment: CrossAxisAlignment.end, children: [
                            Text(formatPrice(pl), style: TextStyle(fontWeight: FontWeight.w700, color: pl >= 0 ? AppTheme.up : AppTheme.down)),
                            Text('${plPct.toStringAsFixed(2)}%', style: TextStyle(fontSize: 12, color: plPct >= 0 ? AppTheme.up : AppTheme.down)),
                          ]),
                          onLongPress: () async {
                            await ref.read(holdingApiProvider).remove(h.symbol);
                            _load();
                          },
                        ),
                      );
                    },
                  ),
                ),
              ],
            ),
    );
  }

  Widget _summaryBox(String label, String value, {Color? color}) => Column(
    children: [
      Text(label, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
      Text(value, style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700, color: color ?? AppTheme.textPrimary)),
    ],
  );
}
```

- [ ] **Step 2: Update app.dart** — import HoldingsScreen, remove placeholder

- [ ] **Step 3: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/screens/holdings_screen.dart
git commit -m "feat: implement holdings screen with summary and P&L"
```

---

### Task 27: Alerts screen

**Files:**
- Create: `mobile/stock_monitor/lib/presentation/screens/alerts_screen.dart`

- [ ] **Step 1: Write alerts_screen.dart**

```dart
// lib/presentation/screens/alerts_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../data/ws/ws_client.dart';
import '../../domain/model/alert.dart';
import '../providers/api_providers.dart';

class AlertsScreen extends ConsumerStatefulWidget {
  const AlertsScreen({super.key});
  @override
  ConsumerState<AlertsScreen> createState() => _AlertsScreenState();
}

class _AlertsScreenState extends ConsumerState<AlertsScreen> {
  List<AlertRule>? _rules;
  final List<AlertEvent> _logs = [];

  @override
  void initState() {
    super.initState();
    _load();
    ref.read(wsClientProvider).alertStream.listen((event) {
      setState(() { _logs.insert(0, event); if (_logs.length > 50) _logs.removeLast(); });
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(event.message)));
    });
  }

  Future<void> _load() async {
    final api = ref.read(alertApiProvider);
    final list = await api.getAll();
    setState(() => _rules = list);
  }

  Future<void> _add() async {
    final symbolCtrl = TextEditingController();
    String type = 'above';
    final valueCtrl = TextEditingController();

    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(builder: (ctx, setDialogState) => AlertDialog(
        title: const Text('添加提醒'),
        content: Column(mainAxisSize: MainAxisSize.min, children: [
          TextField(controller: symbolCtrl, decoration: const InputDecoration(hintText: '代码 (如 HK:700)')),
          DropdownButtonFormField<String>(
            value: type,
            items: const [
              DropdownMenuItem(value: 'above', child: Text('涨破')),
              DropdownMenuItem(value: 'below', child: Text('跌破')),
              DropdownMenuItem(value: 'change_pct', child: Text('涨跌幅达')),
            ],
            onChanged: (v) => setDialogState(() => type = v!),
          ),
          TextField(controller: valueCtrl, decoration: const InputDecoration(hintText: '阈值'), keyboardType: TextInputType.number),
        ]),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('添加')),
        ],
      )),
    );

    if (ok == true && symbolCtrl.text.isNotEmpty && valueCtrl.text.isNotEmpty) {
      try {
        await ref.read(alertApiProvider).add(symbolCtrl.text.toUpperCase(), type, double.parse(valueCtrl.text));
        _load();
      } catch (e) {
        if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('添加失败: $e')));
      }
    }
  }

  String _typeLabel(String t) => {'above': '涨破', 'below': '跌破', 'change_pct': '涨跌幅达 %'}[t] ?? t;

  @override
  Widget build(BuildContext context) {
    if (_rules == null) return const Center(child: CircularProgressIndicator());

    return Scaffold(
      appBar: AppBar(title: const Text('提醒'), actions: [
        IconButton(onPressed: _add, icon: const Icon(Icons.add)),
      ]),
      body: Column(
        children: [
          // Rules
          Expanded(
            flex: 3,
            child: _rules!.isEmpty
                ? const Center(child: Text('暂无提醒规则', style: TextStyle(color: AppTheme.textSecondary)))
                : ListView.builder(
                    itemCount: _rules!.length,
                    itemBuilder: (_, i) {
                      final r = _rules![i];
                      return Card(
                        margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                        child: ListTile(
                          title: Text(r.symbol, style: const TextStyle(fontWeight: FontWeight.w600)),
                          subtitle: Text('${_typeLabel(r.type)} ${r.value}'),
                          trailing: Row(mainAxisSize: MainAxisSize.min, children: [
                            Switch(
                              value: r.enabled,
                              onChanged: (v) async {
                                await ref.read(alertApiProvider).update(r.id, {'enabled': v});
                                _load();
                              },
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline, color: AppTheme.down, size: 20),
                              onPressed: () async {
                                await ref.read(alertApiProvider).remove(r.id);
                                _load();
                              },
                            ),
                          ]),
                        ),
                      );
                    },
                  ),
          ),
          // Logs
          const Divider(height: 1),
          const Padding(padding: EdgeInsets.all(8), child: Text('触发记录', style: TextStyle(fontWeight: FontWeight.w600, color: AppTheme.textSecondary))),
          Expanded(
            flex: 2,
            child: _logs.isEmpty
                ? const Center(child: Text('暂无触发记录', style: TextStyle(color: AppTheme.textSecondary)))
                : ListView.builder(
                    itemCount: _logs.length,
                    itemBuilder: (_, i) => ListTile(
                      dense: true,
                      title: Text(_logs[i].message, style: const TextStyle(fontSize: 13)),
                      trailing: Text(_logs[i].triggeredAt.substring(11, 19), style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary)),
                    ),
                  ),
          ),
        ],
      ),
    );
  }
}
```

- [ ] **Step 2: Update app.dart** — import AlertsScreen, remove placeholder

- [ ] **Step 3: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/screens/alerts_screen.dart
git commit -m "feat: implement alerts screen with rules and trigger logs"
```

---

### Task 28: Analysis screen

**Files:**
- Create: `mobile/stock_monitor/lib/presentation/screens/analysis_screen.dart`

- [ ] **Step 1: Write analysis_screen.dart**

```dart
// lib/presentation/screens/analysis_screen.dart
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
  List<_AnalysisResult>? _results;
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

    final results = <_AnalysisResult>[];
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

        final sig = evaluateSignals(bars);
        results.add(_AnalysisResult(stock: stock, signal: sig, bars: bars));
      } catch (_) {}
    }

    results.sort((a, b) => b.signal.score.compareTo(a.signal.score));
    setState(() { _results = results; _loading = false; });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('卖出分析'), actions: [
        IconButton(onPressed: _analyze, icon: const Icon(Icons.refresh)),
      ]),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _results == null || _results!.isEmpty
              ? const Center(child: Text('暂无数据', style: TextStyle(color: AppTheme.textSecondary)))
              : Column(
                  children: [
                    // Average score
                    Container(
                      padding: const EdgeInsets.all(12),
                      color: AppTheme.surface,
                      child: _buildSummary(),
                    ),
                    // Results list
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
                              onTap: () => _showDetail(r),
                            ),
                          );
                        },
                      ),
                    ),
                  ],
                ),
    );
  }

  Widget _buildSummary() {
    if (_results!.isEmpty) return const SizedBox.shrink();
    final avgScore = _results!.map((r) => r.signal.score / r.signal.maxScore).reduce((a, b) => a + b) / _results!.length;
    final color = avgScore >= 0.5 ? AppTheme.down : avgScore >= 0.25 ? Colors.orange : AppTheme.up;
    final label = avgScore >= 0.5 ? '强烈偏空' : avgScore >= 0.25 ? '偏弱' : '正常';
    return Row(mainAxisAlignment: MainAxisAlignment.center, children: [
      Text('平均卖出分 ', style: const TextStyle(color: AppTheme.textSecondary)),
      Text('${(avgScore * 100).toStringAsFixed(1)}分', style: TextStyle(fontSize: 24, fontWeight: FontWeight.w800, color: color)),
      const SizedBox(width: 12),
      Container(padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4), decoration: BoxDecoration(color: color.withAlpha(40), borderRadius: BorderRadius.circular(12)), child: Text(label, style: TextStyle(color: color, fontWeight: FontWeight.w600))),
    ]);
  }

  void _showDetail(_AnalysisResult r) {
    showModalBottomSheet(
      context: context,
      builder: (_) => Container(
        padding: const EdgeInsets.all(20),
        child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text('${r.stock.name} (${r.stock.symbol})', style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w700)),
          const SizedBox(height: 4),
          Text('卖出评分: ${(r.signal.score / r.signal.maxScore * 100).toStringAsFixed(0)}分 — ${r.signal.summary}'),
          const SizedBox(height: 12),
          Text('信号数: ${r.signal.count} / 8', style: const TextStyle(color: AppTheme.textSecondary)),
        ]),
      ),
    );
  }
}

class _AnalysisResult {
  final WatchlistItem stock;
  final SignalResult signal;
  final List<Bar> bars;
  _AnalysisResult({required this.stock, required this.signal, required this.bars});
}
```

- [ ] **Step 2: Update app.dart** — import AnalysisScreen, remove placeholder

- [ ] **Step 3: Commit**

```bash
git add mobile/stock_monitor/lib/presentation/screens/analysis_screen.dart mobile/stock_monitor/lib/app.dart
git commit -m "feat: implement analysis screen with sell signal scoring"
```
