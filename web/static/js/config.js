// ============ Config ============
let currentSanitize = {};
let currentModels = {};

async function loadConfig() {
  try {
    const results = await Promise.allSettled([
      api('/api/config'), api('/api/config/models'), api('/api/config/keys'), api('/api/config/sanitize')
    ]);

    const cfg = results[0].status === 'fulfilled' ? results[0].value : {};
    currentModels = results[1].status === 'fulfilled' ? results[1].value : {};
    const keys = results[2].status === 'fulfilled' ? results[2].value : { keys: [] };
    currentSanitize = results[3].status === 'fulfilled' ? results[3].value : {};

    setToggle('toggle-injection', currentSanitize.block_system_prompt_injection);
    setToggle('toggle-strip', currentSanitize.strip_unknown_fields);
    document.getElementById('cfg-max-input').value = currentSanitize.max_input_tokens || 0;

    renderModelMap(currentModels.model_map || {});
    renderKeys(keys.keys || []);

    document.getElementById('cfg-upstream-url').value = cfg.upstream?.base_url || '';
    document.getElementById('cfg-upstream-key').value = '';
    document.getElementById('cfg-upstream-key').placeholder = cfg.upstream?.api_key || 'sk-...';
    document.getElementById('cfg-upstream-timeout').value = cfg.upstream?.timeout || 300;

    document.getElementById('config-info').innerHTML = `
      Listen: ${esc(cfg.server?.listen || '-')}<br>
      Upstream: ${esc(cfg.upstream?.base_url || '-')} ${cfg.upstream?.api_key ? '<span class="badge badge-green">key set</span>' : '<span class="badge badge-red">no key</span>'}<br>
      Client API Keys: ${cfg.auth?.key_count || 0}<br>
      Log Level: ${esc(cfg.log?.level || 'info')}<br>
      Disguise: ${cfg.disguise?.enabled ? '<span class="badge badge-green">enabled</span>' : '<span class="badge badge-red">disabled</span>'}
    `;
  } catch(e) {
    console.error('loadConfig error:', e);
  }
}

function setToggle(id, value) {
  const el = document.getElementById(id);
  if (value) el.classList.add('on'); else el.classList.remove('on');
}

function toggleSanitize(el) { el.classList.toggle('on'); }

async function saveSanitize() {
  try {
    const body = {
      block_system_prompt_injection: document.getElementById('toggle-injection').classList.contains('on'),
      strip_unknown_fields: document.getElementById('toggle-strip').classList.contains('on'),
      max_input_tokens: parseInt(document.getElementById('cfg-max-input').value) || 0,
      allowed_system_prompt_hashes: currentSanitize.allowed_system_prompt_hashes || []
    };
    await api('/api/config/sanitize', { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) });
    alert('Sanitize config saved');
  } catch(e) {
    alert('Save failed: ' + e.message);
  }
}

function renderModelMap(map) {
  const container = document.getElementById('model-map-list');
  container.innerHTML = Object.entries(map).map(([from, to]) =>
    `<div class="model-map-row">
      <input type="text" value="${esc(from)}" class="mm-from" placeholder="Client model name">
      <span style="color:var(--text-dim)">&#8594;</span>
      <input type="text" value="${esc(to)}" class="mm-to" placeholder="Bedrock model ID">
      <button class="btn btn-danger btn-sm" onclick="this.parentElement.remove()">&#10005;</button>
    </div>`
  ).join('');
}

function addModelRow() {
  const container = document.getElementById('model-map-list');
  const row = document.createElement('div');
  row.className = 'model-map-row';
  row.innerHTML = `
    <input type="text" class="mm-from" placeholder="Client model name">
    <span style="color:var(--text-dim)">&#8594;</span>
    <input type="text" class="mm-to" placeholder="Bedrock model ID">
    <button class="btn btn-danger btn-sm" onclick="this.parentElement.remove()">&#10005;</button>
  `;
  container.appendChild(row);
}

async function saveModels() {
  try {
    const map = {};
    document.querySelectorAll('.model-map-row').forEach(row => {
      const from = row.querySelector('.mm-from').value.trim();
      const to = row.querySelector('.mm-to').value.trim();
      if (from && to) map[from] = to;
    });
    await api('/api/config/models', {
      method: 'PUT', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({ model_map: map, default_model: currentModels.default_model || '' })
    });
    alert('Model mapping saved');
  } catch(e) {
    alert('Save failed: ' + e.message);
  }
}

function renderKeys(keys) {
  const container = document.getElementById('keys-list');
  container.innerHTML = keys.map((k, i) =>
    `<div style="display:flex;align-items:center;gap:8px;margin-bottom:6px;">
      <code style="flex:1;font-size:12px;color:var(--text-dim)">${esc(k)}</code>
      <button class="btn btn-danger btn-sm" onclick="deleteKey(${i})">Delete</button>
    </div>`
  ).join('');
}

async function saveUpstream() {
  try {
    const baseUrl = document.getElementById('cfg-upstream-url').value.trim();
    const apiKeyVal = document.getElementById('cfg-upstream-key').value.trim();
    const timeout = parseInt(document.getElementById('cfg-upstream-timeout').value) || 300;

    if (baseUrl && !/^https?:\/\/.+/.test(baseUrl)) {
      alert('Invalid URL format');
      return;
    }

    const body = { base_url: baseUrl, timeout };
    if (apiKeyVal) body.api_key = apiKeyVal;
    await api('/api/config/upstream', { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) });
    alert('Upstream config saved');
    loadConfig();
  } catch(e) {
    alert('Save failed: ' + e.message);
  }
}

async function addKey() {
  const input = document.getElementById('new-key-input');
  const key = input.value.trim();
  if (!key) return;
  try {
    await api('/api/config/keys', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ key }) });
    input.value = '';
    loadConfig();
  } catch(e) {
    alert('Add key failed: ' + e.message);
  }
}

async function deleteKey(index) {
  if (!confirm('Delete this API key?')) return;
  try {
    await api('/api/config/keys', { method: 'DELETE', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ index }) });
    loadConfig();
  } catch(e) {
    alert('Delete failed: ' + e.message);
  }
}
