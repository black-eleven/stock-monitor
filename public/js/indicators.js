// Technical Indicator Calculation Functions
// Used by both K-line chart component and analysis tab

// ============ Moving Averages ============

// Simple Moving Average
// bars: array of { time, open, high, low, close } sorted by time ascending
// period: number (e.g. 5, 20, 60)
// returns: [{ time, value }] starting from index period - 1
function calcMA(bars, period) {
  if (!bars || bars.length < period) return [];
  const result = [];
  for (let i = period - 1; i < bars.length; i++) {
    let sum = 0;
    for (let j = 0; j < period; j++) sum += bars[i - j].close;
    result.push({ time: bars[i].time, value: sum / period });
  }
  return result;
}

// Exponential Moving Average (helper for MACD)
// bars: array of { time, close } or { time, value }
// period: number
// returns: [{ time, value }]
function calcEMA(bars, period) {
  if (!bars || bars.length === 0) return [];
  const k = 2 / (period + 1);
  // Determine if input has .close or .value
  const getValue = (bar) => bar.close !== undefined ? bar.close : bar.value;
  let ema = getValue(bars[0]);
  const result = [];
  for (let i = 0; i < bars.length; i++) {
    ema = (getValue(bars[i]) - ema) * k + ema;
    result.push({ time: bars[i].time, value: ema });
  }
  return result;
}

// ============ RSI ============

// Relative Strength Index using Wilder Smoothing
// bars: array of { time, open, high, low, close }
// period: number (default 14)
// returns: [{ time, value }] starting from index period
function calcRSI(bars, period = 14) {
  if (!bars || bars.length < period + 1) return [];
  let gain = 0, loss = 0;
  for (let i = 1; i <= period; i++) {
    const diff = bars[i].close - bars[i - 1].close;
    if (diff > 0) gain += diff; else loss -= diff;
  }
  let avgGain = gain / period;
  let avgLoss = loss / period;
  const result = [];
  for (let i = period; i < bars.length; i++) {
    if (i > period) {
      const diff = bars[i].close - bars[i - 1].close;
      avgGain = (avgGain * (period - 1) + (diff > 0 ? diff : 0)) / period;
      avgLoss = (avgLoss * (period - 1) + (diff < 0 ? -diff : 0)) / period;
    }
    const rs = avgLoss === 0 ? 100 : avgGain / avgLoss;
    result.push({ time: bars[i].time, value: 100 - 100 / (1 + rs) });
  }
  return result;
}

// ============ MACD ============

// MACD indicator
// bars: array of { time, open, high, low, close }
// fast: number (default 12)
// slow: number (default 26)
// signal: number (default 9)
// returns: { dif: [{ time, value }], dea: [{ time, value }], macd: [{ time, value }] }
//          or null if not enough data
function calcMACD(bars, fast = 12, slow = 26, signal = 9) {
  if (!bars || bars.length < slow) return null;
  const emaFast = calcEMA(bars, fast);
  const emaSlow = calcEMA(bars, slow);
  const dif = [];
  for (let i = 0; i < bars.length; i++) {
    dif.push({ time: bars[i].time, value: emaFast[i].value - emaSlow[i].value });
  }
  const dea = calcEMA(dif, signal);
  const macd = [];
  for (let i = 0; i < dif.length; i++) {
    macd.push({ time: dif[i].time, value: (dif[i].value - dea[i].value) * 2 });
  }
  return { dif, dea, macd };
}

// ============ Signal Evaluation ============

// Helper: detect bearish divergence over last 20 bars
// Checks if price made a higher high but the indicator made a lower high
function detectBearishDivergence(bars, indicatorData) {
  const count = Math.min(20, bars.length, indicatorData.length);
  if (count < 2) return false;

  const recentBars = bars.slice(-count);
  const recentIndicators = indicatorData.slice(-count);

  // Find the bar with the highest high price
  let maxHigh = -Infinity;
  let maxHighIdx = -1;
  for (let i = 0; i < recentBars.length; i++) {
    if (recentBars[i].high > maxHigh) {
      maxHigh = recentBars[i].high;
      maxHighIdx = i;
    }
  }

  // Find the indicator entry with the highest value
  let maxIndicator = -Infinity;
  let maxIndicatorIdx = -1;
  for (let i = 0; i < recentIndicators.length; i++) {
    if (recentIndicators[i].value > maxIndicator) {
      maxIndicator = recentIndicators[i].value;
      maxIndicatorIdx = i;
    }
  }

  // If the bar with highest high is more recent than the indicator's max,
  // and the indicator value at that bar is lower than the max indicator -> divergence
  if (maxHighIdx < 0 || maxIndicatorIdx < 0) return false;
  const indicatorAtMaxHigh = recentIndicators[maxHighIdx].value;
  return maxHighIdx > maxIndicatorIdx && indicatorAtMaxHigh < maxIndicator;
}

// Volume death cross: average volume on down days > up days over last 5 bars
function detectVolumeDeathCross(bars) {
  const n = Math.min(5, bars.length);
  if (n < 3) return false;
  const recent = bars.slice(-n);
  let upVol = 0, upDays = 0, downVol = 0, downDays = 0;
  for (let i = 0; i < recent.length; i++) {
    const change = recent[i].close - recent[i].open;
    if (change >= 0) {
      upVol += recent[i].close * recent[i].volume; // approximate turnover
      upDays++;
    } else {
      downVol += recent[i].close * recent[i].volume;
      downDays++;
    }
  }
  if (upDays === 0 || downDays === 0) return false;
  const avgUpVol = upVol / upDays;
  const avgDownVol = downVol / downDays;
  return avgDownVol > avgUpVol * 1.2;
}

// Helper: detect bullish divergence over last 20 bars
// Checks if price made a lower low but the indicator made a higher low
function detectBullishDivergence(bars, indicatorData) {
  const count = Math.min(20, bars.length, indicatorData.length);
  if (count < 2) return false;

  const recentBars = bars.slice(-count);
  const recentIndicators = indicatorData.slice(-count);

  let minLow = Infinity;
  let minLowIdx = -1;
  for (let i = 0; i < recentBars.length; i++) {
    if (recentBars[i].low < minLow) {
      minLow = recentBars[i].low;
      minLowIdx = i;
    }
  }

  let minIndicator = Infinity;
  let minIndicatorIdx = -1;
  for (let i = 0; i < recentIndicators.length; i++) {
    if (recentIndicators[i].value < minIndicator) {
      minIndicator = recentIndicators[i].value;
      minIndicatorIdx = i;
    }
  }

  if (minLowIdx < 0 || minIndicatorIdx < 0) return false;
  const indicatorAtMinLow = recentIndicators[minLowIdx].value;
  return minLowIdx > minIndicatorIdx && indicatorAtMinLow > minIndicator;
}

// Volume golden cross: average volume on up days > down days over last 5 bars
function detectVolumeGoldenCross(bars) {
  const n = Math.min(5, bars.length);
  if (n < 3) return false;
  const recent = bars.slice(-n);
  let upVol = 0, upDays = 0, downVol = 0, downDays = 0;
  for (let i = 0; i < recent.length; i++) {
    const change = recent[i].close - recent[i].open;
    if (change >= 0) {
      upVol += recent[i].close * recent[i].volume;
      upDays++;
    } else {
      downVol += recent[i].close * recent[i].volume;
      downDays++;
    }
  }
  if (upDays === 0 || downDays === 0) return false;
  const avgUpVol = upVol / upDays;
  const avgDownVol = downVol / downDays;
  return avgUpVol > avgDownVol * 1.2;
}

// Weighted sell signal evaluation (8 signals, max score = 13.0)
// bars: array of { time, open, high, low, close }
// maData: { ma5, ma20, ma60 } where each is [{ time, value }]
// rsiData: [{ time, value }]
// macdData: { dif, dea, macd } or null
// returns: { score, maxScore: 13, count, total: 8, signals: Signal[], summary: string }
function evaluateSignals(bars, maData, rsiData, macdData) {
  // Weights: tier-1 reversal signals 2.0–2.5, tier-2 confirmations 1.0, tier-3 redundant 0.5, volume 1.5
  const W = { ma_cross: 2.0, price_ma20: 1.0, price_ma60: 0.5, rsi_overbought: 1.0, rsi_diverge: 2.5, macd_cross: 2.0, macd_diverge: 2.5, volume_death: 1.5 };
  const MAX_SCORE = W.ma_cross + W.price_ma20 + W.price_ma60 + W.rsi_overbought + W.rsi_diverge + W.macd_cross + W.macd_diverge + W.volume_death;

  const signals = [];
  let score = 0;
  let count = 0;

  const latestBar = bars[bars.length - 1];

  // ---- Extract latest MA values ----
  const ma5 = maData && maData.ma5 ? maData.ma5 : [];
  const ma20 = maData && maData.ma20 ? maData.ma20 : [];
  const ma60 = maData && maData.ma60 ? maData.ma60 : [];

  const latestMA5 = ma5.length >= 1 ? ma5[ma5.length - 1] : null;
  const prevMA5 = ma5.length >= 2 ? ma5[ma5.length - 2] : null;
  const latestMA20 = ma20.length >= 1 ? ma20[ma20.length - 1] : null;
  const prevMA20 = ma20.length >= 2 ? ma20[ma20.length - 2] : null;
  const latestMA60 = ma60.length >= 1 ? ma60[ma60.length - 1] : null;

  // ---- 1. MA5死叉MA20 (weight 2.0) ----
  let s1 = { key: 'ma_cross_dead', name: 'MA5死叉MA20', triggered: false, value: null, status: 'ok', weight: W.ma_cross };
  if (prevMA5 && prevMA20 && latestMA5 && latestMA20) {
    if (prevMA5.value >= prevMA20.value && latestMA5.value < latestMA20.value) {
      s1.triggered = true;
      s1.value = `MA5=${latestMA5.value.toFixed(2)}, MA20=${latestMA20.value.toFixed(2)}`;
      s1.status = 'danger';
      score += W.ma_cross;
      count++;
    }
  }
  signals.push(s1);

  // ---- 2. 收盘 < MA20 (weight 1.0) ----
  let s2 = { key: 'price_below_ma20', name: '收盘价低于MA20', triggered: false, value: null, status: 'ok', weight: W.price_ma20 };
  if (latestBar && latestMA20) {
    if (latestBar.close < latestMA20.value) {
      s2.triggered = true;
      s2.value = `收盘=${latestBar.close.toFixed(2)}, MA20=${latestMA20.value.toFixed(2)}`;
      s2.status = 'warn';
      score += W.price_ma20;
      count++;
    }
  }
  signals.push(s2);

  // ---- 3. 收盘 < MA60 (weight 0.5) ----
  let s3 = { key: 'price_below_ma60', name: '收盘价低于MA60', triggered: false, value: null, status: 'ok', weight: W.price_ma60 };
  if (latestBar && latestMA60) {
    if (latestBar.close < latestMA60.value) {
      s3.triggered = true;
      s3.value = `收盘=${latestBar.close.toFixed(2)}, MA60=${latestMA60.value.toFixed(2)}`;
      s3.status = 'warn';
      score += W.price_ma60;
      count++;
    }
  }
  signals.push(s3);

  // ---- 4. RSI超买 (weight 1.0) ----
  let s4 = { key: 'rsi_overbought', name: 'RSI超买', triggered: false, value: null, status: 'ok', weight: W.rsi_overbought };
  if (rsiData && rsiData.length >= 1) {
    const latestRSI = rsiData[rsiData.length - 1].value;
    if (latestRSI > 70) {
      s4.triggered = true;
      s4.value = `RSI=${latestRSI.toFixed(2)}`;
      s4.status = latestRSI > 80 ? 'danger' : 'warn';
      score += W.rsi_overbought;
      count++;
    }
  }
  signals.push(s4);

  // ---- 5. RSI顶背离 (weight 2.5) ----
  let s5 = { key: 'rsi_divergence', name: 'RSI顶背离', triggered: false, value: null, status: 'ok', weight: W.rsi_diverge };
  if (bars && rsiData && rsiData.length >= 2) {
    const divergent = detectBearishDivergence(bars, rsiData);
    if (divergent) {
      s5.triggered = true;
      s5.value = '价格创新高，RSI未创新高';
      s5.status = 'danger';
      score += W.rsi_diverge;
      count++;
    }
  }
  signals.push(s5);

  // ---- 6. MACD死叉 (weight 2.0) ----
  let s6 = { key: 'macd_dead_cross', name: 'MACD死叉', triggered: false, value: null, status: 'ok', weight: W.macd_cross };
  if (macdData && macdData.dif && macdData.dea) {
    const dif = macdData.dif;
    const dea = macdData.dea;
    if (dif.length >= 2 && dea.length >= 2) {
      const prevDIF = dif[dif.length - 2].value;
      const prevDEA = dea[dea.length - 2].value;
      const currDIF = dif[dif.length - 1].value;
      const currDEA = dea[dea.length - 1].value;
      if (prevDIF >= prevDEA && currDIF < currDEA) {
        s6.triggered = true;
        s6.value = `DIF=${currDIF.toFixed(4)}, DEA=${currDEA.toFixed(4)}`;
        s6.status = 'danger';
        score += W.macd_cross;
        count++;
      }
    }
  }
  signals.push(s6);

  // ---- 7. MACD顶背离 (weight 2.5) ----
  let s7 = { key: 'macd_divergence', name: 'MACD顶背离', triggered: false, value: null, status: 'ok', weight: W.macd_diverge };
  if (bars && macdData && macdData.dif && macdData.dif.length >= 2) {
    const divergent = detectBearishDivergence(bars, macdData.dif);
    if (divergent) {
      s7.triggered = true;
      s7.value = '价格创新高，DIF未创新高';
      s7.status = 'danger';
      score += W.macd_diverge;
      count++;
    }
  }
  signals.push(s7);

  // ---- 8. 成交量死叉 (weight 1.5) ----
  let s8 = { key: 'volume_death_cross', name: '成交量死叉（下跌放量）', triggered: false, value: null, status: 'ok', weight: W.volume_death };
  if (bars && bars.length >= 3) {
    if (detectVolumeDeathCross(bars)) {
      s8.triggered = true;
      s8.value = '近5日下跌日平均成交量 > 上涨日1.2倍';
      s8.status = 'warn';
      score += W.volume_death;
      count++;
    }
  }
  signals.push(s8);

  // ---- Summary ----
  const pct = Math.round((score / MAX_SCORE) * 100);
  let summary;
  if (pct >= 50) {
    summary = '强烈卖出信号';
  } else if (pct >= 25) {
    summary = '偏弱，注意风险';
  } else if (pct > 0) {
    summary = '短期偏弱';
  } else {
    summary = '正常';
  }

  return {
    score,
    maxScore: MAX_SCORE,
    count,
    total: 8,
    signals,
    summary
  };
}

// Weighted buy signal evaluation (10 signals, max score = 17.0)
// bars: array of { time, open, high, low, close, volume }
// maData: { ma5, ma20, ma60 } where each is [{ time, value }]
// rsiData: [{ time, value }]
// macdData: { dif, dea, macd } or null
// returns: { score, maxScore: 17, count, total: 10, signals: Signal[], summary: string }
function evaluateBuySignals(bars, maData, rsiData, macdData) {
  const W = {
    ma_cross_golden: 2.0, price_ma20: 1.0, price_ma60: 0.5,
    rsi_oversold: 1.0, rsi_diverge: 2.5, macd_cross_golden: 2.0,
    macd_diverge: 2.5, volume_golden: 1.5,
    vol_breakout: 2.0, bull_alignment: 2.0
  };
  const MAX_SCORE = W.ma_cross_golden + W.price_ma20 + W.price_ma60 +
    W.rsi_oversold + W.rsi_diverge + W.macd_cross_golden + W.macd_diverge +
    W.volume_golden + W.vol_breakout + W.bull_alignment;

  const signals = [];
  let score = 0;
  let count = 0;

  const latestBar = bars[bars.length - 1];

  const ma5 = maData && maData.ma5 ? maData.ma5 : [];
  const ma20 = maData && maData.ma20 ? maData.ma20 : [];
  const ma60 = maData && maData.ma60 ? maData.ma60 : [];

  const latestMA5 = ma5.length >= 1 ? ma5[ma5.length - 1] : null;
  const prevMA5 = ma5.length >= 2 ? ma5[ma5.length - 2] : null;
  const latestMA20 = ma20.length >= 1 ? ma20[ma20.length - 1] : null;
  const prevMA20 = ma20.length >= 2 ? ma20[ma20.length - 2] : null;
  const latestMA60 = ma60.length >= 1 ? ma60[ma60.length - 1] : null;

  // ---- 1. MA5金叉MA20 (weight 2.0) ----
  let s1 = { key: 'ma_cross_golden', name: 'MA5金叉MA20', triggered: false, value: null, status: 'ok', weight: W.ma_cross_golden };
  if (prevMA5 && prevMA20 && latestMA5 && latestMA20) {
    if (prevMA5.value <= prevMA20.value && latestMA5.value > latestMA20.value) {
      s1.triggered = true;
      s1.value = `MA5=${latestMA5.value.toFixed(2)}, MA20=${latestMA20.value.toFixed(2)}`;
      s1.status = 'danger';
      score += W.ma_cross_golden;
      count++;
    }
  }
  signals.push(s1);

  // ---- 2. 收盘 > MA20 (weight 1.0) ----
  let s2 = { key: 'price_above_ma20', name: '收盘价高于MA20', triggered: false, value: null, status: 'ok', weight: W.price_ma20 };
  if (latestBar && latestMA20) {
    if (latestBar.close > latestMA20.value) {
      s2.triggered = true;
      s2.value = `收盘=${latestBar.close.toFixed(2)}, MA20=${latestMA20.value.toFixed(2)}`;
      s2.status = 'warn';
      score += W.price_ma20;
      count++;
    }
  }
  signals.push(s2);

  // ---- 3. 收盘 > MA60 (weight 0.5) ----
  let s3 = { key: 'price_above_ma60', name: '收盘价高于MA60', triggered: false, value: null, status: 'ok', weight: W.price_ma60 };
  if (latestBar && latestMA60) {
    if (latestBar.close > latestMA60.value) {
      s3.triggered = true;
      s3.value = `收盘=${latestBar.close.toFixed(2)}, MA60=${latestMA60.value.toFixed(2)}`;
      s3.status = 'warn';
      score += W.price_ma60;
      count++;
    }
  }
  signals.push(s3);

  // ---- 4. RSI超卖 (weight 1.0) ----
  let s4 = { key: 'rsi_oversold', name: 'RSI超卖', triggered: false, value: null, status: 'ok', weight: W.rsi_oversold };
  if (rsiData && rsiData.length >= 1) {
    const latestRSI = rsiData[rsiData.length - 1].value;
    if (latestRSI < 30) {
      s4.triggered = true;
      s4.value = `RSI=${latestRSI.toFixed(2)}`;
      s4.status = latestRSI < 20 ? 'danger' : 'warn';
      score += W.rsi_oversold;
      count++;
    }
  }
  signals.push(s4);

  // ---- 5. RSI底背离 (weight 2.5) ----
  let s5 = { key: 'rsi_bullish_divergence', name: 'RSI底背离', triggered: false, value: null, status: 'ok', weight: W.rsi_diverge };
  if (bars && rsiData && rsiData.length >= 2) {
    if (detectBullishDivergence(bars, rsiData)) {
      s5.triggered = true;
      s5.value = '价格创新低，RSI未创新低';
      s5.status = 'danger';
      score += W.rsi_diverge;
      count++;
    }
  }
  signals.push(s5);

  // ---- 6. MACD金叉 (weight 2.0) ----
  let s6 = { key: 'macd_cross_golden', name: 'MACD金叉', triggered: false, value: null, status: 'ok', weight: W.macd_cross_golden };
  if (macdData && macdData.dif && macdData.dea) {
    const dif = macdData.dif;
    const dea = macdData.dea;
    if (dif.length >= 2 && dea.length >= 2) {
      const prevDIF = dif[dif.length - 2].value;
      const prevDEA = dea[dea.length - 2].value;
      const currDIF = dif[dif.length - 1].value;
      const currDEA = dea[dea.length - 1].value;
      if (prevDIF <= prevDEA && currDIF > currDEA) {
        s6.triggered = true;
        s6.value = `DIF=${currDIF.toFixed(4)}, DEA=${currDEA.toFixed(4)}`;
        s6.status = 'danger';
        score += W.macd_cross_golden;
        count++;
      }
    }
  }
  signals.push(s6);

  // ---- 7. MACD底背离 (weight 2.5) ----
  let s7 = { key: 'macd_bullish_divergence', name: 'MACD底背离', triggered: false, value: null, status: 'ok', weight: W.macd_diverge };
  if (bars && macdData && macdData.dif && macdData.dif.length >= 2) {
    if (detectBullishDivergence(bars, macdData.dif)) {
      s7.triggered = true;
      s7.value = '价格创新低，DIF未创新低';
      s7.status = 'danger';
      score += W.macd_diverge;
      count++;
    }
  }
  signals.push(s7);

  // ---- 8. 成交量金叉 (weight 1.5) ----
  let s8 = { key: 'volume_golden_cross', name: '成交量金叉（上涨放量）', triggered: false, value: null, status: 'ok', weight: W.volume_golden };
  if (bars && bars.length >= 3) {
    if (detectVolumeGoldenCross(bars)) {
      s8.triggered = true;
      s8.value = '近5日上涨日平均成交量 > 下跌日1.2倍';
      s8.status = 'warn';
      score += W.volume_golden;
      count++;
    }
  }
  signals.push(s8);

  // ---- 9. 放量突破 (weight 2.0) ----
  let s9 = { key: 'vol_breakout', name: '放量突破', triggered: false, value: null, status: 'ok', weight: W.vol_breakout };
  if (bars && bars.length >= 5 && latestMA20) {
    const recentBars = bars.slice(-5);
    let sumVol = 0;
    for (const b of recentBars) sumVol += b.close * b.volume;
    const avgVol5 = sumVol / 5;
    const curTurnover = latestBar.close * latestBar.volume;
    if (curTurnover > avgVol5 * 1.5 && latestBar.close > latestMA20.value) {
      s9.triggered = true;
      s9.value = `当日成交额=${curTurnover.toFixed(0)}, 5日均额=${avgVol5.toFixed(0)}`;
      s9.status = 'danger';
      score += W.vol_breakout;
      count++;
    }
  }
  signals.push(s9);

  // ---- 10. 多头排列 (weight 2.0) ----
  let s10 = { key: 'bull_alignment', name: '多头均线排列', triggered: false, value: null, status: 'ok', weight: W.bull_alignment };
  if (latestMA5 && latestMA20 && latestMA60) {
    if (latestMA5.value > latestMA20.value && latestMA20.value > latestMA60.value) {
      s10.triggered = true;
      s10.value = `MA5=${latestMA5.value.toFixed(2)} > MA20=${latestMA20.value.toFixed(2)} > MA60=${latestMA60.value.toFixed(2)}`;
      s10.status = 'danger';
      score += W.bull_alignment;
      count++;
    }
  }
  signals.push(s10);

  // ---- Summary ----
  const pct = Math.round((score / MAX_SCORE) * 100);
  let summary;
  if (pct >= 50) {
    summary = '强烈买入信号';
  } else if (pct >= 25) {
    summary = '值得关注';
  } else if (pct > 0) {
    summary = '观望';
  } else {
    summary = '暂无买入信号';
  }

  return {
    score,
    maxScore: MAX_SCORE,
    count,
    total: 10,
    signals,
    summary
  };
}
