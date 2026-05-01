// ============ Probe ============
let lastProbeResult = null;

const verdictColorMap = {
  green: 'var(--green)',
  blue: 'var(--accent)',
  yellow: 'var(--yellow)',
  orange: 'var(--orange)',
  red: 'var(--red)',
};

const gradeColorMap = verdictColorMap;

function colorVar(c) { return verdictColorMap[c] || 'var(--text-dim)'; }

async function loadProbeReport() {
  const listEl = document.getElementById('probe-upstream-list');
  const detailEl = document.getElementById('probe-detail');
  try {
    const resp = await adminFetch('/admin/probe/report');
    if (!resp.ok) {
      listEl.innerHTML = '<div style="color:var(--text-dim);text-align:center;padding:20px;">No probe data yet. Run a probe to get started.</div>';
      detailEl.style.display = 'none';
      return;
    }
    const data = await resp.json();
    if (data.upstreams) {
      renderUpstreamList(data.upstreams);
    } else if (data.ok) {
      renderUpstreamList([{
        target: data.target, model: data.model, summary: data.summary,
        probed_at: data.timestamp,
        checks_total: data.checks?.length || 0,
        checks_passed: (data.checks || []).filter(c => c.pass).length,
        score: data.score?.total_score, grade: data.score?.grade,
        grade_color: data.score?.grade_color, verdict: data.score?.verdict,
        verdict_label: data.score?.verdict_label, verdict_color: data.score?.verdict_color
      }]);
    }
  } catch(e) {
    listEl.innerHTML = '<div style="color:var(--text-dim);text-align:center;padding:20px;">Enter admin token to view probe data.</div>';
  }
}

function renderUpstreamList(upstreams) {
  const el = document.getElementById('probe-upstream-list');
  if (!upstreams || upstreams.length === 0) {
    el.innerHTML = '<div style="color:var(--text-dim);text-align:center;padding:20px;">No probed upstreams yet.</div>';
    return;
  }
  el.innerHTML = upstreams.map(u => {
    const hasScore = u.score !== undefined && u.score !== null;
    const probing = u.probing ? ' <span class="badge badge-yellow">probing...</span>' : '';
    const when = u.probed_at ? timeAgo(u.probed_at) : '';

    if (hasScore) {
      const gc = colorVar(u.grade_color);
      const vc = colorVar(u.verdict_color);
      return `<div class="probe-upstream-item" onclick="viewUpstreamProbe('${esc(u.target)}')">
        <div class="url">${esc(u.target)}${probing}</div>
        <div style="color:var(--text-dim);font-size:11px;">${when}</div>
        <div style="display:flex;align-items:center;gap:8px;">
          <span class="probe-grade" style="color:${gc}">${esc(u.grade)}</span>
          <span class="badge" style="background:${vc}22;color:${vc}">${esc(u.verdict_label || u.verdict)}</span>
          <span class="probe-score-num" style="color:${gc}">${Math.round(u.score)}</span>
        </div>
      </div>`;
    }

    // Fallback: no score data
    const passed = u.checks_passed || 0;
    const total = u.checks_total || 0;
    const pct = total > 0 ? Math.round(passed / total * 100) : 0;
    const color = pct >= 80 ? 'var(--green)' : pct >= 50 ? 'var(--yellow)' : 'var(--red)';
    return `<div class="probe-upstream-item" onclick="viewUpstreamProbe('${esc(u.target)}')">
      <div class="url">${esc(u.target)}${probing}</div>
      <div style="color:var(--text-dim);font-size:11px;">${when}</div>
      <div class="score" style="color:${color}">${passed}/${total}</div>
    </div>`;
  }).join('');
}

async function viewUpstreamProbe(target) {
  try {
    const resp = await adminFetch(`/admin/probe/report?target_base=${encodeURIComponent(target)}`);
    if (!resp.ok) return;
    const data = await resp.json();
    renderProbeDetail(data);
  } catch(e) {
    console.error('viewUpstreamProbe error:', e);
  }
}

function renderProbeDetail(data) {
  const el = document.getElementById('probe-detail');
  el.style.display = 'block';
  lastProbeResult = data;

  const checks = data.checks || [];
  const score = data.score;

  document.getElementById('probe-detail-target').textContent = data.target || '';
  document.getElementById('probe-detail-model').textContent = data.model || '';
  document.getElementById('probe-detail-time').textContent = data.probed_at ? new Date(data.probed_at).toLocaleString() : (data.timestamp ? new Date(data.timestamp).toLocaleString() : '');
  document.getElementById('probe-detail-elapsed').textContent = data.elapsed_ms ? data.elapsed_ms + 'ms' : '';

  // Score overview
  if (score) {
    const gc = colorVar(score.grade_color);
    const vc = colorVar(score.verdict_color);
    const modeLabel = score.mode === 'quick' ? 'Quick Score' : 'Full Score';
    document.getElementById('probe-score').innerHTML =
      `<div class="probe-score-overview">
        <div class="probe-score-main">
          <span class="probe-grade-big" style="color:${gc}">${esc(score.grade)}</span>
          <span class="probe-score-big" style="color:${gc}">${score.total_score.toFixed(1)}</span>
          <span class="probe-score-max" style="color:var(--text-dim)">/ 100</span>
        </div>
        <div class="probe-verdict-row">
          <span class="badge probe-verdict-badge" style="background:${vc}22;color:${vc}">${esc(score.verdict_label)}</span>
          <span class="probe-mode-label">${esc(modeLabel)}</span>
          ${score.critical_penalty > 0 ? `<span class="badge badge-red">-${score.critical_penalty} Critical</span>` : ''}
        </div>
      </div>`;

    // Category progress bars
    const catHtml = (score.categories || []).map(cat => {
      const pct = Math.round(cat.percentage);
      const barColor = pct >= 80 ? 'var(--green)' : pct >= 50 ? 'var(--yellow)' : 'var(--red)';
      const checksHtml = (cat.checks || []).map(c => {
        const icon = c.pass ? '<span style="color:var(--green)">&#10003;</span>' : '<span style="color:var(--red)">&#10007;</span>';
        const fix = c.fix && !c.pass ? `<span class="fix">${esc(c.fix)}</span>` : '';
        return `<div class="probe-check ${c.pass ? 'pass' : 'fail'}">
          <div class="icon">${icon}</div>
          <div class="info">
            <div class="name">${esc(c.name)} ${fix}</div>
            <div class="detail" title="${esc(c.detail)}">${esc(c.detail)}</div>
          </div>
        </div>`;
      }).join('');

      return `<div class="probe-category">
        <div class="probe-cat-header" onclick="this.parentElement.classList.toggle('expanded')">
          <div class="probe-cat-info">
            <span class="probe-cat-label">${esc(cat.label)}</span>
            <span class="probe-cat-stats">${cat.passed}/${cat.total}</span>
            <span class="probe-cat-weight" title="Weight ${cat.weight}">${cat.score.toFixed(1)}/${cat.weight}</span>
          </div>
          <div class="probe-cat-bar-wrap">
            <div class="probe-cat-bar" style="width:${pct}%;background:${barColor}"></div>
          </div>
          <span class="probe-cat-pct" style="color:${barColor}">${pct}%</span>
        </div>
        <div class="probe-cat-checks">${checksHtml}</div>
      </div>`;
    }).join('');

    document.getElementById('probe-checks-grid').innerHTML = catHtml;
  } else {
    // Fallback: no score
    const passed = checks.filter(c => c.pass).length;
    const total = checks.length;
    const pct = total > 0 ? Math.round(passed / total * 100) : 0;
    const scoreColor = pct >= 80 ? 'var(--green)' : pct >= 50 ? 'var(--yellow)' : 'var(--red)';
    document.getElementById('probe-score').innerHTML = `<span style="color:${scoreColor}">${passed}</span><span style="color:var(--text-dim)">/${total}</span>`;

    // Flat checks grid
    const grid = document.getElementById('probe-checks-grid');
    grid.innerHTML = checks.map(c => {
      const cls = c.pass ? 'pass' : 'fail';
      const icon = c.pass ? '<span style="color:var(--green)">&#10003;</span>' : '<span style="color:var(--red)">&#10007;</span>';
      const fix = c.fix && !c.pass ? `<span class="fix">${esc(c.fix)}</span>` : '';
      return `<div class="probe-check ${cls}">
        <div class="icon">${icon}</div>
        <div class="info">
          <div class="name">${esc(c.name)} ${fix}</div>
          <div class="detail" title="${esc(c.detail)}">${esc(c.detail)}</div>
        </div>
      </div>`;
    }).join('');
  }

  document.getElementById('probe-summary-text').textContent = data.summary || '';

  // Show/hide apply button based on failures
  const failCount = checks.filter(c => !c.pass && c.fix).length;
  const applyBtn = document.getElementById('probe-apply-btn');
  const viewBtn = document.getElementById('probe-view-config-btn');
  const infoEl = document.getElementById('probe-apply-info');

  if (failCount > 0) {
    applyBtn.style.display = 'inline-flex';
    viewBtn.style.display = 'inline-flex';
    infoEl.textContent = `${failCount} fixes available`;
  } else {
    applyBtn.style.display = 'none';
    viewBtn.style.display = 'inline-flex';
    infoEl.textContent = 'All checks passed';
  }
}

async function runProbe() {
  const targetBase = document.getElementById('probe-target').value.trim();
  const targetKey = document.getElementById('probe-key').value.trim();
  const model = document.getElementById('probe-model').value;

  const btn = document.getElementById('probe-run-btn');
  const statusEl = document.getElementById('probe-status');
  btn.disabled = true;
  btn.textContent = 'Probing...';
  statusEl.innerHTML = '<span class="badge badge-yellow">Running probe, this may take 10-60s...</span>';

  try {
    const resp = await adminFetch('/admin/probe', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        target_base: targetBase || undefined,
        target_key: targetKey || undefined,
        model: model,
        quick: true
      })
    });
    if (!resp.ok) {
      const errText = await resp.text().catch(() => 'unknown error');
      statusEl.innerHTML = `<span class="badge badge-red">Error: ${esc(errText.slice(0, 200))}</span>`;
      return;
    }
    const data = await resp.json();
    if (data.ok) {
      statusEl.innerHTML = `<span class="badge badge-green">Probe complete in ${data.elapsed_ms}ms</span>`;
      renderProbeDetail(data);
      loadProbeReport();
    } else {
      statusEl.innerHTML = `<span class="badge badge-red">Error: ${esc(data.error || 'unknown')}</span>`;
    }
  } catch(e) {
    statusEl.innerHTML = `<span class="badge badge-red">Error: ${esc(e.message)}</span>`;
  } finally {
    btn.disabled = false;
    btn.textContent = 'Run Probe';
  }
}

async function applyProbeConfig() {
  if (!lastProbeResult || !lastProbeResult.recommended) {
    alert('No probe result to apply');
    return;
  }

  const choice = prompt(
    'Apply recommended config to:\n\n' +
    '1 = This upstream only\n' +
    '2 = This upstream + Global config\n' +
    '3 = View in Gateway page (recommended)\n\n' +
    'Enter 1, 2, or 3:'
  );

  if (choice === '3') {
    // Navigate to Gateway page
    switchPage('gateway');
    return;
  }

  if (choice !== '1' && choice !== '2') {
    return;
  }

  const applyGlobal = choice === '2';

  try {
    const resp = await adminFetch('/admin/probe/apply', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        target_base: lastProbeResult.target || '',
        recommended: lastProbeResult.recommended,
        apply_global: applyGlobal
      })
    });
    if (!resp.ok) {
      alert('Apply failed: ' + resp.status);
      return;
    }
    const data = await resp.json();
    if (data.ok) {
      const scope = applyGlobal ? 'global + upstream' : 'upstream only';
      alert(`Applied ${data.applied} disguise switches (${scope})\n\nGo to Gateway page to view/edit.`);
    } else {
      alert('Error: ' + (data.error || 'unknown'));
    }
  } catch(e) {
    alert('Error: ' + e.message);
  }
}

// Show recommended config details
function showRecommendedConfig() {
  if (!lastProbeResult || !lastProbeResult.recommended) {
    alert('No probe result available');
    return;
  }

  const rec = lastProbeResult.recommended;
  const switches = [];

  if (rec.bodyRewrite || rec.body_rewrite) switches.push('Body Rewrite');
  if (rec.idRewrite || rec.id_rewrite) switches.push('ID Rewrite');
  if (rec.signatureRewrite || rec.signature_rewrite) switches.push('Signature Rewrite');
  if (rec.headersFake || rec.headers_fake) switches.push('Headers Fake');
  if (rec.stripBedrock || rec.strip_bedrock) switches.push('Strip Bedrock');
  if (rec.stripDone || rec.strip_done) switches.push('Strip Done');
  if (rec.stripContainer || rec.strip_container) switches.push('Strip Container');
  if (rec.forceGeo || rec.force_geo) switches.push('Force Geo');
  if (rec.thinkingInject || rec.thinking_inject) switches.push('Thinking Inject');
  if (rec.smallProbeZero || rec.small_probe_zero) switches.push('Small Probe Zero');
  if (rec.cacheFake || rec.cache_fake) switches.push('Cache Fake');

  const msg = switches.length > 0
    ? 'Recommended switches to enable:\n\n' + switches.map(s => '  - ' + s).join('\n') + '\n\nGo to Gateway page to configure.'
    : 'No disguise switches needed! This upstream appears to be compliant.';

  alert(msg);
}
