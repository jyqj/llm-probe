/* ─── bench.js · Benchmark run pages ──────────────────────────────────── */
/*
 * Mirrors the channel design:
 *   - #/bench         → new-run config view (with "Configure & Start" CTA opening a drawer)
 *   - #/bench/run/:id → run page (running | done | history)
 *   - #/bench/history → history list (in history.js)
 *
 * Uses the existing /api/intelligence/datasets/{name}/stream SSE endpoint.
 */

/* ─── New-run config view ─── */
async function renderBenchConfig() {
  setCrumb([{ label: 'Benchmark', href: '#/bench' }, { cur: '新建运行' }],
    el('div', { class: 'crumb-actions' },
      btn('查看历史', { onClick: () => location.hash = '#/bench/history', icon: 'history', size: 'sm', ghost: true })
    ));

  const v = $('#view');
  v.innerHTML = '<div class="empty">加载数据集…</div>';

  try {
    const data = await api('/api/intelligence/datasets');
    State.datasets = data.datasets || [];
    if (!State.currentDataset && State.datasets.length > 0) State.currentDataset = State.datasets[0].name;
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty', style: { color: 'var(--bad-ink)' } },
      el('div', { class: 'glyph' }, '×'), '加载数据集失败: ' + esc(e.message)));
    return;
  }

  v.innerHTML = '';
  v.appendChild(buildBenchConfigBody());
}

function buildBenchConfigBody() {
  const wrap = el('div');

  // Target panel
  const B = State.bench;
  wrap.appendChild(el('div', { class: 'panel', style: { marginBottom: '12px' } },
    el('div', { class: 'panel-head' },
      el('h3', null, 'Target 渠道'),
      el('span', { class: 'meta' }, '配置要做 benchmark 的渠道'),
    ),
    el('div', { class: 'panel-body' },
      buildField('TARGET BASE URL', el('input', {
        class: 'mono', placeholder: 'https://api.example.com/v1/messages', value: B.targetBase,
        oninput: e => { B.targetBase = e.target.value.trim(); updateConn(); },
        style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
      })),
      buildField('API KEY', el('input', {
        type: 'password', class: 'mono', placeholder: 'sk-...', value: B.targetKey,
        oninput: e => { B.targetKey = e.target.value; },
        style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
      })),
      buildField('MODEL', buildBenchModelChips()),
    ),
  ));

  // Datasets panel
  const dsPanel = el('div', { class: 'panel', style: { marginBottom: '12px' } });
  dsPanel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '数据集'),
    el('span', { class: 'meta' }, State.datasets.length + ' 个 · ' + State.datasets.reduce((s, d) => s + (d.total_tasks || 0), 0) + ' tasks'),
    el('div', { class: 'spacer' }),
    btn('添加数据集', { icon: 'add', size: 'sm', ghost: true, onClick: () => openDatasetDrawer() }),
  ));
  const dsBody = el('div', { class: 'panel-body', style: { padding: '12px' } });
  if (State.datasets.length === 0) {
    dsBody.appendChild(el('div', { class: 'empty' }, el('div', { class: 'glyph' }, '∅'), '尚无数据集 · 添加一个以开始'));
  } else {
    const grid = el('div', { style: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '8px' } });
    State.datasets.forEach(d => {
      const isCur = d.name === State.currentDataset;
      const card = el('div', { class: 'model-card' + (isCur ? ' active' : ''),
        onclick: () => { State.currentDataset = d.name; renderBenchConfig(); } });
      card.appendChild(el('div', { class: 'top' },
        el('span', { class: 'led-dot ' + (isCur ? 'pass' : 'pending') }),
        el('div', { class: 'name' }, d.name),
        el('span', { class: 'mono', style: { fontSize: '11px', color: 'var(--ink-3)' } }, d.total_tasks + ' tasks'),
      ));
      grid.appendChild(card);
    });
    dsBody.appendChild(grid);
  }
  dsPanel.appendChild(dsBody);
  wrap.appendChild(dsPanel);

  // Dataset summary
  if (State.currentDataset) {
    const summaryPanel = el('div', { class: 'panel', id: 'dsSummaryPanel', style: { marginBottom: '12px' } });
    summaryPanel.appendChild(el('div', { class: 'panel-head' },
      el('h3', null, State.currentDataset),
    ));
    summaryPanel.appendChild(el('div', { class: 'panel-body', id: 'dsSummaryBody' }, el('div', { class: 'muted' }, '加载中…')));
    wrap.appendChild(summaryPanel);
    loadDatasetSummary(State.currentDataset);

    // Run config panel — thinking / effort / scope / concurrency
    const configPanel = el('div', { class: 'panel', style: { marginBottom: '12px' } });
    configPanel.appendChild(el('div', { class: 'panel-head' },
      el('h3', null, '运行配置'),
      el('span', { class: 'meta', id: 'runConfigModelHint' }, B.model || '—'),
      el('div', { class: 'spacer' }),
      btn('开始运行', { primary: true, icon: 'play', onClick: () => kickoffBenchRun() }),
    ));
    const configBody = el('div', { class: 'panel-body', id: 'runConfigBody' });
    configBody.appendChild(buildRunConfigContent());
    configPanel.appendChild(configBody);
    wrap.appendChild(configPanel);
  }
  return wrap;
}

/* ─── Model chip selector (like channel page) ─── */
function buildBenchModelChips() {
  const ALL = ['claude-sonnet-4-6', 'claude-opus-4-6', 'claude-opus-4-7', 'claude-opus-4-5', 'claude-haiku-4-5'];
  const B = State.bench;
  const wrap = el('div', { class: 'chip-set' });
  ALL.forEach(m => {
    const active = B.model === m;
    const lbl = el('label', { class: 'chip' });
    const rb = el('input', { type: 'radio', name: 'bench-model', value: m });
    if (active) rb.checked = true;
    rb.addEventListener('change', () => {
      B.model = m;
      rebuildRunConfig();
    });
    lbl.appendChild(rb);
    lbl.appendChild(el('span', { class: 'led' }));
    lbl.appendChild(document.createTextNode(m));
    wrap.appendChild(lbl);
  });
  return wrap;
}

/* ─── Inline run config ─── */
function buildRunConfigContent() {
  const B = State.bench;
  const model = B.model;
  const frag = el('div');
  const levels = intensityLevelsFor(model);

  // Normalize current intensity
  if (!B.intensity || !levels.includes(B.intensity)) {
    B.intensity = levels.includes('high') ? 'high' : levels[levels.length - 1];
  }

  // Mode: single vs batch
  const modeSeg = el('div', { class: 'seg', style: { width: 'fit-content' } });
  modeSeg.appendChild(el('button', { class: !B.multiIntensity ? 'active' : '',
    onclick: () => { B.multiIntensity = false; rebuildRunConfig(); } }, '单个'));
  modeSeg.appendChild(el('button', { class: B.multiIntensity ? 'active' : '',
    onclick: () => { B.multiIntensity = true; if (!B.selectedIntensities.length) B.selectedIntensities = [...levels]; rebuildRunConfig(); } }, '批量对比'));
  frag.appendChild(buildField('RUN MODE', modeSeg));

  // Intensity level(s)
  const LEVEL_LABEL = { off: 'OFF', low: 'Low', medium: 'Medium', high: 'High', xhigh: 'X-High', max: 'Max', enabled: 'Enabled', adaptive: 'Adaptive' };

  if (B.multiIntensity) {
    B.selectedIntensities = B.selectedIntensities.filter(e => levels.includes(e));
    if (!B.selectedIntensities.length) B.selectedIntensities = [...levels];
    const checkWrap = el('div', { style: { display: 'flex', gap: '10px', flexWrap: 'wrap' } });
    levels.forEach(lv => {
      const checked = B.selectedIntensities.includes(lv);
      checkWrap.appendChild(el('label', { style: { display: 'flex', gap: '4px', alignItems: 'center', cursor: 'pointer', fontSize: '12px', fontFamily: 'var(--font-mono)' } },
        el('input', { type: 'checkbox', checked: checked ? '' : null,
          onchange: () => {
            if (checked) B.selectedIntensities = B.selectedIntensities.filter(e => e !== lv);
            else B.selectedIntensities.push(lv);
            rebuildRunConfig();
          } }),
        LEVEL_LABEL[lv] || lv));
    });
    frag.appendChild(buildField('思考力度 (' + B.selectedIntensities.length + '/' + levels.length + ')', checkWrap));
  } else {
    const seg = el('div', { class: 'seg', style: { width: 'fit-content', flexWrap: 'wrap' } });
    levels.forEach(lv => {
      seg.appendChild(el('button', { class: B.intensity === lv ? 'active' : '',
        onclick: () => { B.intensity = lv; rebuildRunConfig(); } }, LEVEL_LABEL[lv] || lv));
    });
    frag.appendChild(buildField('思考力度', seg));
  }

  // Scope
  frag.appendChild(el('div', { style: { borderTop: '1px solid var(--line)', margin: '14px 0 10px', opacity: '.4' } }));
  const scopeSeg = el('div', { class: 'seg', style: { width: 'fit-content' } });
  [['all', '全量'], ['custom', '自定义筛选']].forEach(([v, lbl]) => {
    scopeSeg.appendChild(el('button', { class: B.scope === v ? 'active' : '',
      onclick: () => { B.scope = v; rebuildRunConfig(); } }, lbl));
  });
  frag.appendChild(buildField('SCOPE', scopeSeg));

  if (B.scope === 'custom') {
    frag.appendChild(buildField('LANGUAGE', el('input', { value: B.lang, placeholder: 'go / python (留空=全部)',
      oninput: e => { B.lang = e.target.value; },
      style: { width: '100%', background: 'transparent', border: 'none', padding: '0' } })));
    frag.appendChild(buildField('CATEGORY', el('input', { value: B.category, placeholder: 'Security (留空=全部)',
      oninput: e => { B.category = e.target.value; },
      style: { width: '100%', background: 'transparent', border: 'none', padding: '0' } })));
    frag.appendChild(buildField('LIMIT', el('input', { type: 'number', value: B.limit, placeholder: '0=全部',
      oninput: e => { B.limit = parseInt(e.target.value) || 0; },
      class: 'mono', style: { width: '120px', background: 'transparent', border: 'none', padding: '0' } })));
  }

  frag.appendChild(buildField('CONCURRENCY', el('input', { type: 'number', min: 1, max: 20,
    value: B.concurrency, class: 'mono',
    oninput: e => { B.concurrency = parseInt(e.target.value) || 1; },
    style: { width: '80px', background: 'transparent', border: 'none', padding: '0' } })));

  return frag;
}

function rebuildRunConfig() {
  const container = document.getElementById('runConfigBody');
  if (!container) return;
  container.innerHTML = '';
  container.appendChild(buildRunConfigContent());
  const hint = document.getElementById('runConfigModelHint');
  if (hint) hint.textContent = State.bench.model || '—';
}

async function loadDatasetSummary(name) {
  try {
    const d = await api('/api/intelligence/datasets/' + encodeURIComponent(name));
    const s = d.stats || {};
    const body = $('#dsSummaryBody'); if (!body) return;
    body.innerHTML = '';
    const stats = el('div', { class: 'statbar', style: { marginBottom: '0' } });
    stats.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'TOTAL'), el('div', { class: 'v big' }, s.total_tasks || 0)));
    if (s.version) stats.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'VERSION'), el('div', { class: 'v' }, s.version)));
    if (s.languages) stats.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'LANGS'), el('div', { class: 'v' }, Object.keys(s.languages).length)));
    if (s.categories) stats.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'CATEGORIES'), el('div', { class: 'v' }, Object.keys(s.categories).length)));
    body.appendChild(stats);

    const tags = el('div', { class: 'tag-row', style: { marginTop: '10px' } });
    Object.entries(s.languages || {}).sort((a, b) => b[1] - a[1]).slice(0, 12).forEach(([k, v]) =>
      tags.appendChild(el('span', { class: 'itag itag-info' }, k + ' · ' + v)));
    Object.entries(s.categories || {}).sort((a, b) => b[1] - a[1]).slice(0, 12).forEach(([k, v]) =>
      tags.appendChild(el('span', { class: 'itag itag-warn' }, k + ' · ' + v)));
    if (tags.children.length) body.appendChild(tags);
  } catch (e) {
    const body = $('#dsSummaryBody'); if (body) body.innerHTML = '<div class="muted">加载失败</div>';
  }
}

/* openRunConfigDrawer kept as no-op for legacy references */
function openRunConfigDrawer() { location.hash = '#/bench'; }

/* ─── add-dataset drawer ─── */
function openDatasetDrawer() {
  const body = el('div');
  let mode = 'fetch';

  const tabs = el('div', { class: 'seg', style: { width: 'fit-content', marginBottom: '14px' } });
  const tabFetch = el('button', { class: 'active', onclick: () => { mode = 'fetch'; tabFetch.classList.add('active'); tabUpload.classList.remove('active'); render(); } }, '从 HuggingFace');
  const tabUpload = el('button', { onclick: () => { mode = 'upload'; tabUpload.classList.add('active'); tabFetch.classList.remove('active'); render(); } }, '上传文件');
  tabs.appendChild(tabFetch); tabs.appendChild(tabUpload);
  body.appendChild(tabs);

  const pane = el('div');
  body.appendChild(pane);
  const status = el('div', { style: { marginTop: '12px', fontSize: '12px', color: 'var(--ink-3)' } });
  body.appendChild(status);

  function render() {
    pane.innerHTML = '';
    if (mode === 'fetch') {
      pane.appendChild(el('p', { class: 'muted', style: { fontSize: '12px', marginBottom: '12px' } },
        '内置 HuggingFace 数据集拉取(目前仅 SWE-Atlas-QnA)。'));
      const sel = el('select', { id: 'dsFetchName', style: { width: '100%' } },
        el('option', { value: 'SWE-Atlas-QnA' }, 'SWE-Atlas-QnA'));
      pane.appendChild(buildField('NAME', sel));
      pane.appendChild(buildField('LIMIT', el('input', { type: 'number', id: 'dsFetchLimit', value: 0,
        class: 'mono', placeholder: '0=全部', style: { width: '120px', background: 'transparent', border: 'none', padding: '0' } })));
    } else {
      pane.appendChild(el('p', { class: 'muted', style: { fontSize: '12px', marginBottom: '12px' } },
        '支持 CSV (含 task_id, prompt) 或 JSON 格式。'));
      pane.appendChild(buildField('NAME', el('input', { id: 'dsUploadName', placeholder: 'my-custom-bench',
        style: { width: '100%', background: 'transparent', border: 'none', padding: '0' } })));
      pane.appendChild(buildField('FILE', el('input', { type: 'file', id: 'dsUploadFile', accept: '.csv,.json' })));
    }
  }
  render();

  const foot = el('div', null,
    btn('取消', { ghost: true, onClick: closeDrawer }),
    btn('确认', { primary: true, onClick: async () => {
      if (mode === 'fetch') {
        const name = $('#dsFetchName').value;
        const limit = parseInt($('#dsFetchLimit').value) || 0;
        status.textContent = '拉取中…';
        try {
          const data = await api('/api/intelligence/fetch', {
            method: 'POST', body: JSON.stringify({ name, limit: limit > 0 ? limit : undefined })
          });
          State.currentDataset = data.stats.name;
          closeDrawer();
          renderBenchConfig();
          toast('已添加: ' + data.stats.total_tasks + ' tasks', 'good');
        } catch (e) { status.textContent = '失败: ' + e.message; }
      } else {
        const name = $('#dsUploadName').value.trim();
        const file = $('#dsUploadFile').files[0];
        if (!name) { status.textContent = '请输入名称'; return; }
        if (!file) { status.textContent = '请选择文件'; return; }
        status.textContent = '上传中…';
        const fd = new FormData(); fd.append('file', file); fd.append('name', name);
        const h = State.adminToken ? { 'X-Admin-Token': State.adminToken } : {};
        try {
          const resp = await fetch('/api/intelligence/datasets/' + encodeURIComponent(name) + '/upload',
            { method: 'POST', headers: h, body: fd });
          const data = await resp.json();
          if (!resp.ok) throw new Error(data.error || 'upload failed');
          State.currentDataset = name;
          closeDrawer();
          renderBenchConfig();
          toast('已添加: ' + data.stats.total_tasks + ' tasks', 'good');
        } catch (e) { status.textContent = '失败: ' + e.message; }
      }
    } }),
  );
  openDrawer('添加数据集', body, foot);
}

/* ─── kickoff bench run ─── */
async function kickoffBenchRun() {
  const B = State.bench;
  if (!B.targetBase) { toast('请填写 Target Base URL', 'bad'); return; }
  if (!B.targetKey)  { toast('请填写 API Key', 'bad'); return; }
  if (!State.currentDataset) { toast('请选择数据集', 'bad'); return; }

  if (B.multiIntensity && B.selectedIntensities.length > 1) {
    kickoffMultiIntensityRun();
    return;
  }

  const params = intensityToParams(B.model, B.intensity);
  const tempId = 'live_' + Date.now().toString(36);
  const payload = {
    target_base: B.targetBase, target_key: B.targetKey,
    model: B.model,
    concurrency: B.concurrency, thinking: params.thinking,
    effort: params.effort || undefined,
    thinking_mode: params.thinkingMode !== 'off' ? params.thinkingMode : undefined,
  };
  if (B.scope === 'custom') {
    if (B.lang) payload.language = B.lang;
    if (B.category) payload.category = B.category;
    if (B.limit > 0) payload.limit = B.limit;
  }

  State.liveRuns[tempId] = {
    kind: 'bench',
    state: 'running',
    payload,
    dataset: State.currentDataset,
    target: B.targetBase,
    model: B.model,
    startedAt: Date.now(),
    totalTasks: 0,
    completedTasks: 0,
    errorTasks: 0,
    results: [],
    aborter: null,
    finalReport: null,
    error: null,
    thinking: params.thinking,
    effort: params.effort || '',
    thinkingMode: params.thinkingMode || '',
  };
  location.hash = '#/bench/run/' + tempId;
  runBenchStream(tempId, payload).catch(err => {
    const r = State.liveRuns[tempId]; if (!r) return;
    r.state = 'error'; r.error = err.message || String(err);
    maybeRerenderBench(tempId);
  });
}

/* ─── multi-intensity sweep ─── */
async function kickoffMultiIntensityRun() {
  const B = State.bench;
  const intensities = [...B.selectedIntensities];
  const batchId = 'batch_' + Date.now().toString(36);

  State.liveRuns[batchId] = {
    kind: 'bench-batch',
    state: 'running',
    efforts: intensities,
    dataset: State.currentDataset,
    target: B.targetBase,
    model: B.model,
    startedAt: Date.now(),
    subRuns: {},
    completedEfforts: 0,
    totalEfforts: intensities.length,
    thinking: true,
    thinkingMode: '',
    error: null,
  };
  location.hash = '#/bench/run/' + batchId;

  for (const intensity of intensities) {
    if (State.liveRuns[batchId].state === 'cancelled') break;
    const params = intensityToParams(B.model, intensity);
    const subId = batchId + '_' + intensity;
    const payload = {
      target_base: B.targetBase, target_key: B.targetKey,
      model: B.model,
      concurrency: B.concurrency, thinking: params.thinking,
      effort: params.effort || undefined,
      thinking_mode: params.thinkingMode !== 'off' ? params.thinkingMode : undefined,
    };
    if (B.scope === 'custom') {
      if (B.lang) payload.language = B.lang;
      if (B.category) payload.category = B.category;
      if (B.limit > 0) payload.limit = B.limit;
    }

    State.liveRuns[batchId].subRuns[intensity] = {
      subId, state: 'running', payload, effort: intensity,
      totalTasks: 0, completedTasks: 0, errorTasks: 0,
      results: [], finalReport: null, error: null,
    };
    maybeRerenderBench(batchId);

    try {
      await runBatchSubStream(batchId, intensity, subId, payload);
    } catch (err) {
      const sub = State.liveRuns[batchId].subRuns[intensity];
      if (sub) { sub.state = 'error'; sub.error = err.message || String(err); }
    }
    State.liveRuns[batchId].completedEfforts++;
    maybeRerenderBench(batchId);
  }

  const batch = State.liveRuns[batchId];
  if (batch && batch.state === 'running') batch.state = 'done';
  maybeRerenderBench(batchId);
}

async function runBatchSubStream(batchId, effort, subId, payload) {
  const batch = State.liveRuns[batchId];
  const sub = batch.subRuns[effort];
  const ac = new AbortController();
  sub.aborter = ac;
  let resp;
  try {
    resp = await fetch('/api/intelligence/datasets/' + encodeURIComponent(batch.dataset) + '/stream', {
      method: 'POST', headers: headers(),
      body: JSON.stringify(payload), signal: ac.signal,
    });
  } catch (e) {
    if (e.name === 'AbortError') { sub.state = 'cancelled'; return; }
    throw e;
  }
  if (!resp.ok) throw new Error(await resp.text());
  const reader = resp.body.getReader(); const decoder = new TextDecoder();
  let buf = '';
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    const lines = buf.split('\n'); buf = lines.pop();
    for (const line of lines) {
      if (!line.startsWith('data: ')) continue;
      try {
        const ev = JSON.parse(line.slice(6));
        handleBatchSubSSE(batchId, effort, ev);
      } catch {}
    }
  }
  if (buf.startsWith('data: ')) {
    try { handleBatchSubSSE(batchId, effort, JSON.parse(buf.slice(6))); } catch {}
  }
}

function handleBatchSubSSE(batchId, effort, ev) {
  const batch = State.liveRuns[batchId]; if (!batch) return;
  const sub = batch.subRuns[effort]; if (!sub) return;
  if (ev.type === 'start') {
    sub.totalTasks = ev.total || sub.totalTasks;
    maybeRerenderBench(batchId);
    return;
  }
  if (ev.type === 'progress') {
    sub.totalTasks = ev.total || sub.totalTasks;
    sub.completedTasks = ev.completed || sub.completedTasks;
    sub.errorTasks = ev.errors || sub.errorTasks;
    if (ev.result) sub.results.push(ev.result);
    maybeRerenderBench(batchId);
  } else if (ev.type === 'complete') {
    sub.state = 'done';
    sub.finalReport = ev.report;
    if (ev.report) {
      sub.totalTasks = ev.report.task_total;
      sub.completedTasks = ev.report.task_completed;
      sub.errorTasks = ev.report.task_errors;
    }
    maybeRerenderBench(batchId);
  } else if (ev.type === 'error') {
    sub.state = 'error'; sub.error = ev.error_msg || ev.error;
    maybeRerenderBench(batchId);
  }
}

function cancelBatchRun(batchId) {
  const batch = State.liveRuns[batchId]; if (!batch) return;
  batch.state = 'cancelled';
  Object.values(batch.subRuns).forEach(sub => {
    if (sub.aborter) sub.aborter.abort();
    if (sub.state === 'running') sub.state = 'cancelled';
  });
  maybeRerenderBench(batchId);
  toast('已取消批量运行', 'good');
}

async function runBenchStream(runId, payload) {
  const run = State.liveRuns[runId];
  const ac = new AbortController(); run.aborter = ac;
  let resp;
  try {
    resp = await fetch('/api/intelligence/datasets/' + encodeURIComponent(run.dataset) + '/stream', {
      method: 'POST', headers: headers(),
      body: JSON.stringify(payload), signal: ac.signal,
    });
  } catch (e) {
    if (e.name === 'AbortError') { run.state = 'cancelled'; maybeRerenderBench(runId); return; }
    throw e;
  }
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(text);
  }
  const reader = resp.body.getReader(); const decoder = new TextDecoder();
  let buf = '';
  const watchdog = createSSEWatchdog(45000, () => {
    if (run.state === 'running') toast('SSE 连接可能已断开 · 服务端仍在执行', 'warn');
  });
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      watchdog.reset();
      buf += decoder.decode(value, { stream: true });
      const lines = buf.split('\n'); buf = lines.pop();
      for (const line of lines) {
        if (!line.startsWith('data: ')) continue;
        try { handleBenchSSE(runId, JSON.parse(line.slice(6))); } catch {}
      }
    }
    if (buf.startsWith('data: ')) {
      try { handleBenchSSE(runId, JSON.parse(buf.slice(6))); } catch {}
    }
  } finally {
    watchdog.stop();
  }
}

function handleBenchSSE(runId, ev) {
  const run = State.liveRuns[runId];
  if (!run) return;
  if (ev.type === 'start') {
    run.totalTasks = ev.total || run.totalTasks;
    maybeRerenderBench(runId);
    return;
  }
  if (ev.type === 'progress') {
    run.totalTasks = ev.total || run.totalTasks;
    run.completedTasks = ev.completed || run.completedTasks;
    run.errorTasks = ev.errors || run.errorTasks;
    if (ev.result) run.results.push(ev.result);
    maybeRerenderBench(runId);
  } else if (ev.type === 'complete') {
    run.state = 'done';
    run.finalReport = ev.report;
    run.finishedAt = Date.now();
    if (ev.report) {
      run.totalTasks = ev.report.task_total;
      run.completedTasks = ev.report.task_completed;
      run.errorTasks = ev.report.task_errors;
    }
    toast('Benchmark 完成', 'good');
    maybeRerenderBench(runId);
  } else if (ev.type === 'error') {
    run.state = 'error'; run.error = ev.error_msg || ev.error;
    maybeRerenderBench(runId);
  }
}

function cancelBenchRun(runId) {
  const run = State.liveRuns[runId]; if (!run) return;
  if (run.aborter) run.aborter.abort();
  run.state = 'cancelled';
  maybeRerenderBench(runId);
  toast('已取消', 'good');
}

let _benchRenderTimer = null;
function maybeRerenderBench(runId) {
  const route = parseRoute(location.hash);
  if (route.app !== 'bench' || route.kind !== 'run' || route.id !== runId) return;
  if (_benchRenderTimer) return;
  _benchRenderTimer = requestAnimationFrame(() => {
    _benchRenderTimer = null;
    const run = State.liveRuns[runId];
    if (run && run.kind === 'bench-batch') renderBenchBatchPage(runId);
    else renderBenchRunPage(runId);
  });
}

/* ─── Bench run page ─── */
async function renderBenchRunRoute(runId) {
  if (State.liveRuns[runId]) {
    if (State.liveRuns[runId].kind === 'bench-batch') { renderBenchBatchPage(runId); return; }
    renderBenchRunPage(runId); return;
  }
  const v = $('#view');
  v.innerHTML = '<div class="empty">加载历史中…</div>';
  try {
    const data = await api('/api/intelligence/history/' + encodeURIComponent(runId));
    State.liveRuns[runId] = {
      kind: 'bench', state: 'done', historical: true,
      dataset: data.dataset_name, target: data.target || '',
      model: data.model, thinking: data.thinking,
      effort: data.effort || '', thinkingMode: data.thinking_mode || '',
      startedAt: data.started_at ? new Date(data.started_at).getTime() : null,
      finishedAt: data.completed_at ? new Date(data.completed_at).getTime() : null,
      totalTasks: data.task_total, completedTasks: data.task_completed, errorTasks: data.task_errors,
      results: data.results || [],
      finalReport: data,
      payload: null,
    };
    renderBenchRunPage(runId);
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty' },
      el('div', { class: 'glyph' }, '×'), '未找到此次运行: ' + esc(runId),
      el('div', { style: { marginTop: '12px' } },
        btn('返回历史', { onClick: () => location.hash = '#/bench/history', icon: 'history', size: 'sm' })
      )));
  }
}

function renderBenchRunPage(runId) {
  const run = State.liveRuns[runId];
  if (!run) return;
  const crumbs = [{ label: 'Benchmark', href: '#/bench' }];
  if (run.historical) crumbs.push({ label: '历史', href: '#/bench/history' });
  crumbs.push({ cur: run.dataset || 'run' });
  if (run.state !== 'running') crumbs.push({ cur: '报告' });
  const actions = el('div', { class: 'crumb-actions' });
  if (run.state === 'running') {
    actions.appendChild(btn('取消', { icon: 'stop', size: 'sm', danger: true, onClick: () => cancelBenchRun(runId) }));
  } else {
    if (run.payload) actions.appendChild(btn('重新运行', { icon: 'refresh', size: 'sm', ghost: true, onClick: () => rerunBench(runId) }));
    actions.appendChild(btn('导出 JSON', { icon: 'download', size: 'sm', ghost: true,
      onClick: () => downloadJSON(run.finalReport || run, 'bench-' + runId + '.json') }));
  }
  setCrumb(crumbs, actions);

  const v = $('#view'); v.innerHTML = '';

  // status bar
  v.appendChild(renderBenchHeader(run, runId));

  if (run.state === 'running') {
    v.appendChild(renderTaskGrid(run));
    v.appendChild(renderBenchResultsList(run));
  } else {
    v.appendChild(renderBenchReportBanner(run));
    v.appendChild(renderBenchStats(run));
    v.appendChild(renderBenchCategoryBreakdown(run));
    v.appendChild(renderBenchResultsList(run));
  }
}

function renderBenchHeader(run, runId) {
  const wrap = el('div', { class: 'statbar' });
  const statusBadge = (() => {
    if (run.state === 'running') return el('span', { class: 'pill pill-running' }, el('span', { class: 'led' }), '运行中');
    if (run.state === 'error') return el('span', { class: 'pill pill-bad' }, el('span', { class: 'led' }), '失败');
    if (run.state === 'cancelled') return el('span', { class: 'pill pill-warn' }, el('span', { class: 'led' }), '已取消');
    return el('span', { class: 'pill pill-good' }, el('span', { class: 'led' }), '完成');
  })();
  const cells = [];
  cells.push(['STATUS', statusBadge]);
  cells.push(['RUN ID', el('span', { class: 'mono' }, runId.slice(0, 16) + (runId.length > 16 ? '…' : ''))]);
  cells.push(['DATASET', run.dataset || '—']);
  cells.push(['MODEL', el('span', { class: 'mono' }, run.model || '—')]);
  cells.push(['THINKING', run.thinkingMode || (run.thinking ? 'on' : 'off')]);
  if (run.effort) cells.push(['EFFORT', run.effort]);
  if (run.state === 'running') {
    cells.push(['ELAPSED', el('span', { class: 'mono', id: 'benchLiveElapsed' }, fmtMs(Date.now() - run.startedAt))]);
    if (!run._tick) {
      run._tick = setInterval(() => {
        if (run.state !== 'running') { clearInterval(run._tick); run._tick = null; return; }
        const cur = document.getElementById('benchLiveElapsed');
        if (cur) cur.textContent = fmtMs(Date.now() - run.startedAt);
      }, 1000);
    }
  } else if (run.finalReport) {
    cells.push(['ELAPSED', el('span', { class: 'mono' }, fmtMs(run.finalReport.elapsed_ms))]);
  }

  cells.forEach(([k, v]) => {
    wrap.appendChild(el('div', { class: 'cell' },
      el('div', { class: 'k' }, k), el('div', { class: 'v' }, v)));
  });

  if (run.state === 'error' && run.error) {
    const frag = document.createDocumentFragment();
    frag.appendChild(wrap);
    frag.appendChild(el('div', { class: 'panel', style: { marginBottom: '12px', borderColor: 'var(--bad-line)' } },
      el('div', { class: 'panel-head', style: { background: 'var(--bad-soft)', borderColor: 'var(--bad-line)' } },
        el('h3', { style: { color: 'var(--bad-ink)' } }, '运行失败')),
      el('div', { class: 'panel-body', style: { fontFamily: 'var(--font-mono)', fontSize: '12px', color: 'var(--bad-ink)', wordBreak: 'break-all' } }, run.error),
    ));
    return frag;
  }
  return wrap;
}

function renderBenchProgressStrip(run) {
  const pct = run.totalTasks > 0 ? Math.round(run.completedTasks / run.totalTasks * 100) : 0;
  return el('div', { class: 'panel', style: { marginBottom: '12px' } },
    el('div', { class: 'panel-head' },
      el('h3', null, '运行进度'),
      el('span', { class: 'meta' }, run.completedTasks + ' / ' + run.totalTasks + ' tasks'),
      el('div', { class: 'spacer' }),
      el('span', { class: 'mono', style: { color: 'var(--ink-3)', fontSize: '11px' } }, pct + '%'),
    ),
    el('div', { class: 'panel-body', style: { padding: '12px 14px' } },
      el('div', { class: 'progress' }, el('div', { class: 'progress-fill', style: { width: pct + '%' } })),
    ),
  );
}

function renderBenchStats(run) {
  const total = run.totalTasks || (run.finalReport ? run.finalReport.task_total : 0);
  const done = run.completedTasks || (run.finalReport ? run.finalReport.task_completed : 0);
  const errors = run.errorTasks || (run.finalReport ? run.finalReport.task_errors : 0);
  const avgMs = run.results.length > 0
    ? Math.round(run.results.reduce((s, r) => s + (r.elapsed_ms || 0), 0) / run.results.length)
    : 0;
  return el('div', { class: 'statbar' },
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'TOTAL'),  el('div', { class: 'v big' }, total)),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'DONE'),   el('div', { class: 'v big good' }, done)),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'ERRORS'), el('div', { class: 'v big ' + (errors > 0 ? 'bad' : '') }, errors)),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'AVG/TASK'), el('div', { class: 'v' }, avgMs ? fmtMs(avgMs) : '—')),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'CONCURRENCY'), el('div', { class: 'v' }, (run.payload && run.payload.concurrency) || '—')),
  );
}

/* ─── Task grid — visual progress for running state ─── */
function renderTaskGrid(run) {
  const total = run.totalTasks || 0;
  const done = run.completedTasks || 0;
  const errors = run.errorTasks || 0;
  const pct = total > 0 ? Math.round(done / total * 100) : 0;

  const resultsByIndex = {};
  run.results.forEach(r => { resultsByIndex[r.index] = r; });

  const passCount = run.results.filter(r => !r.error && r.pass !== false).length;
  const failCount = run.results.filter(r => !r.error && r.pass === false).length;
  const passRate = done > 0 ? Math.round(passCount / Math.max(passCount + failCount, 1) * 1000) / 10 : 0;
  const avgMs = run.results.length > 0
    ? Math.round(run.results.reduce((s, r) => s + (r.elapsed_ms || 0), 0) / run.results.length)
    : 0;

  const panel = el('div', { class: 'panel', style: { marginBottom: '12px' } });
  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '任务进度'),
    el('span', { class: 'meta' }, done + ' / ' + total + ' tasks · ' + pct + '%'),
    el('div', { class: 'spacer' }),
    el('span', { class: 'pill pill-running' }, el('span', { class: 'led' }), '运行中'),
  ));

  const body = el('div', { class: 'panel-body' });

  // stats row
  body.appendChild(el('div', { class: 'task-stats' },
    el('span', { class: 'task-stat good' }, el('span', { class: 'led-dot pass' }), passCount + ' 通过'),
    el('span', { class: 'task-stat bad' }, el('span', { class: 'led-dot fail' }), failCount + ' 失败'),
    el('span', { class: 'task-stat warn' }, el('span', { class: 'led-dot warn' }), errors + ' 错误'),
    el('span', { class: 'spacer' }),
    el('span', { class: 'mono', style: { fontSize: '11px', color: 'var(--ink-3)' } },
      'pass ' + passRate + '%' + (avgMs ? ' · avg ' + fmtMs(avgMs) : '')),
  ));

  // progress bar
  body.appendChild(el('div', { class: 'progress', style: { margin: '10px 0' } },
    el('div', { class: 'progress-fill', style: { width: pct + '%' } })));

  // task grid cells — grouped by category
  if (total > 0) {
    const catGroups = {};
    run.results.forEach(r => {
      const cat = (r.task && r.task.category) || 'unknown';
      (catGroups[cat] ||= []).push(r);
    });
    const catNames = Object.keys(catGroups).sort();
    const hasCats = catNames.length > 1 || (catNames.length === 1 && catNames[0] !== 'unknown');

    if (hasCats && run.results.length > 0) {
      const legend = el('div', { class: 'task-cat-legend' });
      catNames.forEach(cat => {
        const gr = catGroups[cat];
        const p = gr.filter(r => !r.error && r.pass !== false).length;
        const f = gr.filter(r => !r.error && r.pass === false).length;
        const e = gr.filter(r => !!r.error).length;
        legend.appendChild(el('span', { class: 'task-cat-item' },
          el('span', { class: 'cat-name' }, cat),
          el('span', { class: 'cat-counts' }, p + '✓ ' + f + '✗' + (e > 0 ? ' ' + e + '!' : '')),
        ));
      });
      body.appendChild(legend);
    }

    const grid = el('div', { class: 'task-grid' });
    for (let i = 0; i < total; i++) {
      const result = resultsByIndex[i];
      let state = 'pending';
      let title = '#' + (i + 1) + ' · pending';
      if (result) {
        const tid = (result.task && result.task.task_id) || '#' + (i + 1);
        const cat = (result.task && result.task.category) || '';
        const catHint = cat ? ' [' + cat + ']' : '';
        if (result.error) {
          state = 'error';
          title = tid + catHint + ' · ERROR · ' + fmtMs(result.elapsed_ms);
        } else if (result.pass === false) {
          state = 'fail';
          title = tid + catHint + ' · FAIL' + (result.score != null ? ' ' + Math.round(result.score * 100) + '%' : '') + ' · ' + fmtMs(result.elapsed_ms);
        } else if (result.pass === true) {
          state = 'pass';
          title = tid + catHint + ' · PASS' + (result.score != null ? ' ' + Math.round(result.score * 100) + '%' : '') + ' · ' + fmtMs(result.elapsed_ms);
        } else {
          state = 'neutral';
          title = tid + catHint + ' · done · ' + fmtMs(result.elapsed_ms);
        }
      }
      grid.appendChild(el('div', {
        class: 'task-cell ' + state,
        title,
        onclick: result ? () => openBenchResultModal(result) : null,
      }));
    }
    body.appendChild(grid);
  }

  // live counters summary
  body.appendChild(el('div', { style: { display: 'flex', gap: '14px', marginTop: '10px', fontSize: '11px', fontFamily: 'var(--font-mono)', color: 'var(--ink-3)' } },
    el('span', null, 'CONCURRENCY ' + ((run.payload && run.payload.concurrency) || '—')),
    el('span', null, 'ELAPSED ' + (run.startedAt ? fmtMs(Date.now() - run.startedAt) : '—')),
    run.effort ? el('span', null, 'EFFORT ' + run.effort) : null,
  ));

  panel.appendChild(body);
  return panel;
}

function renderBenchReportBanner(run) {
  const rep = run.finalReport;
  const elapsed = rep ? fmtMs(rep.elapsed_ms) : (run.finishedAt && run.startedAt ? fmtMs(run.finishedAt - run.startedAt) : '—');
  const ts = run.startedAt ? fmtTime(new Date(run.startedAt).toISOString()) : '';

  const left = el('div', { class: 'report-banner-left' },
    el('span', { class: 'report-badge' }, '测试报告'),
    el('span', { class: 'report-meta' },
      (run.model || '—') + ' · ' + (run.dataset || '—') + ' · ' + elapsed));

  const scoreInfo = el('div', { class: 'report-banner-scores' });
  if (rep && rep.pass_rate != null) {
    const rate = Math.round(rep.pass_rate * 1000) / 10;
    scoreInfo.appendChild(el('span', { class: 'pill ' + (rate >= 70 ? 'pill-good' : rate >= 40 ? 'pill-warn' : 'pill-bad') },
      el('span', { class: 'led' }), 'pass rate ' + rate + '%'));
  }
  if (rep && rep.score_total != null) {
    scoreInfo.appendChild(el('span', { class: 'mono', style: { fontSize: '12px', color: 'var(--ink-2)' } },
      'score ' + (Math.round(rep.score_total * 100) / 100)));
  }

  return el('div', { class: 'report-banner' },
    left, scoreInfo,
    ts ? el('span', { class: 'report-ts' }, ts) : null);
}

function renderBenchCategoryBreakdown(run) {
  const results = run.results || [];
  if (results.length === 0) return el('div');

  const catData = {};
  results.forEach(r => {
    const cat = (r.task && r.task.category) || 'unknown';
    if (!catData[cat]) catData[cat] = { total: 0, passed: 0, failed: 0, errors: 0, totalMs: 0 };
    const d = catData[cat];
    d.total++;
    if (r.error) d.errors++;
    else if (r.pass === true) d.passed++;
    else if (r.pass === false) d.failed++;
    else d.passed++;
    d.totalMs += r.elapsed_ms || 0;
  });

  const cats = Object.keys(catData).sort();
  if (cats.length <= 1 && cats[0] === 'unknown') return el('div');

  const panel = el('div', { class: 'panel', style: { marginBottom: '12px' } });
  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '分类分布'),
    el('span', { class: 'meta' }, cats.length + ' categories'),
  ));
  const body = el('div', { class: 'panel-body', style: { padding: '0' } });
  const strip = el('div', { class: 'bench-cat-strip' });

  cats.forEach(cat => {
    const d = catData[cat];
    const rate = d.total > 0 ? Math.round(d.passed / d.total * 100) : 0;
    const rateColor = rate >= 80 ? 'var(--good-ink)' : rate >= 50 ? 'var(--warn-ink)' : 'var(--bad-ink)';
    const avgMs = d.total > 0 ? Math.round(d.totalMs / d.total) : 0;

    strip.appendChild(el('div', { class: 'bench-cat-row' },
      el('span', { class: 'name' }, cat),
      el('div', { class: 'track' },
        el('div', { class: 'fill', style: { width: rate + '%', background: rate >= 80 ? 'var(--good)' : rate >= 50 ? 'var(--warn)' : 'var(--bad)' } })),
      el('span', { class: 'frac' }, d.passed + '/' + d.total),
      el('span', { class: 'pct', style: { color: rateColor } }, rate + '%'),
      el('span', { class: 'avg' }, fmtMs(avgMs)),
    ));
  });

  body.appendChild(strip);
  panel.appendChild(body);
  return panel;
}

function renderBenchResultsList(run) {
  const panel = el('div', { class: 'panel' });
  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '单题结果'),
    el('span', { class: 'meta' }, run.results.length + ' 条 · ' + (run.state === 'running' ? '流式追加' : '完整列表')),
  ));
  const body = el('div', { class: 'panel-body', style: { padding: '0' } });
  const list = el('div', { class: 'live-list' });
  if (run.results.length === 0) {
    list.appendChild(el('div', { class: 'empty' }, run.state === 'running' ? '等待第一条结果…' : '无结果'));
  } else {
    // header
    list.appendChild(el('div', { class: 'live-row',
      style: { background: 'var(--panel-2)', color: 'var(--ink-4)', fontSize: '10px', letterSpacing: '.08em', textTransform: 'uppercase', cursor: 'default' } },
      el('span'), el('span', null, 'lang'), el('span', null, 'cat'), el('span', null, 'task'), el('span', null, 'eval'), el('span', { style: { textAlign: 'right' } }, 'ms'),
    ));
    run.results.forEach(r => list.appendChild(renderBenchResultRow(r)));
  }
  body.appendChild(list);
  panel.appendChild(body);
  return panel;
}

function renderBenchResultRow(r) {
  const t = r.task || {};
  const hasErr = !!r.error;
  const passed = r.pass === true;
  const failed = r.pass === false;
  const dotClass = hasErr ? 'fail' : failed ? 'fail' : passed ? 'pass' : 'pending';
  const row = el('div', { class: 'live-row' + (hasErr ? ' err' : '') + (failed ? ' fail-row' : ''),
    onclick: () => openBenchResultModal(r) });
  row.appendChild(el('span', { class: 'led-dot ' + dotClass }));
  row.appendChild(el('span', { class: 'lang' }, t.language || '—'));
  row.appendChild(el('span', { class: 'cat' }, t.category || '—'));
  row.appendChild(el('span', { class: 'preview' }, t.prompt ? t.prompt.slice(0, 80) : ''));
  // eval info column
  const evalCell = el('span', { class: 'eval-info' });
  if (hasErr) {
    evalCell.appendChild(el('span', { class: 'itag itag-bad' }, 'ERR'));
  } else if (r.score != null) {
    const s = Math.round(r.score * 100);
    evalCell.appendChild(el('span', { class: 'itag ' + (s >= 50 ? 'itag-good' : 'itag-bad') }, s + '%'));
  } else if (passed) {
    evalCell.appendChild(el('span', { class: 'itag itag-good' }, 'PASS'));
  } else if (failed) {
    evalCell.appendChild(el('span', { class: 'itag itag-bad' }, 'FAIL'));
  }
  if (r.eval_type && r.eval_type !== 'manual') {
    evalCell.appendChild(el('span', { class: 'eval-type' }, r.eval_type));
  }
  row.appendChild(evalCell);
  row.appendChild(el('span', { class: 'ms' }, fmtMs(r.elapsed_ms || 0)));
  return row;
}

function openBenchResultModal(r) {
  const t = r.task || {};
  const body = el('div');

  // tags
  const tags = el('div', { class: 'tag-row', style: { marginBottom: '12px' } });
  tags.appendChild(el('span', { class: 'itag itag-info' }, t.language || '—'));
  tags.appendChild(el('span', { class: 'itag itag-warn' }, t.category || '—'));
  if (r.error) tags.appendChild(el('span', { class: 'itag itag-bad' }, 'ERROR'));
  else if (r.pass === true) tags.appendChild(el('span', { class: 'itag itag-good' }, 'PASS'));
  else if (r.pass === false) tags.appendChild(el('span', { class: 'itag itag-bad' }, 'FAIL'));
  else tags.appendChild(el('span', { class: 'itag' }, 'DONE'));
  tags.appendChild(el('span', { class: 'itag' }, fmtMs(r.elapsed_ms || 0)));
  if (r.score != null) tags.appendChild(el('span', { class: 'itag ' + (r.score >= 0.5 ? 'itag-good' : 'itag-bad') }, 'score ' + Math.round(r.score * 100) + '%'));
  if (r.eval_type) tags.appendChild(el('span', { class: 'itag itag-accent' }, r.eval_type));
  tags.appendChild(el('span', { class: 'itag' }, t.task_id || ''));
  body.appendChild(tags);

  // evaluation verdict
  if (r.pass != null || r.score != null) {
    const evalPanel = el('div', { class: 'panel', style: { marginBottom: '12px' } });
    evalPanel.appendChild(el('div', { class: 'panel-head' },
      el('h3', null, '评估结果'),
      el('span', { class: 'meta' }, r.eval_type || 'auto'),
    ));
    const evalBody = el('div', { class: 'panel-body' });
    const evalGrid = el('div', { class: 'statbar', style: { marginBottom: '0' } });
    evalGrid.appendChild(el('div', { class: 'cell' },
      el('div', { class: 'k' }, 'VERDICT'),
      el('div', { class: 'v' },
        el('span', { class: 'pill ' + (r.pass ? 'pill-good' : 'pill-bad') }, el('span', { class: 'led' }), r.pass ? 'PASS' : 'FAIL'))));
    if (r.score != null) {
      evalGrid.appendChild(el('div', { class: 'cell' },
        el('div', { class: 'k' }, 'SCORE'),
        el('div', { class: 'v mono' }, Math.round(r.score * 100) + '%')));
    }
    evalGrid.appendChild(el('div', { class: 'cell' },
      el('div', { class: 'k' }, 'METHOD'),
      el('div', { class: 'v mono' }, r.eval_type || 'manual')));
    evalBody.appendChild(evalGrid);
    if (r.judge_reason) {
      evalBody.appendChild(el('div', { style: { marginTop: '8px', fontSize: '12px', color: 'var(--ink-2)', lineHeight: '1.5' } }, r.judge_reason));
    }
    evalPanel.appendChild(evalBody);
    body.appendChild(evalPanel);
  }

  // prompt
  body.appendChild(el('div', { class: 'eyebrow', style: { marginBottom: '4px' } }, 'PROMPT'));
  body.appendChild(el('pre', { class: 'json-out', style: { maxHeight: '240px' } }, t.prompt || ''));

  // error or answer
  if (r.error) {
    body.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '12px', marginBottom: '4px', color: 'var(--bad-ink)' } }, 'ERROR'));
    body.appendChild(el('pre', { class: 'json-out', style: { color: 'var(--bad-ink)' } }, r.error));
  } else if (r.answer) {
    body.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '12px', marginBottom: '4px' } }, 'ANSWER · ' + r.answer.length + ' chars'));
    body.appendChild(el('pre', { class: 'json-out' }, r.answer));
  }
  openModal('单题结果 · ' + (t.task_id || ''), body);
}

function rerunBench(runId) {
  const run = State.liveRuns[runId]; if (!run) return;
  if (run.payload) {
    const p = run.payload;
    let intensity = 'off';
    if (p.effort) intensity = p.effort;
    else if (p.thinking) intensity = p.thinking_mode || 'adaptive';
    Object.assign(State.bench, {
      targetBase: p.target_base, targetKey: p.target_key,
      model: p.model, concurrency: p.concurrency,
      intensity,
      lang: p.language || '', category: p.category || '',
      limit: p.limit || 0, scope: (p.language || p.category || p.limit) ? 'custom' : 'all',
    });
  }
  if (run.dataset) State.currentDataset = run.dataset;
  location.hash = '#/bench';
  toast('已填回配置,确认后点击开始运行', 'good');
}

/* ═══════════════════════════════════════════════════════════════════════
 * Batch (multi-effort) run page
 * ═══════════════════════════════════════════════════════════════════════ */
function renderBenchBatchPage(batchId) {
  const batch = State.liveRuns[batchId];
  if (!batch) return;
  const crumbs = [{ label: 'Benchmark', href: '#/bench' }, { cur: batch.dataset + ' · 多 Effort 对比' }];
  const actions = el('div', { class: 'crumb-actions' });
  if (batch.state === 'running') {
    actions.appendChild(btn('取消全部', { icon: 'stop', size: 'sm', danger: true, onClick: () => cancelBatchRun(batchId) }));
  } else {
    actions.appendChild(btn('导出 JSON', { icon: 'download', size: 'sm', ghost: true,
      onClick: () => {
        const exportData = {};
        Object.entries(batch.subRuns).forEach(([eff, sub]) => { exportData[eff] = sub.finalReport || { results: sub.results }; });
        downloadJSON(exportData, 'bench-batch-' + batchId + '.json');
      } }));
  }
  setCrumb(crumbs, actions);

  const v = $('#view'); v.innerHTML = '';

  // batch header
  v.appendChild(renderBatchHeader(batch, batchId));

  // per-effort progress
  v.appendChild(renderBatchEffortCards(batch));

  // score comparison chart
  if (batch.state === 'done' || batch.completedEfforts > 0) {
    v.appendChild(renderBatchScoreComparison(batch));
  }
}

function renderBatchHeader(batch, batchId) {
  const wrap = el('div', { class: 'statbar' });
  const statusBadge = (() => {
    if (batch.state === 'running') return el('span', { class: 'pill pill-running' }, el('span', { class: 'led' }), '运行中');
    if (batch.state === 'cancelled') return el('span', { class: 'pill pill-warn' }, el('span', { class: 'led' }), '已取消');
    return el('span', { class: 'pill pill-good' }, el('span', { class: 'led' }), '完成');
  })();
  wrap.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'STATUS'), el('div', { class: 'v' }, statusBadge)));
  wrap.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'DATASET'), el('div', { class: 'v' }, batch.dataset || '—')));
  wrap.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'MODEL'), el('div', { class: 'v mono' }, batch.model || '—')));
  wrap.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'THINKING'), el('div', { class: 'v' }, batch.thinkingMode || (batch.thinking ? 'on' : 'off'))));
  wrap.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'EFFORTS'), el('div', { class: 'v' }, batch.efforts.join(', '))));
  wrap.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'PROGRESS'),
    el('div', { class: 'v mono' }, batch.completedEfforts + ' / ' + batch.totalEfforts)));

  if (batch.state === 'running') {
    const elapsed = el('span', { class: 'mono', id: 'batchLiveElapsed' }, fmtMs(Date.now() - batch.startedAt));
    wrap.appendChild(el('div', { class: 'cell' }, el('div', { class: 'k' }, 'ELAPSED'), el('div', { class: 'v' }, elapsed)));
    if (!batch._tick) {
      batch._tick = setInterval(() => {
        if (batch.state !== 'running') { clearInterval(batch._tick); batch._tick = null; return; }
        const cur = document.getElementById('batchLiveElapsed');
        if (cur) cur.textContent = fmtMs(Date.now() - batch.startedAt);
      }, 1000);
    }
  }
  return wrap;
}

function renderBatchEffortCards(batch) {
  const panel = el('div', { class: 'panel', style: { marginBottom: '12px' } });
  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '各 Effort 级别'),
    el('span', { class: 'meta' }, batch.efforts.length + ' 个'),
  ));
  const body = el('div', { class: 'panel-body', style: { padding: '12px' } });
  const grid = el('div', { style: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '10px' } });

  batch.efforts.forEach(effort => {
    const sub = batch.subRuns[effort];
    const card = el('div', { class: 'model-card' });
    const top = el('div', { class: 'top' });

    if (!sub) {
      top.appendChild(el('span', { class: 'led-dot pending' }));
      top.appendChild(el('div', { class: 'name' }, effort));
      top.appendChild(el('span', { class: 'mono', style: { fontSize: '11px', color: 'var(--ink-4)' } }, '等待中'));
    } else if (sub.state === 'running') {
      top.appendChild(el('span', { class: 'led-dot running' }));
      top.appendChild(el('div', { class: 'name' }, effort));
      top.appendChild(el('span', { class: 'mono', style: { fontSize: '11px', color: 'var(--ink-3)' } },
        sub.completedTasks + '/' + sub.totalTasks));
      const pct = sub.totalTasks > 0 ? Math.round(sub.completedTasks / sub.totalTasks * 100) : 0;
      card.appendChild(top);
      card.appendChild(el('div', { style: { padding: '6px 10px 10px' } },
        el('div', { class: 'progress', style: { height: '3px' } },
          el('div', { class: 'progress-fill', style: { width: pct + '%' } }))));
      grid.appendChild(card);
      return;
    } else if (sub.state === 'error') {
      top.appendChild(el('span', { class: 'led-dot fail' }));
      top.appendChild(el('div', { class: 'name' }, effort));
      top.appendChild(el('span', { style: { fontSize: '11px', color: 'var(--bad-ink)' } }, '失败'));
    } else if (sub.state === 'cancelled') {
      top.appendChild(el('span', { class: 'led-dot pending' }));
      top.appendChild(el('div', { class: 'name' }, effort));
      top.appendChild(el('span', { style: { fontSize: '11px', color: 'var(--warn-ink)' } }, '已取消'));
    } else {
      top.appendChild(el('span', { class: 'led-dot pass' }));
      top.appendChild(el('div', { class: 'name' }, effort));
      const rep = sub.finalReport;
      if (rep) {
        const score = rep.score_total != null ? Math.round(rep.score_total * 100) / 100 : null;
        const rate = rep.pass_rate != null ? Math.round(rep.pass_rate * 1000) / 10 : null;
        top.appendChild(el('span', { class: 'mono', style: { fontSize: '11px', fontWeight: 600 } },
          score != null ? score + ' pts' : sub.completedTasks + '/' + sub.totalTasks));
        if (rate != null) {
          card.appendChild(top);
          card.appendChild(el('div', { style: { padding: '4px 10px 8px', fontSize: '11px', color: 'var(--ink-3)' } },
            'pass rate: ' + rate + '% · ' + fmtMs(rep.elapsed_ms)));
          grid.appendChild(card);
          return;
        }
      }
    }
    card.appendChild(top);
    grid.appendChild(card);
  });

  body.appendChild(grid);
  panel.appendChild(body);
  return panel;
}

function renderBatchScoreComparison(batch) {
  const panel = el('div', { class: 'panel' });
  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, 'Effort 对比'),
    el('span', { class: 'meta' }, '分数 / 通过率 / 耗时'),
  ));

  const body = el('div', { class: 'panel-body', style: { padding: '0' } });
  const t = el('table', { class: 'table' });
  t.appendChild(el('thead', null, el('tr', null,
    el('th', null, 'effort'),
    el('th', { style: { textAlign: 'right' } }, 'score'),
    el('th', { style: { textAlign: 'right' } }, 'pass rate'),
    el('th', { style: { textAlign: 'right' } }, 'tasks'),
    el('th', { style: { textAlign: 'right' } }, 'errors'),
    el('th', { style: { textAlign: 'right' } }, 'avg/task'),
    el('th', { style: { textAlign: 'right' } }, 'total time'),
    el('th', null, 'status'),
  )));

  const tb = el('tbody');
  batch.efforts.forEach(effort => {
    const sub = batch.subRuns[effort];
    if (!sub) return;
    const rep = sub.finalReport;
    const tr = el('tr');
    tr.appendChild(el('td', null, el('span', { class: 'itag itag-warn' }, effort)));

    if (rep) {
      const score = rep.score_total != null ? Math.round(rep.score_total * 100) / 100 : null;
      const rate = rep.pass_rate != null ? Math.round(rep.pass_rate * 1000) / 10 : null;
      tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right', fontWeight: 600 } },
        score != null ? String(score) : '—'));
      tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } },
        rate != null ? rate + '%' : '—'));
      tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } },
        (rep.task_completed || sub.completedTasks) + '/' + (rep.task_total || sub.totalTasks)));
      tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right', color: (rep.task_errors || 0) > 0 ? 'var(--bad-ink)' : 'var(--ink-3)' } },
        rep.task_errors || 0));
      const avgMs = sub.results.length > 0
        ? Math.round(sub.results.reduce((s, r) => s + (r.elapsed_ms || 0), 0) / sub.results.length)
        : 0;
      tr.appendChild(el('td', { class: 'mono', style: { textAlign: 'right' } }, avgMs ? fmtMs(avgMs) : '—'));
      tr.appendChild(el('td', { class: 'mono', style: { textAlign: 'right' } }, fmtMs(rep.elapsed_ms)));
    } else {
      tr.appendChild(el('td', { style: { textAlign: 'right' } }, '—'));
      tr.appendChild(el('td', { style: { textAlign: 'right' } }, '—'));
      tr.appendChild(el('td', { class: 'mono', style: { textAlign: 'right' } },
        sub.completedTasks + '/' + sub.totalTasks));
      tr.appendChild(el('td', { style: { textAlign: 'right' } }, sub.errorTasks || 0));
      tr.appendChild(el('td', { style: { textAlign: 'right' } }, '—'));
      tr.appendChild(el('td', { style: { textAlign: 'right' } }, '—'));
    }

    const statePill = sub.state === 'done' ? el('span', { class: 'pill pill-good' }, el('span', { class: 'led' }), '完成')
      : sub.state === 'running' ? el('span', { class: 'pill pill-running' }, el('span', { class: 'led' }), '运行中')
      : sub.state === 'error' ? el('span', { class: 'pill pill-bad' }, el('span', { class: 'led' }), '失败')
      : el('span', { class: 'pill pill-warn' }, el('span', { class: 'led' }), sub.state);
    tr.appendChild(el('td', null, statePill));
    tb.appendChild(tr);
  });
  t.appendChild(tb);
  body.appendChild(t);

  // bar chart visualization
  const chartWrap = el('div', { style: { padding: '14px', borderTop: '1px solid var(--line)' } });
  const maxScore = Math.max(...batch.efforts.map(e => {
    const sub = batch.subRuns[e];
    return sub && sub.finalReport && sub.finalReport.score_total != null ? sub.finalReport.score_total : 0;
  }), 1);

  batch.efforts.forEach(effort => {
    const sub = batch.subRuns[effort];
    const rep = sub ? sub.finalReport : null;
    const score = rep && rep.score_total != null ? rep.score_total : 0;
    const pct = Math.round(score / maxScore * 100);
    const row = el('div', { style: { display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' } });
    row.appendChild(el('span', { class: 'mono', style: { width: '60px', fontSize: '11px', textAlign: 'right', flexShrink: '0' } }, effort));
    row.appendChild(el('div', { style: { flex: '1', background: 'var(--panel-2)', borderRadius: '3px', height: '18px', overflow: 'hidden' } },
      el('div', { style: { width: pct + '%', height: '100%', background: 'var(--accent)', borderRadius: '3px', transition: 'width .3s' } })));
    row.appendChild(el('span', { class: 'mono', style: { width: '50px', fontSize: '11px', color: 'var(--ink-3)' } },
      score > 0 ? (Math.round(score * 100) / 100) : '—'));
    chartWrap.appendChild(row);
  });
  body.appendChild(chartWrap);

  panel.appendChild(body);
  return panel;
}
