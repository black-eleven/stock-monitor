if (!auth.isLoggedIn() || !auth.isAdmin()) {
  window.location.href = '/login.html';
}

async function loadCodes() {
  const res = await fetch('/api/admin/invite-codes', { headers: auth.getAuthHeaders() });
  if (res.status === 401) { auth.logout(); return; }
  if (!res.ok) return;
  const codes = await res.json();
  renderTable(codes);
}

function copyText(text, btn) {
  navigator.clipboard.writeText(text).then(() => {
    const orig = btn.textContent;
    btn.textContent = '已复制';
    btn.style.color = '#3fb950';
    setTimeout(() => { btn.textContent = orig; btn.style.color = ''; }, 1500);
  });
}

function showGeneratedCodes(codes) {
  const html = `
    <div id="generatedCodes" style="background:#161b22;border:1px solid #30363d;border-radius:6px;padding:12px 16px;margin-bottom:16px;">
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">
        <span style="color:#3fb950;font-weight:bold;">已生成 ${codes.length} 个邀请码</span>
        <button onclick="document.getElementById('generatedCodes').remove()" style="background:none;border:none;color:#8b949e;cursor:pointer;font-size:16px;">×</button>
      </div>
      ${codes.map(c => `
        <div style="display:flex;align-items:center;gap:8px;padding:6px 0;border-bottom:1px solid #30363d;">
          <code style="flex:1;font-size:15px;">${escapeHtml(c.code)}</code>
          <button class="btn-sm copy-btn" onclick="copyText('${escapeHtml(c.code)}', this)">复制</button>
        </div>
      `).join('')}
      <button id="copyAllBtn" style="margin-top:10px;padding:6px 12px;background:#30363d;color:#e6edf3;border:none;border-radius:4px;cursor:pointer;font-size:13px;" onclick="copyText(${JSON.stringify(codes.map(c => c.code).join('\n'))}, this); this.textContent='已全部复制'; setTimeout(()=>this.textContent='一键复制全部',1500);">一键复制全部</button>
    </div>
  `;
  const oldBox = document.getElementById('generatedCodes');
  if (oldBox) oldBox.remove();
  document.getElementById('generateForm').insertAdjacentHTML('afterend', html);
}

function renderTable(codes) {
  const tbody = document.getElementById('codeTableBody');
  tbody.innerHTML = codes.map(c => `
    <tr>
      <td>
        <code>${escapeHtml(c.code)}</code>
        <button class="btn-sm copy-btn" onclick="copyText('${escapeHtml(c.code)}', this)">复制</button>
      </td>
      <td>${c.maxUses === 0 ? '∞' : c.maxUses}</td>
      <td>${c.usedCount}</td>
      <td><span style="color:${c.isActive ? '#3fb950' : '#f85149'}">${c.isActive ? '有效' : '已禁用'}</span></td>
      <td>${escapeHtml(c.createdAt)}</td>
      <td>
        <button class="btn-sm" onclick="toggleCode(${c.id}, ${!c.isActive})" style="background:none;border:1px solid #30363d;color:#e6edf3;padding:4px 8px;border-radius:4px;cursor:pointer;">
          ${c.isActive ? '禁用' : '启用'}
        </button>
      </td>
    </tr>
  `).join('');
}

async function toggleCode(id, active) {
  await fetch(`/api/admin/invite-codes/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...auth.getAuthHeaders() },
    body: JSON.stringify({ isActive: active }),
  });
  loadCodes();
}

document.getElementById('generateForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const data = new FormData(e.target);
  const res = await fetch('/api/admin/invite-codes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...auth.getAuthHeaders() },
    body: JSON.stringify({
      maxUses: parseInt(data.get('maxUses')) || 1,
      count: parseInt(data.get('count')) || 1,
    }),
  });
  if (!res.ok) { alert('生成失败'); return; }
  const codes = await res.json();
  showGeneratedCodes(codes);
  loadCodes();
});

loadCodes();
