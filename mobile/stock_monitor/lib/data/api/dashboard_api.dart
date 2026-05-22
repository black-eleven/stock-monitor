import 'api_client.dart';

class DashboardApi {
  final ApiClient _client;
  DashboardApi(this._client);

  Future<Map<String, dynamic>> get() async {
    final res = await _client.get('/dashboard');
    return Map<String, dynamic>.from(res.data as Map);
  }
}
