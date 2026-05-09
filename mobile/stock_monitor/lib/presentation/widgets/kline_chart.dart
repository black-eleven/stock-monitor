import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';
import '../../core/theme.dart';
import '../../domain/model/kline.dart';

class KlineChartWidget extends StatelessWidget {
  final List<Bar> bars;

  const KlineChartWidget({super.key, required this.bars});

  @override
  Widget build(BuildContext context) {
    if (bars.isEmpty) return const Center(child: Text('无数据'));

    final spots = bars.map((b) => CandlestickData(
          date: b.time,
          open: b.open,
          high: b.high,
          low: b.low,
          close: b.close,
          color: b.close >= b.open ? AppTheme.up : AppTheme.down,
        )).toList();

    final visibleSpots = spots.length > 100 ? spots.sublist(spots.length - 100) : spots;
    final minY = visibleSpots.map((s) => s.low).reduce((a, b) => a < b ? a : b) * 0.995;
    final maxY = visibleSpots.map((s) => s.high).reduce((a, b) => a > b ? a : b) * 1.005;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 4),
      child: CandlestickChart(
        CandlestickChartData(
          minY: minY,
          maxY: maxY,
          lineColor: AppTheme.border,
          dataSets: [CandlestickDataSet(data: visibleSpots)],
        ),
      ),
    );
  }
}
