import 'package:flutter/material.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/indicators.dart';
import '../../domain/model/recommendation.dart';

class RecommendDetailSheet extends StatelessWidget {
  final Recommendation rec;
  final SignalResult? buySignal;
  final SignalResult? sellSignal;
  final VoidCallback onAdd;

  const RecommendDetailSheet({
    super.key,
    required this.rec,
    this.buySignal,
    this.sellSignal,
    required this.onAdd,
  });

  @override
  Widget build(BuildContext context) {
    final changeDir = rec.changePercent >= 0 ? 'up' : 'down';
    final changeColor = changeDir == 'up' ? AppTheme.up : AppTheme.down;

    return DraggableScrollableSheet(
      initialChildSize: 0.75,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      expand: false,
      builder: (_, scrollController) => Container(
        padding: const EdgeInsets.all(20),
        child: ListView(
          controller: scrollController,
          children: [
            // Header
            Row(children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(rec.name.isNotEmpty ? rec.name : rec.symbol,
                        style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w700, color: AppTheme.textPrimary)),
                    const SizedBox(height: 4),
                    Text(rec.symbol, style: const TextStyle(fontSize: 14, color: AppTheme.textSecondary)),
                  ],
                ),
              ),
              if (rec.price > 0)
                Column(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Text(formatPrice(rec.price),
                        style: TextStyle(fontSize: 24, fontWeight: FontWeight.w800, color: changeColor)),
                    Text('${rec.changePercent >= 0 ? '+' : ''}${rec.changePercent.toStringAsFixed(2)}%',
                        style: TextStyle(fontSize: 14, color: changeColor)),
                  ],
                ),
            ]),
            const SizedBox(height: 8),
            Row(children: [
              _metaChip(Icons.auto_awesome, '综合评分 ${(rec.score * 100).toStringAsFixed(0)}'),
              const SizedBox(width: 8),
              _metaChip(Icons.article_outlined, '${rec.newsCount}篇新闻'),
            ]),
            if (rec.highlights.isNotEmpty) ...[
              const SizedBox(height: 12),
              ...rec.highlights.map((h) => Padding(
                    padding: const EdgeInsets.only(bottom: 4),
                    child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      const Text('• ', style: TextStyle(color: AppTheme.textSecondary)),
                      Expanded(child: Text(h, style: const TextStyle(fontSize: 13, color: AppTheme.textSecondary))),
                    ]),
                  )),
            ],

            const SizedBox(height: 16),
            // Add to watchlist button
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: () {
                  onAdd();
                  Navigator.pop(context);
                },
                icon: const Icon(Icons.add),
                label: const Text('加入自选'),
              ),
            ),

            // Buy signals
            if (buySignal != null) ...[
              const SizedBox(height: 20),
              _buildSignalSection('买入信号分析', buySignal!, true),
            ],

            // Sell signals
            if (sellSignal != null) ...[
              const SizedBox(height: 20),
              _buildSignalSection('卖出信号分析', sellSignal!, false),
            ],

            if (buySignal == null && sellSignal == null) ...[
              const SizedBox(height: 20),
              const Center(
                child: Text('技术信号加载中...', style: TextStyle(color: AppTheme.textSecondary)),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _metaChip(IconData icon, String text) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: AppTheme.textSecondary.withAlpha(25),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        Icon(icon, size: 14, color: AppTheme.textSecondary),
        const SizedBox(width: 4),
        Text(text, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
      ]),
    );
  }

  Widget _buildSignalSection(String title, SignalResult signal, bool isBuy) {
    final pct = signal.score / signal.maxScore;
    final color = pct >= 0.5
        ? (isBuy ? AppTheme.up : AppTheme.down)
        : pct > 0
            ? Colors.orange
            : AppTheme.textSecondary;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(children: [
          Text(title, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700, color: AppTheme.textPrimary)),
          const SizedBox(width: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
            decoration: BoxDecoration(
              color: color.withAlpha(40),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text('${(pct * 100).toStringAsFixed(0)}分 · ${signal.summary}',
                style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: color)),
          ),
        ]),
        const SizedBox(height: 8),
        Text('${signal.count} / ${signal.total} 个信号触发',
            style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
        const SizedBox(height: 8),
        ...signal.signals.map((s) => _buildSignalRow(s, isBuy)),
      ],
    );
  }

  Widget _buildSignalRow(SignalInfo s, bool isBuy) {
    Color dotColor;
    if (s.triggered) {
      dotColor = s.status == 'danger' ? (isBuy ? AppTheme.up : AppTheme.down) : Colors.orange;
    } else {
      dotColor = AppTheme.textSecondary.withAlpha(80);
    }

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(children: [
        Icon(Icons.circle, size: 8, color: dotColor),
        const SizedBox(width: 8),
        Expanded(
          child: Text(s.name,
              style: TextStyle(
                  fontSize: 13,
                  color: s.triggered ? AppTheme.textPrimary : AppTheme.textSecondary.withAlpha(120))),
        ),
        if (s.value != null)
          Flexible(
            child: Text(s.value!, style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary), overflow: TextOverflow.ellipsis),
          ),
        const SizedBox(width: 8),
        Text(s.weight.toStringAsFixed(1), style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary)),
      ]),
    );
  }
}
