/* ─── State ─── */
let lastChannelResult = null;
let lastIntelligenceResult = null;
let lastAuditResult = null;

/* ─── Navigation ─── */
function showView(name) {
  document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
  document.getElementById('view-' + name).classList.add('active');
  document.querySelectorAll('.sidebar nav button').forEach(b => {
    b.classList.toggle('active', b.dataset.view === name);
  });
}
function showRawTab(id, btn) {
  document.querySelectorAll('#view-raw pre').forEach(p => p.classList.add('hidden'));
  document.getElementById(id).classList.remove('hidden');
  document.querySelectorAll('#view-raw .tab-btn').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
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

/* ─── Render Helpers ─── */
const colorMap = {green:'var(--green)',blue:'var(--blue)',yellow:'var(--yellow)',orange:'var(--orange)',red:'var(--red)'};
const verdictClassMap = {green:'verdict-green',blue:'verdict-blue',yellow:'verdict-yellow',orange:'verdict-orange',red:'verdict-red'};

function renderScoreRing(arcId, gradeId, ptsId, score) {
  const circumference = 364.4;
  const offset = circumference - (score.total_score / 100) * circumference;
  const arc = document.getElementById(arcId);
  arc.style.strokeDashoffset = offset;
  arc.style.stroke = colorMap[score.grade_color] || 'var(--accent)';
  document.getElementById(gradeId).textContent = score.grade;
  document.getElementById(gradeId).style.color = colorMap[score.grade_color] || 'var(--text)';
  document.getElementById(ptsId).textContent = score.total_score + ' pts';
}

function renderCatBars(containerId, categories) {
  const el = document.getElementById(containerId);
  el.innerHTML = categories.map(c => {
    const barColor = c.percentage >= 80 ? 'var(--green)' : c.percentage >= 50 ? 'var(--yellow)' : 'var(--red)';
    return `<div class="cat-item">
      <div class="cat-label">${c.label}</div>
      <div class="cat-bar-wrap"><div class="cat-bar" style="width:${c.percentage}%;background:${barColor}"></div></div>
      <div class="cat-pct" style="color:${barColor}">${Math.round(c.percentage)}%</div>
    </div>`;
  }).join('');
}

function renderChecks(containerId, checks) {
  // Group by category
  const groups = {};
  const catNames = {'fingerprint':'LLM 指纹验证','structural':'结构完整性','signature':'签名校验','behavioral':'行为验证','multimodal':'多模态能力'};
  const catMap = {};
  // Build category mapping from check names
  checks.forEach(c => {
    const cat = guessCategory(c.name);
    if (!groups[cat]) groups[cat] = [];
    groups[cat].push(c);
  });
  const el = document.getElementById(containerId);
  el.innerHTML = Object.entries(groups).map(([cat, items]) => {
    const passed = items.filter(c => c.pass).length;
    const label = catNames[cat] || cat;
    return `<div class="check-group">
      <div class="check-group-header" onclick="this.querySelector('.arrow').classList.toggle('open');this.nextElementSibling.classList.toggle('hidden')">
        <span class="arrow open">&#9654;</span>
        <span class="cat-name">${label}</span>
        <span class="cat-count">${passed}/${items.length}</span>
      </div>
      <div class="check-items">${items.map(c => `<div class="check-row">
        <span class="check-dot ${c.pass?'pass':'fail'}"></span>
        <span class="check-name">${c.name}</span>
        <span class="check-detail" title="${esc(c.detail)}">${esc(c.detail)}</span>
        ${c.fix && !c.pass ? `<span class="check-fix">${c.fix}</span>` : ''}
      </div>`).join('')}</div>
    </div>`;
  }).join('');
}

const checkCatMap = {
  id_format:'fingerprint',backend_type:'fingerprint',inference_geo:'fingerprint',
  stop_details:'fingerprint',stop_details_structure:'fingerprint',
  small_output_tokens:'fingerprint',small_stop_reason:'fingerprint',
  container:'fingerprint',bedrock_state:'fingerprint',request_id:'fingerprint',
  x_new_api_version:'fingerprint',cf_ray_format:'fingerprint',cookie_domain:'fingerprint',
  hidden_prompt:'fingerprint',
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
  stop_sequence_null:'structural',
  signature:'signature',signature_length:'signature',thinking_present:'signature',
  thinking_order:'signature',thinking_display_omitted:'signature',no_thinking_leak:'signature',
  tag_replay:'behavioral',identity_response:'behavioral',identity_no_leak:'behavioral',
  identity_platform:'behavioral',poison_answer:'behavioral',logic_answer:'behavioral',
  tool_forced_compliance:'behavioral',magic_refusal:'behavioral',
  image_ocr:'multimodal',pdf_extract:'multimodal'
};
function guessCategory(name) { return checkCatMap[name] || 'other'; }
function esc(s) { return s ? s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;') : ''; }

/* ─── Audit ─── */
async function runAudit() {
  const btn = document.getElementById('btnAudit');
  btn.disabled = true;
  document.getElementById('auditPlaceholder').classList.add('hidden');
  document.getElementById('auditResult').classList.add('hidden');
  document.getElementById('auditLoading').classList.remove('hidden');

  try {
    const p = {
      ...targetPayload(),
      quick_channel: document.getElementById('auditMode').value === 'quick',
      intelligence_limit: parseInt(document.getElementById('auditIntelligenceLimit').value) || 0
    };
    const data = await api('/api/audit/run', {method: 'POST', body: JSON.stringify(p)});
    lastAuditResult = data;
    document.getElementById('rawAudit').textContent = JSON.stringify(data, null, 2);

    // Render channel score
    const det = data.channel;
    if (det && det.score) {
      renderScoreRing('scoreArc', 'scoreGrade', 'scorePts', det.score);
      const vc = verdictClassMap[det.score.verdict_color] || 'verdict-blue';
      const vEl = document.getElementById('scoreVerdict');
      vEl.className = 'verdict-badge ' + vc;
      vEl.textContent = det.score.verdict_label;
      document.getElementById('scoreSummary').textContent = det.summary || '';
      renderCatBars('catBars', det.score.categories || []);

      // Quick stats
      document.getElementById('quickStats').innerHTML = `
        <div class="stat"><div class="stat-val" style="color:${colorMap[det.score.grade_color]}">${det.score.total_score}</div><div class="stat-label">检测得分</div></div>
        <div class="stat"><div class="stat-val">${det.score.checks_passed}/${det.score.checks_total}</div><div class="stat-label">检查通过</div></div>
        <div class="stat"><div class="stat-val">${det.elapsed_ms || '-'}ms</div><div class="stat-label">检测耗时</div></div>
        <div class="stat"><div class="stat-val">${data.elapsed_ms || '-'}ms</div><div class="stat-label">总耗时</div></div>
      `;
    }

    // Render intelligence summary
    const intelligence = data.intelligence;
    if (intelligence && intelligence.results) {
      document.getElementById('intelligenceSummaryMeta').textContent =
        `${intelligence.task_total} tasks | ${intelligence.model} | ${intelligence.elapsed_ms}ms`;
      document.getElementById('intelligenceSummaryBody').innerHTML = intelligence.results.map(r => {
        const hasErr = !!r.error;
        return `<div class="task-card">
          <div class="task-meta">
            <span class="tag tag-lang">${r.task.language}</span>
            <span class="tag tag-cat">${r.task.category}</span>
            <span class="text-sm text-muted">${r.elapsed_ms}ms</span>
            ${hasErr ? '<span class="tag" style="background:var(--red-bg);color:var(--red)">Error</span>' : ''}
          </div>
          <div class="task-prompt">${esc(r.task.prompt.substring(0, 200))}...</div>
          ${r.answer ? `<div class="task-answer">${esc(r.answer.substring(0, 500))}</div>` : ''}
          ${r.error ? `<div class="task-answer" style="color:var(--red)">${esc(r.error)}</div>` : ''}
        </div>`;
      }).join('');
    } else if (data.intelligence_error) {
      document.getElementById('intelligenceSummaryBody').innerHTML = `<div class="text-muted">智商测试 error: ${esc(data.intelligence_error)}</div>`;
    }

    document.getElementById('auditResult').classList.remove('hidden');

    // Also update channel page data
    if (det && det.checks) {
      lastChannelResult = det;
      renderChannelResult(det);
    }
  } catch (e) {
    alert('审计失败: ' + (typeof e === 'string' ? e : JSON.stringify(e)));
  } finally {
    document.getElementById('auditLoading').classList.add('hidden');
    btn.disabled = false;
  }
}

/* ─── Channel Test ─── */
async function runChannel() {
  const btn = document.getElementById('btnChannel');
  btn.disabled = true;
  document.getElementById('channelPlaceholder').classList.add('hidden');
  document.getElementById('channelResult').classList.add('hidden');
  document.getElementById('channelLoading').classList.remove('hidden');

  try {
    const p = {...targetPayload(), quick: document.getElementById('channelMode').value === 'true'};
    const data = await api('/api/channel/run', {method: 'POST', body: JSON.stringify(p)});
    lastChannelResult = data;
    document.getElementById('rawChannel').textContent = JSON.stringify(data, null, 2);
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
  if (data.score) {
    renderScoreRing('channelScoreArc', 'channelGrade', 'channelPts', data.score);
    const vc = verdictClassMap[data.score.verdict_color] || 'verdict-blue';
    const vEl = document.getElementById('channelVerdict');
    vEl.className = 'verdict-badge ' + vc;
    vEl.textContent = data.score.verdict_label;
    renderCatBars('channelCatBars', data.score.categories || []);
  }
  document.getElementById('channelMeta').textContent =
    `Target: ${data.target || '-'} | Model: ${data.model || '-'} | ${data.elapsed_ms || '-'}ms`;
  if (data.checks) {
    document.getElementById('channelCheckCount').textContent =
      `${data.checks.filter(c => c.pass).length}/${data.checks.length} passed`;
    renderChecks('channelChecks', data.checks);
  }
  document.getElementById('channelResult').classList.remove('hidden');
}

/* ─── 智商测试 ─── */
let intelligenceAbort = null;
let intelligenceStartTime = null;
let intelligenceTimer = null;
let intelligenceStreamResults = [];
let currentDataset = '';
let allDatasets = [];

function currentDS() { return currentDataset || 'SWE-Atlas-QnA'; }

async function loadIntelligenceList() {
  try {
    const data = await api('/api/intelligence/datasets');
    allDatasets = data.datasets || [];
    const sel = document.getElementById('intelligenceDatasetSelect');
    sel.innerHTML = allDatasets.map(d =>
      `<option value="${esc(d.name)}" ${d.name === currentDataset ? 'selected' : ''}>${esc(d.name)} (${d.total_tasks})</option>`
    ).join('');
    if (allDatasets.length > 0 && !currentDataset) {
      currentDataset = allDatasets[0].name;
    }
    loadIntelligenceInfo();
  } catch (e) {
    console.error('loadIntelligenceList', e);
  }
}

function switchDataset() {
  currentDataset = document.getElementById('intelligenceDatasetSelect').value;
  loadIntelligenceInfo();
}

function showUploadDialog() {
  document.getElementById('uploadCard').classList.toggle('hidden');
}

function showAddTab(tabId, btn) {
  document.getElementById('tabFetch').classList.add('hidden');
  document.getElementById('tabUpload').classList.add('hidden');
  document.getElementById(tabId).classList.remove('hidden');
  btn.parentElement.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
}

async function fetchFromHF() {
  const statusEl = document.getElementById('fetchStatus');
  const name = document.getElementById('fetchName').value;
  const limit = parseInt(document.getElementById('fetchLimit').value) || 0;
  statusEl.textContent = '拉取中，请稍候...';

  try {
    const h = headers();
    const resp = await fetch('/api/intelligence/fetch', {
      method: 'POST', headers: h,
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
  const idsWrap = document.getElementById('runIDsWrap');
  if (scope === 'all') {
    filters.style.display = 'none';
  } else {
    filters.style.display = '';
    idsWrap.style.display = scope === 'ids' ? '' : 'none';
  }
  updateIntelligencePreview();
}

function updateIntelligencePreview() {
  const scope = document.getElementById('runScope').value;
  const el = document.getElementById('intelligencePreview');
  if (scope === 'all') { el.textContent = '将运行全部 124 题'; return; }
  const lang = document.getElementById('runLang').value.trim();
  const cat = document.getElementById('runCategory').value.trim();
  const limit = parseInt(document.getElementById('runLimit').value) || 0;
  let desc = [];
  if (lang) desc.push('lang=' + lang);
  if (cat) desc.push('cat=' + cat);
  if (limit > 0) desc.push('limit=' + limit);
  el.textContent = desc.length ? desc.join(', ') : '全部 124 题';
}

async function loadIntelligenceInfo() {
  const ds = currentDS();
  try {
    const d = await api(`/api/intelligence/datasets/${encodeURIComponent(ds)}`);
    const s = d.stats;
    const langHtml = Object.entries(s.languages || {}).sort((a,b) => b[1]-a[1]).map(([k,v]) => `<span class="tag tag-lang" style="margin:2px">${k}: ${v}</span>`).join('') || '<span class="text-muted">-</span>';
    const catHtml = Object.entries(s.categories || {}).sort((a,b) => b[1]-a[1]).map(([k,v]) => `<span class="tag tag-cat" style="margin:2px">${k}: ${v}</span>`).join('') || '<span class="text-muted">-</span>';
    document.getElementById('intelligenceInfo').innerHTML = `
      <div class="grid-3 mb-8">
        <div class="stat"><div class="stat-val" style="color:var(--green)">${s.total_tasks}</div><div class="stat-label">Total Tasks</div></div>
        <div class="stat"><div class="stat-val">${Object.keys(s.languages || {}).length}</div><div class="stat-label">Languages</div></div>
        <div class="stat"><div class="stat-val">${Object.keys(s.categories || {}).length}</div><div class="stat-label">Categories</div></div>
      </div>
      <div class="mb-8">Languages: ${langHtml}</div>
      <div>Categories: ${catHtml}</div>
    `;
    document.getElementById('intelligenceDatasetLabel').textContent = `${s.name} ${s.version ? 'v'+s.version : ''} (${s.total_tasks} tasks)`;
    // Update run scope label
    const allOpt = document.querySelector('#runScope option[value="all"]');
    if (allOpt) allOpt.textContent = `全量运行 (${s.total_tasks} 题)`;
  } catch (e) {
    document.getElementById('intelligenceInfo').innerHTML = '<span style="color:var(--red)">加载失败: ' + esc(ds) + '</span>';
  }
}

async function listTasks() {
  const el = document.getElementById('taskList');
  el.innerHTML = '<div class="loading-overlay"><div class="spinner"></div></div>';
  try {
    const q = new URLSearchParams();
    const lang = document.getElementById('taskLang').value.trim();
    const cat = document.getElementById('taskCategory').value.trim();
    const limit = document.getElementById('taskLimit').value;
    if (lang) q.set('language', lang);
    if (cat) q.set('category', cat);
    if (limit) q.set('limit', limit);
    q.set('rubric', '1');
    const data = await api(`/api/intelligence/datasets/${encodeURIComponent(currentDS())}/tasks?` + q.toString());
    if (!data.tasks || data.tasks.length === 0) {
      el.innerHTML = '<div class="text-muted">没有匹配的任务</div>';
      return;
    }
    el.innerHTML = `<div class="text-sm text-muted mb-8">${data.total} tasks found</div>` +
      data.tasks.map(t => `<div class="task-card">
        <div class="task-meta">
          <span class="tag tag-lang">${t.language}</span>
          <span class="tag tag-cat">${t.category}</span>
          <span class="text-sm text-muted">${t.task_id.substring(0,12)}...</span>
        </div>
        <div class="task-prompt">${esc(t.prompt.substring(0, 300))}</div>
      </div>`).join('');
  } catch (e) {
    el.innerHTML = `<div style="color:var(--red)">${esc(JSON.stringify(e))}</div>`;
  }
}

function buildRunPayload() {
  const scope = document.getElementById('runScope').value;
  const p = {
    ...targetPayload(),
    concurrency: parseInt(document.getElementById('runConcurrency').value) || 5,
    max_tokens: parseInt(document.getElementById('runMaxTokens').value) || 4096,
  };
  if (scope === 'all') {
    // no filters = run all
    return p;
  }
  const lang = document.getElementById('runLang').value.trim();
  const cat = document.getElementById('runCategory').value.trim();
  const limit = parseInt(document.getElementById('runLimit').value) || 0;
  if (lang) p.language = lang;
  if (cat) p.category = cat;
  if (limit > 0) p.limit = limit;
  if (scope === 'ids') {
    const ids = document.getElementById('runTaskIDs').value.split(',').map(s => s.trim()).filter(Boolean);
    if (ids.length > 0) p.task_ids = ids;
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

  // Reset progress
  document.getElementById('intelligenceProgressBar').style.width = '0%';
  document.getElementById('intelligenceProgressText').textContent = '0/0';
  document.getElementById('intelligenceProgressErrors').textContent = '';
  document.getElementById('intelligenceProgressPct').textContent = '0%';
  document.getElementById('intelligenceResultList').innerHTML = '';

  const payload = buildRunPayload();
  intelligenceAbort = new AbortController();

  try {
    const resp = await fetch(`/api/intelligence/datasets/${encodeURIComponent(currentDS())}/stream`, {
      method: 'POST',
      headers: headers(),
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

      // Parse SSE lines
      const lines = buffer.split('\n');
      buffer = lines.pop(); // keep incomplete line
      for (const line of lines) {
        if (!line.startsWith('data: ')) continue;
        try {
          const ev = JSON.parse(line.substring(6));
          handleIntelligenceEvent(ev);
        } catch {}
      }
    }
    // Process remaining buffer
    if (buffer.startsWith('data: ')) {
      try { handleIntelligenceEvent(JSON.parse(buffer.substring(6))); } catch {}
    }
  } catch (e) {
    if (e.name !== 'AbortError') {
      document.getElementById('intelligenceResultList').innerHTML +=
        `<div style="color:var(--red);padding:12px">Error: ${esc(e.message || JSON.stringify(e))}</div>`;
    }
  } finally {
    clearInterval(intelligenceTimer);
    btn.disabled = false;
    stopBtn.classList.add('hidden');
    intelligenceAbort = null;
  }
}

function stopIntelligence() {
  if (intelligenceAbort) intelligenceAbort.abort();
}

function updateElapsed() {
  if (!intelligenceStartTime) return;
  const sec = ((Date.now() - intelligenceStartTime) / 1000).toFixed(0);
  const min = Math.floor(sec / 60);
  const s = sec % 60;
  document.getElementById('intelligenceElapsed').textContent = min > 0 ? `${min}m ${s}s` : `${s}s`;
}

function handleIntelligenceEvent(ev) {
  if (ev.type === 'progress') {
    const pct = ev.total > 0 ? Math.round(ev.completed / ev.total * 100) : 0;
    document.getElementById('intelligenceProgressBar').style.width = pct + '%';
    document.getElementById('intelligenceProgressText').textContent = `${ev.completed}/${ev.total}`;
    document.getElementById('intelligenceProgressPct').textContent = pct + '%';
    if (ev.errors > 0) {
      document.getElementById('intelligenceProgressErrors').innerHTML = `<span class="err">${ev.errors} errors</span>`;
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
    document.getElementById('intelligenceProgressPct').textContent = '100%';
  }
}

function appendResultCard(r) {
  const el = document.getElementById('intelligenceResultList');
  const hasErr = !!r.error;
  const card = document.createElement('div');
  card.className = 'task-card';
  card.innerHTML = `
    <div class="task-meta">
      <span class="tag tag-lang">${esc(r.task.language)}</span>
      <span class="tag tag-cat">${esc(r.task.category)}</span>
      <span class="text-sm text-muted">${r.task.task_id.substring(0,12)}...</span>
      <span class="text-sm text-muted">${r.elapsed_ms}ms</span>
      ${hasErr ? '<span class="tag" style="background:var(--red-bg);color:var(--red)">Error</span>' : '<span class="tag" style="background:var(--green-bg);color:var(--green)">OK</span>'}
    </div>
    <div class="task-prompt">${esc((r.task.prompt || '').substring(0, 200))}...</div>
    ${r.answer ? `<details><summary class="text-sm text-muted" style="cursor:pointer;margin-top:6px">查看回答 (${r.answer.length} chars)</summary><div class="task-answer">${esc(r.answer)}</div></details>` : ''}
    ${r.error ? `<div class="task-answer" style="color:var(--red)">${esc(r.error)}</div>` : ''}
  `;
  el.appendChild(card);
  // Auto-scroll to bottom
  el.scrollTop = el.scrollHeight;
  // Show result card
  document.getElementById('intelligenceResultCard').classList.remove('hidden');
}

function renderIntelligenceSummary(report) {
  document.getElementById('intelligenceResultMeta').textContent =
    `${report.task_total} tasks | ${report.model} | ${report.elapsed_ms}ms`;

  const avgMs = report.results.length > 0
    ? Math.round(report.results.reduce((s, r) => s + r.elapsed_ms, 0) / report.results.length)
    : 0;

  document.getElementById('intelligenceResultStats').innerHTML = `
    <div class="stat"><div class="stat-val" style="color:var(--green)">${report.task_completed || report.task_total}</div><div class="stat-label">Completed</div></div>
    <div class="stat"><div class="stat-val" style="color:${report.task_errors > 0 ? 'var(--red)' : 'var(--text)'}">${report.task_errors || 0}</div><div class="stat-label">Errors</div></div>
    <div class="stat"><div class="stat-val">${(report.elapsed_ms / 1000).toFixed(1)}s</div><div class="stat-label">Total Time</div></div>
    <div class="stat"><div class="stat-val">${(avgMs / 1000).toFixed(1)}s</div><div class="stat-label">Avg / Task</div></div>
  `;
  document.getElementById('intelligenceResultCard').classList.remove('hidden');
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

/* ─── Init ─── */
loadIntelligenceList();
toggleRunScope();
updateIntelligencePreview();
