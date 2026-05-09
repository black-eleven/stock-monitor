class Holding {
  final String symbol;
  final String name;
  final double shares;
  final double avgCost;
  final String buyDate;
  Holding({required this.symbol, required this.name, required this.shares, required this.avgCost, required this.buyDate});
  factory Holding.fromJson(Map<String, dynamic> json) => Holding(
        symbol: json['symbol'] as String,
        name: json['name'] as String,
        shares: (json['shares'] as num).toDouble(),
        avgCost: (json['avgCost'] as num).toDouble(),
        buyDate: json['buyDate'] as String,
      );
}
