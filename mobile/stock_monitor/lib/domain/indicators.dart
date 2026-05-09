import 'model/kline.dart';

class MA {
  final int time;
  final double value;
  MA(this.time, this.value);
}

List<MA> calcMA(List<Bar> bars, int period) {
  if (bars.length < period) return [];
  final result = <MA>[];
  double sum = 0;
  for (int i = 0; i < bars.length; i++) {
    sum += bars[i].close;
    if (i >= period) sum -= bars[i - period].close;
    if (i >= period - 1) result.add(MA(bars[i].time, sum / period));
  }
  return result;
}

class RSIVal {
  final int time;
  final double value;
  RSIVal(this.time, this.value);
}

List<RSIVal> calcRSI(List<Bar> bars, int period) {
  if (bars.length <= period) return [];
  final result = <RSIVal>[];
  double avgGain = 0, avgLoss = 0;
  for (int i = 1; i <= period; i++) {
    final diff = bars[i].close - bars[i - 1].close;
    if (diff > 0) avgGain += diff; else avgLoss -= diff;
  }
  avgGain /= period;
  avgLoss /= period;
  for (int i = period; i < bars.length; i++) {
    if (avgLoss == 0) {
      result.add(RSIVal(bars[i].time, 100));
    } else {
      result.add(RSIVal(bars[i].time, 100 - 100 / (1 + avgGain / avgLoss)));
    }
    final diff = bars[i].close - bars[i - 1].close;
    final gain = diff > 0 ? diff : 0.0;
    final loss = diff < 0 ? -diff : 0.0;
    avgGain = (avgGain * (period - 1) + gain) / period;
    avgLoss = (avgLoss * (period - 1) + loss) / period;
  }
  return result;
}

class MACDResult {
  final List<MA> dif, dea, macd;
  MACDResult(this.dif, this.dea, this.macd);
}

double _ema(List<double> values, int period, int i) {
  final k = 2.0 / (period + 1);
  double ema = values[0];
  for (int j = 1; j <= i; j++) {
    ema = values[j] * k + ema * (1 - k);
  }
  return ema;
}

MACDResult? calcMACD(List<Bar> bars, {int fast = 12, int slow = 26, int signal = 9}) {
  if (bars.length < slow) return null;
  final closes = bars.map((b) => b.close).toList();
  final dif = <MA>[];
  for (int i = slow - 1; i < bars.length; i++) {
    dif.add(MA(bars[i].time, _ema(closes.sublist(0, i + 1), fast, i) - _ema(closes.sublist(0, i + 1), slow, i)));
  }
  final dea = <MA>[];
  final macd = <MA>[];
  final difValues = dif.map((d) => d.value).toList();
  for (int i = signal - 1; i < dif.length; i++) {
    final deaVal = _ema(difValues.sublist(0, i + 1), signal, i);
    dea.add(MA(dif[i].time, deaVal));
    macd.add(MA(dif[i].time, (dif[i].value - deaVal) * 2));
  }
  return MACDResult(dif, dea, macd);
}

class SignalResult {
  final double score;
  final double maxScore;
  final int count;
  final String summary;
  SignalResult({required this.score, required this.maxScore, required this.count, required this.summary});
}

SignalResult evaluateSignals(List<Bar> bars) {
  final ma5 = calcMA(bars, 5);
  final ma20 = calcMA(bars, 20);
  final ma60 = calcMA(bars, 60);
  final rsi = calcRSI(bars, 14);
  final macd = calcMACD(bars);
  double score = 0;
  int count = 0;
  const maxScore = 13.0;

  if (ma5.length >= 2 && ma20.length >= 2) {
    final prevMA5 = ma5[ma5.length - 2].value;
    final prevMA20 = ma20[ma20.length - 2].value;
    final currMA5 = ma5.last.value;
    final currMA20 = ma20.last.value;
    if (prevMA5 >= prevMA20 && currMA5 < currMA20) { score += 2.0; count++; }
    if (bars.last.close < currMA20) { score += 1.0; count++; }
  }
  if (ma60.isNotEmpty && bars.last.close < ma60.last.value) { score += 0.5; count++; }

  if (rsi.isNotEmpty) {
    if (rsi.last.value > 70) { score += 1.0; count++; }
    if (rsi.length >= 20 && bars.length >= 20) {
      final recentRSI = rsi.sublist(rsi.length - 20);
      final recentBars = bars.sublist(bars.length - 20);
      double maxPrice = 0, maxRsi = 0;
      int pricePeak = 0, rsiPeak = 0;
      for (int i = 0; i < 20; i++) {
        if (recentBars[i].high > maxPrice) { maxPrice = recentBars[i].high; pricePeak = i; }
        if (recentRSI[i].value > maxRsi) { maxRsi = recentRSI[i].value; rsiPeak = i; }
      }
      if (pricePeak > rsiPeak && recentBars.last.high >= maxPrice * 0.98) { score += 2.5; count++; }
    }
  }

  if (macd != null && macd.dea.length >= 2 && macd.dif.length >= 2) {
    final prevDIF = macd.dif[macd.dif.length - 2].value;
    final prevDEA = macd.dea[macd.dea.length - 2].value;
    final currDIF = macd.dif.last.value;
    final currDEA = macd.dea.last.value;
    if (prevDIF >= prevDEA && currDIF < currDEA) { score += 2.0; count++; }
  }

  final pct = score / maxScore;
  String summary;
  if (pct >= 0.5) summary = '强烈卖出信号';
  else if (pct >= 0.25) summary = '偏弱，注意风险';
  else if (pct > 0) summary = '短期偏弱';
  else summary = '正常';

  return SignalResult(score: score, maxScore: maxScore, count: count, summary: summary);
}
