class WatchlistItem {
  final String symbol;
  final String name;
  final String addedAt;
  WatchlistItem({required this.symbol, required this.name, required this.addedAt});
  factory WatchlistItem.fromJson(Map<String, dynamic> json) => WatchlistItem(
        symbol: json['symbol'] as String,
        name: json['name'] as String,
        addedAt: json['addedAt'] as String,
      );
}

class Quote {
  final String code;
  final double price;
  final double yp;
  final double open;
  final double high;
  final double low;
  final double volume;
  final double turnover;
  final int timestamp;
  final String status;
  Quote({
    required this.code, required this.price, required this.yp,
    required this.open, required this.high, required this.low,
    required this.volume, required this.turnover, required this.timestamp,
    required this.status,
  });
  factory Quote.fromJson(Map<String, dynamic> json) => Quote(
        code: json['code'] as String,
        price: (json['price'] as num).toDouble(),
        yp: (json['yp'] as num).toDouble(),
        open: (json['open'] as num).toDouble(),
        high: (json['high'] as num).toDouble(),
        low: (json['low'] as num).toDouble(),
        volume: (json['volume'] as num).toDouble(),
        turnover: (json['turnover'] as num).toDouble(),
        timestamp: json['timestamp'] as int,
        status: json['status'] as String,
      );
}
