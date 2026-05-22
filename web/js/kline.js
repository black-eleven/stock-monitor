class KlineComponent {
  constructor(api, containerId, intervalsContainerId) {
    this.api = api;
    this.containerId = containerId || 'klineChartContainer';
    this.intervalsContainerId = intervalsContainerId || 'klineIntervals';
    this.chart = null;
    this.candleSeries = null;
    this.currentSymbol = null;
    this.currentInterval = '1d';
    this._inited = false;
  }

  init() {
    if (this._inited) return;
    this._inited = true;

    const BJ = 8 * 3600;
    const fmtBeijing = (time) => {
      const d = new Date((time + BJ) * 1000);
      const pad = n => String(n).padStart(2, '0');
      return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
    };

    this.chart = LightweightCharts.createChart(document.getElementById(this.containerId), {
      layout: { background: { color: '#161b22' }, textColor: '#8b949e' },
      grid: { vertLines: { color: '#21262d' }, horzLines: { color: '#21262d' } },
      crosshair: { mode: LightweightCharts.CrosshairMode.Normal },
      timeScale: { timeVisible: true, secondsVisible: false, borderColor: '#30363d' },
      localization: { timeFormatter: fmtBeijing },
      rightPriceScale: { borderColor: '#30363d' },
    });

    this.candleSeries = this.chart.addCandlestickSeries({
      upColor: '#3fb950', downColor: '#f85149',
      borderUpColor: '#3fb950', borderDownColor: '#f85149',
      wickUpColor: '#3fb950', wickDownColor: '#f85149',
    });

    // Setup interval buttons scoped to the intervals container
    const intervalsEl = document.getElementById(this.intervalsContainerId);
    if (intervalsEl) {
      intervalsEl.querySelectorAll('.interval-btn').forEach(btn => {
        btn.addEventListener('click', () => {
          intervalsEl.querySelectorAll('.interval-btn').forEach(b => b.classList.remove('active'));
          btn.classList.add('active');
          this.currentInterval = btn.dataset.kt;
          if (this.currentSymbol) this.loadData();
        });
      });
    }
  }

  updateSymbols(watchlist) {
    if (watchlist.length > 0 && !this.currentSymbol) {
      this.currentSymbol = watchlist[0].symbol;
      this.loadData();
    }
  }

  setSymbol(symbol) {
    this.currentSymbol = symbol;
    this.loadData();
  }

  resize() {
    if (this.chart) {
      const container = document.getElementById(this.containerId);
      if (container && container.offsetParent) {
        this.chart.resize(container.clientWidth, container.clientHeight);
      }
    }
  }

  async loadData(retries = 3) {
    if (!this.currentSymbol) return;
    for (let attempt = 0; attempt < retries; attempt++) {
      try {
        const data = await this.api.getKline(this.currentSymbol, this.currentInterval, 200);
        const bars = [];
        for (const item of data) {
          if (!item.k) continue;
          for (const k of item.k) {
            bars.push({
              time: k.ts,
              open: parseFloat(k.o),
              high: parseFloat(k.h),
              low: parseFloat(k.l),
              close: parseFloat(k.cl),
            });
          }
        }
        bars.sort((a, b) => a.time - b.time);
        this.candleSeries.setData(bars);
        this.chart.timeScale().fitContent();
        this._drawIndicators(bars);
        return;
      } catch (err) {
        console.error(`Kline load error (attempt ${attempt + 1}/${retries}):`, err);
        if (attempt < retries - 1) {
          await new Promise(r => setTimeout(r, 1500));
        }
      }
    }
  }

  _getMAPeriods() {
    switch (this.currentInterval) {
      case '5m':  return [24, 48, 96];
      case '15m': return [16, 32, 64];
      case '30m': return [16, 32, 48];
      case '1h':  return [8, 20, 40];
      case '2h':  return [8, 16, 32];
      case '4h':  return [6, 12, 24];
      case '1w':  return [4, 13, 26];
      case '1M':  return [3, 6, 12];
      default:    return [5, 20, 60];
    }
  }

  _drawIndicators(bars) {
    if (!bars || bars.length < 30) return;
    this._clearIndicators();

    const [p1, p2, p3] = this._getMAPeriods();
    const ma5 = calcMA(bars, p1);
    const ma20 = calcMA(bars, p2);
    const ma60 = calcMA(bars, p3);

    this._maLines = [];

    if (ma5.length > 0) {
      const line = this.chart.addLineSeries({ color: '#ffe066', lineWidth: 1, lastValueVisible: false, priceLineVisible: false });
      line.setData(ma5);
      this._maLines.push(line);
    }
    if (ma20.length > 0) {
      const line = this.chart.addLineSeries({ color: '#58a6ff', lineWidth: 1, lastValueVisible: false, priceLineVisible: false });
      line.setData(ma20);
      this._maLines.push(line);
    }
    if (ma60.length > 0) {
      const line = this.chart.addLineSeries({ color: '#bc8cff', lineWidth: 1, lastValueVisible: false, priceLineVisible: false });
      line.setData(ma60);
      this._maLines.push(line);
    }

    const crossMarkers = [];
    const rsi = calcRSI(bars, 14);
    const macd = calcMACD(bars);
    const sellSignals = evaluateSignals(bars, { ma5, ma20, ma60 }, rsi, macd);
    const buySignals = evaluateBuySignals(bars, { ma5, ma20, ma60 }, rsi, macd);
    const lastBar = bars[bars.length - 1];

    if (sellSignals.count > 0) {
      crossMarkers.push({ time: lastBar.time, position: 'aboveBar', color: '#f85149', shape: 'arrowDown', text: 'S' + sellSignals.count });
    }
    if (buySignals.count > 0) {
      crossMarkers.push({ time: lastBar.time, position: 'belowBar', color: '#3fb950', shape: 'arrowUp', text: 'B' + buySignals.count });
    }

    if (crossMarkers.length > 0) {
      this.candleSeries.setMarkers(crossMarkers);
    }
  }

  _clearIndicators() {
    if (this._maLines) {
      for (const line of this._maLines) {
        this.chart.removeSeries(line);
      }
    }
    this._maLines = [];
    this.candleSeries.setMarkers([]);
  }
}
