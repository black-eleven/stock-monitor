import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/indicators.dart';
import '../../domain/model/kline.dart';
import '../../domain/model/stock.dart';
import '../providers/api_providers.dart';
import '../providers/quote_provider.dart';
import '../widgets/kline_chart.dart';

class StockDetailScreen extends ConsumerStatefulWidget {
  final String symbol;
  const StockDetailScreen({super.key, required this.symbol});

  @override
  ConsumerState<StockDetailScreen> createState() => _StockDetailScreenState();
}

class _StockDetailScreenState extends ConsumerState<StockDetailScreen> {
  String _interval = '1d';
  List<Bar> _bars = [];
  SignalResult? _buySignal;
  SignalResult? _sellSignal;
  bool _loading = true;

  String? _analysis;
  bool _analyzing = false;
  List<String> _strategyKeys = [];
  List<String> _strategyNames = [];
  String _currentStrategy = 'comprehensive';

  static const _intervals = {
    '5M': '5m', '15M': '15m', '30M': '30m', '1H': '1h', '日K': '1d', '周K': '1w', '月K': '1M',
  };

  @override
  void initState() {
    super.initState();
    _loadAll();
  }

  Future<void> _loadAll() async {
    await Future.wait([_loadKline(), _loadStrategies()]);
  }

  Future<void> _loadStrategies() async {
    try {
      final api = ref.read(strategyApiProvider);
      final keys = await api.getStrategyKeys();
      final names = await api.getStrategies();
      if (mounted) setState(() { _strategyKeys = keys; _strategyNames = names; });
    } catch (_) {}
  }

  Future<void> _loadKline() async {
    setState(() => _loading = true);
    try {
      final data = await ref.read(quoteApiProvider).getKline(
        widget.symbol, interval: _interval, count: 200,
      );
      final bars = <Bar>[];
      final rawBars = <KlineBar>[];
      for (final item in data) {
        for (final k in item.k) {
          bars.add(Bar(time: k.ts, open: k.o, high: k.h, low: k.l, close: k.cl, volume: k.v));
          rawBars.add(k);
        }
      }
      bars.sort((a, b) => a.time.compareTo(b.time));
      if (bars.length >= 30) {
        setState(() {
          _bars = bars;
          _buySignal = evaluateBuySignals(bars);
          _sellSignal = evaluateSignals(bars);
          _loading = false;
        });
      } else {
        setState(() { _bars = bars; _loading = false; });
      }
    } catch (e) {
      setState(() => _loading = false);
    }
  }

  Future<void> _runStrategy(String strategy) async {
    setState(() { _analyzing = true; _analysis = null; });
    try {
      final rawBars = _bars.map((b) => KlineBar(
        ts: b.time, o: b.open, cl: b.close, h: b.high, l: b.low, v: b.volume,
      )).toList();
      final result = await ref.read(strategyApiProvider).analyze(strategy, widget.symbol, rawBars);
      if (mounted) setState(() { _analysis = result; _analyzing = false; });
    } catch (e) {
      if (mounted) setState(() { _analysis = '分析失败: $e'; _analyzing = false; });
    }
  }

  void _changeInterval(String val) {
    setState(() => _interval = val);
    _loadKline();
  }

  @override
  Widget build(BuildContext context) {
    final quote = ref.watch(quoteProvider).quotes[widget.symbol];

    return Scaffold(
      appBar: AppBar(title: Text(widget.symbol), actions: [
        IconButton(onPressed: _loadAll, icon: const Icon(Icons.refresh)),
      ]),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : SingleChildScrollView(
              padding: const EdgeInsets.all(12),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                _buildQuoteHeader(quote),
                const SizedBox(height: 12),
                // Interval selector
                SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: Row(
                    children: _intervals.entries.map((e) => Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 2),
                      child: ChoiceChip(
                        label: Text(e.key, style: const TextStyle(fontSize: 12)),
                        selected: _interval == e.value,
                        onSelected: (_) => _changeInterval(e.value),
                      ),
                    )).toList(),
                  ),
                ),
                const SizedBox(height: 8),
                // K-line chart
                SizedBox(
                  height: 300,
                  child: _bars.isNotEmpty
                      ? KlineChartWidget(bars: _bars)
                      : const Center(child: Text('无K线数据')),
                ),
                const SizedBox(height: 16),
                // Signal summary
                if (_buySignal != null || _sellSignal != null)
                  _buildSignalSummary(),
                const SizedBox(height: 12),
                // Signal detail tables
                if (_buySignal != null || _sellSignal != null)
                  _buildSignalTables(),
                const SizedBox(height: 16),
                // AI strategy analysis
                _buildStrategySection(),
              ]),
            ),
    );
  }

  Widget _buildQuoteHeader(Quote? quote) {
    if (quote == null) {
      return Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(color: AppTheme.surface, borderRadius: BorderRadius.circular(8)),
        child: const Text('暂无实时数据', style: TextStyle(color: AppTheme.textSecondary)),
      );
    }
    final dir = changeDir(quote.price, quote.yp);
    final color = dir == 'up' ? AppTheme.up : AppTheme.down;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(color: AppTheme.surface, borderRadius: BorderRadius.circular(8)),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Expanded(child: Text(formatPrice(quote.price), style: TextStyle(fontSize: 36, fontWeight: FontWeight.w800, color: color))),
          Text(formatChange(quote.price, quote.yp), style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600, color: color)),
        ]),
        const SizedBox(height: 12),
        Wrap(spacing: 16, runSpacing: 4, children: [
          _infoText('今开', formatPrice(quote.open)),
          _infoText('最高', formatPrice(quote.high)),
          _infoText('最低', formatPrice(quote.low)),
          _infoText('昨收', formatPrice(quote.yp)),
          _infoText('成交量', formatVolume(quote.volume)),
          _infoText('成交额', formatVolume(quote.turnover)),
        ]),
      ]),
    );
  }

  Widget _infoText(String label, String value) => Row(mainAxisSize: MainAxisSize.min, children: [
    Text('$label ', style: const TextStyle(fontSize: 13, color: AppTheme.textSecondary)),
    Text(value, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500)),
  ]);

  Widget _buildSignalSummary() {
    final buyPct = _buySignal != null ? (_buySignal!.score / _buySignal!.maxScore * 100) : 0.0;
    final sellPct = _sellSignal != null ? (_sellSignal!.score / _sellSignal!.maxScore * 100) : 0.0;
    return Row(children: [
      Expanded(child: _signalBox('买入指数', buyPct, true, _buySignal)),
      const SizedBox(width: 12),
      Expanded(child: _signalBox('卖出指数', sellPct, false, _sellSignal)),
    ]);
  }

  Widget _signalBox(String label, double pct, bool isBuy, SignalResult? signal) {
    Color color;
    String text;
    if (pct >= 50) {
      color = isBuy ? AppTheme.up : AppTheme.down;
      text = isBuy ? '强烈推荐' : '强烈卖出';
    } else if (pct >= 25) {
      color = Colors.orange;
      text = isBuy ? '值得关注' : '偏弱';
    } else {
      color = AppTheme.textSecondary;
      text = '暂无信号';
    }
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppTheme.border),
      ),
      child: Column(children: [
        Text(label, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
        Text('${pct.toStringAsFixed(0)}%', style: TextStyle(fontSize: 24, fontWeight: FontWeight.w800, color: color)),
        Text(text, style: TextStyle(fontSize: 12, color: color)),
        if (signal != null)
          Text('${signal.count}/${signal.total} 信号', style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary)),
      ]),
    );
  }

  Widget _buildSignalTables() {
    return Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Expanded(child: _signalTable('卖出信号', _sellSignal, false)),
      const SizedBox(width: 8),
      Expanded(child: _signalTable('买入信号', _buySignal, true)),
    ]);
  }

  Widget _signalTable(String title, SignalResult? signal, bool isBuy) {
    return Container(
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppTheme.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(title, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: AppTheme.textSecondary)),
        const SizedBox(height: 4),
        if (signal == null)
          const Text('--', style: TextStyle(fontSize: 11, color: AppTheme.textSecondary))
        else
          ...signal.signals.map((s) {
            Color c;
            if (!s.triggered) { c = AppTheme.textSecondary; }
            else if (s.status == 'danger') { c = isBuy ? AppTheme.up : AppTheme.down; }
            else { c = Colors.orange; }
            return Padding(
              padding: const EdgeInsets.symmetric(vertical: 1),
              child: Row(children: [
                Icon(s.triggered ? Icons.circle : Icons.circle_outlined, size: 8, color: c),
                const SizedBox(width: 4),
                Expanded(child: Text(s.name, style: TextStyle(fontSize: 11, color: c))),
              ]),
            );
          }),
      ]),
    );
  }

  Widget _buildStrategySection() {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppTheme.border),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        const Text('AI 策略分析', style: TextStyle(fontWeight: FontWeight.w600, color: AppTheme.accent)),
        const SizedBox(height: 8),
        if (_strategyKeys.isNotEmpty)
          DropdownButton<String>(
            value: _currentStrategy,
            isDense: true,
            dropdownColor: AppTheme.surface,
            style: const TextStyle(color: AppTheme.textPrimary, fontSize: 13),
            underline: Container(height: 0),
            items: List.generate(_strategyKeys.length, (i) => DropdownMenuItem(
              value: _strategyKeys[i], child: Text(_strategyNames[i], style: const TextStyle(fontSize: 12)),
            )),
            onChanged: (v) {
              if (v == null) return;
              _currentStrategy = v;
              _runStrategy(v);
            },
          ),
        const SizedBox(height: 8),
        if (_analysis == null && !_analyzing)
          TextButton(onPressed: () => _runStrategy(_currentStrategy), child: const Text('开始分析'))
        else if (_analyzing)
          const Center(child: CircularProgressIndicator(strokeWidth: 2))
        else
          _buildMarkdown(_analysis!),
      ]),
    );
  }

  Widget _buildMarkdown(String text) {
    final spans = <InlineSpan>[];
    final lines = text.split('\n');
    for (final line in lines) {
      if (line.startsWith('### ')) {
        spans.add(TextSpan(text: '${line.substring(4)}\n',
            style: const TextStyle(color: AppTheme.accent, fontWeight: FontWeight.w600, fontSize: 14)));
      } else if (line.startsWith('## ')) {
        spans.add(TextSpan(text: '${line.substring(3)}\n',
            style: const TextStyle(color: Color(0xFFFFD700), fontWeight: FontWeight.w700, fontSize: 15)));
      } else if (line.startsWith('- ')) {
        spans.add(TextSpan(text: '• ${line.substring(2)}\n',
            style: const TextStyle(color: AppTheme.textPrimary, fontSize: 13)));
      } else {
        final parts = line.split(RegExp(r'(\*\*.+?\*\*|\b1[7-9]\d{8}\b)'));
        for (final part in parts) {
          if (part.startsWith('**') && part.endsWith('**')) {
            spans.add(TextSpan(text: part.substring(2, part.length - 2),
                style: const TextStyle(color: Color(0xFFFFD700), fontWeight: FontWeight.w700)));
          } else if (RegExp(r'^1[7-9]\d{8}$').hasMatch(part)) {
            final ts = int.tryParse(part) ?? 0;
            final d = DateTime.fromMillisecondsSinceEpoch(ts * 1000 + 8 * 3600 * 1000);
            final pad = (int n) => n.toString().padLeft(2, '0');
            spans.add(TextSpan(text: '${d.year}-${pad(d.month)}-${pad(d.day)} ${pad(d.hour)}:${pad(d.minute)}',
                style: const TextStyle(color: Color(0xFFFFD700), fontSize: 12)));
          } else {
            spans.add(TextSpan(text: part, style: const TextStyle(color: AppTheme.textPrimary, fontSize: 13)));
          }
        }
        spans.add(const TextSpan(text: '\n'));
      }
    }
    return RichText(text: TextSpan(children: spans));
  }
}
