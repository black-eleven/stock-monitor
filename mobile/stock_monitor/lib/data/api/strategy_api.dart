import '../../domain/model/kline.dart';
import 'api_client.dart';

class StrategyApi {
  final ApiClient _client;
  StrategyApi(this._client);

  Future<List<String>> getStrategies() async {
    final res = await _client.get('/strategy/list');
    final list = res.data['displayNames'] as List;
    return list.map((e) => e.toString()).toList();
  }

  Future<List<String>> getStrategyKeys() async {
    final res = await _client.get('/strategy/list');
    final list = res.data['strategies'] as List;
    return list.map((e) => e.toString()).toList();
  }

  Future<String> analyze(String strategy, String symbol, List<KlineBar> bars) async {
    final barList = bars.map((b) => {
      'ts': b.ts,
      'o': b.o,
      'cl': b.cl,
      'h': b.h,
      'l': b.l,
      'v': b.v,
    }).toList();

    final res = await _client.post('/strategy/analyze', data: {
      'strategy': strategy,
      'symbol': symbol,
      'bars': barList,
    });
    return res.data['analysis'] as String;
  }
}
