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

// Evaluate 7 sell signals
// bars: array of { time, open, high, low, close }
// maData: { ma5, ma20, ma60 } where each is [{ time, value }]
// rsiData: [{ time, value }]
// macdData: { dif, dea, macd } or null
// returns: { count, total: 7, signals: Signal[], summary: string }
function evaluateSignals(bars, maData, rsiData, macdData) {
  const signals = [];
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

  // ---- 1. MA5 just crossed below MA20 (dead cross) ----
  let signal1 = {
    key: 'ma_cross_dead',
    name: 'MA5死叉MA20',
    triggered: false,
    value: null,
    status: 'ok'
  };
  if (prevMA5 && prevMA20 && latestMA5 && latestMA20) {
    if (prevMA5.value >= prevMA20.value && latestMA5.value < latestMA20.value) {
      signal1.triggered = true;
      signal1.value = `MA5=${latestMA5.value.toFixed(2)}, MA20=${latestMA20.value.toFixed(2)}`;
      signal1.status = 'danger';
      count++;
    }
  }
  signals.push(signal1);

  // ---- 2. Latest close below latest MA20 ----
  let signal2 = {
    key: 'price_below_ma20',
    name: '收盘价低于MA20',
    triggered: false,
    value: null,
    status: 'ok'
  };
  if (latestBar && latestMA20) {
    if (latestBar.close < latestMA20.value) {
      signal2.triggered = true;
      signal2.value = `收盘=${latestBar.close.toFixed(2)}, MA20=${latestMA20.value.toFixed(2)}`;
      signal2.status = 'warn';
      count++;
    }
  }
  signals.push(signal2);

  // ---- 3. Latest close below latest MA60 ----
  let signal3 = {
    key: 'price_below_ma60',
    name: '收盘价低于MA60',
    triggered: false,
    value: null,
    status: 'ok'
  };
  if (latestBar && latestMA60) {
    if (latestBar.close < latestMA60.value) {
      signal3.triggered = true;
      signal3.value = `收盘=${latestBar.close.toFixed(2)}, MA60=${latestMA60.value.toFixed(2)}`;
      signal3.status = 'warn';
      count++;
    }
  }
  signals.push(signal3);

  // ---- 4. RSI overbought ----
  let signal4 = {
    key: 'rsi_overbought',
    name: 'RSI超买',
    triggered: false,
    value: null,
    status: 'ok'
  };
  if (rsiData && rsiData.length >= 1) {
    const latestRSI = rsiData[rsiData.length - 1].value;
    if (latestRSI > 70) {
      signal4.triggered = true;
      signal4.value = `RSI=${latestRSI.toFixed(2)}`;
      signal4.status = latestRSI > 80 ? 'danger' : 'warn';
      count++;
    }
  }
  signals.push(signal4);

  // ---- 5. RSI divergence (bearish) over last 20 bars ----
  let signal5 = {
    key: 'rsi_divergence',
    name: 'RSI顶背离',
    triggered: false,
    value: null,
    status: 'ok'
  };
  if (bars && rsiData && rsiData.length >= 2) {
    const divergent = detectBearishDivergence(bars, rsiData);
    if (divergent) {
      signal5.triggered = true;
      signal5.value = '价格创新高，RSI未创新高';
      signal5.status = 'danger';
      count++;
    }
  }
  signals.push(signal5);

  // ---- 6. MACD dead cross (DIF crossed below DEA) ----
  let signal6 = {
    key: 'macd_dead_cross',
    name: 'MACD死叉',
    triggered: false,
    value: null,
    status: 'ok'
  };
  if (macdData && macdData.dif && macdData.dea) {
    const dif = macdData.dif;
    const dea = macdData.dea;
    if (dif.length >= 2 && dea.length >= 2) {
      const prevDIF = dif[dif.length - 2].value;
      const prevDEA = dea[dea.length - 2].value;
      const currDIF = dif[dif.length - 1].value;
      const currDEA = dea[dea.length - 1].value;
      if (prevDIF >= prevDEA && currDIF < currDEA) {
        signal6.triggered = true;
        signal6.value = `DIF=${currDIF.toFixed(4)}, DEA=${currDEA.toFixed(4)}`;
        signal6.status = 'danger';
        count++;
      }
    }
  }
  signals.push(signal6);

  // ---- 7. MACD divergence (bearish) over last 20 bars ----
  let signal7 = {
    key: 'macd_divergence',
    name: 'MACD顶背离',
    triggered: false,
    value: null,
    status: 'ok'
  };
  if (bars && macdData && macdData.dif && macdData.dif.length >= 2) {
    const divergent = detectBearishDivergence(bars, macdData.dif);
    if (divergent) {
      signal7.triggered = true;
      signal7.value = '价格创新高，DIF未创新高';
      signal7.status = 'danger';
      count++;
    }
  }
  signals.push(signal7);

  // ---- Summary ----
  let summary;
  if (count === 0) {
    summary = '正常';
  } else if (count <= 2) {
    summary = '短期偏弱';
  } else if (count <= 4) {
    summary = '偏弱，注意风险';
  } else {
    summary = '强烈卖出信号';
  }

  return {
    count,
    total: 7,
    signals,
    summary
  };
}
