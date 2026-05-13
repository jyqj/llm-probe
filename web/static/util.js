/* ─── util.js · shared primitives ─────────────────────────────────────── */

/* state singletons */
const State = {
  adminToken: '',
  // ephemeral target config — kept in memory, not localStorage
  channel: {
    targetBase: '', targetKey: '', channelName: '', concurrency: 2,
    models: ['claude-sonnet-4-6'],
  },
  bench: {
    targetBase: '', targetKey: '', model: 'claude-opus-4-6',
    concurrency: 5, thinking: false, effort: '', thinkingMode: '',
    scope: 'custom', lang: '', category: '', limit: 0, runModel: '',
    multiEffort: false, selectedEfforts: [],
  },
  // live runs (not yet persisted to history)
  liveRuns: Object.create(null),   // run_id → {state, reports, events, sse}
  datasets: [],
  currentDataset: '',
  // monitor
  monitor: { targets: [], states: [] },
};

/* ─── escape / format ─── */
function esc(s) {
  if (s == null) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}
function fmtMs(ms) {
  if (ms == null) return '-';
  if (ms < 1000) return ms + 'ms';
  return (ms / 1000).toFixed(1) + 's';
}
function fmtTimeAgo(ts) {
  if (!ts) return '-';
  const d = new Date(ts);
  if (isNaN(d)) return ts;
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff < 60) return Math.floor(diff) + 's ago';
  if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
  if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
  return Math.floor(diff / 86400) + 'd ago';
}
function fmtTime(ts) {
  if (!ts) return '-';
  const d = new Date(ts);
  if (isNaN(d)) return ts;
  return d.toLocaleString('zh-CN', { hour12: false });
}

/* ─── API ─── */
function headers(extra) {
  const h = Object.assign({ 'Content-Type': 'application/json' }, extra || {});
  if (State.adminToken) h['X-Admin-Token'] = State.adminToken;
  return h;
}
async function api(url, opt = {}) {
  const r = await fetch(url, { headers: headers(opt.headers), ...opt });
  const text = await r.text();
  let data;
  try { data = JSON.parse(text); } catch { data = text; }
  if (!r.ok) {
    const msg = (data && data.error) ? data.error : (typeof data === 'string' ? data : JSON.stringify(data));
    const err = new Error(msg);
    err.data = data;
    err.status = r.status;
    throw err;
  }
  return data;
}

/* ─── DOM helpers ─── */
function $(sel) { return document.querySelector(sel); }
function $$(sel) { return Array.from(document.querySelectorAll(sel)); }
function el(tag, attrs, ...children) {
  const e = document.createElement(tag);
  if (attrs) for (const k in attrs) {
    if (k === 'class') e.className = attrs[k];
    else if (k === 'style' && typeof attrs[k] === 'object') Object.assign(e.style, attrs[k]);
    else if (k.startsWith('on') && typeof attrs[k] === 'function') e.addEventListener(k.slice(2), attrs[k]);
    else if (attrs[k] !== false && attrs[k] != null) e.setAttribute(k, attrs[k]);
  }
  for (const c of children.flat()) {
    if (c == null || c === false) continue;
    e.appendChild(c.nodeType ? c : document.createTextNode(c));
  }
  return e;
}

/* ─── Toast ─── */
let toastSeq = 0;
function toast(msg, kind) {
  const id = 'tst-' + (++toastSeq);
  const node = el('div', { class: 'toast ' + (kind || ''), id }, msg);
  $('#toastStack').appendChild(node);
  setTimeout(() => { node.style.opacity = '0'; node.style.transform = 'translateY(8px)'; }, 2400);
  setTimeout(() => node.remove(), 2900);
}

/* ─── Drawer ─── */
function openDrawer(title, bodyNode, footNode) {
  $('#drawerTitle').textContent = title;
  const body = $('#drawerBody'); body.innerHTML = ''; body.appendChild(bodyNode);
  const foot = $('#drawerFoot'); foot.innerHTML = ''; if (footNode) foot.appendChild(footNode);
  $('#drawer').classList.add('open');
  $('#drawerBack').classList.add('open');
  $('#drawer').setAttribute('aria-hidden', 'false');
}
function closeDrawer() {
  $('#drawer').classList.remove('open');
  $('#drawerBack').classList.remove('open');
  $('#drawer').setAttribute('aria-hidden', 'true');
}

/* ─── Modal ─── */
function openModal(title, bodyNode, opts) {
  opts = opts || {};
  const back = el('div', { class: 'modal-back', onclick: ev => { if (ev.target === back) closeModal(); } });
  const modal = el('div', { class: 'modal', style: opts.width ? { width: opts.width } : null });
  const head = el('div', { class: 'modal-head' },
    el('h3', { style: { fontSize: '13px', fontWeight: 600 } }, title),
    el('div', { class: 'spacer' }),
    el('button', { class: 'iconbtn', onclick: closeModal, title: 'Esc' }, mkIcon('x'))
  );
  const body = el('div', { class: 'modal-body' }, bodyNode);
  modal.appendChild(head); modal.appendChild(body);
  back.appendChild(modal);
  $('#modalRoot').innerHTML = '';
  $('#modalRoot').appendChild(back);
}
function closeModal() { $('#modalRoot').innerHTML = ''; }
document.addEventListener('keydown', e => {
  if (e.key === 'Escape') { closeDrawer(); closeModal(); }
});

/* ─── Icons (inline svg helpers) ─── */
const _icoPaths = {
  play:    '<polygon points="6,4 20,12 6,20" fill="currentColor"/>',
  stop:    '<rect x="6" y="6" width="12" height="12" fill="currentColor"/>',
  pause:   '<rect x="6" y="5" width="4" height="14"/><rect x="14" y="5" width="4" height="14"/>',
  x:       '<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>',
  check:   '<polyline points="20 6 9 17 4 12"/>',
  caret:   '<polyline points="9 6 15 12 9 18"/>',
  copy:    '<rect x="9" y="9" width="11" height="11" rx="1"/><path d="M5 15V5a2 2 0 0 1 2-2h10"/>',
  refresh: '<polyline points="23 4 23 10 17 10"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>',
  trash:   '<polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/>',
  edit:    '<path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>',
  link:    '<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>',
  download:'<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>',
  upload:  '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>',
  history: '<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>',
  add:     '<line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>',
  search:  '<circle cx="11" cy="11" r="7"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>',
  filter:  '<polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/>',
  compare: '<path d="M9 3v18M15 3v18M3 9h6M15 9h6M3 15h6M15 15h6"/>',
  activity:'<polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>',
  bell:    '<path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/>',
  target:  '<circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="6"/><circle cx="12" cy="12" r="2"/>',
  power:   '<path d="M18.36 6.64a9 9 0 1 1-12.73 0"/><line x1="12" y1="2" x2="12" y2="12"/>',
};
function mkIcon(name, opts) {
  opts = opts || {};
  const size = opts.size || 13;
  const stroke = opts.stroke != null ? opts.stroke : 1.7;
  const tpl = document.createElement('template');
  tpl.innerHTML = `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="${stroke}" stroke-linecap="round" stroke-linejoin="round" class="ico">${_icoPaths[name] || ''}</svg>`;
  return tpl.content.firstChild;
}

/* ─── Category mapping ─── */
const CAT_COLOR = {
  fingerprint: 'var(--cat-fp)', structural: 'var(--cat-st)',
  signature:   'var(--cat-sg)', behavioral:  'var(--cat-bh)',
  multimodal:  'var(--cat-mm)',  other: 'var(--ink-4)',
};
const CAT_LABEL = {
  fingerprint: 'Fingerprint', structural: 'Structural',
  signature:   'Signature',   behavioral:  'Behavioral',
  multimodal:  'Multimodal',  other: 'Other',
};
const CAT_OF = {
  id_format:'fingerprint',backend_type:'fingerprint',inference_geo:'fingerprint',
  stop_details:'fingerprint',stop_details_structure:'fingerprint',
  small_output_tokens:'fingerprint',small_stop_reason:'fingerprint',
  container:'fingerprint',bedrock_state:'fingerprint',request_id:'fingerprint',
  x_new_api_version:'fingerprint',cf_ray_format:'fingerprint',cookie_domain:'fingerprint',
  hidden_prompt:'fingerprint',token_budget:'fingerprint',service_tier:'fingerprint',
  server_header:'fingerprint',signature_type_leak:'fingerprint',
  minimal_input_tokens:'fingerprint',minimal_output_tokens:'fingerprint',
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
  bash_stop_reason:'structural',bash_tool_name:'structural',bash_tool_rejected:'structural',
  signature:'signature',signature_length:'signature',thinking_present:'signature',
  thinking_order:'signature',thinking_display_omitted:'signature',no_thinking_leak:'signature',
  effort_high_thinking:'signature',effort_high_signature:'signature',
  effort_medium_no_think:'signature',effort_low_no_think:'signature',
  effort_max_thinking:'signature',effort_xhigh_thinking:'signature',
  signature_empty_rejected:'signature',
  tag_replay:'behavioral',identity_response:'behavioral',identity_no_leak:'behavioral',
  identity_platform:'behavioral',poison_answer:'behavioral',logic_answer:'behavioral',
  tool_forced_compliance:'behavioral',magic_refusal:'behavioral',
  intelligence_answer:'behavioral',
  image_ocr:'multimodal',pdf_extract:'multimodal',
};
function catOf(name) { return CAT_OF[name] || 'other'; }
function catColor(name) { return CAT_COLOR[catOf(name)]; }
function catLabel(name) { return CAT_LABEL[catOf(name)]; }

/* grade color map */
const GRADE_COLOR = {
  green: 'var(--good)', blue: 'var(--info)', yellow: 'var(--warn)',
  orange: 'var(--warn)', red: 'var(--bad)',
};
const GRADE_INK = {
  green: 'var(--good-ink)', blue: 'var(--info-ink)', yellow: 'var(--warn-ink)',
  orange: 'var(--warn-ink)', red: 'var(--bad-ink)',
};
const VERDICT_PILL = {
  green: 'pill-good', blue: 'pill-info', yellow: 'pill-warn',
  orange: 'pill-warn', red: 'pill-bad',
};
function gradeColor(g) { return GRADE_COLOR[g] || 'var(--ink-3)'; }
function gradeInk(g)   { return GRADE_INK[g]   || 'var(--ink-2)'; }

/* settings */
function loadSettings() {
  try {
    const s = JSON.parse(localStorage.getItem('llm-probe-settings') || '{}');
    State.adminToken = s.adminToken || '';
  } catch {}
}
function saveSettings() {
  localStorage.setItem('llm-probe-settings', JSON.stringify({ adminToken: State.adminToken }));
}

/* connection status badge */
function updateConn() {
  // connect status pill removed from topbar — no-op
}

/* clipboard */
async function copyText(s) {
  try { await navigator.clipboard.writeText(s); toast('已复制', 'good'); }
  catch { toast('复制失败', 'bad'); }
}

/* download json */
function downloadJSON(data, filename) {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url; a.download = filename; a.click();
  URL.revokeObjectURL(url);
}

/* ─── Probe / report normalisation ─── */
function reportSummary(rep) {
  const checks = rep.checks || [];
  const passed = checks.filter(c => c.pass).length;
  return {
    passed, failed: checks.length - passed, total: checks.length,
    score: rep.score ? rep.score.total_score : null,
    grade: rep.score ? rep.score.grade : '-',
    gradeColor: rep.score ? rep.score.grade_color : null,
    verdictColor: rep.score ? rep.score.verdict_color : null,
    verdictLabel: rep.score ? rep.score.verdict_label : '-',
    categories: rep.score ? (rep.score.categories || []) : [],
    elapsedMs: rep.elapsed_ms || 0,
    failures: checks.filter(c => !c.pass),
  };
}

/* ─── Model capabilities (mirrors Go model_caps.go) ─── */
const MODEL_CAPS = {
  'claude-haiku-4-5':  { thinking: true, thinkingMode: 'enabled', effort: [] },
  'claude-sonnet-4-6': { thinking: true, thinkingMode: 'adaptive', effort: ['low','medium','high','max'] },
  'claude-opus-4-5':   { thinking: true, thinkingMode: 'enabled', effort: ['low','medium','high'] },
  'claude-opus-4-6':   { thinking: true, thinkingMode: 'adaptive', effort: ['low','medium','high','max'] },
  'claude-opus-4-7':   { thinking: true, thinkingMode: 'adaptive_only', effort: ['low','medium','high','xhigh','max'] },
};
function getModelCaps(model) {
  if (MODEL_CAPS[model]) return MODEL_CAPS[model];
  for (const prefix in MODEL_CAPS) {
    if (model && model.startsWith(prefix)) return MODEL_CAPS[prefix];
  }
  return { thinking: false, thinkingMode: '', effort: [] };
}
function thinkingModesFor(model) {
  const c = getModelCaps(model);
  if (!c.thinking) return ['off'];
  if (c.thinkingMode === 'adaptive_only') return ['off', 'adaptive'];
  if (c.thinkingMode === 'adaptive') return ['off', 'adaptive'];
  if (c.thinkingMode === 'enabled') return ['off', 'enabled'];
  return ['off'];
}
function effortLevelsFor(model) {
  return getModelCaps(model).effort;
}
