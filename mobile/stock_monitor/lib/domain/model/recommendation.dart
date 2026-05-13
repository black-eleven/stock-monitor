class Recommendation {
  final String symbol;
  final String name;
  final double score;
  final int newsCount;
  final double price;
  final double changePercent;
  final List<String> highlights;
  final int rank;

  Recommendation({
    required this.symbol,
    required this.name,
    required this.score,
    required this.newsCount,
    required this.price,
    required this.changePercent,
    required this.highlights,
    required this.rank,
  });

  factory Recommendation.fromJson(Map<String, dynamic> json) =>
      Recommendation(
        symbol: json['symbol'] as String,
        name: json['name'] as String,
        score: (json['score'] as num).toDouble(),
        newsCount: json['newsCount'] as int,
        price: (json['price'] as num).toDouble(),
        changePercent: (json['changePercent'] as num).toDouble(),
        highlights: (json['highlights'] as List).cast<String>(),
        rank: json['rank'] as int,
      );
}
