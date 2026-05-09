import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../data/api/api_client.dart';
import '../../data/api/watchlist_api.dart';
import '../../data/api/alert_api.dart';
import '../../data/api/holding_api.dart';
import '../../data/api/quote_api.dart';
import '../../data/ws/ws_client.dart';

final apiClientProvider = Provider((ref) => ApiClient());
final watchlistApiProvider = Provider((ref) => WatchlistApi(ref.watch(apiClientProvider)));
final alertApiProvider = Provider((ref) => AlertApi(ref.watch(apiClientProvider)));
final holdingApiProvider = Provider((ref) => HoldingApi(ref.watch(apiClientProvider)));
final quoteApiProvider = Provider((ref) => QuoteApi(ref.watch(apiClientProvider)));

final wsClientProvider = Provider<WsClient>((ref) {
  final ws = WsClient();
  ref.onDispose(() => ws.dispose());
  return ws;
});
