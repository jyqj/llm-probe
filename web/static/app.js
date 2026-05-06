/* ─── State ─── */
let lastChannelResult = null;
let lastIntelligenceResult = null;
let currentDataset = '';
let allDatasets = [];
let intelligenceAbort = null;
let intelligenceStartTime = null;
let intelligenceTimer = null;
let intelligenceStreamResults = [];
let currentCheckFilter = 'all';

/* ─── Settings persistence ─── */
// Only adminToken is persisted; target/key/model are ephemeral per-session.
function loadSettings() {
  const s = JSON.parse(localStorage.getItem('llm-probe-settings') || '{}');
  if (s.adminToken) document.getElementById('adminToken').value = s.adminToken;
  updateConnStatus();
}
function saveSettings() {
  const s = { adminToken: document.getElementById('adminToken').value.trim() };
  localStorage.setItem('llm-probe-settings', JSON.stringify(s));
}
function updateConnStatus() {
  const base = document.getElementById('targetBase').value.trim();
  const el = document.getElementById('connStatus');
  const label = document.getElementById('connLabel');
  if (base) {
    el.classList.add('connected');
    try {
      const u = new URL(base);
      label.textContent = '已连接 · ' + u.hostname.substring(0, 20);
    } catch {
      label.textContent = '已配置';
    }
  } else {
    el.classList.remove('connected');
    label.textContent = '未配置';
  }
}

/* ─── Navigation ─── */
function switchApp(name) {
  document.querySelectorAll('[data-app-pane]').forEach(el => {
    el.hidden = el.dataset.appPane !== name;
  });
  document.querySelectorAll('.appswitch button').forEach(b => {
    b.classList.toggle('active', b.dataset.app === name);
  });
  if (name === 'settings') updateConnStatus();
}

function showChannelSub(sub) {
  ['Overview', 'Checks', 'Raw', 'History'].forEach(s => {
    const el = document.getElementById('channelSub' + s);
    if (el) el.classList.toggle('hidden', s.toLowerCase() !== sub);
  });
  // Update sidebar active state
  document.querySelectorAll('[data-app-pane="channel"] .side-link').forEach((link, i) => {
    const subs = ['overview', 'checks', 'raw', 'history'];
    link.classList.toggle('channel-active', subs[i] === sub);
    link.classList.toggle('active', false);
  });
  if (sub === 'history') loadChannelHistory();
}

/* ─── API Helpers ─── */
function headers() {
  const h = {'Content-Type': 'application/json'};
  const t = document.getElementById('adminToken').value.trim();
  if (t) h['X-Admin-Token'] = t;
  return h;
}
function targetPayload() {
  return {
    target_base: document.getElementById('targetBase').value.trim(),
    target_key: document.getElementById('targetKey').value.trim(),
    model: document.getElementById('model').value.trim()
  };
}
async function api(url, opt = {}) {
  const r = await fetch(url, {headers: headers(), ...opt});
  const text = await r.text();
  let data;
  try { data = JSON.parse(text); } catch { data = text; }
  if (!r.ok) throw data;
  return data;
}
function esc(s) { return s ? s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;') : ''; }

/* ─── Color maps ─── */
const gradeColorMap = {
  green: 'var(--good)', blue: 'var(--info)', yellow: 'var(--warn)',
  orange: 'var(--accent)', red: 'var(--bad)'
};
const verdictPillMap = {
  green: 'pill-good', blue: 'pill-info', yellow: 'pill-warn',
  orange: 'pill-warn', red: 'pill-bad'
};
const catSwatchMap = {
  fingerprint: 'var(--cat-fp)', structural: 'var(--cat-st)',
  signature: 'var(--cat-sg)', behavioral: 'var(--cat-bh)',
  multimodal: 'var(--cat-mm)'
};
const catNameMap = {
  fingerprint: '指纹 Fingerprint', structural: '结构 Structural',
  signature: '签名 Signature', behavioral: '行为 Behavioral',
  multimodal: '多模态 Multimodal'
};
const checkCatMap = {
  id_format:'fingerprint',backend_type:'fingerprint',inference_geo:'fingerprint',
  stop_details:'fingerprint',stop_details_structure:'fingerprint',
  small_output_tokens:'fingerprint',small_stop_reason:'fingerprint',
  container:'fingerprint',bedrock_state:'fingerprint',request_id:'fingerprint',
  x_new_api_version:'fingerprint',cf_ray_format:'fingerprint',cookie_domain:'fingerprint',
  hidden_prompt:'fingerprint',token_budget:'fingerprint',service_tier:'fingerprint',
  server_header:'fingerprint',signature_type_leak:'fingerprint',
  usage_structure:'structural',field_order:'structural',model_name:'structural',
  stop_reason:'structural',tool_stop_reason:'structural',delta_usage_slim:'structural',
  message_start_usage:'structural',message_start_output_zero:'structural',
  nonstream_fields:'structural',nonstream_type:'structural',nonstream_role:'structural',
  tool_use_id:'structural',web_search_result:'structural',
  structured_json_valid:'structural',structured_schema_match:'structural',
  structured_stop_reason:'structural',headers:'structural',cf_headers:'structural',
  server_timing:'structural',sse_done:'structural',sse_event_order:'structural',
  sse_tailing:'structural',sse_ping_position:'structural',cache_small_probe:'structural',
  cache_fake:'structural',small_ephemeral_zero:'structural',small_cache_zero:'structural',
  stop_sequence_null:'structural',usage_fields_complete:'structural',
  cache_creation_complete:'structural',server_tool_type:'structural',
  citations_present:'structural',body_key_order:'structural',server_tool_usage:'structural',
  signature:'signature',signature_length:'signature',thinking_present:'signature',
  thinking_order:'signature',thinking_display_omitted:'signature',no_thinking_leak:'signature',
  tag_replay:'behavioral',identity_response:'behavioral',identity_no_leak:'behavioral',
  identity_platform:'behavioral',poison_answer:'behavioral',logic_answer:'behavioral',
  tool_forced_compliance:'behavioral',
  image_ocr:'multimodal',pdf_extract:'multimodal'
};
function guessCategory(name) { return checkCatMap[name] || 'other'; }

/* ─── Channel Test ─── */
async function runChannel() {
  const btn = document.getElementById('btnChannel');
  btn.disabled = true;
  showChannelSub('overview');
  document.getElementById('channelPlaceholder').classList.add('hidden');
  document.getElementById('channelResult').classList.add('hidden');
  document.getElementById('channelLoading').classList.remove('hidden');

  try {
    const p = {...targetPayload(), quick: false};
    const data = await api('/api/channel/run', {method: 'POST', body: JSON.stringify(p)});
    lastChannelResult = data;
    renderChannelResult(data);
  } catch (e) {
    alert('检测失败: ' + (typeof e === 'string' ? e : JSON.stringify(e)));
    document.getElementById('channelPlaceholder').classList.remove('hidden');
  } finally {
    document.getElementById('channelLoading').classList.add('hidden');
    btn.disabled = false;
  }
}

function renderChannelResult(data) {
  const checks = data.checks || [];
  const passed = checks.filter(c => c.pass).length;
  const failed = checks.length - passed;

  // Score ring
  if (data.score) {
    const circumference = 603.2;
    const offset = circumference - (data.score.total_score / 100) * circumference;
    const arc = document.getElementById('channelScoreArc');
    arc.style.strokeDashoffset = offset;
    const color = gradeColorMap[data.score.grade_color] || 'var(--accent)';
    arc.style.stroke = color;
    document.getElementById('channelGrade').textContent = data.score.grade;
    document.getElementById('channelGrade').style.color = color;
    document.getElementById('channelPts').textContent = data.score.total_score;

    // Verdict pill
    const vp = document.getElementById('channelVerdict');
    vp.className = 'pill ' + (verdictPillMap[data.score.verdict_color] || 'pill-info');
    vp.innerHTML = '<span class="dot"></span><span>' + esc(data.score.verdict_label) + '</span>';

    // Summary
    document.getElementById('channelSummaryTitle').innerHTML =
      data.score.total_score >= 80
        ? '大概率为<em>官方直连</em>' + (failed > 0 ? ',但有 ' + failed + ' 处偏差' : '')
        : data.score.total_score >= 50
          ? '存在<em>明显偏差</em>,建议检查'
          : '疑似<em>非官方渠道</em>';
    document.getElementById('channelSummaryDesc').textContent = data.summary || '';

    // Score meta
    document.getElementById('channelScoreMeta').innerHTML = `
      <div class="score-meta-item"><div class="k">checks</div><div class="v">${passed} / ${checks.length}</div></div>
      <div class="score-meta-item"><div class="k">elapsed</div><div class="v">${((data.elapsed_ms || 0) / 1000).toFixed(1)} s</div></div>
      <div class="score-meta-item"><div class="k">target</div><div class="v mono" style="font-size:12px">${esc(data.target || '-')}</div></div>
    `;

    // Category bars
    renderCatBars('channelCatBars', data.score.categories || []);
  }

  // Stats
  document.getElementById('chkTotal').textContent = checks.length;
  document.getElementById('chkPassed').innerHTML = passed + '<small>/' + checks.length + '</small>';
  document.getElementById('chkFailed').textContent = failed;
  document.getElementById('chkElapsed').innerHTML =
    ((data.elapsed_ms || 0) / 1000).toFixed(1) + '<small>s</small>';

  // Fix panel
  const failures = checks.filter(c => !c.pass);
  renderFixPanel(failures);

  // Check details (sub checks view)
  renderChecks('channelChecks', checks);
  document.getElementById('channelChecksEmpty').classList.add('hidden');
  document.getElementById('channelChecksContent').classList.remove('hidden');

  // Raw data (sub raw view)
  document.getElementById('rawChannel').textContent = JSON.stringify(data, null, 2);
  document.getElementById('channelRawEmpty').classList.add('hidden');
  document.getElementById('channelRawContent').classList.remove('hidden');

  document.getElementById('channelResult').classList.remove('hidden');
  showChannelSub('overview');
}

function renderCatBars(containerId, categories) {
  const el = document.getElementById(containerId);
  el.innerHTML = categories.map(c => {
    const cat = (c.key || '').toLowerCase();
    const swatch = catSwatchMap[cat] || 'var(--ink-3)';
    const pctColor = c.percentage >= 80 ? 'var(--good)' : c.percentage >= 50 ? 'var(--warn)' : 'var(--bad)';
    return `<div class="cat">
      <div class="cat-name"><span class="swatch" style="background:${swatch}"></span>${esc(c.label)}</div>
      <div class="cat-track"><div class="cat-fill" style="width:${c.percentage}%;background:${swatch}"></div></div>
      <div class="cat-frac">${c.passed}/${c.total}</div>
      <div class="cat-pct" style="color:${pctColor}">${Math.round(c.percentage)}%</div>
    </div>`;
  }).join('');
}

function renderFixPanel(failures) {
  const list = document.getElementById('channelFixList');
  if (failures.length === 0) {
    list.innerHTML = '<div class="muted" style="font-size:13px;padding:8px 0">全部通过,无需修复</div>';
    return;
  }
  list.innerHTML = failures.slice(0, 5).map((c, i) => `
    <div class="fix-item">
      <div class="num">${String(i + 1).padStart(2, '0')}</div>
      <div>
        <div class="what">${esc(c.name)}</div>
        <div class="how">${esc(c.detail)}${c.fix ? ' — fix: ' + esc(c.fix) : ''}</div>
      </div>
    </div>
  `).join('');
  if (failures.length > 5) {
    list.innerHTML += `<div class="muted" style="font-size:12px;padding:4px 0;text-align:center">还有 ${failures.length - 5} 项...</div>`;
  }
}

const svgCheck = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><polyline points="20 6 9 17 4 12"/></svg>';
const svgCross = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>';

function renderChecks(containerId, checks) {
  const groups = {};
  checks.forEach(c => {
    const cat = guessCategory(c.name);
    if (!groups[cat]) groups[cat] = [];
    groups[cat].push(c);
  });

  const el = document.getElementById(containerId);
  const catOrder = ['fingerprint', 'structural', 'signature', 'behavioral', 'multimodal', 'other'];
  let html = '';
  catOrder.forEach(cat => {
    const items = groups[cat];
    if (!items) return;
    const passed = items.filter(c => c.pass).length;
    const swatch = catSwatchMap[cat] || 'var(--ink-3)';
    const label = catNameMap[cat] || cat;

    html += `<div class="checks-head" onclick="this.nextElementSibling.classList.toggle('hidden')">
      <h4><span class="swatch" style="background:${swatch}"></span>${label}</h4>
      <span class="frac">${passed} / ${items.length} 通过</span>
      <span class="toggle">折叠 ▾</span>
    </div>`;
    html += '<div>';
    items.forEach(c => {
      if (currentCheckFilter === 'pass' && !c.pass) return;
      if (currentCheckFilter === 'fail' && c.pass) return;
      html += `<div class="check">
        <div class="ind ${c.pass ? 'pass' : 'fail'}">${c.pass ? svgCheck : svgCross}</div>
        <div class="name">${esc(c.name)}</div>
        <div class="detail">${esc(c.detail)}</div>
        ${!c.pass && c.fix ? '<span class="fix-pill">' + esc(c.fix) + '</span>' : ''}
      </div>`;
    });
    html += '</div>';
  });
  el.innerHTML = html;
}

function filterChecks(mode, btn) {
  currentCheckFilter = mode;
  btn.parentElement.querySelectorAll('button').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  if (lastChannelResult && lastChannelResult.checks) {
    renderChecks('channelChecks', lastChannelResult.checks);
  }
}

function exportChannel() {
  if (!lastChannelResult) return;
  const blob = new Blob([JSON.stringify(lastChannelResult, null, 2)], {type: 'application/json'});
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `channel-${new Date().toISOString().slice(0,19)}.json`;
  a.click();
  URL.revokeObjectURL(url);
}

/* ─── Intelligence / Benchmark ─── */
function currentDS() { return currentDataset || 'SWE-Atlas-QnA'; }

async function loadIntelligenceList() {
  try {
    const data = await api('/api/intelligence/datasets');
    allDatasets = data.datasets || [];
    if (allDatasets.length > 0 && !currentDataset) {
      currentDataset = allDatasets[0].name;
    }
    renderDatasetSwitch();
    renderSideDatasets();
    loadIntelligenceInfo();
  } catch (e) {
    console.error('loadIntelligenceList', e);
  }
}

function renderDatasetSwitch() {
  const el = document.getElementById('datasetSwitch');
  let html = allDatasets.map(d => {
    const active = d.name === currentDataset ? 'active' : '';
    return `<div class="ds-tab ${active}" onclick="switchDataset('${esc(d.name)}')">
      <div class="name">${esc(d.name)}</div>
      <div class="ds-meta">${d.total_tasks} 题</div>
    </div>`;
  }).join('');
  html += '<div class="ds-tab add" onclick="showUploadDialog()" title="添加数据集">+</div>';
  el.innerHTML = html;
}

function renderSideDatasets() {
  const el = document.getElementById('sideDatasets');
  let html = '<div class="side-label">Datasets</div>';
  allDatasets.forEach(d => {
    html += `<a class="side-link" style="font-size:12px" onclick="switchDataset('${esc(d.name)}')">
      <span class="tag" style="background:var(--accent-soft);color:var(--accent-ink)">${esc((d.name || '').substring(0, 3).toUpperCase())}</span>
      ${esc(d.name)}
    </a>`;
  });
  el.innerHTML = html;
  document.getElementById('sideDatasetCount').textContent =
    allDatasets.length + ' datasets · ' + allDatasets.reduce((s, d) => s + (d.total_tasks || 0), 0) + ' tasks';
}

function switchDataset(name) {
  currentDataset = name;
  renderDatasetSwitch();
  loadIntelligenceInfo();
}

async function loadIntelligenceInfo() {
  const ds = currentDS();
  try {
    const d = await api(`/api/intelligence/datasets/${encodeURIComponent(ds)}`);
    const s = d.stats;
    const langTags = Object.entries(s.languages || {}).sort((a,b) => b[1]-a[1])
      .map(([k,v]) => `<span class="tag tag-lang">${k} · ${v}</span>`).join('');
    const catTags = Object.entries(s.categories || {}).sort((a,b) => b[1]-a[1])
      .map(([k,v]) => `<span class="tag tag-cat">${k} · ${v}</span>`).join('');

    document.getElementById('datasetInfo').innerHTML = `
      <div style="display:flex;align-items:flex-start;justify-content:space-between;gap:24px">
        <div style="flex:1">
          <div style="display:flex;align-items:center;gap:10px;margin-bottom:6px">
            <h2 class="serif" style="font-size:24px;font-weight:400;letter-spacing:-0.02em">${esc(s.name)} ${s.version ? '<span class="muted serif" style="font-style:italic">v' + esc(s.version) + '</span>' : ''}</h2>
            <span class="pill pill-info">已加载</span>
          </div>
          <div style="margin-top:14px;display:flex;flex-wrap:wrap;gap:6px">
            ${langTags}
            ${langTags && catTags ? '<span style="width:1px;background:var(--line);margin:0 4px"></span>' : ''}
            ${catTags}
          </div>
        </div>
        <div style="text-align:right">
          <div class="serif tnum" style="font-size:44px;line-height:1;letter-spacing:-0.03em">${s.total_tasks}</div>
          <div class="muted" style="font-size:11px;letter-spacing:0.06em;text-transform:uppercase;margin-top:4px">tasks</div>
        </div>
      </div>
    `;
    updateIntelligencePreview();
  } catch (e) {
    document.getElementById('datasetInfo').innerHTML =
      '<span style="color:var(--bad)">加载失败: ' + esc(ds) + '</span>';
  }
}

function showUploadDialog() {
  document.getElementById('uploadCard').classList.toggle('hidden');
}

function showAddTab(tabId, btn) {
  document.getElementById('tabFetch').classList.add('hidden');
  document.getElementById('tabUpload').classList.add('hidden');
  document.getElementById(tabId).classList.remove('hidden');
  btn.parentElement.querySelectorAll('.upload-tab').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
}

async function fetchFromHF() {
  const statusEl = document.getElementById('fetchStatus');
  const name = document.getElementById('fetchName').value;
  const limit = parseInt(document.getElementById('fetchLimit').value) || 0;
  statusEl.textContent = '拉取中,请稍候...';

  try {
    const resp = await fetch('/api/intelligence/fetch', {
      method: 'POST', headers: headers(),
      body: JSON.stringify({name, limit: limit > 0 ? limit : undefined})
    });
    const data = await resp.json();
    if (!resp.ok) throw data;
    statusEl.textContent = `成功: ${data.stats.total_tasks} tasks`;
    currentDataset = data.stats.name;
    await loadIntelligenceList();
    document.getElementById('uploadCard').classList.add('hidden');
  } catch (e) {
    statusEl.textContent = '拉取失败: ' + (e.error || JSON.stringify(e));
  }
}

async function uploadDataset() {
  const name = document.getElementById('uploadName').value.trim();
  const fileInput = document.getElementById('uploadFile');
  const statusEl = document.getElementById('uploadStatus');
  if (!name) { statusEl.textContent = '请输入数据集名称'; return; }
  if (!fileInput.files.length) { statusEl.textContent = '请选择文件'; return; }

  statusEl.textContent = '上传中...';
  const formData = new FormData();
  formData.append('file', fileInput.files[0]);
  formData.append('name', name);

  try {
    const h = {};
    const t = document.getElementById('adminToken').value.trim();
    if (t) h['X-Admin-Token'] = t;
    const resp = await fetch(`/api/intelligence/datasets/${encodeURIComponent(name)}/upload`, {
      method: 'POST', headers: h, body: formData
    });
    const data = await resp.json();
    if (!resp.ok) throw data;
    statusEl.textContent = `上传成功: ${data.stats.total_tasks} tasks`;
    currentDataset = name;
    await loadIntelligenceList();
    document.getElementById('uploadCard').classList.add('hidden');
  } catch (e) {
    statusEl.textContent = '上传失败: ' + (e.error || JSON.stringify(e));
  }
}

function toggleRunScope() {
  const scope = document.getElementById('runScope').value;
  const filters = document.getElementById('runFilters');
  filters.style.display = scope === 'all' ? 'none' : '';
  updateIntelligencePreview();
}

function updateIntelligencePreview() {
  const scope = document.getElementById('runScope').value;
  const el = document.getElementById('intelligencePreview');
  const el2 = document.getElementById('runPreview');
  if (scope === 'all') {
    el.textContent = '将运行全部题目';
    el2.textContent = '全量运行';
    return;
  }
  const lang = document.getElementById('runLang').value.trim();
  const cat = document.getElementById('runCategory').value.trim();
  const limit = parseInt(document.getElementById('runLimit').value) || 0;
  let desc = [];
  if (lang) desc.push('lang=' + lang);
  if (cat) desc.push('cat=' + cat);
  if (limit > 0) desc.push('limit=' + limit);
  const text = desc.length ? desc.join(', ') : '全量运行';
  el.textContent = text;
  el2.textContent = text;
}

function benchTargetPayload() {
  return {
    target_base: document.getElementById('benchTargetBase').value.trim(),
    target_key: document.getElementById('benchTargetKey').value.trim(),
    model: document.getElementById('benchModel').value.trim()
  };
}

function buildRunPayload() {
  const scope = document.getElementById('runScope').value;
  const p = {
    ...benchTargetPayload(),
    concurrency: parseInt(document.getElementById('runConcurrency').value) || 5,
    thinking: document.getElementById('runThinking').value === 'on',
  };
  // Override model if set in run config
  const runModel = document.getElementById('runModel').value.trim();
  if (runModel) p.model = runModel;

  if (scope !== 'all') {
    const lang = document.getElementById('runLang').value.trim();
    const cat = document.getElementById('runCategory').value.trim();
    const limit = parseInt(document.getElementById('runLimit').value) || 0;
    if (lang) p.language = lang;
    if (cat) p.category = cat;
    if (limit > 0) p.limit = limit;
  }
  return p;
}

async function runIntelligenceStream() {
  const btn = document.getElementById('btnIntelligence');
  const stopBtn = document.getElementById('btnIntelligenceStop');
  btn.disabled = true;
  stopBtn.classList.remove('hidden');

  const progressCard = document.getElementById('intelligenceProgressCard');
  const resultCard = document.getElementById('intelligenceResultCard');
  progressCard.classList.remove('hidden');
  resultCard.classList.add('hidden');

  intelligenceStreamResults = [];
  intelligenceStartTime = Date.now();
  intelligenceTimer = setInterval(updateElapsed, 1000);
  updateElapsed();

  document.getElementById('intelligenceProgressBar').style.width = '0%';
  document.getElementById('progressCompleted').textContent = '0';
  document.getElementById('progressTotal').textContent = '/0';
  document.getElementById('progressDone').textContent = '0';
  document.getElementById('progressErrors').textContent = '0';
  document.getElementById('progressAvg').textContent = '-';
  document.getElementById('progressConcurrency').textContent =
    document.getElementById('runConcurrency').value;
  document.getElementById('intelligenceResultList').innerHTML = '';
  document.getElementById('progressShimmer').style.display = '';

  const payload = buildRunPayload();
  intelligenceAbort = new AbortController();

  try {
    const resp = await fetch(`/api/intelligence/datasets/${encodeURIComponent(currentDS())}/stream`, {
      method: 'POST', headers: headers(),
      body: JSON.stringify(payload),
      signal: intelligenceAbort.signal,
    });

    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(text);
    }

    const reader = resp.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const {done, value} = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, {stream: true});
      const lines = buffer.split('\n');
      buffer = lines.pop();
      for (const line of lines) {
        if (!line.startsWith('data: ')) continue;
        try { handleIntelligenceEvent(JSON.parse(line.substring(6))); } catch {}
      }
    }
    if (buffer.startsWith('data: ')) {
      try { handleIntelligenceEvent(JSON.parse(buffer.substring(6))); } catch {}
    }
  } catch (e) {
    if (e.name !== 'AbortError') {
      document.getElementById('intelligenceResultList').innerHTML +=
        `<div style="color:var(--bad);padding:12px">Error: ${esc(e.message || JSON.stringify(e))}</div>`;
    }
  } finally {
    clearInterval(intelligenceTimer);
    btn.disabled = false;
    stopBtn.classList.add('hidden');
    intelligenceAbort = null;
    document.getElementById('progressShimmer').style.display = 'none';
  }
}

function stopIntelligence() {
  if (intelligenceAbort) intelligenceAbort.abort();
}

function updateElapsed() {
  if (!intelligenceStartTime) return;
  const sec = Math.floor((Date.now() - intelligenceStartTime) / 1000);
  const min = Math.floor(sec / 60);
  const s = sec % 60;
  document.getElementById('intelligenceElapsed').textContent =
    min > 0 ? `${min}:${String(s).padStart(2,'0')}` : `${s}s`;
}

function handleIntelligenceEvent(ev) {
  if (ev.type === 'progress') {
    const pct = ev.total > 0 ? Math.round(ev.completed / ev.total * 100) : 0;
    document.getElementById('intelligenceProgressBar').style.width = pct + '%';
    document.getElementById('progressCompleted').textContent = ev.completed;
    document.getElementById('progressTotal').textContent = '/' + ev.total;
    document.getElementById('progressDone').textContent = ev.completed;
    if (ev.errors > 0) {
      document.getElementById('progressErrors').textContent = ev.errors;
      document.getElementById('progressErrors').classList.add('bad');
    }

    // Avg time
    if (intelligenceStreamResults.length > 0) {
      const avg = intelligenceStreamResults.reduce((s, r) => s + (r.elapsed_ms || 0), 0) / intelligenceStreamResults.length;
      document.getElementById('progressAvg').textContent = (avg / 1000).toFixed(1) + ' s';
    }

    if (ev.result) {
      intelligenceStreamResults.push(ev.result);
      appendResultCard(ev.result);
    }
  } else if (ev.type === 'complete') {
    if (ev.report) {
      lastIntelligenceResult = ev.report;
      document.getElementById('rawIntelligence').textContent = JSON.stringify(ev.report, null, 2);
      renderIntelligenceSummary(ev.report);
    }
    document.getElementById('intelligenceProgressBar').style.width = '100%';
  }
}

function appendResultCard(r) {
  const el = document.getElementById('intelligenceResultList');
  const hasErr = !!r.error;
  const card = document.createElement('div');
  card.className = 'task';
  card.innerHTML = `
    <div class="head">
      <span class="tag tag-lang">${esc(r.task.language)}</span>
      <span class="tag tag-cat">${esc(r.task.category)}</span>
      ${hasErr
        ? '<span class="tag tag-err">Error</span>'
        : '<span class="tag tag-ok">OK</span>'}
      <span class="id">${esc((r.task.task_id || '').substring(0, 12))}...</span>
      <span style="margin-left:auto" class="muted mono" style="font-size:11px">${r.elapsed_ms} ms</span>
    </div>
    <div class="prompt">${esc((r.task.prompt || '').substring(0, 200))}</div>
    <div class="footer">
      ${r.answer ? '<span class="muted">回答 ' + r.answer.length + ' 字符</span>' : ''}
      ${r.error ? '<span style="color:var(--bad)">' + esc(r.error) + '</span>' : ''}
    </div>
    ${r.answer ? '<details><summary class="muted" style="cursor:pointer;font-size:12px;margin-top:6px">查看回答</summary><div class="answer">' + esc(r.answer) + '</div></details>' : ''}
  `;
  el.appendChild(card);
  el.scrollTop = el.scrollHeight;
  document.getElementById('intelligenceResultCard').classList.remove('hidden');
}

function renderIntelligenceSummary(report) {
  document.getElementById('intelligenceResultMeta').textContent =
    `${report.task_total} tasks | ${report.model} | ${(report.elapsed_ms / 1000).toFixed(1)}s`;

  const avgMs = report.results && report.results.length > 0
    ? Math.round(report.results.reduce((s, r) => s + r.elapsed_ms, 0) / report.results.length)
    : 0;

  document.getElementById('intelligenceResultStats').innerHTML = `
    <div><div class="muted" style="font-size:11px">完成</div><div class="serif tnum" style="font-size:22px">${report.task_completed || report.task_total}</div></div>
    <div><div class="muted" style="font-size:11px">错误</div><div class="serif tnum" style="font-size:22px;${report.task_errors > 0 ? 'color:var(--bad)' : ''}">${report.task_errors || 0}</div></div>
    <div><div class="muted" style="font-size:11px">总耗时</div><div class="serif tnum" style="font-size:22px">${(report.elapsed_ms / 1000).toFixed(1)}s</div></div>
    <div><div class="muted" style="font-size:11px">Avg/Task</div><div class="serif tnum" style="font-size:22px">${(avgMs / 1000).toFixed(1)}s</div></div>
  `;
  document.getElementById('distLegend').textContent =
    `${report.task_completed || report.task_total} 已完成`;
}

function exportIntelligenceResult() {
  if (!lastIntelligenceResult) return;
  const blob = new Blob([JSON.stringify(lastIntelligenceResult, null, 2)], {type: 'application/json'});
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `intelligence-${currentDS()}-${new Date().toISOString().slice(0,19)}.json`;
  a.click();
  URL.revokeObjectURL(url);
}

/* ─── History ─── */
async function loadChannelHistory() {
  try {
    const data = await api('/api/channel/history');
    const list = data.history || [];
    const el = document.getElementById('channelHistoryList');
    if (list.length === 0) {
      el.innerHTML = '<div class="muted">暂无历史记录</div>';
      return;
    }
    el.innerHTML = list.map(r => {
      const time = r.timestamp ? new Date(r.timestamp).toLocaleString() : '-';
      const passed = (r.checks || []).filter(c => c.pass).length;
      const total = (r.checks || []).length;
      const score = r.score ? r.score.total_score.toFixed(1) : '-';
      return `<div class="history-row" style="display:flex;align-items:center;gap:12px;padding:8px 0;border-bottom:1px solid var(--line-soft);cursor:pointer" onclick="viewChannelHistory('${esc(r.id)}')">
        <div style="flex:1;min-width:0">
          <div style="font-size:13px;font-weight:500">${esc(r.target || '-')}</div>
          <div class="muted" style="font-size:11px">${time} · ${esc(r.model || '-')}</div>
        </div>
        <div style="font-size:12px">${passed}/${total}</div>
        <div style="font-size:12px;font-weight:500">${score} pts</div>
        <div style="font-size:11px;color:var(--ink-3)">${((r.elapsed_ms || 0) / 1000).toFixed(1)}s</div>
        <button class="btn btn-quiet btn-sm" onclick="event.stopPropagation();deleteChannelHistory('${esc(r.id)}')" title="删除">✕</button>
      </div>`;
    }).join('');
  } catch (e) {
    document.getElementById('channelHistoryList').innerHTML = '<div style="color:var(--bad)">加载失败</div>';
  }
}

async function viewChannelHistory(id) {
  try {
    const data = await api('/api/channel/history/' + encodeURIComponent(id));
    lastChannelResult = data;
    renderChannelResult(data);
    showChannelSub('overview');
  } catch (e) {
    alert('加载历史详情失败');
  }
}

async function deleteChannelHistory(id) {
  if (!confirm('确定删除此条历史记录?')) return;
  try {
    await api('/api/channel/history/' + encodeURIComponent(id), {method: 'DELETE'});
    loadChannelHistory();
  } catch (e) {
    alert('删除失败');
  }
}

/* ─── Benchmark History ─── */
function showBenchSub(sub) {
  const mainEl = document.getElementById('benchSubMain');
  const historyEl = document.getElementById('benchSubHistory');
  if (sub === 'history') {
    mainEl.classList.add('hidden');
    historyEl.classList.remove('hidden');
    loadBenchHistory();
  } else {
    mainEl.classList.remove('hidden');
    historyEl.classList.add('hidden');
  }
  // Update sidebar active state
  const links = document.querySelectorAll('[data-app-pane="bench"] .side-section:first-child .side-link');
  links.forEach(link => {
    const text = link.textContent.trim();
    link.classList.toggle('active', sub === 'history' ? text.includes('历史') : text.includes('本次'));
  });
}

async function loadBenchHistory() {
  try {
    const data = await api('/api/intelligence/history');
    const list = data.history || [];
    const el = document.getElementById('benchHistoryList');
    if (list.length === 0) {
      el.innerHTML = '<div class="muted">暂无历史记录</div>';
      return;
    }
    el.innerHTML = list.map(r => {
      return `<div class="history-row" style="display:flex;align-items:center;gap:12px;padding:8px 0;border-bottom:1px solid var(--line-soft);cursor:pointer" onclick="viewBenchHistory('${esc(r.id)}')">
        <div style="flex:1;min-width:0">
          <div style="font-size:13px;font-weight:500">${esc(r.dataset_name || '-')}</div>
          <div class="muted" style="font-size:11px">${esc(r.started_at || '-')} · ${esc(r.model || '-')}${r.thinking ? ' · thinking' : ''}</div>
        </div>
        <div style="font-size:12px">${r.task_completed}/${r.task_total}</div>
        <div style="font-size:12px;${r.task_errors > 0 ? 'color:var(--bad)' : ''}">${r.task_errors} err</div>
        <div style="font-size:11px;color:var(--ink-3)">${((r.elapsed_ms || 0) / 1000).toFixed(1)}s</div>
        <button class="btn btn-quiet btn-sm" onclick="event.stopPropagation();deleteBenchHistory('${esc(r.id)}')" title="删除">✕</button>
      </div>`;
    }).join('');
  } catch (e) {
    document.getElementById('benchHistoryList').innerHTML = '<div style="color:var(--bad)">加载失败</div>';
  }
}

async function viewBenchHistory(id) {
  try {
    const data = await api('/api/intelligence/history/' + encodeURIComponent(id));
    lastIntelligenceResult = data;
    showBenchSub('main');
    // Render results
    document.getElementById('rawIntelligence').textContent = JSON.stringify(data, null, 2);
    document.getElementById('rawIntelligence').classList.remove('hidden');
    renderIntelligenceSummary(data);
    document.getElementById('intelligenceProgressCard').classList.remove('hidden');
    // Render individual results
    const resultList = document.getElementById('intelligenceResultList');
    resultList.innerHTML = '';
    (data.results || []).forEach(r => appendResultCard(r));
    document.getElementById('intelligenceResultCard').classList.remove('hidden');
    document.getElementById('intelligenceResultMeta').textContent =
      `${data.task_total} tasks | ${data.model} | ${(data.elapsed_ms / 1000).toFixed(1)}s`;
  } catch (e) {
    alert('加载历史详情失败');
  }
}

async function deleteBenchHistory(id) {
  if (!confirm('确定删除此条历史记录?')) return;
  try {
    await api('/api/intelligence/history/' + encodeURIComponent(id), {method: 'DELETE'});
    loadBenchHistory();
  } catch (e) {
    alert('删除失败');
  }
}

/* ─── Init ─── */
loadSettings();
loadIntelligenceList();
toggleRunScope();
updateIntelligencePreview();

// Auto-save admin token; update conn status on target change
document.getElementById('adminToken').addEventListener('change', saveSettings);
document.getElementById('targetBase').addEventListener('input', updateConnStatus);
