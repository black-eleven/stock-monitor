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

function renderTable(codes) {
  const tbody = document.getElementById('codeTableBody');
  tbody.innerHTML = codes.map(c => `
    <tr>
      <td><code>${escapeHtml(c.code)}</code></td>
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
  alert(`已生成 ${codes.length} 个邀请码：\n${codes.map(c => c.code).join('\n')}`);
  loadCodes();
});

loadCodes();
