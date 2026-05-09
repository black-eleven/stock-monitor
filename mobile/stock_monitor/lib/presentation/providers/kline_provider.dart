import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../domain/model/kline.dart';
import 'api_providers.dart';

class KlineState {
  final String symbol;
  final String interval;
  final List<KlineItem> data;
  final bool loading;
  final String? error;
  const KlineState({this.symbol = '', this.interval = '1d', this.data = const [], this.loading = false, this.error});

  KlineState copyWith({String? symbol, String? interval, List<KlineItem>? data, bool? loading, String? error}) =>
      KlineState(symbol: symbol ?? this.symbol, interval: interval ?? this.interval, data: data ?? this.data, loading: loading ?? this.loading, error: error);
}

class KlineNotifier extends StateNotifier<KlineState> {
  final QuoteApi _api;
  KlineNotifier(this._api) : super(const KlineState());

  Future<void> load(String symbol, {String interval = '1d', int count = 200}) async {
    if (state.loading) return;
    state = state.copyWith(symbol: symbol, interval: interval, loading: true, error: null);
    try {
      final data = await _api.getKline(symbol, interval: interval, count: count);
      state = state.copyWith(data: data, loading: false);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }
}

final klineProvider = StateNotifierProvider<KlineNotifier, KlineState>((ref) {
  final api = ref.watch(quoteApiProvider);
  return KlineNotifier(api);
});
