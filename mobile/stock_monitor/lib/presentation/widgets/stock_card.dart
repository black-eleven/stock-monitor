import 'package:flutter/material.dart';
import '../../core/theme.dart';
import '../../core/utils.dart';
import '../../domain/model/stock.dart';

class StockCard extends StatelessWidget {
  final WatchlistItem item;
  final Quote? quote;
  final VoidCallback onTap;
  final VoidCallback onDelete;

  const StockCard({super.key, required this.item, this.quote, required this.onTap, required this.onDelete});

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
                  Text(formatPrice(quote!.price), style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700, color: changeDir(quote!.price, quote!.yp) == 'up' ? AppTheme.up : AppTheme.down)),
                  Text(formatChange(quote!.price, quote!.yp), style: TextStyle(fontSize: 13, color: changeDir(quote!.price, quote!.yp) == 'up' ? AppTheme.up : AppTheme.down)),
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

  const StockDetailSheet({super.key, required this.item, required this.quote, required this.onDelete});

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
            Text(formatPrice(quote.price), style: TextStyle(fontSize: 32, fontWeight: FontWeight.w800, color: changeDir(quote.price, quote.yp) == 'up' ? AppTheme.up : AppTheme.down)),
            const SizedBox(width: 12),
            Text(formatChange(quote.price, quote.yp), style: TextStyle(fontSize: 18, color: changeDir(quote.price, quote.yp) == 'up' ? AppTheme.up : AppTheme.down)),
          ]),
          const SizedBox(height: 20),
          _row('今开', formatPrice(quote.open)),
          _row('最高', formatPrice(quote.high)),
          _row('最低', formatPrice(quote.low)),
          _row('昨收', formatPrice(quote.yp)),
          _row('成交量', formatVolume(quote.volume)),
          _row('成交额', formatVolume(quote.turnover)),
        ],
      ),
    );
  }

  Widget _row(String label, String value) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Row(children: [
          SizedBox(width: 80, child: Text(label, style: const TextStyle(color: AppTheme.textSecondary))),
          Text(value, style: const TextStyle(fontWeight: FontWeight.w500)),
        ]),
      );
}
