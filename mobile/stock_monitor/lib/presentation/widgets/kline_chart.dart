import 'package:flutter/material.dart';
import '../../core/theme.dart';
import '../../domain/model/kline.dart';

class KlineChartWidget extends StatelessWidget {
  final List<Bar> bars;

  const KlineChartWidget({super.key, required this.bars});

  @override
  Widget build(BuildContext context) {
    if (bars.isEmpty) return const Center(child: Text('无数据'));

    final visibleBars = bars.length > 80 ? bars.sublist(bars.length - 80) : bars;
    final minY = visibleBars.map((b) => b.low).reduce((a, b) => a < b ? a : b);
    final maxY = visibleBars.map((b) => b.high).reduce((a, b) => a > b ? a : b);
    final margin = (maxY - minY) * 0.05 + 0.01;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 4),
      child: CustomPaint(
        size: const Size(double.infinity, 280),
        painter: _CandlestickPainter(
          bars: visibleBars,
          minPrice: minY - margin,
          maxPrice: maxY + margin,
        ),
      ),
    );
  }
}

class _CandlestickPainter extends CustomPainter {
  final List<Bar> bars;
  final double minPrice;
  final double maxPrice;

  _CandlestickPainter({required this.bars, required this.minPrice, required this.maxPrice});

  @override
  void paint(Canvas canvas, Size size) {
    if (bars.isEmpty) return;

    final priceRange = maxPrice - minPrice;
    if (priceRange == 0) return;

    final barWidth = size.width / bars.length;
    final wickWidth = 1.0;
    final bodyWidth = barWidth * 0.6;

    final upPaint = Paint()..color = AppTheme.up;
    final downPaint = Paint()..color = AppTheme.down;
    final wickUpPaint = Paint()..color = AppTheme.up..strokeWidth = wickWidth..style = PaintingStyle.fill;
    final wickDownPaint = Paint()..color = AppTheme.down..strokeWidth = wickWidth..style = PaintingStyle.fill;

    for (int i = 0; i < bars.length; i++) {
      final bar = bars[i];
      final centerX = i * barWidth + barWidth / 2;

      final openY = size.height - ((bar.open - minPrice) / priceRange) * size.height;
      final closeY = size.height - ((bar.close - minPrice) / priceRange) * size.height;
      final highY = size.height - ((bar.high - minPrice) / priceRange) * size.height;
      final lowY = size.height - ((bar.low - minPrice) / priceRange) * size.height;

      final isUp = bar.close >= bar.open;
      final bodyPaint = isUp ? upPaint : downPaint;
      final wickPaint = isUp ? wickUpPaint : wickDownPaint;

      // Draw wick (high-low line)
      canvas.drawLine(Offset(centerX, highY), Offset(centerX, lowY), wickPaint);

      // Draw body
      final top = isUp ? closeY : openY;
      final bottom = isUp ? openY : closeY;
      final bodyHeight = (bottom - top).abs();
      final rect = RRect.fromRectAndRadius(
        Rect.fromLTWH(centerX - bodyWidth / 2, top, bodyWidth, bodyHeight < 0.5 ? 0.5 : bodyHeight),
        Radius.zero,
      );
      canvas.drawRect(rect.outerRect, bodyPaint);
    }
  }

  @override
  bool shouldRepaint(covariant _CandlestickPainter oldDelegate) =>
      bars != oldDelegate.bars || minPrice != oldDelegate.minPrice || maxPrice != oldDelegate.maxPrice;
}
