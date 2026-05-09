class AlertRule {
  final int id;
  final String symbol;
  final String type;
  final double value;
  final bool enabled;
  final String createdAt;
  final String? lastTriggeredAt;
  AlertRule({required this.id, required this.symbol, required this.type, required this.value, required this.enabled, required this.createdAt, this.lastTriggeredAt});
  factory AlertRule.fromJson(Map<String, dynamic> json) => AlertRule(
        id: json['id'] as int,
        symbol: json['symbol'] as String,
        type: json['type'] as String,
        value: (json['value'] as num).toDouble(),
        enabled: json['enabled'] as bool,
        createdAt: json['createdAt'] as String,
        lastTriggeredAt: json['lastTriggeredAt'] as String?,
      );
}

class AlertLog {
  final int id;
  final int alertId;
  final String symbol;
  final double price;
  final String message;
  final String triggeredAt;
  AlertLog({required this.id, required this.alertId, required this.symbol, required this.price, required this.message, required this.triggeredAt});
  factory AlertLog.fromJson(Map<String, dynamic> json) => AlertLog(
        id: json['id'] as int,
        alertId: json['alertId'] as int,
        symbol: json['symbol'] as String,
        price: (json['price'] as num).toDouble(),
        message: json['message'] as String,
        triggeredAt: json['triggeredAt'] as String,
      );
}
