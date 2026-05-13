/* ─── history.js · History list (channel + bench) + compare ──────────── */

let _historyState = {
  channel: { items: [], filtered: [], selected: new Set(),
    filter: { q: '', channel: '', model: '', status: '' } },
  bench:   { items: [], filtered: [], selected: new Set(),
    filter: { q: '', dataset: '', model: '', status: '' } },
};

async function renderChannelHistory() {
  setCrumb([{ label: 'Channel', href: '#/channel' }, { cur: '历史' }],
    el('div', { class: 'crumb-actions' },
      btn('新建检测', { primary: true, icon: 'play', size: 'sm', onClick: () => location.hash = '#/channel' }),
    ));
  const v = $('#view');
  v.innerHTML = '<div class="empty">加载中…</div>';
  try {
    const data = await api('/api/channel/history');
    _historyState.channel.items = data.history || [];
    paintChannelHistory();
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty', style: { color: 'var(--bad-ink)' } },
      el('div', { class: 'glyph' }, '×'), '加载失败: ' + esc(e.message)));
  }
}

function paintChannelHistory() {
  const H = _historyState.channel;
  const f = H.filter;
  H.filtered = H.items.filter(r => {
    if (f.q) {
      const s = (r.channel_name + ' ' + r.target + ' ' + r.model).toLowerCase();
      if (!s.includes(f.q.toLowerCase())) return false;
    }
    if (f.channel && (r.channel_name || '') !== f.channel) return false;
    if (f.model   && (r.model || '')        !== f.model) return false;
    if (f.status) {
      const s = (r.score && r.score.grade_color) || '';
      if (f.status === 'pass' && !['green', 'blue'].includes(s)) return false;
      if (f.status === 'warn' && !['yellow', 'orange'].includes(s)) return false;
      if (f.status === 'fail' && s !== 'red') return false;
    }
    return true;
  });

  const v = $('#view'); v.innerHTML = '';
  const panel = el('div', { class: 'panel' });
  panel.appendChild(buildHistoryFilters(H, paintChannelHistory, 'channel'));
  panel.appendChild(buildChannelHistoryTable(H));
  v.appendChild(panel);
}

function buildHistoryFilters(H, repaint, kind) {
  const bar = el('div', { class: 'filterbar' });
  const f = H.filter;

  const q = el('input', { placeholder: 'search…', value: f.q,
    oninput: e => { f.q = e.target.value; repaint(); },
    style: { width: '160px' } });
  q.appendChild = null;
  bar.appendChild(el('span', { class: 'lbl' }, 'q'));
  bar.appendChild(q);

  // channel/dataset selector
  const items = H.items;
  if (kind === 'channel') {
    const channels = Array.from(new Set(items.map(i => i.channel_name).filter(Boolean))).sort();
    bar.appendChild(el('span', { class: 'lbl' }, 'channel'));
    bar.appendChild(selectFilter(f, 'channel', channels, repaint));
  } else {
    const datasets = Array.from(new Set(items.map(i => i.dataset_name).filter(Boolean))).sort();
    bar.appendChild(el('span', { class: 'lbl' }, 'dataset'));
    bar.appendChild(selectFilter(f, 'dataset', datasets, repaint));
  }

  // model
  const models = Array.from(new Set(items.map(i => i.model).filter(Boolean))).sort();
  bar.appendChild(el('span', { class: 'lbl' }, 'model'));
  bar.appendChild(selectFilter(f, 'model', models, repaint));

  // status
  bar.appendChild(el('span', { class: 'lbl' }, 'status'));
  if (kind === 'channel') {
    bar.appendChild(selectFilter(f, 'status', ['pass', 'warn', 'fail'], repaint));
  } else {
    bar.appendChild(selectFilter(f, 'status', ['ok', 'has_errors'], repaint));
  }

  bar.appendChild(el('span', { class: 'spacer' }));
  bar.appendChild(el('span', { class: 'muted', style: { fontSize: '11px', fontFamily: 'var(--font-mono)' } },
    H.filtered.length + ' / ' + H.items.length + ' 条'));

  if (H.selected.size >= 2 && kind === 'channel') {
    bar.appendChild(btn('对比 (' + H.selected.size + ')', {
      size: 'sm', primary: true, icon: 'compare',
      onClick: () => openChannelCompare(Array.from(H.selected)),
    }));
  }
  if (H.selected.size > 0) {
    bar.appendChild(btn('清除', {
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
  const t = el('table', { class: 'table' });
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
    const passed = (r.checks || []).filter(c => c.pass).length;
    const total  = (r.checks || []).length;
    const score  = r.score ? Math.round(r.score.total_score) : null;
    const grade  = r.score ? r.score.grade : '·';
    const gc     = r.score ? gradeColor(r.score.grade_color) : 'var(--ink-3)';
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

/* ─── compare modal ─── */
async function openChannelCompare(ids) {
  const items = await Promise.all(ids.map(async id => {
    try { return await api('/api/channel/history/' + encodeURIComponent(id)); }
    catch { return null; }
  }));
  const reports = items.filter(Boolean);

  // collect all check names
  const checkNames = new Set();
  reports.forEach(r => (r.checks || []).forEach(c => checkNames.add(c.name)));
  const allNames = Array.from(checkNames).sort();

  const body = el('div');

  // header row of report identifiers
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

  // diff toggle
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
      rowsWrap.appendChild(el('div', { class: 'empty' }, '所有 check 在这些渠道上结果一致 ✓'));
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

/* ═══════════════════════════════════════════════════════════════════════
 * Benchmark history list
 * ═══════════════════════════════════════════════════════════════════════ */

async function renderBenchHistory() {
  setCrumb([{ label: 'Benchmark', href: '#/bench' }, { cur: '历史' }],
    el('div', { class: 'crumb-actions' },
      btn('新建运行', { primary: true, icon: 'play', size: 'sm', onClick: () => location.hash = '#/bench' }),
    ));
  const v = $('#view');
  v.innerHTML = '<div class="empty">加载中…</div>';
  try {
    const data = await api('/api/intelligence/history');
    _historyState.bench.items = data.history || [];
    paintBenchHistory();
  } catch (e) {
    v.innerHTML = '';
    v.appendChild(el('div', { class: 'empty', style: { color: 'var(--bad-ink)' } },
      el('div', { class: 'glyph' }, '×'), '加载失败: ' + esc(e.message)));
  }
}

function paintBenchHistory() {
  const H = _historyState.bench;
  const f = H.filter;
  H.filtered = H.items.filter(r => {
    if (f.q) {
      const s = ((r.dataset_name || '') + ' ' + (r.model || '')).toLowerCase();
      if (!s.includes(f.q.toLowerCase())) return false;
    }
    if (f.dataset && (r.dataset_name || '') !== f.dataset) return false;
    if (f.model   && (r.model || '')        !== f.model) return false;
    if (f.status) {
      const ok = (r.task_errors || 0) === 0;
      if (f.status === 'ok' && !ok) return false;
      if (f.status === 'has_errors' && ok) return false;
    }
    return true;
  });

  const v = $('#view'); v.innerHTML = '';
  const panel = el('div', { class: 'panel' });
  panel.appendChild(buildHistoryFilters(H, paintBenchHistory, 'bench'));
  panel.appendChild(buildBenchHistoryTable(H));
  v.appendChild(panel);
}

function buildBenchHistoryTable(H) {
  if (H.filtered.length === 0) {
    return el('div', { class: 'empty' }, el('div', { class: 'glyph' }, '∅'), H.items.length === 0 ? '尚无历史记录' : '没有匹配的记录');
  }
  const t = el('table', { class: 'table' });
  t.appendChild(el('thead', null, el('tr', null,
    el('th', null, 'dataset'),
    el('th', null, 'model'),
    el('th', { style: { width: '90px', textAlign: 'right' } }, 'tasks'),
    el('th', { style: { width: '70px', textAlign: 'right' } }, 'errors'),
    el('th', { style: { width: '80px', textAlign: 'right' } }, 'elapsed'),
    el('th', { style: { width: '60px' } }, 'thinking'),
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
    tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right' } },
      (r.task_completed || 0) + ' / ' + (r.task_total || 0)));
    tr.appendChild(el('td', { class: 'mono tnum', style: { textAlign: 'right', color: r.task_errors > 0 ? 'var(--bad-ink)' : 'var(--ink-3)' } }, r.task_errors || 0));
    tr.appendChild(el('td', { class: 'mono', style: { textAlign: 'right' } }, fmtMs(r.elapsed_ms)));
    tr.appendChild(el('td', null, r.thinking ? el('span', { class: 'itag itag-info' }, 'on') : el('span', { class: 'itag' }, 'off')));
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

async function deleteBenchHistoryConfirm(r) {
  if (!confirm('确定删除此条历史记录?')) return;
  try {
    await api('/api/intelligence/history/' + encodeURIComponent(r.id), { method: 'DELETE' });
    toast('已删除', 'good');
    renderBenchHistory();
  } catch (e) { toast('删除失败: ' + e.message, 'bad'); }
}
