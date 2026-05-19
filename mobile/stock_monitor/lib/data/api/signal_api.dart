import 'api_client.dart';

class SignalApi {
  final ApiClient _client;
  SignalApi(this._client);

  Future<void> record({
    required String symbol,
    required double buyScore,
    required int buyPct,
    required double sellScore,
    required int sellPct,
    required int buyCount,
    required int sellCount,
  }) async {
    await _client.post('/signals/record', data: {
      'symbol': symbol,
      'buyScore': buyScore,
      'buyPct': buyPct,
      'sellScore': sellScore,
      'sellPct': sellPct,
      'buyCount': buyCount,
      'sellCount': sellCount,
    });
  }
}
