/* ─── monitor.js · Continuous monitoring dashboard ────────────────────── */

const STATUS_PILL = {
  ok:       'pill-good',
  warning:  'pill-warn',
  critical: 'pill-bad',
  unknown:  'pill-info',
};
const STATUS_LED = {
  ok:       'pass',
  warning:  'warn',
  critical: 'fail',
  unknown:  'pending',
};
const STATUS_LABEL = {
  ok:       'OK',
  warning:  '警告',
  critical: '严重',
  unknown:  '未知',
};

/* ═══════════════════════════════════════════════════════════════════════
 * Dashboard — target grid + health overview + activity feed
 * ═══════════════════════════════════════════════════════════════════════ */

let _monitorPollTimer = null;
function stopMonitorPoll() {
  if (_monitorPollTimer) { clearInterval(_monitorPollTimer); _monitorPollTimer = null; }
}

async function renderMonitorDashboard() {
  await ensureModelMeta();
  stopMonitorPoll();
  setCrumb([{ label: 'Monitor', href: '#/monitor' }, { cur: '监控面板' }],
    el('div', { class: 'crumb-actions' },
      btn('告警事件', { onClick: () => location.hash = '#/monitor/alerts', icon: 'bell', size: 'sm', ghost: true }),
    ));

  const v = $('#view');
  v.innerHTML = '<div class="empty">加载中…</div>';

  try {
    const [targData, runData] = await Promise.all([
      api('/api/monitor/targets'),
      api('/api/monitor/runs?target_id='),
    ]);
    State.monitor.targets = targData.targets || [];
    State.monitor.states = targData.states || [];
    State.monitor.recentRuns = runData.runs || [];
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty', style: { color: 'var(--bad-ink)' } },
      el('div', { class: 'glyph' }, '×'), '加载失败: ' + esc(e.message)));
    return;
  }

  v.innerHTML = '';
  v.appendChild(buildDashboardBody());
  paintRail(parseRoute(location.hash));

  _monitorPollTimer = setInterval(async () => {
    const route = parseRoute(location.hash);
    if (route.app !== 'monitor' || route.kind !== 'dashboard') { stopMonitorPoll(); return; }
    try {
      const [targData, runData] = await Promise.all([
        api('/api/monitor/targets'),
        api('/api/monitor/runs?target_id='),
      ]);
      const oldStates = JSON.stringify(State.monitor.states);
      const oldRuns = JSON.stringify((State.monitor.recentRuns || []).map(r => r.id));
      State.monitor.targets = targData.targets || [];
      State.monitor.states = targData.states || [];
      State.monitor.recentRuns = runData.runs || [];
      const newRuns = JSON.stringify((State.monitor.recentRuns || []).map(r => r.id));
      if (JSON.stringify(State.monitor.states) !== oldStates || oldRuns !== newRuns) {
        const v = $('#view'); if (v) { v.innerHTML = ''; v.appendChild(buildDashboardBody()); }
        paintRail(parseRoute(location.hash));
      }
    } catch {}
  }, 30000);
}

function targetWorstStatus(target, states) {
  const targetStates = states.filter(s => s.target_id === target.id);
  return targetStates.reduce((w, s) => {
    const order = { critical: 3, warning: 2, unknown: 1, ok: 0 };
    return (order[s.status] || 0) > (order[w] || 0) ? s.status : w;
  }, targetStates.length > 0 ? 'ok' : 'unknown');
}

function buildDashboardBody() {
  const wrap = el('div');
  const targets = State.monitor.targets;
  const states = State.monitor.states;

  // per-channel status counts
  const chCounts = { total: targets.length, ok: 0, warning: 0, critical: 0, unknown: 0, enabled: 0, paused: 0 };
  targets.forEach(t => {
    const ws = targetWorstStatus(t, states);
    if (chCounts[ws] != null) chCounts[ws]++;
    if (t.enabled) chCounts.enabled++; else chCounts.paused++;
  });

  if (targets.length > 0) {
    wrap.appendChild(el('div', { class: 'statbar' },
      el('div', { class: 'cell' }, el('div', { class: 'k' }, '渠道总数'), el('div', { class: 'v big' }, chCounts.total)),
      el('div', { class: 'cell' }, el('div', { class: 'k' }, '正常'), el('div', { class: 'v big good' }, chCounts.ok)),
      el('div', { class: 'cell' }, el('div', { class: 'k' }, '告警'), el('div', { class: 'v big', style: { color: 'var(--warn-ink)' } }, chCounts.warning)),
      el('div', { class: 'cell' }, el('div', { class: 'k' }, '异常'), el('div', { class: 'v big bad' }, chCounts.critical)),
      el('div', { class: 'cell' }, el('div', { class: 'k' }, '运行中'), el('div', { class: 'v big' }, chCounts.enabled)),
      el('div', { class: 'cell' }, el('div', { class: 'k' }, '已暂停'), el('div', { class: 'v big' }, chCounts.paused)),
    ));
  }

  // target cards — sorted: critical > warning > unknown > ok, then paused last
  const panel = el('div', { class: 'panel' });
  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '渠道列表'),
    el('span', { class: 'meta' }, targets.length + ' 个渠道'),
    el('div', { class: 'spacer' }),
    btn('+ 添加渠道', { icon: 'add', size: 'sm', primary: true, onClick: () => openTargetDrawer() }),
  ));

  if (targets.length === 0) {
    panel.appendChild(el('div', { class: 'empty' },
      el('div', { class: 'glyph' }, '◎'),
      '尚无监控渠道 · 点击「添加渠道」开始持续检测'));
  } else {
    const statusOrder = { critical: 0, warning: 1, unknown: 2, ok: 3 };
    const sorted = targets.slice().sort((a, b) => {
      if (a.enabled !== b.enabled) return a.enabled ? -1 : 1;
      const sa = statusOrder[targetWorstStatus(a, states)] ?? 2;
      const sb = statusOrder[targetWorstStatus(b, states)] ?? 2;
      return sa - sb;
    });
    const body = el('div', { class: 'panel-body', style: { padding: '12px' } });
    const grid = el('div', { class: 'mon-grid' });
    sorted.forEach(t => grid.appendChild(buildTargetCard(t, states)));
    body.appendChild(grid);
    panel.appendChild(body);
  }
  wrap.appendChild(panel);

  // recent activity feed
  const runs = State.monitor.recentRuns || [];
  if (runs.length > 0) {
    wrap.appendChild(buildActivityPanel(runs, targets));
  }

  return wrap;
}

function buildTargetCard(target, states) {
  const targetStates = states.filter(s => s.target_id === target.id);
  const worstStatus = targetWorstStatus(target, states);
  const borderColor = worstStatus === 'ok' ? 'var(--good)' : worstStatus === 'warning' ? 'var(--warn)' : worstStatus === 'critical' ? 'var(--bad)' : 'var(--ink-5)';

  const card = el('div', { class: 'mon-card' + (target.enabled ? '' : ' paused') });
  card.style.borderLeftColor = borderColor;

  // header: name + status + actions
  const header = el('div', { class: 'mon-card-head' });
  header.appendChild(el('div', { class: 'mon-card-name', onclick: () => location.hash = '#/monitor/target/' + target.id },
    el('span', { class: 'led-dot ' + (STATUS_LED[worstStatus] || 'pending') }),
    el('span', { class: 'name-text' }, target.name || '未命名渠道'),
  ));
  const actions = el('div', { class: 'mon-card-actions' });
  actions.appendChild(el('button', {
    class: 'iconbtn', title: '手动运行',
    onclick: ev => { ev.stopPropagation(); manualRunTarget(target.id); },
  }, mkIcon('play', { size: 12 })));
  actions.appendChild(el('button', {
    class: 'iconbtn', title: target.enabled ? '暂停' : '启用',
    onclick: async ev => { ev.stopPropagation(); await toggleTarget(target.id, !target.enabled); renderMonitorDashboard(); },
  }, mkIcon(target.enabled ? 'pause' : 'power', { size: 12 })));
  actions.appendChild(el('button', {
    class: 'iconbtn', title: '编辑',
    onclick: ev => { ev.stopPropagation(); openTargetDrawer(target); },
  }, mkIcon('edit', { size: 12 })));
  header.appendChild(actions);
  card.appendChild(header);

  // url + badges row
  const meta = el('div', { class: 'mon-card-meta' });
  meta.appendChild(el('span', { class: 'mono url-text' }, (target.base_url || '').replace(/^https?:\/\//, '')));
  meta.appendChild(el('span', { class: 'itag itag-accent' }, target.check_type || 'channel'));
  if (!target.enabled) meta.appendChild(el('span', { class: 'itag' }, '已暂停'));
  card.appendChild(meta);

  // per-model health dots
  if (targetStates.length > 0) {
    const modelRow = el('div', { class: 'mon-model-row' });
    targetStates.forEach(s => {
      const sc = s.status === 'ok' ? 'var(--good)' : s.status === 'warning' ? 'var(--warn)' : s.status === 'critical' ? 'var(--bad)' : 'var(--ink-5)';
      const modelPill = el('div', { class: 'mon-model-pill',
        onclick: () => location.hash = '#/monitor/target/' + target.id,
      });
      modelPill.appendChild(el('span', { class: 'led-dot ' + (STATUS_LED[s.status] || 'pending'), style: { width: '5px', height: '5px' } }));
      modelPill.appendChild(el('span', { class: 'model-name' }, s.model.replace('claude-', '').replace(/-/g, ' ')));
      modelPill.appendChild(el('span', { class: 'model-score', style: { color: sc } }, s.grade || '–'));
      if (s.score != null && s.score > 0) {
        const bar = el('div', { class: 'mini-bar' });
        bar.appendChild(el('div', { class: 'mini-fill', style: { width: Math.min(s.score, 100) + '%', background: sc } }));
        modelPill.appendChild(bar);
      }
      modelRow.appendChild(modelPill);
    });
    card.appendChild(modelRow);
  } else {
    card.appendChild(el('div', { class: 'muted', style: { fontSize: '11px', padding: '4px 0' } }, '尚未运行'));
  }

  // footer: interval + models count + last check
  const lastCheck = targetStates.reduce((latest, s) =>
    s.last_check && new Date(s.last_check) > new Date(latest || 0) ? s.last_check : latest, null);
  const foot = el('div', { class: 'mon-card-foot' });
  foot.appendChild(el('span', null, formatInterval(target.interval)));
  foot.appendChild(el('span', null, (target.models || []).length + ' models'));
  foot.appendChild(el('span', null, lastCheck ? fmtTimeAgo(lastCheck) : '—'));
  card.appendChild(foot);

  return card;
}

function buildActivityPanel(runs, targets) {
  const targetMap = {};
  targets.forEach(t => { targetMap[t.id] = t; });

  const panel = el('div', { class: 'panel', style: { marginTop: '12px' } });
  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '最近活动'),
    el('span', { class: 'meta' }, '最近 ' + runs.length + ' 条运行记录'),
  ));

  const t = el('table', { class: 'table' });
  t.appendChild(el('thead', null, el('tr', null,
    el('th', { style: { width: '24px' } }, ''),
    el('th', null, '渠道'),
    el('th', null, 'model'),
    el('th', { style: { width: '80px' } }, 'status'),
    el('th', { style: { width: '70px', textAlign: 'right' } }, 'score'),
    el('th', { style: { width: '50px', textAlign: 'right' } }, 'grade'),
    el('th', { style: { width: '80px', textAlign: 'right' } }, 'elapsed'),
    el('th', { style: { width: '50px' } }, 'changed'),
    el('th', { style: { width: '110px' } }, 'when'),
  )));
  const tb = el('tbody');
  runs.slice(0, 30).forEach(r => {
    const sc = r.status === 'ok' ? 'var(--good)' : r.status === 'warning' ? 'var(--warn)' : r.status === 'critical' ? 'var(--bad)' : 'var(--ink-4)';
    const tgt = targetMap[r.target_id];
    const channelName = tgt ? (tgt.name || tgt.base_url) : r.target_id;
    const tr = el('tr', { onclick: () => {
      if (r.report || r.intelligence_report) openRunDetailModal(r);
    }});
    tr.appendChild(el('td', null, el('span', { class: 'led-dot ' + (STATUS_LED[r.status] || 'pending') })));
    tr.appendChild(el('td', { class: 'name-cell', style: { maxWidth: '160px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }, channelName));
    tr.appendChild(el('td', { class: 'mono' }, r.model || '—'));
    tr.appendChild(el('td', null,
      el('span', { class: 'pill ' + (STATUS_PILL[r.status] || ''), style: { fontSize: '10px' } },
        el('span', { class: 'led' }), STATUS_LABEL[r.status] || r.status)));
    tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right', color: sc, fontWeight: 600 } },
      r.score != null ? Math.round(r.score) : '—'));
    tr.appendChild(el('td', { class: 'mono', style: { textAlign: 'right' } }, r.grade || '—'));
    tr.appendChild(el('td', { class: 'mono', style: { textAlign: 'right' } }, fmtMs(r.elapsed_ms)));
    tr.appendChild(el('td', null, r.changed
      ? el('span', { class: 'itag itag-warn' }, r.prev_state + '→' + r.status)
      : el('span', { class: 'itag' }, '—')));
    tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } }, fmtTimeAgo(r.started_at)));
    tb.appendChild(tr);
  });
  t.appendChild(tb);
  panel.appendChild(t);
  return panel;
}

async function toggleTarget(targetId, enabled) {
  try {
    await api('/api/monitor/targets/' + targetId, {
      method: 'PATCH',
      body: JSON.stringify({ enabled }),
    });
    toast(enabled ? '已启用' : '已暂停', 'good');
  } catch (e) {
    toast('操作失败: ' + e.message, 'bad');
  }
}

function formatInterval(ns) {
  if (!ns) return '—';
  const sec = ns / 1e9;
  if (sec < 60) return sec + 's';
  if (sec < 3600) return Math.round(sec / 60) + 'm';
  return Math.round(sec / 3600) + 'h';
}

/* ═══════════════════════════════════════════════════════════════════════
 * Target Drawer — add/edit target
 * ═══════════════════════════════════════════════════════════════════════ */

async function openTargetDrawer(existingTarget) {
  const isEdit = !!existingTarget;
  const t = existingTarget || {};
  let name = t.name || '';
  let baseURL = t.base_url || '';
  let apiKey = '';
  let models = t.models ? t.models.slice() : ['claude-sonnet-4-6'];
  let interval = t.interval ? formatInterval(t.interval) : '5m';
  let jitter = t.jitter ? formatInterval(t.jitter) : '';
  let enabled = t.enabled !== false;
  let checkType = t.check_type || 'channel';
  let intDataset = t.intelligence_dataset || '';
  let intLimit = t.intelligence_limit || 3;
  let intMaxLimit = t.intelligence_max_limit != null ? t.intelligence_max_limit : 0;
  let intThreshold = t.intelligence_threshold || 1.0;
  let baselineID = t.baseline_id || '';
  let effort = t.effort || '';
  let thinkingMode = t.thinking_mode || '';
  let maxTokens = t.max_tokens || 0;

  let datasets = [];
  let baselines = [];
  try {
    const [dsData, blData] = await Promise.all([
      api('/api/intelligence/datasets').catch(() => ({ datasets: [] })),
      api('/api/monitor/baselines').catch(() => ({ baselines: [] })),
    ]);
    datasets = dsData.datasets || [];
    baselines = blData.baselines || [];
  } catch {}

  const body = el('div');
  const intSection = el('div');

  function renderIntSection() {
    intSection.innerHTML = '';
    if (checkType === 'channel') return;

    intSection.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '14px', marginBottom: '6px' } }, 'INTELLIGENCE CONFIG'));

    const dsSel = el('select', { style: { width: '100%' }, onchange: e => { intDataset = e.target.value; } });
    dsSel.appendChild(el('option', { value: '' }, '(默认 / 第一个)'));
    datasets.forEach(d => {
      const opt = el('option', { value: d.name }, d.name + ' · ' + (d.total_tasks || 0) + ' tasks');
      if (d.name === intDataset) opt.selected = true;
      dsSel.appendChild(opt);
    });
    intSection.appendChild(buildField('DATASET', dsSel));

    intSection.appendChild(buildField('初始采样 / TICK', el('input', {
      type: 'number', min: 1, max: 100, value: intLimit, class: 'mono',
      oninput: e => { intLimit = parseInt(e.target.value) || 3; },
      style: { width: '80px', background: 'transparent', border: 'none', padding: '0' }
    })));
    intSection.appendChild(buildField('升级采样 / 0=全量', el('input', {
      type: 'number', min: 0, max: 500, value: intMaxLimit, class: 'mono',
      oninput: e => { intMaxLimit = parseInt(e.target.value) || 0; },
      style: { width: '80px', background: 'transparent', border: 'none', padding: '0' }
    })));
    intSection.appendChild(buildField('偏差阈值 / TASK', el('input', {
      type: 'number', step: 0.1, value: intThreshold, class: 'mono',
      oninput: e => { intThreshold = parseFloat(e.target.value) || 1.0; },
      style: { width: '80px', background: 'transparent', border: 'none', padding: '0' }
    })));

    intSection.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '14px', marginBottom: '6px' } }, 'EFFORT PROFILE'));

    const effortSeg = el('div', { class: 'seg', style: { width: 'fit-content' } });
    ['', 'low', 'medium', 'high', 'xhigh', 'max'].forEach(v => {
      const b = el('button', { class: effort === v ? 'active' : '',
        onclick: () => { effort = v; effortSeg.querySelectorAll('button').forEach(x => x.classList.remove('active')); b.classList.add('active'); }
      }, v || 'off');
      effortSeg.appendChild(b);
    });
    intSection.appendChild(buildField('EFFORT', effortSeg));

    const tmSeg = el('div', { class: 'seg', style: { width: 'fit-content' } });
    ['', 'adaptive', 'adaptive_only', 'enabled'].forEach(v => {
      const b = el('button', { class: thinkingMode === v ? 'active' : '',
        onclick: () => { thinkingMode = v; tmSeg.querySelectorAll('button').forEach(x => x.classList.remove('active')); b.classList.add('active'); }
      }, v || 'auto');
      tmSeg.appendChild(b);
    });
    intSection.appendChild(buildField('THINKING MODE', tmSeg));

    intSection.appendChild(buildField('MAX TOKENS (0=default)', el('input', {
      type: 'number', min: 0, max: 128000, value: maxTokens, class: 'mono',
      oninput: e => { maxTokens = parseInt(e.target.value) || 0; },
      style: { width: '100px', background: 'transparent', border: 'none', padding: '0' }
    })));
  }

  body.appendChild(buildField('渠道名称', el('input', {
    value: name, placeholder: '例: 官方直连 / 某某中转 / 测试渠道A',
    oninput: e => { name = e.target.value.trim(); },
    style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
  })));
  body.appendChild(buildField('BASE URL', el('input', {
    class: 'mono', value: baseURL, placeholder: 'https://api.example.com',
    oninput: e => { baseURL = e.target.value.trim(); },
    style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
  })));
  body.appendChild(buildField('API KEY', el('input', {
    type: 'password', class: 'mono', value: apiKey, placeholder: 'sk-...',
    oninput: e => { apiKey = e.target.value; },
    style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
  })));
  body.appendChild(buildField('MODELS', buildMonitorModelChips(models)));

  // check type radio
  const ctSeg = el('div', { class: 'seg', style: { width: 'fit-content' } });
  ['channel', 'intelligence', 'both'].forEach(v => {
    const b = el('button', { class: checkType === v ? 'active' : '',
      onclick: () => { checkType = v; ctSeg.querySelectorAll('button').forEach(x => x.classList.remove('active')); b.classList.add('active'); renderIntSection(); }
    }, { channel: '指纹', intelligence: '智商', both: '两者' }[v]);
    ctSeg.appendChild(b);
  });
  body.appendChild(buildField('CHECK TYPE', ctSeg));

  body.appendChild(intSection);
  renderIntSection();

  body.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '14px', marginBottom: '6px' } }, 'SCHEDULE'));
  body.appendChild(buildField('INTERVAL', el('input', {
    value: interval, placeholder: '5m',
    oninput: e => { interval = e.target.value.trim(); },
    class: 'mono', style: { width: '120px', background: 'transparent', border: 'none', padding: '0' }
  })));
  body.appendChild(buildField('JITTER (optional)', el('input', {
    value: jitter, placeholder: '30s',
    oninput: e => { jitter = e.target.value.trim(); },
    class: 'mono', style: { width: '120px', background: 'transparent', border: 'none', padding: '0' }
  })));

  // baseline selector
  const blSel = el('select', { style: { width: '100%' }, onchange: e => { baselineID = e.target.value; } });
  blSel.appendChild(el('option', { value: '' }, '(无基线)'));
  baselines.forEach(b => {
    const label = (b.name || b.id) + ' · ' + b.model + ' · ' + (b.thinking_effort || 'off');
    const opt = el('option', { value: b.id }, label);
    if (b.id === baselineID) opt.selected = true;
    blSel.appendChild(opt);
  });
  body.appendChild(buildField('BASELINE (对比基准)', blSel));

  const foot = el('div', null,
    btn('取消', { ghost: true, onClick: closeDrawer }),
    btn(isEdit ? '保存' : '添加', { primary: true, icon: isEdit ? 'check' : 'add', onClick: async () => {
      if (!baseURL) { toast('请填写 Base URL', 'bad'); return; }
      if (!isEdit && !apiKey) { toast('请填写 API Key', 'bad'); return; }

      const payload = {
        name: name || undefined, base_url: baseURL, models, interval,
        jitter: jitter || undefined, enabled, check_type: checkType,
        intelligence_dataset: intDataset || undefined,
        intelligence_limit: intLimit, intelligence_max_limit: intMaxLimit,
        intelligence_threshold: intThreshold,
        baseline_id: baselineID || undefined,
        effort: effort || undefined,
        thinking_mode: thinkingMode || undefined,
        max_tokens: maxTokens || undefined,
      };
      if (apiKey) payload.api_key = apiKey;

      try {
        if (isEdit) {
          await api('/api/monitor/targets/' + t.id, { method: 'PATCH', body: JSON.stringify(payload) });
          toast('已更新', 'good');
        } else {
          payload.api_key = apiKey;
          await api('/api/monitor/targets', { method: 'POST', body: JSON.stringify(payload) });
          toast('已添加', 'good');
        }
        closeDrawer();
        renderMonitorDashboard();
      } catch (e) { toast('失败: ' + e.message, 'bad'); }
    }}),
  );

  openDrawer(isEdit ? '编辑渠道' : '添加渠道', body, foot);
}

function buildMonitorModelChips(models) {
  const ALL = modelIDs();
  const set = new Set(models);
  const wrap = el('div', { class: 'chip-set' });
  ALL.forEach(m => {
    const lbl = el('label', { class: 'chip' });
    const cb = el('input', { type: 'checkbox', value: m });
    if (set.has(m)) cb.checked = true;
    cb.addEventListener('change', () => {
      if (cb.checked) set.add(m); else set.delete(m);
      models.length = 0;
      models.push(...set);
    });
    lbl.appendChild(cb);
    lbl.appendChild(el('span', { class: 'led' }));
    lbl.appendChild(document.createTextNode(m));
    wrap.appendChild(lbl);
  });
  return wrap;
}

/* ═══════════════════════════════════════════════════════════════════════
 * Target Detail — info + manual run + run history
 * ═══════════════════════════════════════════════════════════════════════ */

async function renderMonitorTarget(targetId) {
  setCrumb([{ label: 'Monitor', href: '#/monitor' }, { cur: '加载中…' }]);
  const v = $('#view');
  v.innerHTML = '<div class="empty">加载中…</div>';

  let target, states, runs;
  try {
    const [tData, sData, rData] = await Promise.all([
      api('/api/monitor/targets/' + targetId),
      api('/api/monitor/targets'),
      api('/api/monitor/runs?target_id=' + targetId),
    ]);
    target = tData;
    State.monitor.targets = sData.targets || [];
    State.monitor.states = sData.states || [];
    states = (sData.states || []).filter(s => s.target_id === targetId);
    runs = rData.runs || [];
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty' },
      el('div', { class: 'glyph' }, '×'), '加载失败: ' + esc(e.message),
      el('div', { style: { marginTop: '12px' } },
        btn('返回面板', { onClick: () => location.hash = '#/monitor', icon: 'activity', size: 'sm' })
      )));
    return;
  }

  setCrumb([{ label: 'Monitor', href: '#/monitor' }, { cur: target.name || target.base_url }],
    el('div', { class: 'crumb-actions' },
      btn(target.enabled ? '暂停' : '启用', {
        icon: target.enabled ? 'pause' : 'power', size: 'sm', ghost: true,
        onClick: async () => { await toggleTarget(targetId, !target.enabled); renderMonitorTarget(targetId); },
      }),
      btn('编辑', { icon: 'edit', size: 'sm', ghost: true, onClick: () => openTargetDrawer(target) }),
      btn('手动运行', { icon: 'play', size: 'sm', primary: true, onClick: () => manualRunTarget(targetId) }),
      btn('删除', { icon: 'trash', size: 'sm', danger: true, onClick: () => deleteTarget(targetId) }),
    ));

  v.innerHTML = '';

  // statbar
  const worstStatus = states.reduce((w, s) => {
    const order = { critical: 3, warning: 2, unknown: 1, ok: 0 };
    return (order[s.status] || 0) > (order[w] || 0) ? s.status : w;
  }, states.length > 0 ? 'ok' : 'unknown');
  const statusBadge = el('span', { class: 'pill ' + (STATUS_PILL[worstStatus] || 'pill-info') },
    el('span', { class: 'led' }), STATUS_LABEL[worstStatus] || worstStatus);

  v.appendChild(el('div', { class: 'statbar' },
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'STATUS'), el('div', { class: 'v' }, statusBadge)),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'TARGET'), el('div', { class: 'v' }, target.name || '—')),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'URL'), el('div', { class: 'v mono', style: { fontSize: '11px' } }, (target.base_url || '').replace(/^https?:\/\//, ''))),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'MODELS'), el('div', { class: 'v' }, (target.models || []).length)),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'CHECK TYPE'), el('div', { class: 'v' },
      el('span', { class: 'itag itag-accent' }, target.check_type || 'channel'))),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'INTERVAL'), el('div', { class: 'v mono' }, formatInterval(target.interval))),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'ENABLED'), el('div', { class: 'v' }, target.enabled ? 'YES' : 'NO')),
    target.effort ? el('div', { class: 'cell' }, el('div', { class: 'k' }, 'EFFORT'), el('div', { class: 'v' }, el('span', { class: 'itag itag-accent' }, target.effort))) : null,
    target.thinking_mode ? el('div', { class: 'cell' }, el('div', { class: 'k' }, 'THINKING'), el('div', { class: 'v' }, el('span', { class: 'itag itag-info' }, target.thinking_mode))) : null,
  ));

  // per-model health states
  if (states.length > 0) {
    const statePanel = el('div', { class: 'panel', style: { marginBottom: '12px' } });
    statePanel.appendChild(el('div', { class: 'panel-head' },
      el('h3', null, '模型健康状态'),
      el('span', { class: 'meta' }, states.length + ' 个模型'),
    ));
    const stateBody = el('div', { class: 'panel-body', style: { padding: '12px' } });
    const grid = el('div', { class: 'model-grid', style: { margin: 0 } });
    states.forEach(s => {
      const sc = s.status === 'ok' ? 'var(--good)' : s.status === 'warning' ? 'var(--warn)' : s.status === 'critical' ? 'var(--bad)' : 'var(--ink-4)';
      const card = el('div', { class: 'model-card' });
      card.appendChild(el('div', { class: 'top' },
        el('span', { class: 'led-dot ' + (STATUS_LED[s.status] || 'pending') }),
        el('div', { class: 'name' }, s.model),
        el('div', { class: 'grade', style: { color: sc } }, s.grade || '–'),
      ));
      card.appendChild(el('div', { class: 'score-line' },
        el('span', { class: 'num', style: { color: sc } }, s.score != null ? Math.round(s.score) : '–'),
        el('span', { class: 'denom' }, '/ 100'),
        el('span', { class: 'spacer' }),
        el('span', { class: 'pill ' + (STATUS_PILL[s.status] || 'pill-info'), style: { fontSize: '10px' } },
          el('span', { class: 'led' }), STATUS_LABEL[s.status] || s.status),
      ));
      const info = el('div', { style: { fontSize: '10px', fontFamily: 'var(--font-mono)', color: 'var(--ink-4)', marginTop: '6px', display: 'grid', gap: '2px' } });
      info.appendChild(el('div', null, '连续失败: ' + (s.consec_fails || 0) + ' · 连续OK: ' + (s.consec_ok || 0)));
      if (s.last_check) info.appendChild(el('div', null, '上次检测: ' + fmtTimeAgo(s.last_check)));
      if (s.last_change) info.appendChild(el('div', null, '上次变化: ' + fmtTimeAgo(s.last_change)));
      card.appendChild(info);
      grid.appendChild(card);
    });
    stateBody.appendChild(grid);
    statePanel.appendChild(stateBody);
    v.appendChild(statePanel);
  }

  // run history
  v.appendChild(buildRunHistoryPanel(runs));
}

function buildRunHistoryPanel(runs) {
  const panel = el('div', { class: 'panel' });
  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '运行历史'),
    el('span', { class: 'meta' }, runs.length + ' 条记录'),
  ));

  if (runs.length === 0) {
    panel.appendChild(el('div', { class: 'empty' },
      el('div', { class: 'glyph' }, '∅'), '尚无运行记录'));
    return panel;
  }

  const t = el('table', { class: 'table' });
  t.appendChild(el('thead', null, el('tr', null,
    el('th', { style: { width: '24px' } }, ''),
    el('th', null, 'model'),
    el('th', { style: { width: '80px' } }, 'status'),
    el('th', { style: { width: '70px', textAlign: 'right' } }, 'score'),
    el('th', { style: { width: '50px', textAlign: 'right' } }, 'grade'),
    el('th', { style: { width: '80px', textAlign: 'right' } }, 'elapsed'),
    el('th', { style: { width: '50px' } }, 'changed'),
    el('th', { style: { width: '110px' } }, 'when'),
  )));
  const tb = el('tbody');
  runs.forEach(r => {
    const sc = r.status === 'ok' ? 'var(--good)' : r.status === 'warning' ? 'var(--warn)' : r.status === 'critical' ? 'var(--bad)' : 'var(--ink-4)';
    const tr = el('tr', { onclick: () => {
      if (r.report) openRunDetailModal(r);
    }});
    tr.appendChild(el('td', null, el('span', { class: 'led-dot ' + (STATUS_LED[r.status] || 'pending') })));
    tr.appendChild(el('td', { class: 'mono' }, r.model || '—'));
    tr.appendChild(el('td', null,
      el('span', { class: 'pill ' + (STATUS_PILL[r.status] || ''), style: { fontSize: '10px' } },
        el('span', { class: 'led' }), STATUS_LABEL[r.status] || r.status)));
    tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right', color: sc, fontWeight: 600 } },
      r.score != null ? Math.round(r.score) : '—'));
    tr.appendChild(el('td', { class: 'mono', style: { textAlign: 'right' } }, r.grade || '—'));
    tr.appendChild(el('td', { class: 'mono', style: { textAlign: 'right' } }, fmtMs(r.elapsed_ms)));
    tr.appendChild(el('td', null, r.changed
      ? el('span', { class: 'itag itag-warn' }, r.prev_state + '→' + r.status)
      : el('span', { class: 'itag' }, '—')));
    tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } }, fmtTimeAgo(r.started_at)));
    tb.appendChild(tr);
  });
  t.appendChild(tb);
  panel.appendChild(t);
  return panel;
}

function openRunDetailModal(run) {
  const body = el('div');
  body.appendChild(el('div', { class: 'tag-row', style: { marginBottom: '12px' } },
    el('span', { class: 'itag itag-' + (run.status === 'ok' ? 'good' : run.status === 'warning' ? 'warn' : 'bad') }, run.status),
    el('span', { class: 'itag itag-info' }, run.model || ''),
    el('span', { class: 'itag itag-accent' }, run.check_type || 'channel'),
    el('span', { class: 'itag' }, fmtMs(run.elapsed_ms)),
    run.escalated ? el('span', { class: 'itag itag-warn' }, '已升级采样') : null,
    run.changed ? el('span', { class: 'itag itag-warn' }, run.prev_state + ' → ' + run.status) : null,
  ));

  if (run.error) {
    body.appendChild(el('div', { class: 'eyebrow', style: { marginBottom: '4px', color: 'var(--bad-ink)' } }, 'CHANNEL ERROR'));
    body.appendChild(el('pre', { class: 'json-out', style: { color: 'var(--bad-ink)' } }, run.error));
  }
  if (run.intelligence_error) {
    body.appendChild(el('div', { class: 'eyebrow', style: { marginBottom: '4px', color: 'var(--bad-ink)' } }, 'INTELLIGENCE ERROR'));
    body.appendChild(el('pre', { class: 'json-out', style: { color: 'var(--bad-ink)' } }, run.intelligence_error));
  }

  // ─── Baseline diff panel ───
  if (run.baseline_diff) {
    const diff = run.baseline_diff;
    body.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '8px', marginBottom: '4px' } },
      '基线对比 · ' + (diff.baseline_name || diff.baseline_id)));

    if (diff.channel) {
      const cd = diff.channel;
      const deltaColor = cd.score_delta >= 0 ? 'var(--good-ink)' : 'var(--bad-ink)';
      body.appendChild(el('div', { style: { display: 'flex', gap: '12px', marginBottom: '8px', fontSize: '12px', fontFamily: 'var(--font-mono)' } },
        el('span', null, '基线: ' + Math.round(cd.baseline_score)),
        el('span', null, '当前: ' + Math.round(cd.current_score)),
        el('span', { style: { color: deltaColor, fontWeight: 600 } }, 'Δ ' + (cd.score_delta >= 0 ? '+' : '') + cd.score_delta.toFixed(1)),
      ));
      if (cd.check_diffs && cd.check_diffs.some(d => d.deviated)) {
        const diffTable = el('div', { style: { border: '1px solid var(--line)', borderRadius: 'var(--r-sm)', marginBottom: '8px', overflow: 'hidden' } });
        cd.check_diffs.filter(d => d.deviated).forEach(d => {
          diffTable.appendChild(el('div', { style: { display: 'flex', gap: '8px', padding: '4px 10px', fontSize: '11px', fontFamily: 'var(--font-mono)', borderBottom: '1px solid var(--line-soft)', background: 'var(--bad-soft)' } },
            el('span', { class: 'led-dot ' + (d.current_pass ? 'pass' : 'fail'), style: { marginTop: '3px' } }),
            el('span', { style: { flex: 1 } }, d.name),
            el('span', { style: { color: 'var(--ink-3)' } }, (d.baseline_pass ? 'PASS' : 'FAIL') + ' → ' + (d.current_pass ? 'PASS' : 'FAIL')),
          ));
        });
        body.appendChild(diffTable);
      }
    }

    if (diff.intelligence) {
      const id = diff.intelligence;
      const devColor = Math.abs(id.aggregate_deviation) >= 4 ? 'var(--bad-ink)' : Math.abs(id.aggregate_deviation) >= 2 ? 'var(--warn-ink)' : 'var(--good-ink)';
      body.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '8px', marginBottom: '4px' } }, 'INTELLIGENCE DIFF'));
      body.appendChild(el('div', { style: { display: 'flex', gap: '12px', marginBottom: '8px', fontSize: '12px', fontFamily: 'var(--font-mono)' } },
        el('span', null, '重叠 tasks: ' + id.overlapping_tasks),
        el('span', { style: { color: devColor, fontWeight: 600 } }, '聚合偏差: ' + (id.aggregate_deviation >= 0 ? '+' : '') + id.aggregate_deviation.toFixed(2)),
        el('span', null, '绝对偏差: ' + id.abs_aggregate_deviation.toFixed(2)),
      ));
      if (id.task_diffs && id.task_diffs.length > 0) {
        const diffTable = el('div', { style: { border: '1px solid var(--line)', borderRadius: 'var(--r-sm)', marginBottom: '8px', overflow: 'hidden', maxHeight: '200px', overflowY: 'auto' } });
        id.task_diffs.forEach(td => {
          const dc = Math.abs(td.deviation) > 1 ? 'var(--bad-soft)' : 'var(--panel)';
          diffTable.appendChild(el('div', { style: { display: 'grid', gridTemplateColumns: '1fr 60px 60px 60px', gap: '8px', padding: '4px 10px', fontSize: '10px', fontFamily: 'var(--font-mono)', borderBottom: '1px solid var(--line-soft)', background: dc } },
            el('span', { style: { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }, td.task_id),
            el('span', { style: { textAlign: 'right' } }, td.baseline_score.toFixed(1)),
            el('span', { style: { textAlign: 'right' } }, td.current_score.toFixed(1)),
            el('span', { style: { textAlign: 'right', color: Math.abs(td.deviation) > 1 ? 'var(--bad-ink)' : 'var(--ink-3)' } },
              (td.deviation >= 0 ? '+' : '') + td.deviation.toFixed(2)),
          ));
        });
        body.appendChild(diffTable);
      }
    }
  }

  if (run.report) {
    body.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '8px', marginBottom: '4px' } }, 'CHANNEL REPORT'));
    body.appendChild(el('pre', { class: 'json-out', style: { maxHeight: '300px' } }, JSON.stringify(run.report, null, 2)));
  }
  if (run.intelligence_report) {
    body.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '8px', marginBottom: '4px' } }, 'INTELLIGENCE REPORT'));
    body.appendChild(el('pre', { class: 'json-out', style: { maxHeight: '300px' } }, JSON.stringify(run.intelligence_report, null, 2)));
  }

  openModal('运行详情 · ' + (run.model || ''), body);
}

async function manualRunTarget(targetId) {
  toast('正在运行…', 'good');
  try {
    await api('/api/monitor/targets/' + targetId + '/run', { method: 'POST', body: '{}' });
    toast('运行完成', 'good');
    renderMonitorTarget(targetId);
  } catch (e) {
    toast('运行失败: ' + e.message, 'bad');
  }
}

async function deleteTarget(targetId) {
  if (!confirm('确定删除此监控目标？')) return;
  try {
    await api('/api/monitor/targets/' + targetId, { method: 'DELETE' });
    toast('已删除', 'good');
    location.hash = '#/monitor';
  } catch (e) {
    toast('删除失败: ' + e.message, 'bad');
  }
}

/* ═══════════════════════════════════════════════════════════════════════
 * Alerts page — events + rules
 * ═══════════════════════════════════════════════════════════════════════ */

async function renderMonitorAlerts() {
  setCrumb([{ label: 'Monitor', href: '#/monitor' }, { cur: '告警事件' }]);
  const v = $('#view');
  v.innerHTML = '<div class="empty">加载中…</div>';

  let events, active, rules;
  try {
    const [eData, rData] = await Promise.all([
      api('/api/alert/events'),
      api('/api/alert/rules'),
    ]);
    events = eData.events || [];
    active = eData.active || [];
    rules = rData.rules || [];
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty', style: { color: 'var(--bad-ink)' } },
      el('div', { class: 'glyph' }, '×'), '加载失败: ' + esc(e.message)));
    return;
  }

  v.innerHTML = '';

  // active alerts statbar
  v.appendChild(el('div', { class: 'statbar' },
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'ACTIVE'), el('div', { class: 'v big bad' }, active.length)),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'TOTAL EVENTS'), el('div', { class: 'v big' }, events.length)),
    el('div', { class: 'cell' }, el('div', { class: 'k' }, 'RULES'), el('div', { class: 'v big' }, rules.length)),
  ));

  // events panel
  const evPanel = el('div', { class: 'panel', style: { marginBottom: '12px' } });
  evPanel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '告警事件'),
    el('span', { class: 'meta' }, events.length + ' 条 · ' + active.length + ' 条活跃'),
  ));

  if (events.length === 0) {
    evPanel.appendChild(el('div', { class: 'empty' },
      el('div', { class: 'glyph' }, '✓'), '暂无告警事件'));
  } else {
    const t = el('table', { class: 'table' });
    t.appendChild(el('thead', null, el('tr', null,
      el('th', { style: { width: '24px' } }, ''),
      el('th', null, 'rule'),
      el('th', { style: { width: '80px' } }, 'severity'),
      el('th', { style: { width: '80px' } }, 'status'),
      el('th', null, 'model'),
      el('th', null, 'message'),
      el('th', { style: { width: '70px', textAlign: 'right' } }, 'score'),
      el('th', { style: { width: '110px' } }, 'when'),
    )));
    const tb = el('tbody');
    events.forEach(ev => {
      const sevClass = ev.severity === 'critical' ? 'itag-bad' : ev.severity === 'warning' ? 'itag-warn' : 'itag-info';
      const statusClass = ev.status === 'firing' ? 'pill-bad' : 'pill-good';
      const tr = el('tr');
      tr.appendChild(el('td', null, el('span', { class: 'led-dot ' + (ev.status === 'firing' ? 'fail' : 'pass') })));
      tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } }, ev.rule_name || '—'));
      tr.appendChild(el('td', null, el('span', { class: 'itag ' + sevClass }, ev.severity)));
      tr.appendChild(el('td', null, el('span', { class: 'pill ' + statusClass, style: { fontSize: '10px' } },
        el('span', { class: 'led' }), ev.status)));
      tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } }, ev.model || '—'));
      tr.appendChild(el('td', { style: { fontSize: '12px', color: 'var(--ink-2)', maxWidth: '300px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } }, ev.message || ''));
      tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } }, ev.score != null ? Math.round(ev.score) : '—'));
      tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } }, fmtTimeAgo(ev.fired_at)));
      tb.appendChild(tr);
    });
    t.appendChild(tb);
    evPanel.appendChild(t);
  }
  v.appendChild(evPanel);

  // rules panel
  const rulesPanel = el('div', { class: 'panel' });
  rulesPanel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '告警规则'),
    el('span', { class: 'meta' }, rules.length + ' 条规则'),
  ));

  if (rules.length === 0) {
    rulesPanel.appendChild(el('div', { class: 'empty' },
      el('div', { class: 'glyph' }, '∅'), '暂无告警规则'));
  } else {
    const t = el('table', { class: 'table' });
    t.appendChild(el('thead', null, el('tr', null,
      el('th', null, 'name'),
      el('th', null, 'metric'),
      el('th', { style: { width: '50px' } }, 'op'),
      el('th', { style: { width: '60px', textAlign: 'right' } }, 'value'),
      el('th', null, 'check'),
      el('th', { style: { width: '80px', textAlign: 'right' } }, 'consec.'),
      el('th', { style: { width: '80px' } }, 'severity'),
      el('th', { style: { width: '80px' } }, 'cooldown'),
    )));
    const tb = el('tbody');
    rules.forEach(r => {
      const sevClass = r.severity === 'critical' ? 'itag-bad' : r.severity === 'warning' ? 'itag-warn' : 'itag-info';
      const tr = el('tr');
      tr.appendChild(el('td', { class: 'mono', style: { fontWeight: 500 } }, r.name));
      tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } }, r.metric));
      tr.appendChild(el('td', { class: 'mono' }, r.op));
      tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } }, r.value));
      tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } }, r.check || '—'));
      tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } }, r.consecutive));
      tr.appendChild(el('td', null, el('span', { class: 'itag ' + sevClass }, r.severity)));
      tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } }, r.cooldown || '30m'));
      tb.appendChild(tr);
    });
    t.appendChild(tb);
    rulesPanel.appendChild(t);
  }
  v.appendChild(rulesPanel);
}

/* ═══════════════════════════════════════════════════════════════════════
 * Baselines page — record / list / delete
 * ═══════════════════════════════════════════════════════════════════════ */

async function renderMonitorBaselines() {
  setCrumb([{ label: 'Monitor', href: '#/monitor' }, { cur: '基线管理' }],
    el('div', { class: 'crumb-actions' },
      btn('录制基线', { primary: true, icon: 'add', size: 'sm', onClick: () => openBaselineDrawer() }),
    ));

  const v = $('#view');
  v.innerHTML = '<div class="empty">加载中…</div>';

  let baselines;
  try {
    const data = await api('/api/monitor/baselines');
    baselines = data.baselines || [];
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty', style: { color: 'var(--bad-ink)' } },
      el('div', { class: 'glyph' }, '×'), '加载失败: ' + esc(e.message)));
    return;
  }

  v.innerHTML = '';

  const panel = el('div', { class: 'panel' });
  panel.appendChild(el('div', { class: 'panel-head' },
    el('h3', null, '基线库'),
    el('span', { class: 'meta' }, baselines.length + ' 条基线'),
    el('div', { class: 'spacer' }),
    btn('录制基线', { icon: 'add', size: 'sm', primary: true, onClick: () => openBaselineDrawer() }),
  ));

  if (baselines.length === 0) {
    panel.appendChild(el('div', { class: 'empty' },
      el('div', { class: 'glyph' }, '◎'),
      '尚无基线 · 使用官方 key 录制参考基准'));
  } else {
    const t = el('table', { class: 'table' });
    t.appendChild(el('thead', null, el('tr', null,
      el('th', null, 'name'),
      el('th', null, 'model'),
      el('th', { style: { width: '80px' } }, 'effort'),
      el('th', { style: { width: '80px', textAlign: 'right' } }, 'ch. score'),
      el('th', { style: { width: '80px', textAlign: 'right' } }, 'int. tasks'),
      el('th', { style: { width: '120px' } }, 'created'),
      el('th', { style: { width: '60px' } }, ''),
    )));
    const tb = el('tbody');
    baselines.forEach(b => {
      const chScore = b.channel_report && b.channel_report.score ? Math.round(b.channel_report.score.total_score) : '—';
      const intTasks = b.intelligence_report ? b.intelligence_report.task_completed + '/' + b.intelligence_report.task_total : '—';
      const tr = el('tr', { onclick: () => openBaselineDetailModal(b) });
      tr.appendChild(el('td', { class: 'name-cell' }, b.name || b.id));
      tr.appendChild(el('td', { class: 'mono' }, b.model));
      tr.appendChild(el('td', null, el('span', { class: 'itag itag-info' }, b.thinking_effort || 'off')));
      tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } }, chScore));
      tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } }, intTasks));
      tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } }, fmtTimeAgo(b.created_at)));
      tr.appendChild(el('td', null, el('div', { class: 'row-actions' },
        el('button', { class: 'btn btn-ghost btn-xs', title: '删除',
          onclick: ev => { ev.stopPropagation(); deleteBaseline(b.id); } }, mkIcon('trash', { size: 11 })),
      )));
      tb.appendChild(tr);
    });
    t.appendChild(tb);
    panel.appendChild(t);
  }
  v.appendChild(panel);
}

function openBaselineDetailModal(b) {
  const body = el('div');
  body.appendChild(el('div', { class: 'tag-row', style: { marginBottom: '12px' } },
    el('span', { class: 'itag itag-info' }, b.model),
    el('span', { class: 'itag' }, b.thinking_effort || 'off'),
    el('span', { class: 'itag' }, 'ID: ' + b.id),
  ));

  if (b.channel_report) {
    const s = b.channel_report.score;
    body.appendChild(el('div', { class: 'eyebrow', style: { marginBottom: '4px' } },
      'CHANNEL · ' + (s ? s.grade + ' · ' + Math.round(s.total_score) + ' pts' : '—')));
    body.appendChild(el('pre', { class: 'json-out', style: { maxHeight: '250px' } }, JSON.stringify(b.channel_report, null, 2)));
  }
  if (b.intelligence_report) {
    body.appendChild(el('div', { class: 'eyebrow', style: { marginTop: '8px', marginBottom: '4px' } },
      'INTELLIGENCE · ' + b.intelligence_report.task_completed + '/' + b.intelligence_report.task_total + ' tasks'));
    body.appendChild(el('pre', { class: 'json-out', style: { maxHeight: '250px' } }, JSON.stringify(b.intelligence_report, null, 2)));
  }

  openModal('基线详情 · ' + (b.name || b.id), body);
}

async function openBaselineDrawer() {
  let name = '';
  let baseURL = '';
  let apiKey = '';
  let model = 'claude-sonnet-4-6';
  let effort = 'off';
  let blMaxTokens = 0;
  let dataset = '';

  let datasets = [];
  try {
    const d = await api('/api/intelligence/datasets').catch(() => ({ datasets: [] }));
    datasets = d.datasets || [];
  } catch {}

  const body = el('div');
  body.appendChild(el('p', { class: 'muted', style: { fontSize: '12px', marginBottom: '12px' } },
    '使用官方 API Key 录制参考基准。结果将作为后续持续检测的对比基线。'));

  body.appendChild(buildField('NAME', el('input', {
    value: name, placeholder: '官方基线 · sonnet-4-6',
    oninput: e => { name = e.target.value.trim(); },
    style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
  })));
  body.appendChild(buildField('BASE URL (官方)', el('input', {
    class: 'mono', value: baseURL, placeholder: 'https://api.anthropic.com',
    oninput: e => { baseURL = e.target.value.trim(); },
    style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
  })));
  body.appendChild(buildField('API KEY (官方)', el('input', {
    type: 'password', class: 'mono', placeholder: 'sk-ant-...',
    oninput: e => { apiKey = e.target.value; },
    style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
  })));
  body.appendChild(buildField('MODEL', el('input', {
    class: 'mono', value: model, list: 'blModelList',
    oninput: e => { model = e.target.value.trim(); },
    style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
  })));
  const dl = el('datalist', { id: 'blModelList' });
  modelIDs().forEach(m => dl.appendChild(el('option', { value: m })));
  body.appendChild(dl);

  const effortSeg = el('div', { class: 'seg', style: { width: 'fit-content' } });
  ['off', 'low', 'medium', 'high', 'max', 'xhigh'].forEach(v => {
    const b = el('button', { class: effort === v ? 'active' : '',
      onclick: () => { effort = v; effortSeg.querySelectorAll('button').forEach(x => x.classList.remove('active')); b.classList.add('active'); }
    }, v);
    effortSeg.appendChild(b);
  });
  body.appendChild(buildField('EFFORT', effortSeg));
  body.appendChild(buildField('MAX TOKENS (0=default)', el('input', {
    type: 'number', min: 0, max: 128000, value: blMaxTokens, class: 'mono',
    oninput: e => { blMaxTokens = parseInt(e.target.value) || 0; },
    style: { width: '100px', background: 'transparent', border: 'none', padding: '0' }
  })));

  const dsSel = el('select', { style: { width: '100%' }, onchange: e => { dataset = e.target.value; } });
  dsSel.appendChild(el('option', { value: '' }, '(不录制智商基线)'));
  datasets.forEach(d => dsSel.appendChild(el('option', { value: d.name }, d.name + ' · ' + (d.total_tasks || 0) + ' tasks')));
  body.appendChild(buildField('INTELLIGENCE DATASET (optional)', dsSel));

  const status = el('div', { style: { marginTop: '8px', fontSize: '12px', color: 'var(--ink-3)' } });
  body.appendChild(status);

  const foot = el('div', null,
    btn('取消', { ghost: true, onClick: closeDrawer }),
    btn('开始录制', { primary: true, icon: 'play', onClick: async () => {
      if (!baseURL) { toast('请填写 Base URL', 'bad'); return; }
      if (!apiKey) { toast('请填写 API Key', 'bad'); return; }
      status.textContent = '正在录制基线…（可能需要数分钟）';
      try {
        await api('/api/monitor/baselines', {
          method: 'POST',
          body: JSON.stringify({
            name, base_url: baseURL, api_key: apiKey, model,
            thinking_effort: effort,
            effort: effort !== 'off' ? effort : undefined,
            max_tokens: blMaxTokens || undefined,
            dataset: dataset || undefined,
          }),
        });
        toast('基线录制完成', 'good');
        closeDrawer();
        renderMonitorBaselines();
      } catch (e) {
        status.textContent = '录制失败: ' + e.message;
        toast('录制失败', 'bad');
      }
    }}),
  );

  openDrawer('录制基线', body, foot);
}

async function deleteBaseline(id) {
  if (!confirm('确定删除此基线？')) return;
  try {
    await api('/api/monitor/baselines/' + id, { method: 'DELETE' });
    toast('已删除', 'good');
    renderMonitorBaselines();
  } catch (e) {
    toast('删除失败: ' + e.message, 'bad');
  }
}
