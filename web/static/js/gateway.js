// ============ Gateway Config ============

let globalDisguise = {};
let upstreamDisguises = [];
let switchMeta = [];

const switchGroups = {
  body: { label: 'Body Rewrite', icon: '{}' },
  headers: { label: 'Headers', icon: 'H' },
  sse: { label: 'SSE Stream', icon: '~' },
  thinking: { label: 'Thinking', icon: '?' },
  cache: { label: 'Cache', icon: '$' },
  tokens: { label: 'Tokens', icon: '#' },
  advanced: { label: 'Advanced', icon: '*' },
  passthrough: { label: 'Passthrough', icon: '>' },
  debug: { label: 'Debug', icon: 'D' },
};

async function loadGatewayConfig() {
  try {
    const [disguiseResp, upstreamResp] = await Promise.all([
      adminFetch('/api/config/disguise'),
      adminFetch('/api/config/disguise/upstream')
    ]);

    if (disguiseResp.ok) {
      const data = await disguiseResp.json();
      globalDisguise = data.config || {};
      switchMeta = data.switches || [];
      renderGlobalDisguise();
    }

    if (upstreamResp.ok) {
      const data = await upstreamResp.json();
      upstreamDisguises = data.upstreams || [];
      renderUpstreamList();
    }
  } catch (e) {
    console.error('loadGatewayConfig error:', e);
    document.getElementById('gateway-global-switches').innerHTML =
      '<div style="color:var(--text-dim);padding:20px;">Enter admin token to view config.</div>';
  }
}

function renderGlobalDisguise() {
  const container = document.getElementById('gateway-global-switches');

  // Group switches
  const grouped = {};
  for (const sw of switchMeta) {
    if (!grouped[sw.group]) grouped[sw.group] = [];
    grouped[sw.group].push(sw);
  }

  let html = `
    <div class="disguise-master-toggle">
      <div class="toggle ${globalDisguise.enabled ? 'on' : ''}"
           onclick="toggleGlobalMaster(this)" data-key="enabled"></div>
      <span class="toggle-label" style="font-weight:600;">Disguise Enabled (Master Switch)</span>
    </div>
    <div class="disguise-groups ${globalDisguise.enabled ? '' : 'disabled'}">
  `;

  for (const [groupKey, groupInfo] of Object.entries(switchGroups)) {
    const switches = grouped[groupKey] || [];
    if (switches.length === 0) continue;

    html += `
      <div class="disguise-group">
        <div class="disguise-group-header">
          <span class="disguise-group-icon">${groupInfo.icon}</span>
          <span class="disguise-group-label">${groupInfo.label}</span>
        </div>
        <div class="disguise-group-switches">
    `;

    for (const sw of switches) {
      const configKey = snakeToCamel(sw.key);
      const isOn = globalDisguise[configKey] || false;
      html += `
        <div class="disguise-switch-row">
          <div class="toggle ${isOn ? 'on' : ''}"
               onclick="toggleGlobalSwitch(this, '${configKey}')"
               data-key="${configKey}"></div>
          <div class="disguise-switch-info">
            <span class="disguise-switch-label">${sw.label}</span>
            <span class="disguise-switch-desc">${sw.desc}</span>
          </div>
        </div>
      `;
    }

    html += '</div></div>';
  }

  html += '</div>';
  container.innerHTML = html;
}

function snakeToCamel(str) {
  return str.replace(/_([a-z])/g, (_, c) => c.toUpperCase());
}

function toggleGlobalMaster(el) {
  el.classList.toggle('on');
  globalDisguise.enabled = el.classList.contains('on');
  const groups = document.querySelector('.disguise-groups');
  if (groups) {
    groups.classList.toggle('disabled', !globalDisguise.enabled);
  }
}

function toggleGlobalSwitch(el, key) {
  el.classList.toggle('on');
  globalDisguise[key] = el.classList.contains('on');
}

async function saveGlobalDisguise() {
  const btn = document.getElementById('gateway-save-btn');
  btn.disabled = true;
  btn.textContent = 'Saving...';

  try {
    const resp = await adminFetch('/api/config/disguise', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(globalDisguise)
    });

    if (resp.ok) {
      showGatewayStatus('Global config saved', 'green');
    } else {
      const err = await resp.text();
      showGatewayStatus('Save failed: ' + err.slice(0, 100), 'red');
    }
  } catch (e) {
    showGatewayStatus('Error: ' + e.message, 'red');
  } finally {
    btn.disabled = false;
    btn.textContent = 'Save Global Config';
  }
}

function showGatewayStatus(msg, color) {
  const el = document.getElementById('gateway-status');
  el.innerHTML = `<span class="badge badge-${color}">${esc(msg)}</span>`;
  setTimeout(() => { el.innerHTML = ''; }, 3000);
}

function renderUpstreamList() {
  const container = document.getElementById('gateway-upstream-list');

  if (upstreamDisguises.length === 0) {
    container.innerHTML = '<div style="color:var(--text-dim);padding:20px;text-align:center;">No per-upstream configs. Run Probe to auto-detect settings.</div>';
    return;
  }

  container.innerHTML = upstreamDisguises.map(u => {
    const score = u.score !== undefined ? Math.round(u.score) : '-';
    const grade = u.grade || '-';
    const gradeColor = u.score >= 80 ? 'var(--green)' : u.score >= 50 ? 'var(--yellow)' : 'var(--red)';

    return `
      <div class="upstream-config-item">
        <div class="upstream-config-header" onclick="toggleUpstreamDetail(this)">
          <div class="upstream-url">${esc(u.upstream_base)}</div>
          <div class="upstream-meta">
            ${u.has_report ? `<span class="badge" style="color:${gradeColor}">${grade} (${score})</span>` : '<span class="badge badge-dim">No probe</span>'}
            <span class="upstream-toggle-icon">+</span>
          </div>
        </div>
        <div class="upstream-config-detail" style="display:none;" data-upstream="${esc(u.upstream_base)}">
          ${renderUpstreamSwitches(u.upstream_base, u.config)}
          <div style="margin-top:12px;display:flex;gap:8px;">
            <button class="btn btn-primary btn-sm" onclick="saveUpstreamDisguise('${esc(u.upstream_base)}')">Save</button>
            <button class="btn btn-danger btn-sm" onclick="deleteUpstreamDisguise('${esc(u.upstream_base)}')">Delete</button>
            <button class="btn btn-ghost btn-sm" onclick="copyFromGlobal('${esc(u.upstream_base)}')">Copy from Global</button>
          </div>
        </div>
      </div>
    `;
  }).join('');
}

function renderUpstreamSwitches(upstreamBase, config) {
  const grouped = {};
  for (const sw of switchMeta) {
    if (!grouped[sw.group]) grouped[sw.group] = [];
    grouped[sw.group].push(sw);
  }

  let html = '<div class="upstream-switches">';

  for (const [groupKey, groupInfo] of Object.entries(switchGroups)) {
    const switches = grouped[groupKey] || [];
    if (switches.length === 0) continue;

    html += `<div class="upstream-switch-group"><span class="upstream-group-label">${groupInfo.label}:</span>`;

    for (const sw of switches) {
      const configKey = snakeToCamel(sw.key);
      const isOn = config[configKey] || false;
      html += `
        <div class="toggle-mini ${isOn ? 'on' : ''}"
             onclick="toggleUpstreamSwitch(this, '${upstreamBase}', '${configKey}')"
             title="${sw.label}: ${sw.desc}"
             data-key="${configKey}"></div>
      `;
    }

    html += '</div>';
  }

  html += '</div>';
  return html;
}

function toggleUpstreamDetail(header) {
  const detail = header.nextElementSibling;
  const icon = header.querySelector('.upstream-toggle-icon');
  if (detail.style.display === 'none') {
    detail.style.display = 'block';
    icon.textContent = '-';
  } else {
    detail.style.display = 'none';
    icon.textContent = '+';
  }
}

function toggleUpstreamSwitch(el, upstreamBase, key) {
  el.classList.toggle('on');
  // Update local state
  const upstream = upstreamDisguises.find(u => u.upstream_base === upstreamBase);
  if (upstream) {
    upstream.config[key] = el.classList.contains('on');
  }
}

async function saveUpstreamDisguise(upstreamBase) {
  const upstream = upstreamDisguises.find(u => u.upstream_base === upstreamBase);
  if (!upstream) return;

  try {
    const resp = await adminFetch('/api/config/disguise/upstream', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        upstream_base: upstreamBase,
        config: upstream.config
      })
    });

    if (resp.ok) {
      showGatewayStatus('Upstream config saved: ' + upstreamBase, 'green');
    } else {
      showGatewayStatus('Save failed', 'red');
    }
  } catch (e) {
    showGatewayStatus('Error: ' + e.message, 'red');
  }
}

async function deleteUpstreamDisguise(upstreamBase) {
  if (!confirm('Delete per-upstream config for ' + upstreamBase + '?\nThis upstream will use global config.')) return;

  try {
    const resp = await adminFetch('/api/config/disguise/upstream?upstream_base=' + encodeURIComponent(upstreamBase), {
      method: 'DELETE'
    });

    if (resp.ok) {
      upstreamDisguises = upstreamDisguises.filter(u => u.upstream_base !== upstreamBase);
      renderUpstreamList();
      showGatewayStatus('Deleted: ' + upstreamBase, 'green');
    }
  } catch (e) {
    showGatewayStatus('Error: ' + e.message, 'red');
  }
}

function copyFromGlobal(upstreamBase) {
  const upstream = upstreamDisguises.find(u => u.upstream_base === upstreamBase);
  if (!upstream) return;

  // Copy all switches from global
  for (const sw of switchMeta) {
    const key = snakeToCamel(sw.key);
    upstream.config[key] = globalDisguise[key] || false;
  }
  upstream.config.enabled = globalDisguise.enabled || false;

  // Re-render
  const detail = document.querySelector(`[data-upstream="${upstreamBase}"]`);
  if (detail) {
    detail.innerHTML = renderUpstreamSwitches(upstreamBase, upstream.config) + `
      <div style="margin-top:12px;display:flex;gap:8px;">
        <button class="btn btn-primary btn-sm" onclick="saveUpstreamDisguise('${esc(upstreamBase)}')">Save</button>
        <button class="btn btn-danger btn-sm" onclick="deleteUpstreamDisguise('${esc(upstreamBase)}')">Delete</button>
        <button class="btn btn-ghost btn-sm" onclick="copyFromGlobal('${esc(upstreamBase)}')">Copy from Global</button>
      </div>
    `;
  }
  showGatewayStatus('Copied global config to ' + upstreamBase, 'green');
}

// Quick probe for testing new upstream
async function runQuickProbe() {
  const targetBase = document.getElementById('quick-probe-url').value.trim();
  const targetKey = document.getElementById('quick-probe-key').value.trim();

  if (!targetBase) {
    alert('Enter upstream URL');
    return;
  }

  const btn = document.getElementById('quick-probe-btn');
  const resultEl = document.getElementById('quick-probe-result');
  btn.disabled = true;
  btn.textContent = 'Testing...';
  resultEl.innerHTML = '<span class="badge badge-yellow">Running quick probe...</span>';

  try {
    const resp = await adminFetch('/admin/probe/quick', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        target_base: targetBase,
        target_key: targetKey,
        model: 'claude-opus-4-6'
      })
    });

    const data = await resp.json();

    if (data.ok) {
      const score = data.score ? Math.round(data.score.total_score) : '-';
      const grade = data.score ? data.score.grade : '-';
      const needed = data.needed_switches || [];

      let html = `
        <div class="quick-probe-summary">
          <div class="quick-probe-score">
            <span class="grade">${esc(grade)}</span>
            <span class="score">${score}/100</span>
          </div>
          <div class="quick-probe-stats">
            ${data.checks_passed}/${data.checks_total} checks passed in ${data.elapsed_ms}ms
          </div>
        </div>
      `;

      if (needed.length > 0) {
        html += `
          <div class="quick-probe-needed">
            <strong>Recommended switches:</strong>
            ${needed.map(s => `<span class="badge badge-yellow">${esc(s)}</span>`).join(' ')}
          </div>
        `;
      } else {
        html += '<div class="quick-probe-needed"><span class="badge badge-green">No disguise needed!</span></div>';
      }

      html += `
        <div style="margin-top:12px;">
          <button class="btn btn-primary btn-sm" onclick="applyQuickProbeRecommended('${esc(targetBase)}')">Apply Recommended Config</button>
          <button class="btn btn-ghost btn-sm" onclick="switchPage('probe')">Run Full Probe</button>
        </div>
      `;

      resultEl.innerHTML = html;

      // Store recommended for apply
      window._quickProbeRecommended = {
        target: targetBase,
        config: data.recommended
      };

      // Refresh upstream list
      loadGatewayConfig();
    } else {
      resultEl.innerHTML = `<span class="badge badge-red">Error: ${esc(data.error || 'unknown')}</span>`;
    }
  } catch (e) {
    resultEl.innerHTML = `<span class="badge badge-red">Error: ${esc(e.message)}</span>`;
  } finally {
    btn.disabled = false;
    btn.textContent = 'Quick Probe';
  }
}

async function applyQuickProbeRecommended(targetBase) {
  if (!window._quickProbeRecommended || window._quickProbeRecommended.target !== targetBase) {
    alert('No recommendation available');
    return;
  }

  try {
    const resp = await adminFetch('/api/config/disguise/upstream', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        upstream_base: targetBase,
        config: window._quickProbeRecommended.config
      })
    });

    if (resp.ok) {
      showGatewayStatus('Applied recommended config to ' + targetBase, 'green');
      loadGatewayConfig();
    } else {
      showGatewayStatus('Apply failed', 'red');
    }
  } catch (e) {
    showGatewayStatus('Error: ' + e.message, 'red');
  }
}
