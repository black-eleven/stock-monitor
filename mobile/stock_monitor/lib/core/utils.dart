String formatPrice(double? price) {
  if (price == null) return '--';
  return price.toStringAsFixed(2);
}

String formatVolume(double v) {
  if (v >= 100000000) return '${(v / 100000000).toStringAsFixed(2)}亿';
  if (v >= 10000) return '${(v / 10000).toStringAsFixed(0)}万';
  return v.toStringAsFixed(0);
}

String shortCode(String code) {
  return code.replaceFirst(RegExp(r'^(HK|SH|SZ|US):'), '');
}

double calcChangePct(double price, double yp) {
  if (yp == 0) return 0;
  return (price - yp) / yp * 100;
}

String formatChange(double price, double yp) {
  final pct = calcChangePct(price, yp);
  final sign = pct >= 0 ? '+' : '';
  return '$sign${pct.toStringAsFixed(2)}%';
}

String changeDir(double price, double yp) => price >= yp ? 'up' : 'down';
