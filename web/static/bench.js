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
      buildField('MODEL', el('input', {
        class: 'mono', placeholder: 'claude-opus-4-6', value: B.model,
        oninput: e => { B.model = e.target.value.trim(); },
        list: 'benchModelList',
        style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
      })),
      (() => {
        const dl = el('datalist', { id: 'benchModelList' });
        ['claude-opus-4-6', 'claude-opus-4-7', 'claude-sonnet-4-6', 'claude-haiku-4-5'].forEach(m => dl.appendChild(el('option', { value: m })));
        return dl;
      })(),
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

  // Current dataset summary + CTA
  if (State.currentDataset) {
    const summaryPanel = el('div', { class: 'panel', id: 'dsSummaryPanel' });
    summaryPanel.appendChild(el('div', { class: 'panel-head' },
      el('h3', null, State.currentDataset),
      el('div', { class: 'spacer' }),
      btn('开始运行', { primary: true, icon: 'play', onClick: () => openRunConfigDrawer() }),
    ));
    summaryPanel.appendChild(el('div', { class: 'panel-body', id: 'dsSummaryBody' }, el('div', { class: 'muted' }, '加载中…')));
    wrap.appendChild(summaryPanel);
    loadDatasetSummary(State.currentDataset);
  }
  return wrap;
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

/* ─── Run config drawer (shown when clicking "开始运行") ─── */
function openRunConfigDrawer() {
  const B = State.bench;
  const body = el('div');

  const onChange = (k, parse) => e => { B[k] = parse ? parse(e.target.value) : e.target.value; };

  body.appendChild(el('div', { class: 'eyebrow', style: { marginBottom: '6px' } }, 'SCOPE'));
  body.appendChild((() => {
    const seg = el('div', { class: 'seg', style: { width: 'fit-content', marginBottom: '12px' } });
    [['all', '全量运行'], ['custom', '自定义筛选']].forEach(([v, lbl]) => {
      const b = el('button', { class: B.scope === v ? 'active' : '',
        onclick: () => { B.scope = v; openRunConfigDrawer(); } }, lbl);
      seg.appendChild(b);
    });
    return seg;
  })());

  if (B.scope === 'custom') {
    body.appendChild(buildField('LANGUAGE', el('input', { value: B.lang, placeholder: 'go / python (留空=全部)',
      oninput: onChange('lang'),
      style: { width: '100%', background: 'transparent', border: 'none', padding: '0' } })));
    body.appendChild(buildField('CATEGORY', el('input', { value: B.category, placeholder: 'Security (留空=全部)',
      oninput: onChange('category'),
      style: { width: '100%', background: 'transparent', border: 'none', padding: '0' } })));
    body.appendChild(buildField('LIMIT', el('input', { type: 'number', value: B.limit, placeholder: '0=全部',
      oninput: onChange('limit', v => parseInt(v) || 0),
      class: 'mono', style: { width: '120px', background: 'transparent', border: 'none', padding: '0' } })));
  }

  body.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '14px', marginBottom: '6px' } }, 'EXECUTION'));
  body.appendChild(buildField('CONCURRENCY', el('input', { type: 'number', min: 1, max: 20,
    value: B.concurrency, class: 'mono',
    oninput: onChange('concurrency', v => parseInt(v) || 1),
    style: { width: '80px', background: 'transparent', border: 'none', padding: '0' } })));

  body.appendChild(buildField('THINKING', (() => {
    const seg = el('div', { class: 'seg', style: { width: 'fit-content' } });
    [['off', 'OFF'], ['on', 'ON']].forEach(([v, lbl]) => {
      const b = el('button', { class: (B.thinking ? 'on' : 'off') === v ? 'active' : '',
        onclick: () => { B.thinking = v === 'on'; openRunConfigDrawer(); } }, lbl);
      seg.appendChild(b);
    });
    return seg;
  })()));

  body.appendChild(buildField('MODEL OVERRIDE (optional)', el('input', { value: B.runModel, placeholder: '留空=使用上方 Model',
    oninput: onChange('runModel'), class: 'mono',
    style: { width: '100%', background: 'transparent', border: 'none', padding: '0' } })));

  const foot = el('div', null,
    btn('取消', { ghost: true, onClick: closeDrawer }),
    btn('开始运行', { primary: true, icon: 'play', onClick: () => { closeDrawer(); kickoffBenchRun(); } }),
  );

  openDrawer('运行配置 · ' + State.currentDataset, body, foot);
}

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

  const tempId = 'live_' + Date.now().toString(36);
  const payload = {
    target_base: B.targetBase, target_key: B.targetKey,
    model: B.runModel || B.model,
    concurrency: B.concurrency, thinking: B.thinking,
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
    model: payload.model,
    startedAt: Date.now(),
    totalTasks: 0,
    completedTasks: 0,
    errorTasks: 0,
    results: [],
    aborter: null,
    finalReport: null,
    error: null,
    thinking: B.thinking,
  };
  location.hash = '#/bench/run/' + tempId;
  runBenchStream(tempId, payload).catch(err => {
    const r = State.liveRuns[tempId]; if (!r) return;
    r.state = 'error'; r.error = err.message || String(err);
    maybeRerenderBench(tempId);
  });
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
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
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
}

function handleBenchSSE(runId, ev) {
  const run = State.liveRuns[runId];
  if (!run) return;
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
    maybeRerenderBench(runId);
  } else if (ev.type === 'error') {
    run.state = 'error'; run.error = ev.error;
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
  _benchRenderTimer = requestAnimationFrame(() => { _benchRenderTimer = null; renderBenchRunPage(runId); });
}

/* ─── Bench run page ─── */
async function renderBenchRunRoute(runId) {
  if (State.liveRuns[runId]) { renderBenchRunPage(runId); return; }
  const v = $('#view');
  v.innerHTML = '<div class="empty">加载历史中…</div>';
  try {
    const data = await api('/api/intelligence/history/' + encodeURIComponent(runId));
    State.liveRuns[runId] = {
      kind: 'bench', state: 'done', historical: true,
      dataset: data.dataset_name, target: data.target || '',
      model: data.model, thinking: data.thinking,
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

  // progress strip (running only)
  if (run.state === 'running') v.appendChild(renderBenchProgressStrip(run));

  // task stats
  v.appendChild(renderBenchStats(run));

  // results list
  v.appendChild(renderBenchResultsList(run));
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
  cells.push(['THINKING', run.thinking ? 'on' : 'off']);
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
      el('span'), el('span', null, 'lang'), el('span', null, 'cat'), el('span', null, 'task'), el('span', { style: { textAlign: 'right' } }, 'ms'), el('span'),
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
  const row = el('div', { class: 'live-row' + (hasErr ? ' err' : ''),
    onclick: () => openBenchResultModal(r) });
  row.appendChild(el('span', { class: 'led-dot ' + (hasErr ? 'fail' : 'pass') }));
  row.appendChild(el('span', { class: 'lang' }, t.language || '—'));
  row.appendChild(el('span', { class: 'cat' }, t.category || '—'));
  row.appendChild(el('span', { class: 'preview' }, t.prompt ? t.prompt.slice(0, 80) : ''));
  row.appendChild(el('span', { class: 'ms' }, fmtMs(r.elapsed_ms || 0)));
  row.appendChild(el('span', { class: 'id', style: { textAlign: 'right' } }, (t.task_id || '').slice(0, 10)));
  return row;
}

function openBenchResultModal(r) {
  const t = r.task || {};
  const body = el('div');
  body.appendChild(el('div', { class: 'tag-row', style: { marginBottom: '12px' } },
    el('span', { class: 'itag itag-info' }, t.language || '—'),
    el('span', { class: 'itag itag-warn' }, t.category || '—'),
    r.error ? el('span', { class: 'itag itag-bad' }, 'ERROR') : el('span', { class: 'itag itag-good' }, 'OK'),
    el('span', { class: 'itag' }, fmtMs(r.elapsed_ms || 0)),
    el('span', { class: 'itag' }, t.task_id || ''),
  ));
  body.appendChild(el('div', { class: 'eyebrow', style: { marginBottom: '4px' } }, 'PROMPT'));
  body.appendChild(el('pre', { class: 'json-out', style: { maxHeight: '240px' } }, t.prompt || ''));
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
    Object.assign(State.bench, {
      targetBase: run.payload.target_base, targetKey: run.payload.target_key,
      model: run.payload.model, concurrency: run.payload.concurrency,
      thinking: run.payload.thinking, lang: run.payload.language || '', category: run.payload.category || '',
      limit: run.payload.limit || 0, scope: (run.payload.language || run.payload.category || run.payload.limit) ? 'custom' : 'all',
    });
    if (run.dataset) State.currentDataset = run.dataset;
    location.hash = '#/bench';
    setTimeout(() => openRunConfigDrawer(), 200);
  } else {
    if (run.dataset) State.currentDataset = run.dataset;
    location.hash = '#/bench';
    toast('已填回部分配置,请确认 API Key 后重新开始', 'good');
  }
}
