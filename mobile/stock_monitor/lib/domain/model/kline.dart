class KlineBar {
  final int ts;
  final double o;
  final double cl;
  final double h;
  final double l;
  final double v;
  KlineBar({required this.ts, required this.o, required this.cl, required this.h, required this.l, required this.v});
  factory KlineBar.fromJson(Map<String, dynamic> json) => KlineBar(
        ts: json['ts'] as int,
        o: (json['o'] as num).toDouble(),
        cl: (json['cl'] as num).toDouble(),
        h: (json['h'] as num).toDouble(),
        l: (json['l'] as num).toDouble(),
        v: (json['v'] as num).toDouble(),
      );
}

class KlineItem {
  final String c;
  final List<KlineBar> k;
  KlineItem({required this.c, required this.k});
  factory KlineItem.fromJson(Map<String, dynamic> json) => KlineItem(
        c: json['c'] as String,
        k: (json['k'] as List).map((e) => KlineBar.fromJson(e as Map<String, dynamic>)).toList(),
      );
}

class Bar {
  final int time;
  final double open;
  final double high;
  final double low;
  final double close;
  Bar({required this.time, required this.open, required this.high, required this.low, required this.close});
}
