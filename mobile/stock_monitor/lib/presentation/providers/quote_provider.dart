import 'dart:async';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../domain/model/stock.dart';
import 'api_providers.dart';

class QuoteState {
  final Map<String, Quote> quotes;
  const QuoteState(this.quotes);
}

class QuoteNotifier extends StateNotifier<QuoteState> {
  final WsClient _ws;
  StreamSubscription? _quoteSub;
  StreamSubscription? _snapshotSub;

  QuoteNotifier(this._ws) : super(const QuoteState({})) {
    _snapshotSub = _ws.snapshotStream.listen((quotes) {
      final map = {...state.quotes};
      for (final q in quotes) {
        map[q.code] = q;
      }
      state = QuoteState(map);
    });
    _quoteSub = _ws.quoteStream.listen((quote) {
      final map = {...state.quotes};
      map[quote.code] = quote;
      state = QuoteState(map);
    });
  }

  Quote? getQuote(String symbol) => state.quotes[symbol];

  @override
  void dispose() {
    _quoteSub?.cancel();
    _snapshotSub?.cancel();
    super.dispose();
  }
}

final quoteProvider = StateNotifierProvider<QuoteNotifier, QuoteState>((ref) {
  final ws = ref.watch(wsClientProvider);
  return QuoteNotifier(ws);
});
