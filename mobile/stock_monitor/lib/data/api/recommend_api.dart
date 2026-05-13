import '../../domain/model/recommendation.dart';
import 'api_client.dart';

class RecommendApi {
  final ApiClient _client;
  RecommendApi(this._client);

  Future<List<Recommendation>> recommend(String industry) async {
    final res = await _client.post('/recommendations', data: {'industry': industry});
    final list = res.data['recommendations'] as List;
    return list.map((e) => Recommendation.fromJson(e)).toList();
  }
}
