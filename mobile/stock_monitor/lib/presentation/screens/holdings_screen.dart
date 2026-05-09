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
    final list = await ref.read(holdingApiProvider).getAll();
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
      appBar: AppBar(title: const Text('持仓'), actions: [IconButton(onPressed: _add, icon: const Icon(Icons.add))]),
      body: _holdings!.isEmpty
          ? const Center(child: Text('暂无持仓', style: TextStyle(color: AppTheme.textSecondary)))
          : Column(children: [
              Container(
                padding: const EdgeInsets.all(12),
                color: AppTheme.surface,
                child: Row(mainAxisAlignment: MainAxisAlignment.spaceAround, children: [
                  _box('总成本', formatPrice(totalCost)),
                  _box('总市值', formatPrice(totalValue)),
                  _box('总盈亏', formatPrice(totalPL), color: totalPL >= 0 ? AppTheme.up : AppTheme.down),
                  _box('盈亏率', '${totalPLPct.toStringAsFixed(2)}%', color: totalPLPct >= 0 ? AppTheme.up : AppTheme.down),
                ]),
              ),
              Expanded(
                child: ListView.builder(
                  itemCount: _holdings!.length,
                  itemBuilder: (_, i) {
                    final h = _holdings![i];
                    final q = quotes[h.symbol];
                    final mv = q != null ? h.shares * q.price : 0.0;
                    final pl = mv - h.shares * h.avgCost;
                    final plPct = h.avgCost > 0 ? ((q?.price ?? 0) - h.avgCost) / h.avgCost * 100 : 0;
                    return Card(
                      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                      child: ListTile(
                        title: Text('${h.name} (${shortCode(h.symbol)})', style: const TextStyle(fontWeight: FontWeight.w600)),
                        subtitle: Text('${h.shares.toStringAsFixed(0)}股 × ${formatPrice(h.avgCost)}'),
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
            ]),
    );
  }

  Widget _box(String label, String value, {Color? color}) => Column(children: [
        Text(label, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
        Text(value, style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700, color: color ?? AppTheme.textPrimary)),
      ]);
}
