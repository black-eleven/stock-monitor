import 'package:flutter/material.dart';
import '../../core/theme.dart';
import '../../domain/indicators.dart';
import '../../domain/model/kline.dart';

class KlineChartWidget extends StatelessWidget {
  final List<Bar> bars;

  const KlineChartWidget({super.key, required this.bars});

  @override
  Widget build(BuildContext context) {
    if (bars.isEmpty) return const Center(child: Text('无数据'));

    final visibleBars =
        bars.length > 80 ? bars.sublist(bars.length - 80) : bars;
    final minY =
        visibleBars.map((b) => b.low).reduce((a, b) => a < b ? a : b);
    final maxY =
        visibleBars.map((b) => b.high).reduce((a, b) => a > b ? a : b);
    final margin = (maxY - minY) * 0.05 + 0.01;

    // Calculate MAs from full bars for cross detection accuracy
    final ma5 = calcMA(bars, 5);
    final ma20 = calcMA(bars, 20);
    final ma60 = calcMA(bars, 60);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 4),
      child: CustomPaint(
        size: const Size(double.infinity, 280),
        painter: _CandlestickPainter(
          bars: visibleBars,
          minPrice: minY - margin,
          maxPrice: maxY + margin,
          allBars: bars,
          ma5: ma5,
          ma20: ma20,
          ma60: ma60,
        ),
      ),
    );
  }
}

class _CandlestickPainter extends CustomPainter {
  final List<Bar> bars;
  final double minPrice;
  final double maxPrice;
  final List<Bar> allBars;
  final List<MA> ma5;
  final List<MA> ma20;
  final List<MA> ma60;

  _CandlestickPainter({
    required this.bars,
    required this.minPrice,
    required this.maxPrice,
    required this.allBars,
    required this.ma5,
    required this.ma20,
    required this.ma60,
  });

  double _toY(double price, double priceRange, double height) {
    return height - ((price - minPrice) / priceRange) * height;
  }

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
    final wickUpPaint = Paint()
      ..color = AppTheme.up
      ..strokeWidth = wickWidth
      ..style = PaintingStyle.fill;
    final wickDownPaint = Paint()
      ..color = AppTheme.down
      ..strokeWidth = wickWidth
      ..style = PaintingStyle.fill;

    for (int i = 0; i < bars.length; i++) {
      final bar = bars[i];
      final centerX = i * barWidth + barWidth / 2;

      final openY = _toY(bar.open, priceRange, size.height);
      final closeY = _toY(bar.close, priceRange, size.height);
      final highY = _toY(bar.high, priceRange, size.height);
      final lowY = _toY(bar.low, priceRange, size.height);

      final isUp = bar.close >= bar.open;
      final bodyPaint = isUp ? upPaint : downPaint;
      final wickPaint = isUp ? wickUpPaint : wickDownPaint;

      // Draw wick
      canvas.drawLine(
          Offset(centerX, highY), Offset(centerX, lowY), wickPaint);

      // Draw body
      final top = isUp ? closeY : openY;
      final bottom = isUp ? openY : closeY;
      final bodyHeight = (bottom - top).abs();
      final rect = RRect.fromRectAndRadius(
        Rect.fromLTWH(centerX - bodyWidth / 2, top, bodyWidth,
            bodyHeight < 0.5 ? 0.5 : bodyHeight),
        Radius.zero,
      );
      canvas.drawRect(rect.outerRect, bodyPaint);
    }

    // Draw MA lines and cross markers
    _drawMALine(canvas, size, ma5, const Color(0xFFFFE066), barWidth);
    _drawMALine(canvas, size, ma20, const Color(0xFF58A6FF), barWidth);
    _drawMALine(canvas, size, ma60, const Color(0xFFBC8CFF), barWidth);

    _drawCrossMarkers(canvas, size, barWidth);
  }

  void _drawMALine(
      Canvas canvas, Size size, List<MA> maData, Color color, double barWidth) {
    if (maData.isEmpty) return;
    final priceRange = maxPrice - minPrice;
    final paint = Paint()
      ..color = color
      ..strokeWidth = 1.0
      ..style = PaintingStyle.stroke;

    final path = Path();
    bool first = true;
    for (final ma in maData) {
      // Map time to x position within visible bars
      final idx = bars.indexWhere((b) => b.time == ma.time);
      if (idx < 0) continue;

      final x = idx * barWidth + barWidth / 2;
      final y = _toY(ma.value, priceRange, size.height);
      if (first) {
        path.moveTo(x, y);
        first = false;
      } else {
        path.lineTo(x, y);
      }
    }
    canvas.drawPath(path, paint);
  }

  void _drawCrossMarkers(Canvas canvas, Size size, double barWidth) {
    if (ma5.length < 2 || ma20.length < 2) return;
    final priceRange = maxPrice - minPrice;

    for (int i = 1; i < ma5.length && i < ma20.length; i++) {
      if (ma5[i].value <= 0 || ma20[i].value <= 0) continue;

      final idx = bars.indexWhere((b) => b.time == ma5[i].time);
      if (idx < 0) continue;

      final x = idx * barWidth + barWidth / 2;
      final prev5 = ma5[i - 1].value;
      final prev20 = ma20[i - 1].value;
      final curr5 = ma5[i].value;
      final curr20 = ma20[i].value;

      Paint markerPaint;
      String label;
      double yOffset;

      if (prev5 <= prev20 && curr5 > curr20) {
        // Golden cross
        markerPaint = Paint()..color = AppTheme.up;
        label = '金叉';
        yOffset = -18;
      } else if (prev5 >= prev20 && curr5 < curr20) {
        // Death cross
        markerPaint = Paint()..color = AppTheme.down;
        label = '死叉';
        yOffset = 18;
      } else {
        continue;
      }

      final bar = bars[idx];
      final baseY = prev5 <= prev20 && curr5 > curr20
          ? _toY(bar.low, priceRange, size.height)
          : _toY(bar.high, priceRange, size.height);

      // Draw triangle marker
      final path = Path();
      if (prev5 <= prev20 && curr5 > curr20) {
        // Up arrow (golden cross below bar)
        path.moveTo(x, baseY + 8);
        path.lineTo(x - 5, baseY + 16);
        path.lineTo(x + 5, baseY + 16);
      } else {
        // Down arrow (death cross above bar)
        path.moveTo(x, baseY - 8);
        path.lineTo(x - 5, baseY - 16);
        path.lineTo(x + 5, baseY - 16);
      }
      path.close();
      canvas.drawPath(path, markerPaint);

      // Draw label
      final textPainter = TextPainter(
        text: TextSpan(
          text: label,
          style: TextStyle(
              color: markerPaint.color, fontSize: 10, fontWeight: FontWeight.w600),
        ),
        textDirection: TextDirection.ltr,
      );
      textPainter.layout();
      textPainter.paint(
        canvas,
        Offset(x - textPainter.width / 2, baseY + yOffset),
      );
    }
  }

  @override
  bool shouldRepaint(covariant _CandlestickPainter oldDelegate) => true;
}
