/* ─── app.js · router & init ──────────────────────────────────────────── */

/* parse #/path/segments into { app, kind, id, model, anchor } */
function parseRoute(hash) {
  hash = (hash || '').replace(/^#/, '').replace(/^\/+/, '');
  const seg = hash.split('/').filter(Boolean);
  if (seg.length === 0) return { app: 'channel', kind: 'config' };
  const app = seg[0];

  if (app === 'settings') return { app: 'settings' };

  if (app === 'channel') {
    if (seg[1] === 'history') return { app, kind: 'history' };
    if (seg[1] === 'baselines') return { app, kind: 'baselines' };
    if (seg[1] === 'run' && seg[2]) {
      const out = { app, kind: 'run', id: decodeURIComponent(seg[2]) };
      if (seg[3] === 'm' && seg[4]) out.model = decodeURIComponent(seg[4]);
      if (seg[5] === 'check' && seg[6]) out.anchor = decodeURIComponent(seg[6]);
      return out;
    }
    return { app, kind: 'config' };
  }

  if (app === 'bench') {
    if (seg[1] === 'history') return { app, kind: 'history' };
    if (seg[1] === 'baselines') return { app, kind: 'baselines' };
    if (seg[1] === 'run' && seg[2]) return { app, kind: 'run', id: decodeURIComponent(seg[2]) };
    return { app, kind: 'config' };
  }

  if (app === 'monitor') {
    if (seg[1] === 'alerts') return { app, kind: 'alerts' };
    if (seg[1] === 'baselines') return { app, kind: 'baselines' };
    if (seg[1] === 'target' && seg[2]) return { app, kind: 'target', id: decodeURIComponent(seg[2]) };
    return { app, kind: 'dashboard' };
  }

  return { app: 'channel', kind: 'config' };
}

/* ─── Top-tab + rail painting (per current app) ─── */
function paintTopTabs(app) {
  $$('.app-tabs button').forEach(b => {
    b.classList.toggle('active', b.dataset.tab === app);
  });
}

function paintRail(route) {
  const rail = $('#rail');
  rail.innerHTML = '';

  if (route.app === 'channel') {
    rail.appendChild(railSection('CHANNEL', [
      railLink('新建检测', '#/channel', route.kind === 'config', 'play'),
      railLink('历史记录', '#/channel/history', route.kind === 'history', 'history'),
      railLink('基线管理', '#/channel/baselines', route.kind === 'baselines', 'compare'),
    ]));
    // active live runs
    const liveChannelIds = Object.entries(State.liveRuns)
      .filter(([id, r]) => r.kind === 'channel' && r.state === 'running')
      .map(([id]) => id);
    if (liveChannelIds.length) {
      const sec = el('div', { class: 'rail-section' });
      sec.appendChild(el('div', { class: 'rail-label' }, 'RUNNING'));
      liveChannelIds.forEach(id => {
        const r = State.liveRuns[id];
        sec.appendChild(el('div', { class: 'rail-active-run', onclick: () => location.hash = '#/channel/run/' + id },
          el('div', { class: 'label' }, el('span', { class: 'pulse' }), 'live'),
          el('div', { class: 'name' }, r.channelName || (r.target || '').replace(/^https?:\/\//, '').split('/')[0] || id),
          el('div', { style: { fontFamily: 'var(--font-mono)', fontSize: '10px', color: 'var(--ink-3)', marginTop: '2px' } },
            (r.models || []).length + ' models · ' + fmtMs(Date.now() - r.startedAt)),
        ));
      });
      rail.appendChild(sec);
    }
  } else if (route.app === 'bench') {
    rail.appendChild(railSection('BENCHMARK', [
      railLink('新建运行', '#/bench', route.kind === 'config', 'play'),
      railLink('历史记录', '#/bench/history', route.kind === 'history', 'history'),
      railLink('基线管理', '#/bench/baselines', route.kind === 'baselines', 'compare'),
    ]));
    const liveBenchIds = Object.entries(State.liveRuns)
      .filter(([id, r]) => (r.kind === 'bench' || r.kind === 'bench-batch') && r.state === 'running')
      .map(([id]) => id);
    if (liveBenchIds.length) {
      const sec = el('div', { class: 'rail-section' });
      sec.appendChild(el('div', { class: 'rail-label' }, 'RUNNING'));
      liveBenchIds.forEach(id => {
        const r = State.liveRuns[id];
        sec.appendChild(el('div', { class: 'rail-active-run', onclick: () => location.hash = '#/bench/run/' + id },
          el('div', { class: 'label' }, el('span', { class: 'pulse' }), 'live'),
          el('div', { class: 'name' }, r.dataset || id),
          el('div', { style: { fontFamily: 'var(--font-mono)', fontSize: '10px', color: 'var(--ink-3)', marginTop: '2px' } },
            r.completedTasks + '/' + r.totalTasks + ' · ' + fmtMs(Date.now() - r.startedAt)),
        ));
      });
      rail.appendChild(sec);
    }
    // datasets quick links
    if (State.datasets.length > 0) {
      rail.appendChild(railSection('DATASETS',
        State.datasets.map(d => railLink(d.name, '#/bench',
          d.name === State.currentDataset && route.kind === 'config',
          null,
          () => { State.currentDataset = d.name; renderBenchConfig(); },
          d.total_tasks)),
      ));
    }
  } else if (route.app === 'monitor') {
    rail.appendChild(railSection('MONITOR', [
      railLink('监控面板', '#/monitor', route.kind === 'dashboard', 'activity'),
      railLink('基线管理', '#/monitor/baselines', route.kind === 'baselines', 'compare'),
      railLink('告警事件', '#/monitor/alerts', route.kind === 'alerts', 'bell'),
    ]));
    if (State.monitor.targets.length > 0) {
      const sec = el('div', { class: 'rail-section' });
      sec.appendChild(el('div', { class: 'rail-label' }, '渠道'));
      State.monitor.targets.forEach(t => {
        const ws = targetWorstStatus(t, State.monitor.states);
        const ledClass = STATUS_LED[ws] || 'pending';
        const isActive = route.kind === 'target' && route.id === t.id;
        const link = el('a', { class: 'rail-link' + (isActive ? ' active' : ''), href: '#/monitor/target/' + t.id });
        link.appendChild(el('span', { class: 'led-dot ' + ledClass, style: { width: '6px', height: '6px' } }));
        const nameSpan = el('span', { style: { flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } },
          t.name || t.base_url);
        link.appendChild(nameSpan);
        if (!t.enabled) link.appendChild(el('span', { class: 'itag', style: { fontSize: '8px', padding: '0 3px' } }, '停'));
        sec.appendChild(link);
      });
      rail.appendChild(sec);
    }
  } else if (route.app === 'settings') {
    rail.appendChild(railSection('SETTINGS', [
      railLink('连接 & 鉴权', '#/settings', true, null),
    ]));
  }

  // foot intentionally omitted
}

function railSection(label, links) {
  const sec = el('div', { class: 'rail-section' });
  sec.appendChild(el('div', { class: 'rail-label' }, label));
  links.forEach(l => sec.appendChild(l));
  return sec;
}
function railLink(label, href, active, icon, onclick, count) {
  const a = el('a', { class: 'rail-link' + (active ? ' active' : ''),
    href: onclick ? null : href,
    onclick: onclick ? ev => { ev.preventDefault(); onclick(); } : null,
  });
  if (icon) a.appendChild(mkIcon(icon, { size: 13 }));
  else a.appendChild(el('span', { class: 'ico', style: { width: '14px' } }));
  a.appendChild(el('span', null, label));
  if (count != null) a.appendChild(el('span', { class: 'count' }, count));
  return a;
}

/* ─── Settings view ─── */
function renderSettings() {
  setCrumb([{ cur: 'Settings' }]);
  const v = $('#view');
  v.innerHTML = '';

  const tokenInput = el('input', {
    type: 'password', class: 'mono', value: State.adminToken,
    placeholder: 'optional · 仅在调用受限接口时需要',
    oninput: e => { State.adminToken = e.target.value.trim(); saveSettings(); updateConn(); },
    style: { width: '100%', background: 'transparent', border: 'none', padding: '0' }
  });

  v.appendChild(el('div', { class: 'panel', style: { marginBottom: '12px' } },
    el('div', { class: 'panel-head' },
      el('h3', null, '鉴权'),
      el('span', { class: 'meta' }, '本地保存,不上传'),
    ),
    el('div', { class: 'panel-body' },
      el('div', { class: 'field' },
        el('div', { class: 'field-label' }, 'ADMIN TOKEN'),
        tokenInput,
        el('div', { class: 'muted', style: { fontSize: '11px', marginTop: '4px' } },
          '仅在调用 /api/audit/run、/api/intelligence/* 等受限接口时需要。留空 = 不启用鉴权。'),
      ),
    ),
  ));

  v.appendChild(el('div', { class: 'panel' },
    el('div', { class: 'panel-head' },
      el('h3', null, '关于'),
    ),
    el('div', { class: 'panel-body', style: { fontSize: '12px', color: 'var(--ink-3)', lineHeight: 1.7 } },
      el('div', null, 'LLM Probe · v3 · dark console'),
      el('div', null, 'Target Base URL / API Key 在各工作区顶部直接填写,不持久化。'),
      el('div', null, '快捷键 · ',
        el('span', { class: 'kbd' }, '/'), ' 检测 · ',
        el('span', { class: 'kbd' }, 'g h'), ' 历史 · ',
        el('span', { class: 'kbd' }, 'Esc'), ' 关闭'),
    ),
  ));
}

/* ─── Router ─── */
let _activeApp = null;
function routerOnChange() {
  const route = parseRoute(location.hash);
  if (route.app === 'settings') {
    paintTopTabs(_activeApp || 'channel');
  } else {
    paintTopTabs(route.app);
    _activeApp = route.app;
  }
  paintRail(route);

  if (route.app === 'settings') return renderSettings();
  if (route.app === 'channel') {
    if (route.kind === 'history') return renderChannelHistory();
    if (route.kind === 'baselines') return renderMonitorBaselines(route.app);
    if (route.kind === 'run')     return renderChannelRunRoute(route.id, route.model, route.anchor);
    return renderChannelConfig();
  }
  if (route.app === 'bench') {
    if (route.kind === 'history') return renderBenchHistory();
    if (route.kind === 'baselines') return renderMonitorBaselines(route.app);
    if (route.kind === 'run')     return renderBenchRunRoute(route.id);
    return renderBenchConfig();
  }
  if (route.app === 'monitor') {
    if (route.kind === 'alerts') return renderMonitorAlerts();
    if (route.kind === 'baselines') return renderMonitorBaselines(route.app);
    if (route.kind === 'target') return renderMonitorTarget(route.id);
    return renderMonitorDashboard();
  }
}

window.addEventListener('hashchange', routerOnChange);
$$('.app-tabs button').forEach(b => {
  b.addEventListener('click', () => { location.hash = '#/' + b.dataset.tab; });
});

/* ─── Keyboard shortcuts ─── */
let _kbdBuf = '';
let _kbdTimer = null;
document.addEventListener('keydown', e => {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') return;
  if (e.metaKey || e.ctrlKey || e.altKey) return;
  if (e.key === '/') {
    e.preventDefault();
    location.hash = _activeApp === 'bench' ? '#/bench' : '#/channel';
    return;
  }
  if (e.key === 'g') {
    _kbdBuf = 'g';
    clearTimeout(_kbdTimer);
    _kbdTimer = setTimeout(() => { _kbdBuf = ''; }, 700);
    return;
  }
  if (_kbdBuf === 'g' && e.key === 'h') {
    _kbdBuf = '';
    location.hash = _activeApp === 'bench' ? '#/bench/history' : '#/channel/history';
    return;
  }
});

/* ─── Init ─── */
loadSettings();
updateConn();

if (!location.hash) location.hash = '#/channel';
routerOnChange();
