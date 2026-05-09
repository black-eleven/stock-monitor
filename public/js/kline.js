class KlineComponent {
  constructor(api) {
    this.api = api;
    this.chart = null;
    this.candleSeries = null;
    this.currentSymbol = null;
    this.currentInterval = '1d';
  }

  init() {
    this.chart = LightweightCharts.createChart(document.getElementById('klineChartContainer'), {
      layout: { background: { color: '#161b22' }, textColor: '#8b949e' },
      grid: { vertLines: { color: '#21262d' }, horzLines: { color: '#21262d' } },
      crosshair: { mode: LightweightCharts.CrosshairMode.Normal },
      timeScale: { timeVisible: true, secondsVisible: false, borderColor: '#30363d' },
      rightPriceScale: { borderColor: '#30363d' },
    });

    this.candleSeries = this.chart.addCandlestickSeries({
      upColor: '#f85149', downColor: '#3fb950',
      borderUpColor: '#f85149', borderDownColor: '#3fb950',
      wickUpColor: '#f85149', wickDownColor: '#3fb950',
    });

    // Setup interval buttons
    document.querySelectorAll('.interval-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        document.querySelectorAll('.interval-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        this.currentInterval = btn.dataset.kt;
        if (this.currentSymbol) this.loadData();
      });
    });

    // Setup symbol selector
    const select = document.getElementById('klineSymbol');
    select.addEventListener('change', () => {
      this.currentSymbol = select.value;
      if (this.currentSymbol) this.loadData();
    });
  }

  // Update the symbol dropdown from watchlist
  updateSymbols(watchlist) {
    const select = document.getElementById('klineSymbol');
    select.innerHTML = watchlist.map(w =>
      `<option value="${escapeHtml(w.symbol)}">${escapeHtml(w.name)} (${escapeHtml(shortCode(w.symbol))})</option>`
    ).join('');
    if (watchlist.length > 0 && !this.currentSymbol) {
      this.currentSymbol = watchlist[0].symbol;
      this.loadData();
    }
  }

  // Set symbol externally (from watchlist tab click)
  setSymbol(symbol) {
    this.currentSymbol = symbol;
    const select = document.getElementById('klineSymbol');
    if (select) select.value = symbol;
    this.loadData();
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
        // Sort by time ascending (QOS returns descending)
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

  _drawIndicators(bars) {
    if (!bars || bars.length < 30) return;
    this._clearIndicators();

    // Calculate indicators
    const ma5 = calcMA(bars, 5);
    const ma20 = calcMA(bars, 20);
    const ma60 = calcMA(bars, 60);

    // Store for cleanup
    this._maLines = [];

    // Draw MA lines (lineWidth 1, no last value label, no price line)
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

    // Draw sell signal markers
    const rsi = calcRSI(bars, 14);
    const macd = calcMACD(bars);
    const signals = evaluateSignals(bars, { ma5, ma20, ma60 }, rsi, macd);
    if (signals.count > 0) {
      const lastBar = bars[bars.length - 1];
      this.candleSeries.setMarkers([{
        time: lastBar.time,
        position: 'aboveBar',
        color: '#f85149',
        shape: 'arrowDown',
        text: 'S ' + signals.count,
      }]);
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

  resize() {
    if (this.chart) {
      const container = document.getElementById('klineChartContainer');
      if (container) {
        this.chart.resize(container.clientWidth, container.clientHeight);
      }
    }
  }
}
