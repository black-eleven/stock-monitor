class AlertsComponent {
  constructor(api) {
    this.api = api;
    this.alerts = [];
    this.logs = [];
  }

  async init() {
    this.alerts = await this.api.getAlerts();
    this.render();
  }

  addLog(log) {
    this.logs.unshift(log);
    if (this.logs.length > 100) this.logs.pop();
    this._renderLogs();
  }

  render() {
    this._renderList();
    this._renderLogs();
  }

  _renderList() {
    const el = document.getElementById('alertList');
    if (this.alerts.length === 0) {
      el.innerHTML = '<div class="empty-state">暂无预警规则</div>';
      return;
    }

    const typeLabels = { above: '涨破', below: '跌破', change_pct: '涨跌幅达 %' };

    el.innerHTML = this.alerts.map(a => `
      <div class="alert-item">
        <div class="info">
          <strong>${escapeHtml(a.symbol)}</strong>
          ${escapeHtml(typeLabels[a.type] || a.type)} ${a.value}
          <span style="color:${a.enabled ? '#3fb950' : '#8b949e'};margin-left:8px">${a.enabled ? '启用' : '禁用'}</span>
        </div>
        <div class="actions">
          <button class="btn btn-sm" data-id="${a.id}" data-toggle="1">${a.enabled ? '禁用' : '启用'}</button>
          <button class="btn btn-danger btn-sm" data-id="${a.id}" data-delete="1">删除</button>
        </div>
      </div>
    `).join('');

    el.querySelectorAll('[data-toggle]').forEach(btn => {
      btn.addEventListener('click', async () => {
        const id = parseInt(btn.dataset.id, 10);
        const alert = this.alerts.find(a => a.id === id);
        if (!alert) return;
        await this.api.updateAlert(id, { enabled: !alert.enabled });
        alert.enabled = !alert.enabled;
        this._renderList();
      });
    });

    el.querySelectorAll('[data-delete]').forEach(btn => {
      btn.addEventListener('click', async () => {
        const id = parseInt(btn.dataset.id, 10);
        this.alerts = this.alerts.filter(a => a.id !== id);
        await this.api.deleteAlert(id);
        this._renderList();
      });
    });
  }

  _renderLogs() {
    const el = document.getElementById('alertLog');
    if (this.logs.length === 0) {
      el.innerHTML = '<div style="color:#8b949e;padding:8px">暂无触发记录</div>';
      return;
    }
    el.innerHTML = this.logs.slice(0, 50).map(l =>
      `<div class="alert-log-item">[${new Date(l.triggeredAt).toLocaleString('zh-HK')}] ${escapeHtml(l.message)}</div>`
    ).join('');
  }
}
