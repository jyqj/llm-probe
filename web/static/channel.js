/* ─── channel.js · Channel test pages ─────────────────────────────────── */
/*
 * SSE protocol expected at POST /api/channel/run/stream:
 *
 *   data: {"type":"start","run_id":"abc","models":["m1",...],"total_probes":N}
 *   data: {"type":"probe_start","model":"m1","probe_id":"id","label":"..."}
 *   data: {"type":"probe_done","model":"m1","probe_id":"id","probe":{...probeResult}}
 *   data: {"type":"model_done","model":"m1","report":{...full single-model report}}
 *   data: {"type":"done","reports":[...]}      // final list
 *   data: {"type":"error","error":"..."}
 *
 * Frontend falls back gracefully to the sync /api/channel/run endpoint
 * if /stream returns 404 — in that case it shows an indeterminate progress
 * state until the sync response lands.
 */

/* ─── New-run config view ─── */
function renderChannelConfig() {
  ensureModelMeta().then(() => {
    const holder = document.getElementById('channelModelChips');
    if (holder) {
      holder.replaceWith(buildModelChips());
    }
  });
  setCrumb([{ label: 'Channel', href: '#/channel' }, { cur: '新建检测' }],
    el('div', { class: 'crumb-actions' },
      btn('查看历史', { onClick: () => location.hash = '#/channel/history', icon: 'history', size: 'sm', ghost: true })
    )
  );

  const v = $('#view');
  v.innerHTML = '';
  v.appendChild(buildChannelConfigPanel());
}

function buildChannelConfigPanel() {
  const C = State.channel;
  const wrap = el('div');

  // info strip
  wrap.appendChild(el('div', { class: 'panel', style: { marginBottom: '12px' } },
    el('div', { class: 'panel-head' },
      el('h3', null, '新建渠道检测'),
      el('span', { class: 'meta' }, '通过指纹 / 结构 / 签名 / 行为 / 多模态 五维探测核验渠道是否官方'),
    ),
    el('div', { class: 'panel-body' },
      buildField('TARGET BASE URL', el('input', {
        class: 'mono', id: 'cfgTargetBase', placeholder: 'https://api.example.com', value: C.targetBase,
        oninput: e => { C.targetBase = e.target.value.trim(); updateConn(); },
        style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
      })),
      buildField('API KEY', el('input', {
        type: 'password', class: 'mono', id: 'cfgTargetKey', placeholder: 'sk-...', value: C.targetKey,
        oninput: e => { C.targetKey = e.target.value; },
        style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
      })),
      buildField('CHANNEL NAME (optional)', el('input', {
        id: 'cfgChannelName', placeholder: '(auto)', value: C.channelName,
        oninput: e => { C.channelName = e.target.value.trim(); },
        style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
      })),
      buildField('MODELS', buildModelChips()),
      buildField('PROFILE', buildProfileSelect()),
      buildField('BASELINE', buildBaselineSelect()),
      buildField('CONCURRENCY', el('input', {
        type: 'number', min: 0, max: 10, value: C.concurrency, class: 'mono',
        style: { width: '70px', background: 'transparent', border: 'none', padding: '0' },
        oninput: e => { C.concurrency = parseInt(e.target.value) || 0; },
      })),
    ),
    el('div', {
      class: 'panel-head',
      style: { borderTop: '1px solid var(--line)', borderBottom: 'none', justifyContent: 'flex-end' }
    },
      btn('运行渠道检测', {
        primary: true, icon: 'play',
        onClick: () => kickoffChannelRun(),
      })
    )
  ));

  return wrap;
}

function buildField(label, input) {
  return el('div', { class: 'field', style: { borderBottom: '1px solid var(--line-soft)' } },
    el('div', { class: 'field-label' }, label),
    input
  );
}

function buildModelChips() {
  const ALL = modelIDs();
  const set = new Set(State.channel.models);
  const wrap = el('div', { class: 'chip-set', id: 'channelModelChips' });
  ALL.forEach(m => {
    const lbl = el('label', { class: 'chip' });
    const cb = el('input', { type: 'checkbox', value: m });
    if (set.has(m)) cb.checked = true;
    cb.addEventListener('change', () => {
      if (cb.checked) set.add(m); else set.delete(m);
      State.channel.models = Array.from(set);
    });
    lbl.appendChild(cb);
    lbl.appendChild(el('span', { class: 'led' }));
    lbl.appendChild(document.createTextNode(m));
    wrap.appendChild(lbl);
  });
  return wrap;
}

function buildProfileSelect() {
  const profiles = [
    { id: '', label: '自动 (Console 标准)' },
    { id: 'console', label: 'Console 直连' },
    { id: 'bedrock', label: 'AWS Bedrock' },
    { id: 'vertex', label: 'Google Vertex AI' },
    { id: 'max', label: 'Max 订阅' },
  ];
  const sel = el('select', {
    class: 'mono',
    style: { background: 'transparent', border: 'none', padding: '0', color: 'var(--ink-1)' },
    onchange: e => { State.channel.profile = e.target.value; },
  });
  profiles.forEach(p => {
    const opt = el('option', { value: p.id }, p.label);
    if (p.id === (State.channel.profile || '')) opt.selected = true;
    sel.appendChild(opt);
  });
  return sel;
}

function buildBaselineSelect() {
  const sel = el('select', {
    class: 'mono',
    style: { background: 'transparent', border: 'none', padding: '0', color: 'var(--ink-1)' },
    onchange: e => { State.channel.baselineId = e.target.value; },
  });
  sel.appendChild(el('option', { value: '' }, '无 (不对比)'));
  api('/api/monitor/baselines').then(data => {
    if (data && data.baselines) {
      data.baselines.forEach(b => {
        const label = (b.name || b.id) + ' · ' + b.model + (b.profile ? ' [' + b.profile + ']' : '');
        const opt = el('option', { value: b.id }, label);
        if (b.id === State.channel.baselineId) opt.selected = true;
        sel.appendChild(opt);
      });
    }
  });
  return sel;
}

/* ─── Kick off a run ─── */
async function kickoffChannelRun() {
  const C = State.channel;
  if (!C.targetBase) { toast('请填写 Target Base URL', 'bad'); return; }
  if (!C.targetKey)  { toast('请填写 API Key', 'bad'); return; }
  if (!C.models.length) { toast('请至少勾选一个 Model', 'bad'); return; }

  const tempId = 'live_' + Date.now().toString(36);
  const payload = {
    target_base: C.targetBase,
    target_key:  C.targetKey,
    model:       C.models[0],
    models:      C.models,
    channel_name: C.channelName,
    concurrency: C.concurrency,
    profile:     C.profile || '',
    baseline_id: C.baselineId || undefined,
  };

  State.liveRuns[tempId] = {
    kind: 'channel',
    state: 'running',
    payload,
    models: C.models.slice(),
    channelName: C.channelName,
    targetBase: C.targetBase,
    target: C.targetBase,
    startedAt: Date.now(),
    perModel: {},               // model → { reports, probes: {probeId → probeResult}, status }
    aborter: null,
    progressTotal: 0,
    progressDone: 0,
    error: null,
    finalReports: null,
    realRunId: null,
  };
  C.models.forEach(m => {
    State.liveRuns[tempId].perModel[m] = { status: 'pending', probes: {}, probeOrder: [], report: null };
  });

  // navigate immediately
  location.hash = '#/channel/run/' + tempId;
  // start work
  runChannelStream(tempId, payload).catch(err => {
    const r = State.liveRuns[tempId]; if (!r) return;
    r.state = 'error'; r.error = err.message || String(err);
    renderRunPage(tempId);
  });
}

async function runChannelStream(runId, payload) {
  const run = State.liveRuns[runId];
  const ac = new AbortController();
  run.aborter = ac;

  // try SSE first
  let resp;
  try {
    resp = await fetch('/api/channel/run/stream', {
      method: 'POST', headers: headers(),
      body: JSON.stringify(payload), signal: ac.signal,
    });
  } catch (e) {
    if (e.name === 'AbortError') throw e;
    resp = null;
  }

  if (resp && resp.ok && (resp.headers.get('content-type') || '').includes('text/event-stream')) {
    await consumeChannelSSE(runId, resp);
    return;
  }
  if (resp && resp.status === 404) {
    // fall back to sync
    return runChannelSync(runId, payload, ac);
  }
  // some other response — try sync as last resort
  return runChannelSync(runId, payload, ac);
}

async function consumeChannelSSE(runId, resp) {
  const run = State.liveRuns[runId];
  const reader = resp.body.getReader();
  const decoder = new TextDecoder();
  let buf = '';
  const watchdog = createSSEWatchdog(45000, () => {
    if (run.state === 'running') {
      toast('SSE 连接可能已断开 · 服务端仍在执行', 'warn');
    }
  });
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      watchdog.reset();
      buf += decoder.decode(value, { stream: true });
      const lines = buf.split('\n');
      buf = lines.pop();
      for (const line of lines) {
        if (!line.startsWith('data: ')) continue;
        try { handleChannelSSE(runId, JSON.parse(line.slice(6))); } catch {}
      }
    }
    if (buf.startsWith('data: ')) {
      try { handleChannelSSE(runId, JSON.parse(buf.slice(6))); } catch {}
    }
  } finally {
    watchdog.stop();
  }
}

function handleChannelSSE(runId, ev) {
  const run = State.liveRuns[runId];
  if (!run) return;
  switch (ev.type) {
    case 'start':
      if (ev.models) run.models = ev.models;
      run.progressTotal = ev.total_probes || 0;
      if (ev.run_id && ev.run_id !== runId) {
        run.realRunId = ev.run_id;
      }
      if (ev.model_probes) {
        for (const [model, probeList] of Object.entries(ev.model_probes)) {
          const pm = run.perModel[model] ||= { status: 'pending', probes: {}, probeOrder: [], report: null };
          probeList.forEach(p => {
            if (!pm.probes[p.id]) {
              pm.probes[p.id] = { state: 'pending', probe_id: p.id, label: p.label, checks: [], latency_ms: null };
              pm.probeOrder.push(p.id);
            }
          });
        }
      }
      maybeRerender(runId);
      break;
    case 'probe_start': {
      const pm = run.perModel[ev.model] ||= { status: 'pending', probes: {}, probeOrder: [], report: null };
      pm.status = 'running';
      const pid = ev.probe_id || ev.label;
      if (!pm.probes[pid]) {
        pm.probes[pid] = { state: 'running', probe_id: pid, label: ev.label || pid, checks: [], latency_ms: null };
        pm.probeOrder.push(pid);
      } else { pm.probes[pid].state = 'running'; }
      maybeRerender(runId);
      break;
    }
    case 'probe_done': {
      const pm = run.perModel[ev.model];
      if (!pm) break;
      const pid = ev.probe && ev.probe.probe_id ? ev.probe.probe_id : ev.probe_id;
      const prev = pm.probes[pid] || { probe_id: pid, label: pid };
      pm.probes[pid] = Object.assign(prev, ev.probe, { state: 'done' });
      if (!pm.probeOrder.includes(pid)) pm.probeOrder.push(pid);
      run.progressDone++;
      maybeRerender(runId);
      break;
    }
    case 'model_done': {
      const pm = run.perModel[ev.model] ||= { status: 'pending', probes: {}, probeOrder: [], report: null };
      pm.report = ev.report;
      pm.status = 'done';
      maybeRerender(runId);
      break;
    }
    case 'done':
      run.state = 'done';
      run.finalReports = ev.reports || [];
      run.finishedAt = Date.now();
      (ev.reports || []).forEach(rep => {
        const pm = run.perModel[rep.model] ||= { status: 'done', probes: {}, probeOrder: [], report: rep };
        pm.report = rep;
        pm.status = 'done';
      });
      if (ev.reports && ev.reports.length && ev.reports[0].id) {
        const backendId = ev.reports[0].id;
        State.liveRuns[backendId] = run;
        const route = parseRoute(location.hash);
        if (route.id === runId) {
          history.replaceState(null, '', '#/channel/run/' + backendId);
        }
        runId = backendId;
      }
      toast('渠道检测完成', 'good');
      maybeRerender(runId);
      break;
    case 'error':
      run.state = 'error'; run.error = ev.error || 'unknown error';
      maybeRerender(runId);
      break;
  }
}

async function runChannelSync(runId, payload, ac) {
  const run = State.liveRuns[runId];
  let resp, data;
  try {
    resp = await fetch('/api/channel/run', {
      method: 'POST', headers: headers(),
      body: JSON.stringify(payload), signal: ac.signal,
    });
    const text = await resp.text();
    try { data = JSON.parse(text); } catch { data = { error: text }; }
    if (!resp.ok) throw new Error((data && data.error) || 'request failed');
  } catch (e) {
    if (e.name === 'AbortError') { run.state = 'cancelled'; maybeRerender(runId); return; }
    throw e;
  }
  // shape: single report or {reports:[...]}
  const reports = data.reports ? data.reports : [data];
  run.state = 'done';
  run.finalReports = reports;
  run.finishedAt = Date.now();
  reports.forEach(rep => {
    const pm = run.perModel[rep.model] ||= { status: 'done', probes: {}, probeOrder: [], report: rep };
    pm.report = rep;
    pm.status = 'done';
    (rep.probe_results || []).forEach(p => {
      pm.probes[p.probe_id] = Object.assign({}, p, { state: 'done' });
      if (!pm.probeOrder.includes(p.probe_id)) pm.probeOrder.push(p.probe_id);
    });
  });
  // Redirect to backend ID for persistence across refresh
  if (reports.length && reports[0].id) {
    const backendId = reports[0].id;
    State.liveRuns[backendId] = run;
    const route = parseRoute(location.hash);
    if (route.id === runId) {
      history.replaceState(null, '', '#/channel/run/' + backendId);
      runId = backendId;
    }
  }
  maybeRerender(runId);
}

function cancelChannelRun(runId) {
  const run = State.liveRuns[runId];
  if (!run) return;
  if (run.aborter) run.aborter.abort();
  run.state = 'cancelled';
  maybeRerender(runId);
  toast(run.progressTotal > 0 ? '已取消' : '已断开 · 服务端可能仍在执行', 'warn');
}

/* ─── Re-render guard — only re-render if route still on this run ─── */
let _renderTimer = null;
function maybeRerender(runId) {
  const route = parseRoute(location.hash);
  // Refresh rail when run state changes (clears stale RUNNING entries)
  const run = State.liveRuns[runId];
  if (run && run.state !== 'running') paintRail(route);
  if (route.app !== 'channel' || route.kind !== 'run' || route.id !== runId) return;
  if (_renderTimer) return;
  _renderTimer = requestAnimationFrame(() => {
    _renderTimer = null;
    renderRunPage(runId);
  });
}

/* ═══════════════════════════════════════════════════════════════════════
 * Run page — shared by RUNNING / DONE / HISTORY-DETAIL
 * ═══════════════════════════════════════════════════════════════════════ */

async function renderChannelRunRoute(runId, model, anchor) {
  if (State.liveRuns[runId]) { renderRunPage(runId, model, anchor); return; }
  const v = $('#view');
  v.innerHTML = '<div class="empty"><div class="glyph">/</div>加载历史中...</div>';
  try {
    const data = await api('/api/channel/history/' + encodeURIComponent(runId));
    let reports = Array.isArray(data) ? data : (data.reports ? data.reports : [data]);

    // If report belongs to a multi-model run group, load all grouped reports
    const runGroup = reports[0] && reports[0].run_group;
    if (runGroup && reports.length === 1) {
      try {
        const groupData = await api('/api/channel/history/group/' + encodeURIComponent(runGroup));
        if (groupData.reports && groupData.reports.length > 1) {
          reports = groupData.reports;
        }
      } catch (_) { /* fall back to single report */ }
    }

    const run = {
      kind: 'channel', state: 'done', historical: true,
      models: reports.map(r => r.model),
      channelName: reports[0] ? reports[0].channel_name : '',
      target: reports[0] ? reports[0].target : '',
      startedAt: reports[0] && reports[0].timestamp ? new Date(reports[0].timestamp).getTime() : null,
      finishedAt: null,
      perModel: {},
      finalReports: reports,
      progressTotal: 0, progressDone: 0,
      payload: null,
    };
    reports.forEach(rep => {
      const probes = {}; const order = [];
      (rep.probe_results || []).forEach(p => { probes[p.probe_id] = Object.assign({}, p, { state: 'done' }); order.push(p.probe_id); });
      run.perModel[rep.model] = { status: 'done', probes, probeOrder: order, report: rep };
    });
    State.liveRuns[runId] = run;
    renderRunPage(runId, model, anchor);
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty' },
      el('div', { class: 'glyph' }, '×'),
      '未找到此次运行: ' + esc(runId),
      el('div', { style: { marginTop: '12px' } },
        btn('返回历史', { onClick: () => location.hash = '#/channel/history', icon: 'history', size: 'sm' })
      )
    ));
  }
}

function renderRunPage(runId, focusedModel, anchor) {
  const run = State.liveRuns[runId];
  if (!run) return;
  const models = Array.from(new Set(run.models || Object.keys(run.perModel)));
  if (!focusedModel || !models.includes(focusedModel)) focusedModel = models[0];

  // crumb
  const crumbs = [{ label: 'Channel', href: '#/channel' }];
  if (run.historical) crumbs.push({ label: '历史', href: '#/channel/history' });
  crumbs.push({ cur: run.channelName || (run.target || '').replace(/^https?:\/\//, '').split('/')[0] || 'run' });
  if (run.state !== 'running') crumbs.push({ cur: '报告' });
  if (models.length > 1) crumbs.push({ cur: focusedModel, mono: true });

  const actions = el('div', { class: 'crumb-actions' });
  if (run.state === 'running') {
    actions.appendChild(btn('取消', { icon: 'stop', size: 'sm', danger: true, onClick: () => cancelChannelRun(runId) }));
  } else {
    const failedProbes = countFailedProbes(run, focusedModel);
    if (failedProbes > 0) {
      actions.appendChild(btn('重试失败 (' + failedProbes + ')', { icon: 'refresh', size: 'sm',
        onClick: () => retryAllFailedProbes(runId, focusedModel) }));
    }
    if (run.payload) actions.appendChild(btn('重新运行', { icon: 'refresh', size: 'sm', ghost: true, onClick: () => rerunChannel(runId) }));
    actions.appendChild(btn('复制失败为 MD', { icon: 'copy', size: 'sm', ghost: true, onClick: () => copyFailuresMd(runId, focusedModel) }));
    actions.appendChild(btn('导出 JSON', { icon: 'download', size: 'sm', ghost: true, onClick: () => exportRunJson(runId) }));
  }
  setCrumb(crumbs, actions);

  const v = $('#view');
  v.innerHTML = '';

  // ─── Status bar ───
  v.appendChild(renderRunHeader(run, runId));

  // ─── Multi-model overview grid ───
  if (models.length > 1) v.appendChild(renderModelGrid(run, models, focusedModel, runId));

  // ─── Running: probe board · Done: report ───
  const pm = run.perModel[focusedModel] || { status: 'pending' };
  if (run.state === 'running' && pm.status !== 'done') {
    v.appendChild(renderProbeBoard(run, focusedModel, runId));
  } else {
    if (run.state !== 'running') v.appendChild(renderReportBanner(run));
    v.appendChild(renderModelDetail(run, focusedModel, runId, anchor));
  }
}

function renderRunHeader(run, runId) {
  const wrap = el('div', { class: 'statbar' });
  const cells = [];

  const statusBadge = (() => {
    if (run.state === 'running') return el('span', { class: 'pill pill-running' }, el('span', { class: 'led' }), '运行中');
    if (run.state === 'error')   return el('span', { class: 'pill pill-bad' }, el('span', { class: 'led' }), '失败');
    if (run.state === 'cancelled') return el('span', { class: 'pill pill-warn' }, el('span', { class: 'led' }), '已取消');
    return el('span', { class: 'pill pill-good' }, el('span', { class: 'led' }), '完成');
  })();

  cells.push(['STATUS', statusBadge]);
  cells.push(['RUN ID', el('span', { class: 'mono' }, runId.slice(0, 16) + (runId.length > 16 ? '…' : ''))]);
  cells.push(['CHANNEL', run.channelName || '—']);
  cells.push(['TARGET', el('span', { class: 'mono', style: { fontSize: '11px' } }, (run.target || '').replace(/^https?:\/\//, '') || '—')]);
  cells.push(['MODELS', el('span', { class: 'mono' }, (run.models || []).length)]);
  if (run.startedAt) cells.push(['STARTED', el('span', { class: 'mono', style: { fontSize: '11px' } }, fmtTimeAgo(new Date(run.startedAt).toISOString()))]);
  if (run.state !== 'running') {
    let elapsed = 0;
    if (run.finishedAt && run.startedAt) {
      elapsed = run.finishedAt - run.startedAt;
    } else if (run.finalReports && run.finalReports.length) {
      elapsed = Math.max(...run.finalReports.map(r => r.elapsed_ms || 0));
    }
    cells.push(['ELAPSED', el('span', { class: 'mono' }, elapsed ? fmtMs(elapsed) : '—')]);
  } else {
    const now = Date.now();
    cells.push(['ELAPSED', el('span', { class: 'mono', id: 'liveElapsed' }, fmtMs(now - run.startedAt))]);
    // tick
    if (!run._tick) {
      run._tick = setInterval(() => {
        if (run.state !== 'running') { clearInterval(run._tick); run._tick = null; return; }
        const cur = document.getElementById('liveElapsed');
        if (cur) cur.textContent = fmtMs(Date.now() - run.startedAt);
      }, 1000);
    }
  }

  cells.forEach(([k, v]) => {
    wrap.appendChild(el('div', { class: 'cell' },
      el('div', { class: 'k' }, k),
      el('div', { class: 'v' }, v),
    ));
  });

  if (run.state === 'error' && run.error) {
    const err = el('div', { class: 'panel', style: { marginBottom: '12px', borderColor: 'var(--bad-line)' } },
      el('div', { class: 'panel-head', style: { background: 'var(--bad-soft)', borderColor: 'var(--bad-line)' } },
        el('h3', { style: { color: 'var(--bad-ink)' } }, '运行失败'),
      ),
      el('div', { class: 'panel-body', style: { fontFamily: 'var(--font-mono)', fontSize: '12px', color: 'var(--bad-ink)', wordBreak: 'break-all' } }, run.error),
    );
    const frag = document.createDocumentFragment();
    frag.appendChild(wrap);
    frag.appendChild(err);
    return frag;
  }

  return wrap;
}

function renderProgressStrip(run) {
  const hasRealProgress = run.progressTotal > 0;
  const pct = hasRealProgress ? Math.round(run.progressDone / run.progressTotal * 100) : 0;
  const meta = hasRealProgress
    ? `${run.progressDone} / ${run.progressTotal} probes`
    : '同步模式 · 无探针级进度';
  return el('div', { class: 'panel', style: { marginBottom: '12px' } },
    el('div', { class: 'panel-head' },
      el('h3', null, '运行进度'),
      el('span', { class: 'meta' }, meta),
      el('div', { class: 'spacer' }),
      hasRealProgress
        ? el('span', { class: 'mono', style: { color: 'var(--ink-3)', fontSize: '11px' } }, pct + '%')
        : null,
    ),
    el('div', { class: 'panel-body', style: { padding: '12px 14px' } },
      el('div', { class: 'progress' + (hasRealProgress ? '' : ' indeterminate') },
        hasRealProgress
          ? el('div', { class: 'progress-fill', style: { width: pct + '%' } })
          : null
      ),
    ),
  );
}

/* ─── Probe board — live progress for running state ─── */

const CAT_ORDER = ['fingerprint', 'structural', 'signature', 'behavioral', 'multimodal'];

function collectLiveCatScores(pm) {
  const cats = {};
  CAT_ORDER.forEach(c => { cats[c] = { passed: 0, total: 0 }; });
  for (const pid of pm.probeOrder) {
    const probe = pm.probes[pid];
    if (!probe || probe.state !== 'done' || !probe.checks) continue;
    probe.checks.forEach(c => {
      const cat = catOf(c.name);
      if (!cats[cat]) cats[cat] = { passed: 0, total: 0 };
      cats[cat].total++;
      if (c.pass) cats[cat].passed++;
    });
  }
  return cats;
}

function probePrimaryCat(probe) {
  if (probe.checks && probe.checks.length > 0) {
    const counts = {};
    probe.checks.forEach(c => { const cat = catOf(c.name); counts[cat] = (counts[cat] || 0) + 1; });
    return Object.entries(counts).sort((a, b) => b[1] - a[1])[0][0];
  }
  const FALLBACK = {
    precheck:'structural', tag_replay:'fingerprint', mini_probe:'fingerprint',
    identity_probe:'behavioral', self_intro:'structural', tool_use:'structural',
    logic:'behavioral', hidden_prompt:'fingerprint', image_ocr:'multimodal',
    pdf_extract:'multimodal', magic_refusal:'behavioral', effort_thinking:'signature',
    signature_reject:'signature', bash_tool:'structural', minimal_tokens:'fingerprint',
  };
  return FALLBACK[probe.probe_id] || 'other';
}

function renderProbeBoard(run, model, runId) {
  const pm = run.perModel[model] || { status: 'pending', probes: {}, probeOrder: [] };
  const order = pm.probeOrder.slice();
  const doneCount = order.filter(pid => pm.probes[pid] && pm.probes[pid].state === 'done').length;
  const runningCount = order.filter(pid => pm.probes[pid] && pm.probes[pid].state === 'running').length;
  const totalCount = order.length;
  const pct = totalCount > 0 ? Math.round(doneCount / totalCount * 100) : 0;

  const panel = el('div', { class: 'panel' });

  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '探针进度'),
    el('span', { class: 'meta' },
      doneCount + ' / ' + totalCount + ' probes' + (runningCount > 0 ? ' · ' + runningCount + ' 并发' : '')),
    el('div', { class: 'spacer' }),
    el('span', { class: 'mono', style: { color: 'var(--ink-3)', fontSize: '11px' } }, pct + '%'),
    el('span', { class: 'pill pill-running', style: { marginLeft: '8px' } },
      el('span', { class: 'led' }), '运行中'),
  ));

  // progress bar
  const progWrap = el('div', { style: { padding: '10px 14px', borderBottom: '1px solid var(--line)' } });
  progWrap.appendChild(el('div', { class: 'progress' + (totalCount === 0 ? ' indeterminate' : '') },
    totalCount > 0 ? el('div', { class: 'progress-fill', style: { width: pct + '%' } }) : null));
  panel.appendChild(progWrap);

  // live category scores
  if (doneCount > 0) {
    panel.appendChild(renderLiveCatStrip(pm));
  }

  // probe rows
  const body = el('div', { class: 'panel-body', style: { padding: '0' } });
  const board = el('div', { class: 'probe-board' });

  if (order.length === 0) {
    board.appendChild(el('div', { class: 'empty', style: { padding: '24px' } }, '等待探针启动…'));
  } else {
    order.forEach(pid => {
      const probe = pm.probes[pid] || { state: 'pending', probe_id: pid, label: pid, checks: [] };
      board.appendChild(renderProbeBoardRow(probe));
    });
  }

  body.appendChild(board);
  panel.appendChild(body);
  return panel;
}

function renderLiveCatStrip(pm) {
  const cats = collectLiveCatScores(pm);
  const wrap = el('div', { style: { padding: '10px 14px', borderBottom: '1px solid var(--line)', background: 'var(--panel-2)' } });
  const strip = el('div', { class: 'live-cat-strip' });

  CAT_ORDER.forEach(cat => {
    const d = cats[cat];
    const color = CAT_COLOR[cat] || 'var(--ink-4)';
    const pctVal = d.total > 0 ? Math.round(d.passed / d.total * 100) : -1;
    const pctColor = pctVal >= 80 ? 'var(--good-ink)' : pctVal >= 50 ? 'var(--warn-ink)' : pctVal >= 0 ? 'var(--bad-ink)' : 'var(--ink-5)';

    strip.appendChild(el('div', { class: 'live-cat-row' },
      el('span', { class: 'swatch', style: { background: color } }),
      el('span', { class: 'name' }, CAT_LABEL[cat] || cat),
      el('div', { class: 'track' },
        d.total > 0 ? el('div', { class: 'fill', style: { width: pctVal + '%', background: color } }) : null),
      el('span', { class: 'frac' }, d.total > 0 ? d.passed + '/' + d.total : '—'),
      el('span', { class: 'pct', style: { color: pctColor } }, pctVal >= 0 ? pctVal + '%' : ''),
    ));
  });

  wrap.appendChild(strip);
  return wrap;
}

function renderProbeBoardRow(probe) {
  const checks = probe.checks || [];
  const passed = checks.filter(c => c.pass).length;
  const failed = checks.filter(c => !c.pass).length;
  const total = checks.length;
  const isRunning = probe.state === 'running';
  const isDone = probe.state === 'done';
  const isPending = probe.state === 'pending';
  const allPass = total > 0 && passed === total;

  const dotClass = isRunning ? 'run' : isDone ? (allPass ? 'pass' : 'fail') : 'pending';
  const cat = probePrimaryCat(probe);
  const catBorder = CAT_COLOR[cat] || 'transparent';

  const row = el('div', {
    class: 'pb-row' + (isPending ? ' pending' : '') + (isRunning ? ' running' : ''),
    style: { borderLeftColor: catBorder },
  });

  row.appendChild(el('span', { class: 'led-dot ' + dotClass }));
  row.appendChild(el('div', { class: 'pb-name' },
    el('span', { class: 'label' }, probe.label || probe.probe_id),
    el('span', { class: 'id' }, probe.probe_id)));

  if (isDone) {
    row.appendChild(el('span', { class: 'pb-checks ' + (allPass ? 'allpass' : 'hasfail') },
      passed + '/' + total));
    row.appendChild(el('span', { class: 'pb-latency' }, fmtMs(probe.latency_ms)));
  } else if (isRunning) {
    row.appendChild(el('span', { class: 'pb-checks running' }, '…'));
    row.appendChild(el('span', { class: 'pb-latency' }));
  } else {
    row.appendChild(el('span', { class: 'pb-checks' }));
    row.appendChild(el('span', { class: 'pb-latency' }));
  }

  if (isDone && failed > 0) {
    const failStrip = el('div', { class: 'pb-fails' });
    checks.filter(c => !c.pass).slice(0, 3).forEach(c => {
      const item = el('div', { class: 'pb-fail-item' },
        el('span', { class: 'led-dot fail', style: { width: '5px', height: '5px' } }),
        el('span', { class: 'name' }, c.label || c.name));
      if (c.expected || c.actual) {
        const exp = el('span', { class: 'expect-pair' });
        if (c.expected) exp.appendChild(el('span', { class: 'exp' }, c.expected));
        if (c.actual) exp.appendChild(el('span', { class: 'act' }, c.actual));
        item.appendChild(exp);
      } else if (c.detail) {
        item.appendChild(el('span', { class: 'actual' }, c.detail.slice(0, 80)));
      }
      failStrip.appendChild(item);
    });
    if (failed > 3) {
      failStrip.appendChild(el('div', { class: 'pb-fail-more' }, '+' + (failed - 3) + ' more'));
    }
    row.appendChild(failStrip);
  }

  return row;
}

function renderModelGrid(run, models, focusedModel, runId) {
  const wrap = el('div', { class: 'panel', style: { marginBottom: '12px' } });
  wrap.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '模型对比'),
    el('span', { class: 'meta' }, models.length + ' 个模型 · 点击进入详细 probe'),
  ));
  const grid = el('div', { class: 'panel-body', style: { padding: '12px' } });
  const inner = el('div', { class: 'model-grid', style: { margin: '0' } });
  models.forEach(m => {
    const pm = run.perModel[m] || { status: 'pending', probes: {}, probeOrder: [] };
    inner.appendChild(renderModelCard(m, pm, run, m === focusedModel, runId));
  });
  grid.appendChild(inner);
  wrap.appendChild(grid);
  return wrap;
}

function renderModelCard(model, pm, run, isActive, runId) {
  const card = el('div', { class: 'model-card' + (isActive ? ' active' : ''),
    onclick: () => location.hash = '#/channel/run/' + runId + '/m/' + encodeURIComponent(model),
  });

  if (pm.status !== 'done') {
    const runningModel = Object.entries(run.perModel).find(([, p]) => p.status === 'running');
    const hint = pm.status === 'running' ? '运行中'
      : runningModel ? '排队中 · 当前 ' + runningModel[0].split('-').pop()
      : '等待';
    card.appendChild(el('div', { class: 'running-indicator' }, hint));
  }
  const rep = pm.report;
  const sum = rep ? reportSummary(rep) : null;
  const grade = sum ? sum.grade : '·';
  const gc = sum && sum.gradeColor ? gradeColor(sum.gradeColor) : 'var(--ink-4)';

  card.appendChild(el('div', { class: 'top' },
    el('span', { class: 'led-dot ' + (pm.status === 'running' ? 'run' : pm.status === 'done' ? (sum && sum.failed === 0 ? 'pass' : 'warn') : 'pending') }),
    el('div', { class: 'name' }, model),
    el('div', { class: 'grade', style: { color: gc } }, grade),
  ));

  if (sum) {
    card.appendChild(el('div', { class: 'score-line' },
      el('span', { class: 'num', style: { color: gc } }, sum.score != null ? Math.round(sum.score) : '–'),
      el('span', { class: 'denom' }, '/ 100'),
      el('span', { class: 'spacer' }),
      el('span', { class: 'pill ' + (VERDICT_PILL[sum.verdictColor] || 'pill-info'), style: { fontSize: '10px' } },
        el('span', { class: 'led' }), sum.verdictLabel
      ),
    ));
    const cats = el('div', { class: 'cats' });
    sum.categories.forEach(c => {
      const cat = (c.key || '').toLowerCase();
      const color = CAT_COLOR[cat] || 'var(--ink-3)';
      cats.appendChild(el('div', { class: 'cat-bar' },
        el('span', { class: 'label' }, (c.label || cat).slice(0, 8)),
        el('div', { class: 'track' },
          el('div', { class: 'fill', style: { width: c.percentage + '%', background: color } })),
        el('span', { class: 'pct' }, Math.round(c.percentage) + '%'),
      ));
    });
    card.appendChild(cats);
  } else {
    const doneCount = pm.probeOrder.filter(pid => pm.probes[pid] && pm.probes[pid].state === 'done').length;
    const startedCount = pm.probeOrder.length;
    card.appendChild(el('div', { class: 'score-line' },
      el('span', { class: 'num', style: { color: 'var(--ink-4)', fontSize: '14px' } },
        startedCount > 0 ? doneCount + '/' + startedCount : '—'),
      el('span', { class: 'denom' }, startedCount > 0 ? ' probes' : ''),
    ));
    if (startedCount > 0) {
      const pct = Math.round(doneCount / startedCount * 100);
      card.appendChild(el('div', { style: { padding: '4px 10px 8px' } },
        el('div', { class: 'progress', style: { height: '3px' } },
          el('div', { class: 'progress-fill', style: { width: pct + '%' } }))));
    } else {
      card.appendChild(el('div', { style: { padding: '8px 10px', fontSize: '11px', color: 'var(--ink-4)' } },
        '等待探针启动…'));
    }
  }

  return card;
}

function renderReportBanner(run) {
  const elapsed = run.finishedAt && run.startedAt ? fmtMs(run.finishedAt - run.startedAt)
    : run.finalReports && run.finalReports.length ? fmtMs(Math.max(...run.finalReports.map(r => r.elapsed_ms || 0)))
    : '—';
  const ts = run.startedAt ? fmtTime(new Date(run.startedAt).toISOString()) : '';
  return el('div', { class: 'report-banner' },
    el('div', { class: 'report-banner-left' },
      el('span', { class: 'report-badge' }, '检测报告'),
      el('span', { class: 'report-meta' }, (run.models || []).length + ' 模型 · ' + elapsed),
    ),
    ts ? el('span', { class: 'report-ts' }, ts) : null,
  );
}

function renderAPIErrorBanner(errors) {
  const cats = {};
  errors.forEach(e => { cats[e.category] = (cats[e.category] || 0) + 1; });
  const catLabels = { auth: '认证错误', forbidden: '权限不足', rate_limit: '频率限制', overloaded: '服务过载', client: '请求错误', server: '服务端错误', unknown: '未知错误' };

  const panel = el('div', { class: 'panel', style: { marginBottom: '12px', borderColor: 'var(--bad-line)' } });
  panel.appendChild(el('div', { class: 'panel-head', style: { background: 'var(--bad-soft)', borderColor: 'var(--bad-line)' } },
    el('h3', { style: { color: 'var(--bad-ink)' } }, 'API 错误'),
    el('span', { class: 'meta', style: { color: 'var(--bad-ink)' } },
      errors.length + ' 个探针遇到错误 · 后续检查已跳过'),
  ));

  const body = el('div', { class: 'panel-body', style: { padding: '0' } });
  errors.forEach(e => {
    const statusClass = e.status_code >= 500 || e.status_code === 529 ? 'err-server' : 'err-client';
    body.appendChild(el('div', { class: 'api-error-row', style: { display: 'flex', alignItems: 'center', gap: '10px', padding: '8px 14px', borderBottom: '1px solid var(--line-soft)' } },
      el('span', { class: 'mono', style: { color: 'var(--bad-ink)', fontWeight: 700, minWidth: '42px' } }, String(e.status_code)),
      el('span', { class: 'pill pill-bad', style: { fontSize: '10px' } }, catLabels[e.category] || e.category),
      el('span', { class: 'mono', style: { fontSize: '11px', color: 'var(--ink-3)' } }, e.probe_id),
      el('span', { style: { flex: 1, fontSize: '11px', color: 'var(--ink-2)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }, e.message || e.error_type || ''),
    ));
  });
  panel.appendChild(body);
  return panel;
}

function renderChannelIdentBanner(hits) {
  const channels = {};
  hits.forEach(h => { channels[h.channel] = (channels[h.channel] || []).concat(h); });

  const panel = el('div', { class: 'panel', style: { marginBottom: '12px', borderColor: 'var(--warn-line)' } });
  panel.appendChild(el('div', { class: 'panel-head', style: { background: 'var(--warn-soft)', borderColor: 'var(--warn-line)' } },
    el('h3', { style: { color: 'var(--warn-ink)' } }, '渠道溯源'),
    el('span', { class: 'meta', style: { color: 'var(--warn-ink)' } },
      '检测到 ' + Object.keys(channels).length + ' 个渠道特征 · ' + hits.length + ' 处匹配'),
  ));

  const body = el('div', { class: 'panel-body', style: { padding: '0' } });
  hits.forEach(h => {
    body.appendChild(el('div', { style: { display: 'flex', alignItems: 'center', gap: '10px', padding: '8px 14px', borderBottom: '1px solid var(--line-soft)' } },
      el('span', { class: 'pill pill-warn', style: { fontSize: '10px', flexShrink: 0 } }, h.channel),
      el('span', { class: 'mono', style: { fontSize: '11px', color: 'var(--ink-3)', flexShrink: 0 } }, h.source),
      el('span', { class: 'mono', style: { fontSize: '10px', color: 'var(--ink-4)', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }, h.context || h.keyword),
    ));
  });
  panel.appendChild(body);
  return panel;
}

function renderChannelBaselineComparison(comp) {
  const relScore = Math.round(comp.relative_score * 10) / 10;
  const delta = Math.round(comp.score_delta * 100) / 100;
  const deltaSign = delta >= 0 ? '+' : '';
  const color = relScore >= 90 ? 'var(--good)' : relScore >= 70 ? 'var(--warn)' : 'var(--bad)';
  const inkColor = relScore >= 90 ? 'var(--good-ink)' : relScore >= 70 ? 'var(--warn-ink)' : 'var(--bad-ink)';

  const panel = el('div', { class: 'panel', style: { marginBottom: '12px', borderColor: color } });
  panel.appendChild(el('div', { class: 'panel-head', style: { borderColor: color } },
    el('h3', null, '基线对比'),
    el('span', { class: 'meta' }, '与 ' + (comp.baseline_name || comp.baseline_id) + (comp.baseline_profile ? ' [' + comp.baseline_profile + ']' : '') + ' 对比'),
    el('div', { class: 'spacer' }),
    el('span', { class: 'mono', style: { fontSize: '18px', fontWeight: 700, color: inkColor } }, relScore + '%'),
    el('span', { class: 'mono', style: { fontSize: '11px', color: 'var(--ink-3)', marginLeft: '6px' } }, '相对基线'),
  ));

  const body = el('div', { class: 'panel-body' });
  const summary = el('div', { style: { display: 'flex', gap: '24px', padding: '8px 0', flexWrap: 'wrap' } });
  summary.appendChild(el('div', null,
    el('div', { class: 'meta', style: { fontSize: '10px' } }, 'BASELINE'),
    el('div', { class: 'mono', style: { fontSize: '14px' } }, Math.round(comp.baseline_score * 100) / 100),
  ));
  summary.appendChild(el('div', null,
    el('div', { class: 'meta', style: { fontSize: '10px' } }, 'CURRENT'),
    el('div', { class: 'mono', style: { fontSize: '14px' } }, Math.round(comp.current_score * 100) / 100),
  ));
  summary.appendChild(el('div', null,
    el('div', { class: 'meta', style: { fontSize: '10px' } }, 'DELTA'),
    el('div', { class: 'mono', style: { fontSize: '14px', color: inkColor } }, deltaSign + delta),
  ));
  summary.appendChild(el('div', null,
    el('div', { class: 'meta', style: { fontSize: '10px' } }, 'CHECKS'),
    el('div', { class: 'mono', style: { fontSize: '14px' } }, comp.overlapping_checks + ' 重叠 / ' + comp.deviated_checks + ' 偏离'),
  ));
  body.appendChild(summary);

  if (comp.check_comparisons && comp.check_comparisons.length > 0) {
    const deviated = comp.check_comparisons.filter(c => c.deviated);
    if (deviated.length > 0) {
      const list = el('div', { style: { borderTop: '1px solid var(--line)', paddingTop: '8px', marginTop: '8px' } });
      list.appendChild(el('div', { class: 'meta', style: { marginBottom: '6px', fontSize: '10px', color: 'var(--warn-ink)' } },
        deviated.length + ' 个检查项与基线不一致'));
      deviated.slice(0, 12).forEach(c => {
        const baseIcon = c.baseline_pass ? '✓' : '✗';
        const curIcon = c.current_pass ? '✓' : '✗';
        const curColor = c.current_pass ? 'var(--good-ink)' : 'var(--bad-ink)';
        list.appendChild(el('div', { style: { display: 'flex', gap: '8px', padding: '3px 0', fontSize: '11px', alignItems: 'center' } },
          el('span', { class: 'mono', style: { width: '180px', overflow: 'hidden', textOverflow: 'ellipsis' } }, c.label || c.name),
          el('span', { class: 'mono', style: { color: 'var(--ink-3)' } }, baseIcon),
          el('span', null, '→'),
          el('span', { class: 'mono', style: { color: curColor, fontWeight: 600 } }, curIcon),
          el('span', { class: 'mono', style: { fontSize: '10px', color: 'var(--ink-4)' } }, c.category || ''),
        ));
      });
      if (deviated.length > 12) {
        list.appendChild(el('div', { class: 'meta', style: { fontSize: '10px', textAlign: 'center', padding: '4px' } },
          '+ ' + (deviated.length - 12) + ' more'));
      }
      body.appendChild(list);
    } else {
      body.appendChild(el('div', { style: { color: 'var(--good-ink)', fontSize: '11px', padding: '8px 0', borderTop: '1px solid var(--line)' } },
        '✓ 所有重叠检查项与基线一致'));
    }
  }

  panel.appendChild(body);
  return panel;
}

function renderModelDetail(run, model, runId, anchor) {
  const wrap = document.createDocumentFragment();
  const pm = run.perModel[model] || { status: 'pending', probes: {}, probeOrder: [] };
  const rep = pm.report;
  const sum = rep ? reportSummary(rep) : null;

  // ─── API error banner (prominent, above everything) ───
  if (rep && rep.errors && rep.errors.length > 0) {
    wrap.appendChild(renderAPIErrorBanner(rep.errors));
  }

  // ─── Channel identification banner ───
  if (rep && rep.identified_channels && rep.identified_channels.length > 0) {
    wrap.appendChild(renderChannelIdentBanner(rep.identified_channels));
  }

  // ─── Channel baseline comparison ───
  if (rep && rep.baseline_comparison) {
    wrap.appendChild(renderChannelBaselineComparison(rep.baseline_comparison));
  }

  // ─── verdict-hero (first thing user sees) ───
  if (sum) {
    wrap.appendChild(renderVerdictHero(sum, model, runId));
  }

  // ─── Score + categories (compact, below the verdict) ───
  if (sum) {
    wrap.appendChild(renderScoreCategories(sum, rep));
  }

  // ─── Probe details ───
  wrap.appendChild(renderProbeListPanel(pm, run, runId, model, anchor));

  // ─── Raw JSON ───
  if (rep) {
    const raw = el('details', { style: { marginTop: '14px' } },
      el('summary', { class: 'btn btn-quiet btn-sm', style: { display: 'inline-flex', cursor: 'pointer' } }, '原始 JSON'),
      el('pre', { class: 'json-out', style: { marginTop: '8px' } }, JSON.stringify(rep, null, 2)),
    );
    wrap.appendChild(raw);
  }

  return wrap;
}

function renderVerdictHero(sum, model, runId) {
  const gc = gradeColor(sum.gradeColor);
  const titleMain = sum.score >= 80 ? '大概率<em>官方直连</em>'
                  : sum.score >= 50 ? '存在<em>明显偏差</em>'
                                    : '疑似<em>非官方渠道</em>';
  const titleSub = sum.failed > 0 ? `,有 ${sum.failed} 项偏差` : '';
  const title = el('div', { class: 'title' });
  title.innerHTML = titleMain + esc(titleSub);

  // failure list
  const failBody = el('div', { class: 'fail-list' });
  if (sum.failures.length === 0) {
    failBody.appendChild(el('div', { class: 'empty' }, '✓ 全部 ' + sum.total + ' 项检查通过'));
  } else {
    sum.failures.slice(0, 6).forEach(c => {
      const cat = catOf(c.name);
      const node = el('div', { class: 'fail-item', onclick: () => {
        location.hash = '#/channel/run/' + runId + '/m/' + encodeURIComponent(model) + '/check/' + encodeURIComponent(c.name);
      }});
      node.appendChild(el('span', { class: 'led-dot fail' }));
      const body = el('div', { class: 'body', style: { minWidth: '0' } });
      const name = el('div', { class: 'name' });
      if (c.label) name.appendChild(el('span', { class: 'label' }, c.label));
      name.appendChild(el('span', null, c.name));
      body.appendChild(name);
      body.appendChild(el('div', { class: 'detail' }, c.detail || ''));
      node.appendChild(body);
      node.appendChild(el('span', { class: 'cat-tag', style: { color: CAT_COLOR[cat] } }, cat));
      failBody.appendChild(node);
    });
    if (sum.failures.length > 6) {
      failBody.appendChild(el('div', { class: 'muted', style: { fontSize: '11px', textAlign: 'center', padding: '6px' } },
        '还有 ' + (sum.failures.length - 6) + ' 项 · 见下方完整列表'));
    }
  }

  return el('div', { class: 'verdict-hero' },
    el('div', { class: 'verdict-side' },
      el('div', { class: 'label' }, 'VERDICT · ' + model),
      el('div', { class: 'head' },
        el('span', { class: 'grade', style: { color: gc } }, sum.grade),
        el('span', { class: 'score' }, (sum.score != null ? Math.round(sum.score) : '–') + ' / 100 pts'),
      ),
      el('span', { class: 'pill ' + (VERDICT_PILL[sum.verdictColor] || 'pill-info'), style: { alignSelf: 'flex-start' } },
        el('span', { class: 'led' }), sum.verdictLabel),
      title,
      el('div', { class: 'desc' }, sum.passed + ' / ' + sum.total + ' 通过 · ' + (sum.elapsedMs ? fmtMs(sum.elapsedMs) : '—')),
    ),
    el('div', { class: 'fail-side' },
      el('div', { class: 'head' },
        el('span', null, '失败检查项'),
        el('span', { class: 'spacer' }),
        el('span', { class: 'count' }, sum.failed),
      ),
      failBody,
    ),
  );
}

function renderScoreCategories(sum, rep) {
  const panel = el('div', { class: 'panel', style: { marginBottom: '12px' } });
  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '类别得分'),
    el('span', { class: 'meta' }, '五维探测 · 按权重展示'),
  ));
  const body = el('div', { class: 'panel-body' });
  const strip = el('div', { class: 'cat-strip' });
  sum.categories.forEach(c => {
    const cat = (c.key || '').toLowerCase();
    const color = CAT_COLOR[cat] || 'var(--ink-3)';
    const pctCol = c.percentage >= 80 ? 'var(--good)' : c.percentage >= 50 ? 'var(--warn)' : 'var(--bad)';
    strip.appendChild(el('div', { class: 'cat' },
      el('span', { class: 'swatch', style: { background: color } }),
      el('span', { class: 'name' }, c.label || cat),
      el('div', { class: 'track' },
        el('div', { class: 'fill', style: { width: c.percentage + '%', background: color } })),
      el('span', { class: 'frac' }, (c.passed || 0) + '/' + (c.total || 0)),
      el('span', { class: 'pct', style: { color: pctCol } }, Math.round(c.percentage) + '%'),
    ));
  });
  body.appendChild(strip);
  panel.appendChild(body);
  return panel;
}

let _checkFilter = 'all';
function renderProbeListPanel(pm, run, runId, model, anchor) {
  const panel = el('div', { class: 'panel' });
  const seg = el('div', { class: 'seg' });
  ['all', 'fail', 'pass'].forEach(k => {
    const b = el('button', { class: _checkFilter === k ? 'active' : '',
      onclick: () => { _checkFilter = k; renderRunPage(runId, model, anchor); } },
      ({ all: '全部', fail: '仅失败', pass: '仅通过' })[k]);
    seg.appendChild(b);
  });

  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '探针 / probes'),
    el('span', { class: 'meta' }, (pm.probeOrder || []).length + ' probes · ' + (pm.report ? '完成' : pm.status)),
    el('div', { class: 'spacer' }),
    seg,
  ));

  const body = el('div', { class: 'panel-body', style: { padding: '0' } });
  const list = el('div', { class: 'probes' });

  const order = pm.probeOrder.slice();
  if (order.length === 0) {
    list.appendChild(el('div', { class: 'empty' }, run.state === 'running' ? '等待 probe 结果…' : '无 probe 数据'));
  } else {
    order.forEach(pid => list.appendChild(renderProbeRow(pm.probes[pid], anchor, { run, runId, model })));
  }
  body.appendChild(list);
  panel.appendChild(body);

  // scroll to anchor
  if (anchor) {
    setTimeout(() => {
      const target = document.querySelector('[data-check="' + CSS.escape(anchor) + '"]');
      if (target) {
        // open its parent
        let parent = target.closest('.probe-row');
        if (parent) parent.classList.add('open');
        target.classList.add('highlight');
        target.scrollIntoView({ block: 'center', behavior: 'smooth' });
        setTimeout(() => target.classList.remove('highlight'), 2200);
      }
    }, 120);
  }

  return panel;
}

function renderProbeRow(probe, anchor, ctx) {
  const checks = probe.checks || [];
  const passed = checks.filter(c => c.pass).length;
  const total = checks.length;
  const isRunning = probe.state === 'running';
  const isRetrying = probe._retrying;
  const allPass = total > 0 && passed === total;
  const hasFail = passed < total;

  const dot = el('span', { class: 'led-dot ' + (isRunning || isRetrying ? 'run' : total === 0 ? 'pending' : allPass ? 'pass' : 'fail') });

  const head = el('div', { class: 'probe-head' },
    dot,
    el('div', { class: 'name' }, probe.label || probe.probe_id,
      el('span', { class: 'id' }, probe.probe_id)),
    el('div', { class: 'checks-count ' + (total === 0 ? '' : allPass ? 'allpass' : 'hasfail') },
      isRunning || isRetrying ? '…' : (passed + '/' + total)),
    el('div', { class: 'latency' }, probe.latency_ms != null ? fmtMs(probe.latency_ms) : (isRunning ? '·' : '—')),
    el('div'),
    el('span', { class: 'caret' }, '›'),
  );
  head.onclick = () => row.classList.toggle('open');

  const body = el('div', { class: 'probe-body' });

  // checks — each check gets its own retry + exchange viewer
  checks.forEach(c => {
    if (_checkFilter === 'pass' && !c.pass) return;
    if (_checkFilter === 'fail' && c.pass) return;
    body.appendChild(renderCheckRow(c, probe, ctx));
  });

  const row = el('div', { class: 'probe-row' + ((anchor && checks.some(c => c.name === anchor)) || hasFail ? ' open' : ''),
    'data-probe': probe.probe_id });
  row.appendChild(head); row.appendChild(body);
  return row;
}

/* ─── Exchange rendering ─── */
function fmtJSON(raw) {
  if (raw == null) return null;
  if (typeof raw === 'object') return JSON.stringify(raw, null, 2);
  if (typeof raw === 'string') {
    try { return JSON.stringify(JSON.parse(raw), null, 2); } catch { return raw; }
  }
  return String(raw);
}

function renderExchangePanel(container, exchanges, probe) {
  container.innerHTML = '';
  const header = el('div', { class: 'ex-panel-head' },
    el('span', { class: 'ex-panel-title' },
      mkIcon('code', { size: 12 }),
      ' 请求 / 响应',
      el('span', { class: 'ex-panel-meta' }, probe ? probe.label || probe.probe_id : ''),
    ),
    el('span', { class: 'ex-panel-count' }, exchanges.length + ' 次调用'),
  );
  container.appendChild(header);

  exchanges.forEach((ex, i) => {
    const reqJSON = fmtJSON(ex.request);
    const respJSON = fmtJSON(ex.response);
    const status = ex.status || 200;
    const statusOk = status >= 200 && status < 300;

    const card = el('div', { class: 'ex-card' });

    // card header
    const cardHead = el('div', { class: 'ex-card-head' },
      el('span', { class: 'ex-num' }, '#' + (i + 1)),
      el('span', { class: 'ex-method' }, 'POST /v1/messages'),
      el('span', { class: 'ex-status ' + (statusOk ? 'ok' : 'err') }, status),
    );
    card.appendChild(cardHead);

    // request section
    if (reqJSON) {
      const reqSection = el('div', { class: 'ex-section' });
      const reqHead = el('div', { class: 'ex-section-head' },
        el('span', { class: 'ex-section-label' }, 'REQUEST BODY'),
        el('button', { class: 'btn btn-ghost btn-xs', title: '复制',
          onclick: ev => { ev.stopPropagation(); copyText(reqJSON); toast('已复制', 'good'); },
        }, mkIcon('copy', { size: 10 })),
      );
      reqSection.appendChild(reqHead);
      reqSection.appendChild(el('pre', { class: 'ex-json' }, reqJSON));
      card.appendChild(reqSection);
    }

    // response section
    const respSection = el('div', { class: 'ex-section' });
    const respHead = el('div', { class: 'ex-section-head' },
      el('span', { class: 'ex-section-label' }, 'RESPONSE BODY'),
    );
    if (respJSON) {
      respHead.appendChild(el('button', { class: 'btn btn-ghost btn-xs', title: '复制',
        onclick: ev => { ev.stopPropagation(); copyText(respJSON); toast('已复制', 'good'); },
      }, mkIcon('copy', { size: 10 })));
      respSection.appendChild(respHead);
      respSection.appendChild(el('pre', { class: 'ex-json' }, respJSON));
    } else {
      respSection.appendChild(respHead);
      respSection.appendChild(el('div', { class: 'ex-none' }, status >= 200 && status < 300 ? 'streaming — 响应体已合并至检查结果' : '未捕获'));
    }
    card.appendChild(respSection);

    container.appendChild(card);
  });
}

async function loadProbeExchanges(probe, ctx) {
  if (probe._exchangesLoaded || probe._exchangesLoading) return probe.exchanges || [];
  if (probe.exchanges && probe.exchanges.length) { probe._exchangesLoaded = true; return probe.exchanges; }
  if (!ctx || !ctx.run || !ctx.run.historical) return probe.exchanges || [];

  probe._exchangesLoading = true;
  try {
    const reportId = ctx.run.finalReports && ctx.run.finalReports.length
      ? (ctx.run.finalReports.find(r => r.model === ctx.model) || {}).id || ctx.runId
      : ctx.runId;
    const data = await api('/api/channel/history/' + encodeURIComponent(reportId) + '/probe/' + encodeURIComponent(probe.probe_id));
    probe.exchanges = data.exchanges || [];
    probe._exchangesLoaded = true;
    return probe.exchanges;
  } catch (e) {
    throw e;
  } finally {
    probe._exchangesLoading = false;
  }
}

/* ─── Check-level retry ─── */
async function retryProbe(runId, model, probeId, run) {
  const targetKey = (run && run.payload && run.payload.target_key) || State.channel.targetKey;
  if (!targetKey) {
    toast('请先在 Channel 配置页填写 API Key', 'warn');
    return;
  }

  const reportId = run.finalReports && run.finalReports.length
    ? (run.finalReports.find(r => r.model === model) || {}).id || runId
    : runId;

  const pm = run.perModel[model];
  if (!pm || !pm.probes[probeId]) return;
  pm.probes[probeId]._retrying = true;
  const route = parseRoute(location.hash);
  renderRunPage(runId, model, route.anchor);

  try {
    const data = await api('/api/channel/history/' + encodeURIComponent(reportId) + '/probe/' + encodeURIComponent(probeId) + '/retry', {
      method: 'POST',
      body: JSON.stringify({ target_key: targetKey }),
    });

    if (data.probe) {
      pm.probes[probeId] = Object.assign({}, data.probe, { state: 'done', _exchangesLoaded: true });
    }
    if (pm.report && data.score) pm.report.score = data.score;
    if (pm.report && data.checks) pm.report.checks = data.checks;
    toast(probeId + ' 重试完成', 'good');
  } catch (e) {
    toast('重试失败: ' + e.message, 'bad');
  } finally {
    pm.probes[probeId]._retrying = false;
    const route2 = parseRoute(location.hash);
    renderRunPage(runId, model, route2.anchor);
  }
}

function renderCheckRow(c, probe, ctx) {
  const body = el('div', { class: 'body' });
  const name = el('div', { class: 'name' });
  if (c.label) name.appendChild(el('span', { class: 'label' }, c.label));
  name.appendChild(el('span', null, c.name));
  body.appendChild(name);
  if (c.detail) body.appendChild(el('div', { class: 'detail' }, c.detail));
  if (c.expected || c.actual) {
    const exp = el('div', { class: 'expect' });
    if (c.expected) { exp.appendChild(el('span', { class: 'k' }, '期望')); exp.appendChild(el('span', { class: 'v exp mono' }, c.expected)); }
    if (c.actual)   { exp.appendChild(el('span', { class: 'k' }, '实际')); exp.appendChild(el('span', { class: 'v act mono' }, c.actual)); }
    body.appendChild(exp);
  }
  if (!c.pass && c.fix) body.appendChild(el('div', { class: 'fix' }, 'fix · ' + c.fix));

  const actions = el('div', { class: 'actions' });

  // retry this check (re-runs its parent probe)
  if (ctx && probe.state === 'done' && !probe._retrying) {
    actions.appendChild(el('button', { class: 'btn btn-ghost btn-xs',
      title: '重试 (重跑 ' + probe.probe_id + ')',
      onclick: ev => { ev.stopPropagation(); retryProbe(ctx.runId, ctx.model, probe.probe_id, ctx.run); },
    }, mkIcon('refresh', { size: 11 })));
  }

  // view exchange (request/response) for this check's parent probe
  actions.appendChild(el('button', { class: 'btn btn-ghost btn-xs',
    title: '查看请求/响应',
    onclick: ev => { ev.stopPropagation(); toggleCheckExchange(item, probe, ctx); },
  }, mkIcon('code', { size: 11 })));

  actions.appendChild(el('button', { class: 'btn btn-ghost btn-xs',
    title: '深链',
    onclick: ev => {
      ev.stopPropagation();
      const url = location.origin + location.pathname + location.hash.split('/check/')[0] + '/check/' + encodeURIComponent(c.name);
      copyText(url);
    },
  }, mkIcon('link', { size: 11 })));
  actions.appendChild(el('button', { class: 'btn btn-ghost btn-xs',
    title: '复制 Markdown',
    onclick: ev => { ev.stopPropagation(); copyCheckMd(c, probe); },
  }, mkIcon('copy', { size: 11 })));

  // expandable exchange panel (hidden by default)
  const exPanel = el('div', { class: 'check-exchange' });

  const item = el('div', { class: 'check-item', 'data-check': c.name },
    el('div', { class: 'check-row' },
      el('span', { class: 'led-dot ' + (c.pass ? 'pass' : 'fail') }),
      body,
      actions,
    ),
    exPanel,
  );
  return item;
}

async function toggleCheckExchange(item, probe, ctx) {
  const panel = item.querySelector('.check-exchange');
  if (!panel) return;

  if (panel.classList.contains('open')) {
    panel.classList.remove('open');
    return;
  }
  panel.classList.add('open');
  if (panel.dataset.loaded) return;

  panel.innerHTML = '';
  panel.appendChild(el('div', { class: 'ex-loading' }, '加载请求/响应…'));

  try {
    let exchanges = probe.exchanges;
    if (!exchanges || !exchanges.length) {
      exchanges = await loadProbeExchanges(probe, ctx);
    }
    panel.innerHTML = '';
    if (exchanges && exchanges.length) {
      renderExchangePanel(panel, exchanges, probe);
    } else {
      panel.appendChild(el('div', { class: 'ex-empty' }, '无请求/响应数据'));
    }
    panel.dataset.loaded = '1';
  } catch (e) {
    panel.innerHTML = '';
    panel.appendChild(el('div', { class: 'ex-empty' }, '加载失败: ' + e.message));
  }
}

/* ─── copy helpers ─── */
function copyCheckMd(c, probe) {
  const md = `### \`${c.name}\` · ${c.pass ? 'PASS' : 'FAIL'}\n\n` +
    (c.label ? `**${c.label}**  \n` : '') +
    (probe && probe.label ? `Probe: ${probe.label} (\`${probe.probe_id}\`)\n` : '') +
    (c.detail ? `\n${c.detail}\n` : '') +
    (c.expected ? `\n- expected: \`${c.expected}\`\n` : '') +
    (c.actual ? `- actual: \`${c.actual}\`\n` : '') +
    (c.fix ? `\n_fix_: ${c.fix}\n` : '');
  copyText(md);
}

function copyFailuresMd(runId, model) {
  const run = State.liveRuns[runId]; if (!run) return;
  const pm = run.perModel[model]; if (!pm || !pm.report) return;
  const failures = (pm.report.checks || []).filter(c => !c.pass);
  if (failures.length === 0) { toast('没有失败项', 'good'); return; }
  const title = `# Channel probe failures — ${pm.report.channel_name || pm.report.target || ''} (${model})\n\n`;
  const body = failures.map(c =>
    `### \`${c.name}\` · FAIL` +
    (c.label ? ` — ${c.label}` : '') + `\n` +
    (c.detail ? `\n${c.detail}\n` : '') +
    (c.expected ? `\n- expected: \`${c.expected}\`` : '') +
    (c.actual ? `\n- actual: \`${c.actual}\`` : '') +
    (c.fix ? `\n- fix: ${c.fix}` : '')
  ).join('\n\n');
  copyText(title + body);
}

function exportRunJson(runId) {
  const run = State.liveRuns[runId]; if (!run) return;
  const data = run.finalReports || Object.values(run.perModel).map(p => p.report).filter(Boolean);
  downloadJSON({ run_id: runId, reports: data }, 'channel-' + runId + '.json');
}

function rerunChannel(runId) {
  const run = State.liveRuns[runId];
  if (!run || !run.payload) {
    // history view: read from final reports to reconstruct payload
    const rep0 = (run && run.finalReports && run.finalReports[0]) || null;
    if (!rep0) { toast('无法重新运行 (缺少配置)', 'bad'); return; }
    State.channel.targetBase = rep0.target || State.channel.targetBase;
    State.channel.channelName = rep0.channel_name || '';
    State.channel.models = run.finalReports.map(r => r.model);
    toast('已填回配置,请确认 API Key', 'good');
    location.hash = '#/channel';
    return;
  }
  State.channel.targetBase = run.payload.target_base;
  State.channel.targetKey = run.payload.target_key;
  State.channel.channelName = run.payload.channel_name;
  State.channel.models = run.payload.models || [run.payload.model];
  State.channel.concurrency = run.payload.concurrency || 0;
  kickoffChannelRun();
}

/* ─── Channel: retry all failed probes ─── */
function countFailedProbes(run, model) {
  const pm = run.perModel[model];
  if (!pm || !pm.report) return 0;
  let count = 0;
  for (const pid of pm.probeOrder) {
    const probe = pm.probes[pid];
    if (!probe || !probe.checks) continue;
    if (probe.checks.some(c => !c.pass)) count++;
  }
  return count;
}

async function retryAllFailedProbes(runId, model) {
  const run = State.liveRuns[runId];
  if (!run) return;
  const pm = run.perModel[model];
  if (!pm || !pm.report) return;

  const reportId = pm.report.id || runId;
  const targetKey = run.payload ? run.payload.target_key : State.channel.targetKey;
  if (!targetKey) { toast('需要 API Key 才能重试', 'bad'); return; }

  const failedPids = [];
  for (const pid of pm.probeOrder) {
    const probe = pm.probes[pid];
    if (probe && probe.checks && probe.checks.some(c => !c.pass)) failedPids.push(pid);
  }
  if (failedPids.length === 0) { toast('无失败探针', 'good'); return; }

  toast('正在重试 ' + failedPids.length + ' 个失败探针…');

  for (const pid of failedPids) {
    if (pm.probes[pid]) pm.probes[pid]._retrying = true;
  }
  maybeRerender(runId);

  for (const pid of failedPids) {
    try {
      const data = await api('/api/channel/history/' + encodeURIComponent(reportId) + '/probe/' + encodeURIComponent(pid) + '/retry', {
        method: 'POST',
        body: JSON.stringify({ target_key: targetKey }),
      });
      if (data.probe) {
        data.probe.state = 'done';
        data.probe._retrying = false;
        pm.probes[pid] = data.probe;
      }
      if (data.score && pm.report) pm.report.score = data.score;
      if (data.checks && pm.report) pm.report.checks = data.checks;
    } catch (e) {
      if (pm.probes[pid]) pm.probes[pid]._retrying = false;
      toast('重试 ' + pid + ' 失败: ' + e.message, 'bad');
    }
    maybeRerender(runId);
  }
  toast('所有失败探针重试完成', 'good');
}

/* ─── micro: button helper ─── */
function btn(label, opts) {
  opts = opts || {};
  const classes = ['btn'];
  if (opts.primary) classes.push('btn-primary');
  if (opts.danger)  classes.push('btn-danger');
  if (opts.ghost)   classes.push('btn-ghost');
  if (opts.quiet)   classes.push('btn-quiet');
  if (opts.size === 'sm') classes.push('btn-sm');
  if (opts.size === 'xs') classes.push('btn-xs');
  const b = el('button', { class: classes.join(' '), onclick: opts.onClick, title: opts.title });
  if (opts.icon) b.appendChild(mkIcon(opts.icon, { size: opts.size === 'xs' ? 11 : 13 }));
  if (label) b.appendChild(document.createTextNode(label));
  return b;
}

/* ─── breadcrumb setter ─── */
function setCrumb(items, actions) {
  const bar = $('#crumbBar');
  bar.innerHTML = '';
  bar.classList.remove('hidden');
  const crumb = el('div', { class: 'crumb' });
  items.forEach((it, i) => {
    if (i > 0) crumb.appendChild(el('span', { class: 'sep' }, '/'));
    if (it.cur) crumb.appendChild(el('span', { class: 'cur' + (it.mono ? ' mono' : '') }, it.cur));
    else crumb.appendChild(el('a', { href: it.href }, it.label));
  });
  bar.appendChild(crumb);
  if (actions) bar.appendChild(actions);
}
function hideCrumb() { $('#crumbBar').classList.add('hidden'); }
