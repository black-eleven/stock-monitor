import 'package:flutter/material.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/indicators.dart';
import '../../domain/model/stock.dart';

class StockCard extends StatelessWidget {
  final WatchlistItem item;
  final Quote? quote;
  final SignalResult? buySignal;
  final SignalResult? sellSignal;
  final VoidCallback onTap;
  final VoidCallback onDelete;

  const StockCard({
    super.key,
    required this.item,
    this.quote,
    this.buySignal,
    this.sellSignal,
    required this.onTap,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: ListTile(
        onTap: onTap,
        title: Row(
          children: [
            Expanded(
              child: Text(item.name,
                  style: const TextStyle(fontWeight: FontWeight.w600, color: AppTheme.textPrimary)),
            ),
            if (buySignal != null || sellSignal != null) _buildSignalBadge(),
          ],
        ),
        subtitle: Text(shortCode(item.symbol),
            style: const TextStyle(color: AppTheme.textSecondary)),
        trailing: quote != null
            ? Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(formatPrice(quote!.price),
                      style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w700,
                          color: changeDir(quote!.price, quote!.yp) == 'up'
                              ? AppTheme.up
                              : AppTheme.down)),
                  Text(formatChange(quote!.price, quote!.yp),
                      style: TextStyle(
                          fontSize: 13,
                          color: changeDir(quote!.price, quote!.yp) == 'up'
                              ? AppTheme.up
                              : AppTheme.down)),
                ],
              )
            : const Icon(Icons.chevron_right, color: AppTheme.textSecondary),
      ),
    );
  }

  Widget _buildSignalBadge() {
    final buyPct =
        buySignal != null ? (buySignal!.score / buySignal!.maxScore * 100) : 0.0;
    final sellPct =
        sellSignal != null ? (sellSignal!.score / sellSignal!.maxScore * 100) : 0.0;

    final hasBuy = buyPct >= 25;
    final hasSell = sellPct >= 25;

    Color color;
    String text;
    if (hasBuy && buyPct >= 50) {
      color = AppTheme.up;
      text = 'B ${buyPct.toStringAsFixed(0)}';
    } else if (hasSell && sellPct >= 50) {
      color = AppTheme.down;
      text = 'S ${sellPct.toStringAsFixed(0)}';
    } else if (hasBuy) {
      color = Colors.orange;
      text = 'B ${buyPct.toStringAsFixed(0)}';
    } else if (hasSell) {
      color = Colors.orange;
      text = 'S ${sellPct.toStringAsFixed(0)}';
    } else {
      return const SizedBox.shrink();
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withAlpha(30),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withAlpha(80), width: 0.5),
      ),
      child: Text(text,
          style: TextStyle(fontSize: 11, fontWeight: FontWeight.w700, color: color)),
    );
  }
}

class StockDetailSheet extends StatelessWidget {
  final WatchlistItem item;
  final Quote quote;
  final SignalResult? buySignal;
  final SignalResult? sellSignal;
  final VoidCallback onDelete;

  const StockDetailSheet({
    super.key,
    required this.item,
    required this.quote,
    this.buySignal,
    this.sellSignal,
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
            Expanded(
                child: Text(item.name,
                    style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w700))),
            IconButton(
                onPressed: () {
                  Navigator.pop(context);
                  onDelete();
                },
                icon: const Icon(Icons.delete_outline, color: AppTheme.down)),
          ]),
          const SizedBox(height: 8),
          Row(children: [
            Text(formatPrice(quote.price),
                style: TextStyle(
                    fontSize: 32,
                    fontWeight: FontWeight.w800,
                    color: changeDir(quote.price, quote.yp) == 'up'
                        ? AppTheme.up
                        : AppTheme.down)),
            const SizedBox(width: 12),
            Text(formatChange(quote.price, quote.yp),
                style: TextStyle(
                    fontSize: 18,
                    color: changeDir(quote.price, quote.yp) == 'up'
                        ? AppTheme.up
                        : AppTheme.down)),
          ]),
          const SizedBox(height: 20),
          _row('今开', formatPrice(quote.open)),
          _row('最高', formatPrice(quote.high)),
          _row('最低', formatPrice(quote.low)),
          _row('昨收', formatPrice(quote.yp)),
          _row('成交量', formatVolume(quote.volume)),
          _row('成交额', formatVolume(quote.turnover)),
          if (buySignal != null || sellSignal != null) ...[
            const SizedBox(height: 16),
            const Divider(color: AppTheme.border),
            const SizedBox(height: 8),
            if (buySignal != null) _buildSignalSummary(buySignal!, true),
            if (sellSignal != null) ...[
              const SizedBox(height: 8),
              _buildSignalSummary(sellSignal!, false),
            ],
          ],
        ],
      ),
    );
  }

  Widget _buildSignalSummary(SignalResult signal, bool isBuy) {
    final pct = signal.score / signal.maxScore;
    final color = pct >= 0.5
        ? (isBuy ? AppTheme.up : AppTheme.down)
        : pct > 0
            ? Colors.orange
            : AppTheme.textSecondary;
    return Row(children: [
      Icon(isBuy ? Icons.trending_up : Icons.trending_down, size: 16, color: color),
      const SizedBox(width: 8),
      Text(isBuy ? '买入信号' : '卖出信号',
          style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600)),
      const SizedBox(width: 8),
      Text('${(pct * 100).toStringAsFixed(0)}分',
          style: TextStyle(fontSize: 13, fontWeight: FontWeight.w700, color: color)),
      const SizedBox(width: 4),
      Text('· ${signal.summary}',
          style: TextStyle(fontSize: 12, color: color.withAlpha(200))),
      const SizedBox(width: 8),
      Text('${signal.count}/${signal.total}信号',
          style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary)),
    ]);
  }

  Widget _row(String label, String value) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Row(children: [
          SizedBox(
              width: 80,
              child:
                  Text(label, style: const TextStyle(color: AppTheme.textSecondary))),
          Text(value, style: const TextStyle(fontWeight: FontWeight.w500)),
        ]),
      );
}
