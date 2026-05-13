/* ─── history.js · History list (channel + bench) + compare ──────────── */

const HISTORY_PAGE_SIZE = 40;

let _historyState = {
  channel: { items: [], total: 0, filtered: [], selected: new Set(),
    filter: { q: '', channel: '', model: '', status: '' },
    loaded: 0, loading: false, hasMore: true },
  bench:   { items: [], total: 0, filtered: [], selected: new Set(),
    filter: { q: '', dataset: '', model: '', effort: '', status: '' },
    loaded: 0, loading: false, hasMore: true },
};

/* ═══════════════════════════════════════════════════════════════════════
 * Channel history
 * ═══════════════════════════════════════════════════════════════════════ */

async function renderChannelHistory() {
  setCrumb([{ label: 'Channel', href: '#/channel' }, { cur: '历史' }],
    el('div', { class: 'crumb-actions' },
      btn('新建检测', { primary: true, icon: 'play', size: 'sm', onClick: () => location.hash = '#/channel' }),
    ));
  const H = _historyState.channel;
  H.items = []; H.loaded = 0; H.hasMore = true; H.total = 0;
  const v = $('#view');
  v.innerHTML = '<div class="empty">加载中…</div>';
  try {
    await loadChannelPage(H);
    paintChannelHistory();
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty', style: { color: 'var(--bad-ink)' } },
      el('div', { class: 'glyph' }, '×'), '加载失败: ' + esc(e.message)));
  }
}

async function loadChannelPage(H) {
  if (H.loading || !H.hasMore) return;
  H.loading = true;
  try {
    const data = await api('/api/channel/history?limit=' + HISTORY_PAGE_SIZE + '&offset=' + H.loaded);
    const page = data.history || [];
    H.total = data.total || 0;
    H.items = H.items.concat(page);
    H.loaded += page.length;
    H.hasMore = H.loaded < H.total;
  } finally {
    H.loading = false;
  }
}

function applyChannelFilter(H) {
  const f = H.filter;
  H.filtered = H.items.filter(r => {
    if (f.q) {
      const s = (r.channel_name + ' ' + r.target + ' ' + r.model).toLowerCase();
      if (!s.includes(f.q.toLowerCase())) return false;
    }
    if (f.channel && (r.channel_name || '') !== f.channel) return false;
    if (f.model   && (r.model || '')        !== f.model) return false;
    if (f.status) {
      const s = r.grade_color || '';
      if (f.status === 'pass' && !['green', 'blue'].includes(s)) return false;
      if (f.status === 'warn' && !['yellow', 'orange'].includes(s)) return false;
      if (f.status === 'fail' && s !== 'red') return false;
    }
    return true;
  });
}

function paintChannelHistory() {
  const H = _historyState.channel;
  applyChannelFilter(H);

  const v = $('#view'); v.innerHTML = '';

  // sticky filterbar — outside panel
  v.appendChild(buildHistoryFilters(H, paintChannelHistory, 'channel'));

  // table
  const wrap = el('div', { class: 'hist-table-wrap' });
  wrap.appendChild(buildChannelHistoryTable(H));

  // load more / infinite scroll sentinel
  if (H.hasMore) {
    const sentinel = el('div', { class: 'hist-load-more' });
    const loadBtn = btn('加载更多 (' + H.loaded + '/' + H.total + ')', {
      ghost: true, size: 'sm',
      onClick: async () => {
        await loadChannelPage(H);
        paintChannelHistory();
      }
    });
    sentinel.appendChild(loadBtn);
    wrap.appendChild(sentinel);
    observeLoadMore(sentinel, async () => {
      await loadChannelPage(H);
      paintChannelHistory();
    });
  } else if (H.total > 0) {
    wrap.appendChild(el('div', { class: 'hist-end' }, '已加载全部 ' + H.total + ' 条'));
  }

  v.appendChild(wrap);
}

/* ═══════════════════════════════════════════════════════════════════════
 * Bench history
 * ═══════════════════════════════════════════════════════════════════════ */

async function renderBenchHistory() {
  setCrumb([{ label: 'Benchmark', href: '#/bench' }, { cur: '历史' }],
    el('div', { class: 'crumb-actions' },
      btn('新建运行', { primary: true, icon: 'play', size: 'sm', onClick: () => location.hash = '#/bench' }),
    ));
  const H = _historyState.bench;
  H.items = []; H.loaded = 0; H.hasMore = true; H.total = 0;
  const v = $('#view');
  v.innerHTML = '<div class="empty">加载中…</div>';
  try {
    await loadBenchPage(H);
    paintBenchHistory();
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty', style: { color: 'var(--bad-ink)' } },
      el('div', { class: 'glyph' }, '×'), '加载失败: ' + esc(e.message)));
  }
}

async function loadBenchPage(H) {
  if (H.loading || !H.hasMore) return;
  H.loading = true;
  try {
    const data = await api('/api/intelligence/history?limit=' + HISTORY_PAGE_SIZE + '&offset=' + H.loaded);
    const page = data.history || [];
    H.total = data.total || 0;
    H.items = H.items.concat(page);
    H.loaded += page.length;
    H.hasMore = H.loaded < H.total;
  } finally {
    H.loading = false;
  }
}

function applyBenchFilter(H) {
  const f = H.filter;
  H.filtered = H.items.filter(r => {
    if (f.q) {
      const s = ((r.dataset_name || '') + ' ' + (r.model || '') + ' ' + (r.effort || '')).toLowerCase();
      if (!s.includes(f.q.toLowerCase())) return false;
    }
    if (f.dataset && (r.dataset_name || '') !== f.dataset) return false;
    if (f.model   && (r.model || '')        !== f.model) return false;
    if (f.effort  && (r.effort || '')       !== f.effort) return false;
    if (f.status) {
      const ok = (r.task_errors || 0) === 0;
      if (f.status === 'ok' && !ok) return false;
      if (f.status === 'has_errors' && ok) return false;
    }
    return true;
  });
}

function paintBenchHistory() {
  const H = _historyState.bench;
  applyBenchFilter(H);

  const v = $('#view'); v.innerHTML = '';
  v.appendChild(buildHistoryFilters(H, paintBenchHistory, 'bench'));

  const wrap = el('div', { class: 'hist-table-wrap' });
  wrap.appendChild(buildBenchHistoryTable(H));

  if (H.hasMore) {
    const sentinel = el('div', { class: 'hist-load-more' });
    sentinel.appendChild(btn('加载更多 (' + H.loaded + '/' + H.total + ')', {
      ghost: true, size: 'sm',
      onClick: async () => { await loadBenchPage(H); paintBenchHistory(); }
    }));
    wrap.appendChild(sentinel);
    observeLoadMore(sentinel, async () => { await loadBenchPage(H); paintBenchHistory(); });
  } else if (H.total > 0) {
    wrap.appendChild(el('div', { class: 'hist-end' }, '已加载全部 ' + H.total + ' 条'));
  }

  v.appendChild(wrap);
}

/* ═══════════════════════════════════════════════════════════════════════
 * Shared: filters, tables, infinite scroll
 * ═══════════════════════════════════════════════════════════════════════ */

function buildHistoryFilters(H, repaint, kind) {
  const bar = el('div', { class: 'hist-filterbar' });
  const f = H.filter;

  const q = el('input', { placeholder: 'search…', value: f.q,
    oninput: e => { f.q = e.target.value; repaint(); },
    style: { width: '160px' } });
  bar.appendChild(el('span', { class: 'lbl' }, 'Q'));
  bar.appendChild(q);

  const items = H.items;
  if (kind === 'channel') {
    const channels = Array.from(new Set(items.map(i => i.channel_name).filter(Boolean))).sort();
    bar.appendChild(el('span', { class: 'lbl' }, 'CHANNEL'));
    bar.appendChild(selectFilter(f, 'channel', channels, repaint));
  } else {
    const datasets = Array.from(new Set(items.map(i => i.dataset_name).filter(Boolean))).sort();
    bar.appendChild(el('span', { class: 'lbl' }, 'DATASET'));
    bar.appendChild(selectFilter(f, 'dataset', datasets, repaint));
  }

  const models = Array.from(new Set(items.map(i => i.model).filter(Boolean))).sort();
  bar.appendChild(el('span', { class: 'lbl' }, 'MODEL'));
  bar.appendChild(selectFilter(f, 'model', models, repaint));

  if (kind === 'bench') {
    const efforts = Array.from(new Set(items.map(i => i.effort).filter(Boolean))).sort();
    if (efforts.length > 0) {
      bar.appendChild(el('span', { class: 'lbl' }, 'EFFORT'));
      bar.appendChild(selectFilter(f, 'effort', efforts, repaint));
    }
  }

  bar.appendChild(el('span', { class: 'lbl' }, 'STATUS'));
  if (kind === 'channel') {
    bar.appendChild(selectFilter(f, 'status', ['pass', 'warn', 'fail'], repaint));
  } else {
    bar.appendChild(selectFilter(f, 'status', ['ok', 'has_errors'], repaint));
  }

  bar.appendChild(el('span', { class: 'spacer' }));
  bar.appendChild(el('span', { class: 'hist-count' },
    H.filtered.length + ' / ' + H.total + ' 条'));

  if (H.selected.size >= 2 && kind === 'channel') {
    bar.appendChild(btn('对比 (' + H.selected.size + ')', {
      size: 'sm', primary: true, icon: 'compare',
      onClick: () => openChannelCompare(Array.from(H.selected)),
    }));
  }
  if (H.selected.size > 0) {
    bar.appendChild(btn('删除所选 (' + H.selected.size + ')', {
      size: 'sm', danger: true, icon: 'trash',
      onClick: async () => {
        if (!confirm('确定删除所选的 ' + H.selected.size + ' 条记录？')) return;
        const ids = Array.from(H.selected);
        const endpoint = kind === 'channel' ? '/api/channel/history/' : '/api/intelligence/history/';
        let ok = 0;
        for (const id of ids) {
          try { await api(endpoint + encodeURIComponent(id), { method: 'DELETE' }); ok++; } catch {}
        }
        toast('已删除 ' + ok + ' 条', 'good');
        H.selected.clear();
        if (kind === 'channel') renderChannelHistory(); else renderBenchHistory();
      }
    }));
    bar.appendChild(btn('取消选择', {
      size: 'sm', ghost: true, onClick: () => { H.selected.clear(); repaint(); }
    }));
  }
  return bar;
}

function selectFilter(filter, key, options, repaint) {
  const s = el('select', { onchange: e => { filter[key] = e.target.value; repaint(); }, style: { minWidth: '120px' } });
  s.appendChild(el('option', { value: '' }, 'all'));
  options.forEach(o => {
    const opt = el('option', { value: o }, o);
    if (filter[key] === o) opt.selected = true;
    s.appendChild(opt);
  });
  return s;
}

function buildChannelHistoryTable(H) {
  if (H.filtered.length === 0) {
    return el('div', { class: 'empty' },
      el('div', { class: 'glyph' }, '∅'),
      H.items.length === 0 ? '尚无历史记录' : '没有匹配的记录');
  }
  const t = el('table', { class: 'table hist-table' });
  t.appendChild(el('thead', null, el('tr', null,
    el('th', { style: { width: '32px' } }, ''),
    el('th', { style: { width: '40px' } }, 'grade'),
    el('th', null, 'channel'),
    el('th', null, 'model'),
    el('th', { style: { width: '90px', textAlign: 'right' } }, 'score'),
    el('th', { style: { width: '70px', textAlign: 'right' } }, 'pass/total'),
    el('th', { style: { width: '70px', textAlign: 'right' } }, 'elapsed'),
    el('th', { style: { width: '110px' } }, 'when'),
    el('th', { style: { width: '90px' } }, ''),
  )));
  const tb = el('tbody');
  H.filtered.forEach(r => {
    const isSelected = H.selected.has(r.id);
    const passed = r.checks_passed || 0;
    const total  = r.checks_total || 0;
    const score  = r.total_score != null ? Math.round(r.total_score) : null;
    const grade  = r.grade || '·';
    const gc     = r.grade_color ? gradeColor(r.grade_color) : 'var(--ink-3)';
    const tr = el('tr', { class: isSelected ? 'selected' : null,
      onclick: ev => {
        if (ev.target.closest('.row-actions, input[type=checkbox]')) return;
        location.hash = '#/channel/run/' + encodeURIComponent(r.id);
      } });
    tr.appendChild(el('td', null,
      el('input', { type: 'checkbox',
        checked: isSelected ? '' : null,
        onclick: ev => { ev.stopPropagation();
          if (ev.target.checked) H.selected.add(r.id); else H.selected.delete(r.id);
          paintChannelHistory();
        },
        style: { width: '14px', height: '14px' } })));
    tr.appendChild(el('td', { class: 'grade-cell', style: { color: gc } }, grade));
    tr.appendChild(el('td', { class: 'name-cell' },
      el('div', null, r.channel_name || '—'),
      el('div', { class: 'mono', style: { fontSize: '10px', color: 'var(--ink-4)' } }, (r.target || '').replace(/^https?:\/\//, '')),
    ));
    tr.appendChild(el('td', { class: 'mono' }, r.model || '—'));
    tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right', color: gc, fontWeight: 600 } }, score != null ? score : '—'));
    tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } }, passed + '/' + total));
    tr.appendChild(el('td', { class: 'mono', style: { textAlign: 'right' } }, fmtMs(r.elapsed_ms)));
    tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } },
      fmtTimeAgo(r.timestamp), el('div', { style: { fontSize: '10px', color: 'var(--ink-4)' } },
        fmtTime(r.timestamp))));

    const actions = el('div', { class: 'row-actions' },
      el('button', { class: 'btn btn-ghost btn-xs', title: '重命名',
        onclick: ev => { ev.stopPropagation(); renameChannelHistoryDialog(r); } },
        mkIcon('edit', { size: 11 })),
      el('button', { class: 'btn btn-ghost btn-xs', title: '删除',
        onclick: ev => { ev.stopPropagation(); deleteChannelHistoryConfirm(r); } },
        mkIcon('trash', { size: 11 })),
    );
    tr.appendChild(el('td', null, actions));
    tb.appendChild(tr);
  });
  t.appendChild(tb);
  return t;
}

function buildBenchHistoryTable(H) {
  if (H.filtered.length === 0) {
    return el('div', { class: 'empty' }, el('div', { class: 'glyph' }, '∅'), H.items.length === 0 ? '尚无历史记录' : '没有匹配的记录');
  }
  const t = el('table', { class: 'table hist-table' });
  t.appendChild(el('thead', null, el('tr', null,
    el('th', null, 'dataset'),
    el('th', null, 'model'),
    el('th', { style: { width: '60px' } }, 'effort'),
    el('th', { style: { width: '65px' } }, 'thinking'),
    el('th', { style: { width: '80px', textAlign: 'right' } }, 'score'),
    el('th', { style: { width: '80px', textAlign: 'right' } }, 'pass rate'),
    el('th', { style: { width: '90px', textAlign: 'right' } }, 'tasks'),
    el('th', { style: { width: '70px', textAlign: 'right' } }, 'errors'),
    el('th', { style: { width: '80px', textAlign: 'right' } }, 'elapsed'),
    el('th', { style: { width: '120px' } }, 'when'),
    el('th', { style: { width: '60px' } }, ''),
  )));
  const tb = el('tbody');
  H.filtered.forEach(r => {
    const tr = el('tr', { onclick: ev => {
      if (ev.target.closest('.row-actions')) return;
      location.hash = '#/bench/run/' + encodeURIComponent(r.id);
    } });
    tr.appendChild(el('td', { class: 'name-cell' }, r.dataset_name || '—'));
    tr.appendChild(el('td', { class: 'mono' }, r.model || '—'));
    tr.appendChild(el('td', null,
      r.effort ? el('span', { class: 'itag itag-warn' }, r.effort) : el('span', { class: 'itag' }, 'default')));
    tr.appendChild(el('td', null,
      r.thinking_mode ? el('span', { class: 'itag itag-info' }, r.thinking_mode)
        : r.thinking ? el('span', { class: 'itag itag-info' }, 'on')
        : el('span', { class: 'itag' }, 'off')));
    const scoreVal = r.score_total != null ? Math.round(r.score_total * 100) / 100 : null;
    const passVal = r.pass_rate != null ? Math.round(r.pass_rate * 1000) / 10 : null;
    tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right', fontWeight: 600 } },
      scoreVal != null ? String(scoreVal) : '—'));
    tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } },
      passVal != null ? passVal + '%' : '—'));
    tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } },
      (r.task_completed || 0) + ' / ' + (r.task_total || 0)));
    tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right', color: r.task_errors > 0 ? 'var(--bad-ink)' : 'var(--ink-3)' } }, r.task_errors || 0));
    tr.appendChild(el('td', { class: 'mono', style: { textAlign: 'right' } }, fmtMs(r.elapsed_ms)));
    tr.appendChild(el('td', { class: 'mono', style: { fontSize: '11px' } }, fmtTimeAgo(r.started_at),
      el('div', { style: { fontSize: '10px', color: 'var(--ink-4)' } }, fmtTime(r.started_at))));
    tr.appendChild(el('td', null, el('div', { class: 'row-actions' },
      el('button', { class: 'btn btn-ghost btn-xs', title: '删除',
        onclick: ev => { ev.stopPropagation(); deleteBenchHistoryConfirm(r); } }, mkIcon('trash', { size: 11 })),
    )));
    tb.appendChild(tr);
  });
  t.appendChild(tb);
  return t;
}

/* ─── Infinite scroll observer ─── */
let _historyObserver = null;
function observeLoadMore(sentinel, loadFn) {
  if (_historyObserver) _historyObserver.disconnect();
  _historyObserver = new IntersectionObserver(entries => {
    if (entries[0].isIntersecting) {
      _historyObserver.disconnect();
      loadFn();
    }
  }, { rootMargin: '200px' });
  _historyObserver.observe(sentinel);
}

/* ═══════════════════════════════════════════════════════════════════════
 * Actions: rename, delete
 * ═══════════════════════════════════════════════════════════════════════ */

async function renameChannelHistoryDialog(r) {
  const name = prompt('渠道名称:', r.channel_name || '');
  if (name == null) return;
  try {
    await api('/api/channel/history/' + encodeURIComponent(r.id), {
      method: 'PATCH', body: JSON.stringify({ channel_name: name })
    });
    toast('已更新', 'good');
    renderChannelHistory();
  } catch (e) { toast('更新失败: ' + e.message, 'bad'); }
}

async function deleteChannelHistoryConfirm(r) {
  if (!confirm('确定删除此条历史记录?')) return;
  try {
    await api('/api/channel/history/' + encodeURIComponent(r.id), { method: 'DELETE' });
    toast('已删除', 'good');
    renderChannelHistory();
  } catch (e) { toast('删除失败: ' + e.message, 'bad'); }
}

async function deleteBenchHistoryConfirm(r) {
  if (!confirm('确定删除此条历史记录?')) return;
  try {
    await api('/api/intelligence/history/' + encodeURIComponent(r.id), { method: 'DELETE' });
    toast('已删除', 'good');
    renderBenchHistory();
  } catch (e) { toast('删除失败: ' + e.message, 'bad'); }
}

/* ═══════════════════════════════════════════════════════════════════════
 * Compare modal
 * ═══════════════════════════════════════════════════════════════════════ */

async function openChannelCompare(ids) {
  const items = await Promise.all(ids.map(async id => {
    try { return await api('/api/channel/history/' + encodeURIComponent(id)); }
    catch { return null; }
  }));
  const reports = items.filter(Boolean);

  const checkNames = new Set();
  reports.forEach(r => (r.checks || []).forEach(c => checkNames.add(c.name)));
  const allNames = Array.from(checkNames).sort();

  const body = el('div');

  const head = el('div', { style: { display: 'grid',
    gridTemplateColumns: '240px repeat(' + reports.length + ', 1fr)',
    gap: '1px', background: 'var(--line)', marginBottom: '8px',
    border: '1px solid var(--line)', borderRadius: 'var(--r-md)', overflow: 'hidden' } });
  head.appendChild(el('div', { style: { background: 'var(--panel-2)', padding: '10px 14px',
    fontSize: '11px', fontFamily: 'var(--font-mono)', color: 'var(--ink-4)',
    letterSpacing: '.1em', textTransform: 'uppercase' } }, 'CHECK'));
  reports.forEach(r => {
    head.appendChild(el('div', { style: { background: 'var(--panel)', padding: '10px 14px' } },
      el('div', { class: 'mono', style: { fontSize: '12px', fontWeight: 600 } }, r.channel_name || r.target || '-'),
      el('div', { class: 'mono', style: { fontSize: '10px', color: 'var(--ink-4)' } }, (r.model || '') + ' · ' + fmtTime(r.timestamp)),
      el('div', { style: { marginTop: '6px' } },
        el('span', { class: 'pill ' + (VERDICT_PILL[(r.score && r.score.verdict_color) || ''] || 'pill-info'),
          style: { fontSize: '10px' } },
          el('span', { class: 'led' }),
          (r.score && r.score.verdict_label) || '-'),
        el('span', { class: 'mono', style: { fontSize: '11px', color: 'var(--ink-3)', marginLeft: '6px' } },
          r.score ? Math.round(r.score.total_score) + ' pts' : ''),
      )));
  });
  body.appendChild(head);

  let onlyDiff = true;
  const repaint = () => {
    rowsWrap.innerHTML = '';
    const rows = onlyDiff
      ? allNames.filter(n => {
          const statuses = reports.map(r => (r.checks || []).find(c => c.name === n));
          const passSet = new Set(statuses.map(s => s ? s.pass : null));
          return passSet.size > 1;
        })
      : allNames;
    if (rows.length === 0) {
      rowsWrap.appendChild(el('div', { class: 'empty' }, '所有 check 在这些渠道上结果一致'));
      return;
    }
    rows.forEach(name => {
      const row = el('div', { style: { display: 'grid',
        gridTemplateColumns: '240px repeat(' + reports.length + ', 1fr)',
        gap: '1px', background: 'var(--line-soft)', marginBottom: '1px' } });
      const cat = catOf(name);
      row.appendChild(el('div', {
        style: { background: 'var(--panel)', padding: '8px 14px',
          fontFamily: 'var(--font-mono)', fontSize: '12px', color: 'var(--ink-2)',
          display: 'flex', alignItems: 'center', gap: '8px' } },
        el('span', { class: 'swatch', style: { width: '7px', height: '7px', background: CAT_COLOR[cat], borderRadius: '2px' } }),
        el('span', null, name),
      ));
      reports.forEach(r => {
        const c = (r.checks || []).find(c => c.name === name);
        if (!c) {
          row.appendChild(el('div', { style: { background: 'var(--panel)', padding: '8px 14px', color: 'var(--ink-5)', fontSize: '11px', fontFamily: 'var(--font-mono)' } }, '—'));
        } else {
          row.appendChild(el('div', { style: { background: 'var(--panel)', padding: '8px 14px',
            display: 'flex', alignItems: 'center', gap: '6px' } },
            el('span', { class: 'led-dot ' + (c.pass ? 'pass' : 'fail') }),
            el('span', { class: 'mono', style: { fontSize: '11px', color: c.pass ? 'var(--good-ink)' : 'var(--bad-ink)' } },
              c.pass ? 'PASS' : 'FAIL'),
            c.actual ? el('span', { class: 'mono', style: { fontSize: '10px', color: 'var(--ink-3)', marginLeft: '6px',
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '160px' } }, c.actual) : null,
          ));
        }
      });
      rowsWrap.appendChild(row);
    });
  };

  const toggle = el('div', { style: { display: 'flex', gap: '8px', alignItems: 'center', marginBottom: '8px' } },
    el('span', { class: 'eyebrow' }, 'show'),
    (() => {
      const seg = el('div', { class: 'seg' });
      const b1 = el('button', { class: 'active', onclick: () => { onlyDiff = true; b1.classList.add('active'); b2.classList.remove('active'); repaint(); } }, '仅差异');
      const b2 = el('button', { onclick: () => { onlyDiff = false; b2.classList.add('active'); b1.classList.remove('active'); repaint(); } }, '全部');
      seg.appendChild(b1); seg.appendChild(b2);
      return seg;
    })(),
  );
  body.appendChild(toggle);

  const rowsWrap = el('div', { style: { maxHeight: '60vh', overflow: 'auto', border: '1px solid var(--line)', borderRadius: 'var(--r-md)' } });
  body.appendChild(rowsWrap);
  repaint();

  openModal('对比 ' + reports.length + ' 条记录', body);
}
