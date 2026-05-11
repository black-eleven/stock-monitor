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
    final list = await ref.read(alertApiProvider).getAll();
    setState(() => _rules = list);
  }

  Future<void> _add() async {
    final symbolCtrl = TextEditingController();
    String type = 'above';
    final valueCtrl = TextEditingController();

    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => StatefulBuilder(builder: (ctx, setD) => AlertDialog(
            title: const Text('添加提醒'),
            content: Column(mainAxisSize: MainAxisSize.min, children: [
              TextField(controller: symbolCtrl, decoration: const InputDecoration(hintText: '代码')),
              DropdownButtonFormField<String>(
                value: type,
                items: const [DropdownMenuItem(value: 'above', child: Text('涨破')), DropdownMenuItem(value: 'below', child: Text('跌破')), DropdownMenuItem(value: 'change_pct', child: Text('涨跌幅达'))],
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

    if (ok == true && symbolCtrl.text.isNotEmpty && valueCtrl.text.isNotEmpty) {
      try {
        await ref.read(alertApiProvider).add(symbolCtrl.text.toUpperCase(), type, double.parse(valueCtrl.text));
        _load();
      } catch (e) {
        if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('添加失败: $e')));
      }
    }
  }

  String _label(String t) => {'above': '涨破', 'below': '跌破', 'change_pct': '涨跌幅达 %'}[t] ?? t;

  @override
  Widget build(BuildContext context) {
    if (_rules == null) return const Center(child: CircularProgressIndicator());
    return Scaffold(
      appBar: AppBar(title: const Text('提醒'), actions: [IconButton(onPressed: _add, icon: const Icon(Icons.add))]),
      body: Column(children: [
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
                        subtitle: Text('${_label(r.type)} ${r.value}'),
                        trailing: Row(mainAxisSize: MainAxisSize.min, children: [
                          Switch(value: r.enabled, onChanged: (v) async { await ref.read(alertApiProvider).update(r.id, {'enabled': v}); _load(); }),
                          IconButton(icon: const Icon(Icons.delete_outline, color: AppTheme.down, size: 20), onPressed: () async { await ref.read(alertApiProvider).remove(r.id); _load(); }),
                        ]),
                      ),
                    );
                  },
                ),
        ),
        const Divider(height: 1),
        const Padding(padding: EdgeInsets.all(8), child: Text('触发记录', style: TextStyle(fontWeight: FontWeight.w600, color: AppTheme.textSecondary))),
        Expanded(
          flex: 2,
          child: _logs.isEmpty
              ? const Center(child: Text('暂无触发记录', style: TextStyle(color: AppTheme.textSecondary)))
              : ListView.builder(
                  itemCount: _logs.length,
                  itemBuilder: (_, i) => ListTile(dense: true, title: Text(_logs[i].message, style: const TextStyle(fontSize: 13)), trailing: Text(_logs[i].triggeredAt.substring(11, 19), style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary))),
                ),
        ),
      ]),
    );
  }
}
