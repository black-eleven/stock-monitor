import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../data/ws/ws_client.dart';
import '../../domain/model/alert.dart';
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
  List<AlertRule>? _alerts;
  final List<AlertLog> _logs = [];

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
    final results = await Future.wait([
      ref.read(holdingApiProvider).getAll(),
      ref.read(alertApiProvider).getAll(),
    ]);
    setState(() {
      _holdings = results[0] as List<Holding>;
      _alerts = results[1] as List<AlertRule>;
    });
  }

  List<AlertRule> _alertsFor(String symbol) =>
      _alerts?.where((a) => a.symbol == symbol).toList() ?? [];

  Future<void> _addHolding() async {
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

    if (ok == true && symbolCtrl.text.isNotEmpty) {
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

  Future<void> _addAlert(String symbol) async {
    String type = 'above';
    final valueCtrl = TextEditingController();

    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(builder: (ctx, setD) => AlertDialog(
        title: Text('添加预警 — $symbol'),
        content: Column(mainAxisSize: MainAxisSize.min, children: [
          DropdownButtonFormField<String>(
            value: type,
            items: const [
              DropdownMenuItem(value: 'above', child: Text('涨破')),
              DropdownMenuItem(value: 'below', child: Text('跌破')),
              DropdownMenuItem(value: 'change_pct', child: Text('涨跌幅达 %')),
            ],
            onChanged: (v) => setD(() => type = v!),
          ),
          TextField(controller: valueCtrl, decoration: const InputDecoration(hintText: '阈值'), keyboardType: TextInputType.number),
        ]),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('添加')),
        ],
      )),
    );

    if (ok == true && valueCtrl.text.isNotEmpty) {
      try {
        await ref.read(alertApiProvider).add(symbol, type, double.parse(valueCtrl.text));
        _load();
      } catch (e) {
        if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('添加失败: $e')));
      }
    }
  }

  String _typeLabel(String t) => {'above': '涨破', 'below': '跌破', 'change_pct': '涨跌幅达 %'}[t] ?? t;

  void _navigateToStock(String symbol) {
    Navigator.of(context).pushNamed('/stock-detail', arguments: {'symbol': symbol});
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
      appBar: AppBar(title: const Text('持仓'), actions: [IconButton(onPressed: _addHolding, icon: const Icon(Icons.add))]),
      body: _holdings!.isEmpty
          ? const Center(child: Text('暂无持仓', style: TextStyle(color: AppTheme.textSecondary)))
          : Column(children: [
              Container(
                padding: const EdgeInsets.all(12),
                color: AppTheme.surface,
                child: Row(mainAxisAlignment: MainAxisAlignment.spaceAround, children: [
                  _statBox('总成本', formatPrice(totalCost)),
                  _statBox('总市值', formatPrice(totalValue)),
                  _statBox('总盈亏', formatPrice(totalPL), color: totalPL >= 0 ? AppTheme.up : AppTheme.down),
                  _statBox('盈亏率', '${totalPLPct.toStringAsFixed(2)}%', color: totalPLPct >= 0 ? AppTheme.up : AppTheme.down),
                ]),
              ),
              Expanded(
                flex: 3,
                child: ListView.builder(
                  itemCount: _holdings!.length,
                  itemBuilder: (_, i) => _buildHoldingCard(_holdings![i], quotes),
                ),
              ),
              const Divider(height: 1),
              const Padding(
                padding: EdgeInsets.all(8),
                child: Text('预警触发记录', style: TextStyle(fontWeight: FontWeight.w600, color: AppTheme.textSecondary)),
              ),
              Expanded(
                flex: 2,
                child: _logs.isEmpty
                    ? const Center(child: Text('暂无触发记录', style: TextStyle(color: AppTheme.textSecondary)))
                    : ListView.builder(
                        itemCount: _logs.length,
                        itemBuilder: (_, i) => ListTile(
                          dense: true,
                          title: Text(_logs[i].message, style: const TextStyle(fontSize: 13)),
                          trailing: Text(
                            _logs[i].triggeredAt.length >= 19 ? _logs[i].triggeredAt.substring(11, 19) : '',
                            style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary),
                          ),
                        ),
                      ),
              ),
            ]),
    );
  }

  Widget _statBox(String label, String value, {Color? color}) => Column(children: [
    Text(label, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
    Text(value, style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700, color: color ?? AppTheme.textPrimary)),
  ]);

  Widget _buildHoldingCard(Holding h, Map quotes) {
    final q = quotes[h.symbol];
    final mv = q != null ? h.shares * q.price : 0.0;
    final pl = mv - h.shares * h.avgCost;
    final plPct = h.avgCost > 0 ? ((q?.price ?? 0) - h.avgCost) / h.avgCost * 100 : 0;
    final holdingAlerts = _alertsFor(h.symbol);

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Expanded(
              child: GestureDetector(
                onTap: () => _navigateToStock(h.symbol),
                child: Text('${h.name} (${shortCode(h.symbol)})', style: const TextStyle(fontWeight: FontWeight.w600, color: AppTheme.accent)),
              ),
            ),
            Column(crossAxisAlignment: CrossAxisAlignment.end, children: [
              Text(formatPrice(pl), style: TextStyle(fontWeight: FontWeight.w700, color: pl >= 0 ? AppTheme.up : AppTheme.down)),
              Text('${plPct.toStringAsFixed(2)}%', style: TextStyle(fontSize: 12, color: plPct >= 0 ? AppTheme.up : AppTheme.down)),
            ]),
          ]),
          Text('${h.shares.toStringAsFixed(0)}股 × ${formatPrice(h.avgCost)}', style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
          // Alert rules for this holding
          if (holdingAlerts.isNotEmpty) ...[
            const SizedBox(height: 6),
            ...holdingAlerts.map((a) => Row(children: [
              GestureDetector(
                onTap: () async {
                  await ref.read(alertApiProvider).update(a.id, {'enabled': !a.enabled});
                  _load();
                },
                child: Text('${_typeLabel(a.type)} ${a.value}',
                    style: TextStyle(fontSize: 12, color: a.enabled ? AppTheme.up : AppTheme.textSecondary)),
              ),
              GestureDetector(
                onTap: () async {
                  await ref.read(alertApiProvider).remove(a.id);
                  _load();
                },
                child: const Padding(
                  padding: EdgeInsets.only(left: 8),
                  child: Icon(Icons.close, size: 14, color: AppTheme.down),
                ),
              ),
            ])),
          ],
          const SizedBox(height: 4),
          Row(children: [
            GestureDetector(
              onTap: () => _addAlert(h.symbol),
              child: const Text('+ 添加预警', style: TextStyle(fontSize: 12, color: AppTheme.accent)),
            ),
            const Spacer(),
            GestureDetector(
              onLongPress: () async {
                await ref.read(holdingApiProvider).remove(h.symbol);
                _load();
              },
              child: const Text('长按删除', style: TextStyle(fontSize: 11, color: AppTheme.textSecondary)),
            ),
          ]),
        ]),
      ),
    );
  }
}
