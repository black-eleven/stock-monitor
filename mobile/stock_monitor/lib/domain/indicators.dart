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

// ---- Divergence helpers ----

bool _detectBearishDivergence(List<Bar> bars, List<MA> indicatorData) {
  final count = [20, bars.length, indicatorData.length].reduce((a, b) => a < b ? a : b);
  if (count < 2) return false;
  final recentBars = bars.sublist(bars.length - count);
  final recentInd = indicatorData.sublist(indicatorData.length - count);

  int maxHighIdx = -1;
  double maxHigh = double.negativeInfinity;
  for (int i = 0; i < recentBars.length; i++) {
    if (recentBars[i].high > maxHigh) { maxHigh = recentBars[i].high; maxHighIdx = i; }
  }

  int maxIndIdx = -1;
  double maxInd = double.negativeInfinity;
  for (int i = 0; i < recentInd.length; i++) {
    if (recentInd[i].value > maxInd) { maxInd = recentInd[i].value; maxIndIdx = i; }
  }

  if (maxHighIdx < 0 || maxIndIdx < 0) return false;
  return maxHighIdx > maxIndIdx && recentInd[maxHighIdx].value < maxInd;
}

bool _detectBullishDivergence(List<Bar> bars, List<MA> indicatorData) {
  final count = [20, bars.length, indicatorData.length].reduce((a, b) => a < b ? a : b);
  if (count < 2) return false;
  final recentBars = bars.sublist(bars.length - count);
  final recentInd = indicatorData.sublist(indicatorData.length - count);

  int minLowIdx = -1;
  double minLow = double.infinity;
  for (int i = 0; i < recentBars.length; i++) {
    if (recentBars[i].low < minLow) { minLow = recentBars[i].low; minLowIdx = i; }
  }

  int minIndIdx = -1;
  double minInd = double.infinity;
  for (int i = 0; i < recentInd.length; i++) {
    if (recentInd[i].value < minInd) { minInd = recentInd[i].value; minIndIdx = i; }
  }

  if (minLowIdx < 0 || minIndIdx < 0) return false;
  return minLowIdx > minIndIdx && recentInd[minLowIdx].value > minInd;
}

bool _detectVolumeDeathCross(List<Bar> bars) {
  final n = [5, bars.length].reduce((a, b) => a < b ? a : b);
  if (n < 3) return false;
  final recent = bars.sublist(bars.length - n);
  double upVol = 0, downVol = 0;
  int upDays = 0, downDays = 0;
  for (final b in recent) {
    if (b.close >= b.open) { upVol += b.close * b.volume; upDays++; }
    else { downVol += b.close * b.volume; downDays++; }
  }
  if (upDays == 0 || downDays == 0) return false;
  return (downVol / downDays) > (upVol / upDays) * 1.2;
}

bool _detectVolumeGoldenCross(List<Bar> bars) {
  final n = [5, bars.length].reduce((a, b) => a < b ? a : b);
  if (n < 3) return false;
  final recent = bars.sublist(bars.length - n);
  double upVol = 0, downVol = 0;
  int upDays = 0, downDays = 0;
  for (final b in recent) {
    if (b.close >= b.open) { upVol += b.close * b.volume; upDays++; }
    else { downVol += b.close * b.volume; downDays++; }
  }
  if (upDays == 0 || downDays == 0) return false;
  return (upVol / upDays) > (downVol / downDays) * 1.2;
}

// ---- Signal detail class ----

class SignalInfo {
  final String key;
  final String name;
  final bool triggered;
  final String? value;
  final String status;
  final double weight;
  SignalInfo({required this.key, required this.name, required this.triggered, this.value, required this.status, required this.weight});
}

class SignalResult {
  final double score;
  final double maxScore;
  final int count;
  final int total;
  final List<SignalInfo> signals;
  final String summary;
  SignalResult({required this.score, required this.maxScore, required this.count, required this.total, required this.signals, required this.summary});
}

// ---- Sell signal evaluation (8 signals, maxScore=13.0) ----

SignalResult evaluateSignals(List<Bar> bars) {
  final ma5 = calcMA(bars, 5);
  final ma20 = calcMA(bars, 20);
  final ma60 = calcMA(bars, 60);
  final rsi = calcRSI(bars, 14);
  final macd = calcMACD(bars);

  const W = {
    'ma_cross': 2.0, 'price_ma20': 1.0, 'price_ma60': 0.5,
    'rsi_overbought': 1.0, 'rsi_diverge': 2.5, 'macd_cross': 2.0,
    'macd_diverge': 2.5, 'volume_death': 1.5,
  };
  const maxScore = 13.0;
  final signals = <SignalInfo>[];
  double score = 0;
  int count = 0;

  final latestBar = bars.last;
  final latestMA5 = ma5.isNotEmpty ? ma5.last : null;
  final prevMA5 = ma5.length >= 2 ? ma5[ma5.length - 2] : null;
  final latestMA20 = ma20.isNotEmpty ? ma20.last : null;
  final prevMA20 = ma20.length >= 2 ? ma20[ma20.length - 2] : null;
  final latestMA60 = ma60.isNotEmpty ? ma60.last : null;

  // 1. MA5 dead cross MA20
  bool t1 = false; String? v1; String s1 = 'ok';
  if (prevMA5 != null && prevMA20 != null && latestMA5 != null && latestMA20 != null) {
    if (prevMA5.value >= prevMA20.value && latestMA5.value < latestMA20.value) {
      t1 = true; s1 = 'danger';
      v1 = 'MA5=${latestMA5.value.toStringAsFixed(2)}, MA20=${latestMA20.value.toStringAsFixed(2)}';
      score += W['ma_cross']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'ma_cross_dead', name: 'MA5死叉MA20', triggered: t1, value: v1, status: s1, weight: W['ma_cross']!));

  // 2. Close < MA20
  bool t2 = false; String? v2; String s2 = 'ok';
  if (latestMA20 != null && latestBar.close < latestMA20.value) {
    t2 = true; s2 = 'warn';
    v2 = '收盘=${latestBar.close.toStringAsFixed(2)}, MA20=${latestMA20.value.toStringAsFixed(2)}';
    score += W['price_ma20']!; count++;
  }
  signals.add(SignalInfo(key: 'price_below_ma20', name: '收盘价低于MA20', triggered: t2, value: v2, status: s2, weight: W['price_ma20']!));

  // 3. Close < MA60
  bool t3 = false; String? v3; String s3 = 'ok';
  if (latestMA60 != null && latestBar.close < latestMA60.value) {
    t3 = true; s3 = 'warn';
    v3 = '收盘=${latestBar.close.toStringAsFixed(2)}, MA60=${latestMA60.value.toStringAsFixed(2)}';
    score += W['price_ma60']!; count++;
  }
  signals.add(SignalInfo(key: 'price_below_ma60', name: '收盘价低于MA60', triggered: t3, value: v3, status: s3, weight: W['price_ma60']!));

  // 4. RSI > 70
  bool t4 = false; String? v4; String s4 = 'ok';
  if (rsi.isNotEmpty && rsi.last.value > 70) {
    t4 = true;
    s4 = rsi.last.value > 80 ? 'danger' : 'warn';
    v4 = 'RSI=${rsi.last.value.toStringAsFixed(2)}';
    score += W['rsi_overbought']!; count++;
  }
  signals.add(SignalInfo(key: 'rsi_overbought', name: 'RSI超买', triggered: t4, value: v4, status: s4, weight: W['rsi_overbought']!));

  // 5. RSI bearish divergence
  bool t5 = false; String? v5; String s5 = 'ok';
  if (rsi.length >= 2 && _detectBearishDivergence(bars, rsi.map((r) => MA(r.time, r.value)).toList())) {
    t5 = true; s5 = 'danger'; v5 = '价格创新高，RSI未创新高';
    score += W['rsi_diverge']!; count++;
  }
  signals.add(SignalInfo(key: 'rsi_divergence', name: 'RSI顶背离', triggered: t5, value: v5, status: s5, weight: W['rsi_diverge']!));

  // 6. MACD dead cross
  bool t6 = false; String? v6; String s6 = 'ok';
  if (macd != null && macd.dif.length >= 2 && macd.dea.length >= 2) {
    final prevDIF = macd.dif[macd.dif.length - 2].value;
    final prevDEA = macd.dea[macd.dea.length - 2].value;
    final currDIF = macd.dif.last.value;
    final currDEA = macd.dea.last.value;
    if (prevDIF >= prevDEA && currDIF < currDEA) {
      t6 = true; s6 = 'danger';
      v6 = 'DIF=${currDIF.toStringAsFixed(4)}, DEA=${currDEA.toStringAsFixed(4)}';
      score += W['macd_cross']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'macd_dead_cross', name: 'MACD死叉', triggered: t6, value: v6, status: s6, weight: W['macd_cross']!));

  // 7. MACD bearish divergence
  bool t7 = false; String? v7; String s7 = 'ok';
  if (macd != null && macd.dif.length >= 2 && _detectBearishDivergence(bars, macd.dif)) {
    t7 = true; s7 = 'danger'; v7 = '价格创新高，DIF未创新高';
    score += W['macd_diverge']!; count++;
  }
  signals.add(SignalInfo(key: 'macd_divergence', name: 'MACD顶背离', triggered: t7, value: v7, status: s7, weight: W['macd_diverge']!));

  // 8. Volume death cross
  bool t8 = false; String? v8; String s8 = 'ok';
  if (bars.length >= 3 && _detectVolumeDeathCross(bars)) {
    t8 = true; s8 = 'warn'; v8 = '近5日下跌日平均成交量 > 上涨日1.2倍';
    score += W['volume_death']!; count++;
  }
  signals.add(SignalInfo(key: 'volume_death_cross', name: '成交量死叉（下跌放量）', triggered: t8, value: v8, status: s8, weight: W['volume_death']!));

  final pct = score / maxScore;
  String summary;
  if (pct >= 0.5) summary = '强烈卖出信号';
  else if (pct >= 0.25) summary = '偏弱，注意风险';
  else if (pct > 0) summary = '短期偏弱';
  else summary = '正常';

  return SignalResult(score: score, maxScore: maxScore, count: count, total: 8, signals: signals, summary: summary);
}

// ---- Buy signal evaluation (10 signals, maxScore=17.0) ----

SignalResult evaluateBuySignals(List<Bar> bars) {
  final ma5 = calcMA(bars, 5);
  final ma20 = calcMA(bars, 20);
  final ma60 = calcMA(bars, 60);
  final rsi = calcRSI(bars, 14);
  final macd = calcMACD(bars);

  const W = {
    'ma_cross_golden': 2.0, 'price_ma20': 1.0, 'price_ma60': 0.5,
    'rsi_oversold': 1.0, 'rsi_diverge': 2.5, 'macd_cross_golden': 2.0,
    'macd_diverge': 2.5, 'volume_golden': 1.5,
    'vol_breakout': 2.0, 'bull_alignment': 2.0,
  };
  const maxScore = 17.0;
  final signals = <SignalInfo>[];
  double score = 0;
  int count = 0;

  final latestBar = bars.last;
  final latestMA5 = ma5.isNotEmpty ? ma5.last : null;
  final prevMA5 = ma5.length >= 2 ? ma5[ma5.length - 2] : null;
  final latestMA20 = ma20.isNotEmpty ? ma20.last : null;
  final prevMA20 = ma20.length >= 2 ? ma20[ma20.length - 2] : null;
  final latestMA60 = ma60.isNotEmpty ? ma60.last : null;

  // 1. MA5 golden cross MA20
  bool t1 = false; String? v1; String st1 = 'ok';
  if (prevMA5 != null && prevMA20 != null && latestMA5 != null && latestMA20 != null) {
    if (prevMA5.value <= prevMA20.value && latestMA5.value > latestMA20.value) {
      t1 = true; st1 = 'danger';
      v1 = 'MA5=${latestMA5.value.toStringAsFixed(2)}, MA20=${latestMA20.value.toStringAsFixed(2)}';
      score += W['ma_cross_golden']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'ma_cross_golden', name: 'MA5金叉MA20', triggered: t1, value: v1, status: st1, weight: W['ma_cross_golden']!));

  // 2. Close > MA20
  bool t2 = false; String? v2; String st2 = 'ok';
  if (latestMA20 != null && latestBar.close > latestMA20.value) {
    t2 = true; st2 = 'warn';
    v2 = '收盘=${latestBar.close.toStringAsFixed(2)}, MA20=${latestMA20.value.toStringAsFixed(2)}';
    score += W['price_ma20']!; count++;
  }
  signals.add(SignalInfo(key: 'price_above_ma20', name: '收盘价高于MA20', triggered: t2, value: v2, status: st2, weight: W['price_ma20']!));

  // 3. Close > MA60
  bool t3 = false; String? v3; String st3 = 'ok';
  if (latestMA60 != null && latestBar.close > latestMA60.value) {
    t3 = true; st3 = 'warn';
    v3 = '收盘=${latestBar.close.toStringAsFixed(2)}, MA60=${latestMA60.value.toStringAsFixed(2)}';
    score += W['price_ma60']!; count++;
  }
  signals.add(SignalInfo(key: 'price_above_ma60', name: '收盘价高于MA60', triggered: t3, value: v3, status: st3, weight: W['price_ma60']!));

  // 4. RSI < 30
  bool t4 = false; String? v4; String st4 = 'ok';
  if (rsi.isNotEmpty && rsi.last.value < 30) {
    t4 = true;
    st4 = rsi.last.value < 20 ? 'danger' : 'warn';
    v4 = 'RSI=${rsi.last.value.toStringAsFixed(2)}';
    score += W['rsi_oversold']!; count++;
  }
  signals.add(SignalInfo(key: 'rsi_oversold', name: 'RSI超卖', triggered: t4, value: v4, status: st4, weight: W['rsi_oversold']!));

  // 5. RSI bullish divergence
  bool t5 = false; String? v5; String st5 = 'ok';
  if (rsi.length >= 2 && _detectBullishDivergence(bars, rsi.map((r) => MA(r.time, r.value)).toList())) {
    t5 = true; st5 = 'danger'; v5 = '价格创新低，RSI未创新低';
    score += W['rsi_diverge']!; count++;
  }
  signals.add(SignalInfo(key: 'rsi_bullish_divergence', name: 'RSI底背离', triggered: t5, value: v5, status: st5, weight: W['rsi_diverge']!));

  // 6. MACD golden cross
  bool t6 = false; String? v6; String st6 = 'ok';
  if (macd != null && macd.dif.length >= 2 && macd.dea.length >= 2) {
    final prevDIF = macd.dif[macd.dif.length - 2].value;
    final prevDEA = macd.dea[macd.dea.length - 2].value;
    final currDIF = macd.dif.last.value;
    final currDEA = macd.dea.last.value;
    if (prevDIF <= prevDEA && currDIF > currDEA) {
      t6 = true; st6 = 'danger';
      v6 = 'DIF=${currDIF.toStringAsFixed(4)}, DEA=${currDEA.toStringAsFixed(4)}';
      score += W['macd_cross_golden']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'macd_cross_golden', name: 'MACD金叉', triggered: t6, value: v6, status: st6, weight: W['macd_cross_golden']!));

  // 7. MACD bullish divergence
  bool t7 = false; String? v7; String st7 = 'ok';
  if (macd != null && macd.dif.length >= 2 && _detectBullishDivergence(bars, macd.dif)) {
    t7 = true; st7 = 'danger'; v7 = '价格创新低，DIF未创新低';
    score += W['macd_diverge']!; count++;
  }
  signals.add(SignalInfo(key: 'macd_bullish_divergence', name: 'MACD底背离', triggered: t7, value: v7, status: st7, weight: W['macd_diverge']!));

  // 8. Volume golden cross
  bool t8 = false; String? v8; String st8 = 'ok';
  if (bars.length >= 3 && _detectVolumeGoldenCross(bars)) {
    t8 = true; st8 = 'warn'; v8 = '近5日上涨日平均成交量 > 下跌日1.2倍';
    score += W['volume_golden']!; count++;
  }
  signals.add(SignalInfo(key: 'volume_golden_cross', name: '成交量金叉（上涨放量）', triggered: t8, value: v8, status: st8, weight: W['volume_golden']!));

  // 9. Volume breakout
  bool t9 = false; String? v9; String st9 = 'ok';
  if (bars.length >= 5 && latestMA20 != null) {
    final recentBars = bars.sublist(bars.length - 5);
    double sumVol = 0;
    for (final b in recentBars) { sumVol += b.close * b.volume; }
    final avgVol5 = sumVol / 5;
    final curTurnover = latestBar.close * latestBar.volume;
    if (curTurnover > avgVol5 * 1.5 && latestBar.close > latestMA20.value) {
      t9 = true; st9 = 'danger';
      v9 = '当日成交额=${curTurnover.toStringAsFixed(0)}, 5日均额=${avgVol5.toStringAsFixed(0)}';
      score += W['vol_breakout']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'vol_breakout', name: '放量突破', triggered: t9, value: v9, status: st9, weight: W['vol_breakout']!));

  // 10. Bull alignment
  bool t10 = false; String? v10; String st10 = 'ok';
  if (latestMA5 != null && latestMA20 != null && latestMA60 != null) {
    if (latestMA5.value > latestMA20.value && latestMA20.value > latestMA60.value) {
      t10 = true; st10 = 'danger';
      v10 = 'MA5=${latestMA5.value.toStringAsFixed(2)} > MA20=${latestMA20.value.toStringAsFixed(2)} > MA60=${latestMA60.value.toStringAsFixed(2)}';
      score += W['bull_alignment']!; count++;
    }
  }
  signals.add(SignalInfo(key: 'bull_alignment', name: '多头均线排列', triggered: t10, value: v10, status: st10, weight: W['bull_alignment']!));

  final pct = score / maxScore;
  String summary;
  if (pct >= 0.5) summary = '强烈买入信号';
  else if (pct >= 0.25) summary = '值得关注';
  else if (pct > 0) summary = '观望';
  else summary = '暂无买入信号';

  return SignalResult(score: score, maxScore: maxScore, count: count, total: 10, signals: signals, summary: summary);
}
