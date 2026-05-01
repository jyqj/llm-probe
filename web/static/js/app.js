// ============ Navigation ============
function switchPage(page) {
  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
  document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
  document.getElementById('page-' + page).classList.add('active');
  document.querySelector(`.nav-item[data-page="${page}"]`).classList.add('active');

  if (page === 'dashboard') loadStats();
  if (page === 'gateway') loadGatewayConfig();
  if (page === 'logs') loadLogs();
  if (page === 'config') loadConfig();
  if (page === 'probe') loadProbeReport();
}

// ============ Helpers ============
function fmt(n) {
  if (n >= 1000000) return (n/1000000).toFixed(1) + 'M';
  if (n >= 1000) return (n/1000).toFixed(1) + 'K';
  return String(n);
}

function esc(s) {
  if (s == null) return '';
  const d = document.createElement('div');
  d.textContent = String(s);
  return d.innerHTML;
}

function timeAgo(ts) {
  const d = new Date(ts);
  const now = Date.now();
  const diff = now - d.getTime();
  if (diff < 60000) return Math.round(diff/1000) + 's ago';
  if (diff < 3600000) return Math.round(diff/60000) + 'm ago';
  if (diff < 86400000) return Math.round(diff/3600000) + 'h ago';
  return d.toLocaleDateString();
}

// ============ Unified API wrapper ============
async function api(url, opts = {}) {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), opts.timeout || 15000);
  try {
    const resp = await fetch(url, { signal: controller.signal, ...opts });
    clearTimeout(timeoutId);
    if (!resp.ok) {
      const text = await resp.text().catch(() => '');
      throw new Error(`${resp.status} ${resp.statusText}: ${text.slice(0, 200)}`);
    }
    return resp.json();
  } catch(e) {
    clearTimeout(timeoutId);
    if (e.name === 'AbortError') throw new Error('Request timeout');
    throw e;
  }
}

// Admin token for probe endpoints
let adminToken = sessionStorage.getItem('gw_admin_token') || localStorage.getItem('gw_admin_token') || '';

function getAdminToken() {
  if (!adminToken) {
    adminToken = prompt('Enter Admin Token (X-Admin-Token):') || '';
    if (adminToken) {
      sessionStorage.setItem('gw_admin_token', adminToken);
      localStorage.setItem('gw_admin_token', adminToken);
    }
  }
  return adminToken;
}

function adminFetch(url, opts = {}) {
  const token = getAdminToken();
  if (!token) return Promise.reject(new Error('No admin token'));
  opts.headers = { ...opts.headers, 'X-Admin-Token': token };
  return fetch(url, opts);
}

// Auto-refresh dashboard every 5s
setInterval(() => {
  if (document.getElementById('page-dashboard').classList.contains('active')) loadStats();
}, 5000);

// Initial load
loadStats();
