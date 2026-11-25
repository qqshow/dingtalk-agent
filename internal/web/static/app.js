/* 钉钉助手 — single-page app (no build step). Routes live in location.hash. */
'use strict';

// ------------------------------------------------------------------ icons
const ICONS = {
  chat: '<path d="M4 5.5A2.5 2.5 0 0 1 6.5 3h11A2.5 2.5 0 0 1 20 5.5v8a2.5 2.5 0 0 1-2.5 2.5H10l-4.2 3.4c-.5.4-1.3 0-1.3-.6V16A2.5 2.5 0 0 1 4 13.5v-8Z"/><path d="M8.5 8.5h7M8.5 11.5h4.5"/>',
  dashboard: '<rect x="3.5" y="3.5" width="7" height="8" rx="1.8"/><rect x="13.5" y="3.5" width="7" height="5" rx="1.8"/><rect x="13.5" y="11.5" width="7" height="9" rx="1.8"/><rect x="3.5" y="14.5" width="7" height="6" rx="1.8"/>',
  todo: '<rect x="3.5" y="3.5" width="17" height="17" rx="4"/><path d="m8 12.2 2.8 2.8L16.2 9.5"/>',
  calendar: '<rect x="3.5" y="5" width="17" height="15.5" rx="3"/><path d="M3.5 10h17M8 3v4M16 3v4"/><circle cx="8.5" cy="14.5" r=".6" fill="currentColor"/><circle cx="12" cy="14.5" r=".6" fill="currentColor"/>',
  search: '<circle cx="11" cy="11" r="6.5"/><path d="m16 16 4 4"/>',
  download: '<path d="M12 4v11M7.5 10.5 12 15l4.5-4.5"/><path d="M4.5 16.5v1.5A2.5 2.5 0 0 0 7 20.5h10a2.5 2.5 0 0 0 2.5-2.5v-1.5"/>',
  sync: '<path d="M19.5 12a7.5 7.5 0 0 1-13.2 4.9M4.5 12a7.5 7.5 0 0 1 13.2-4.9"/><path d="M17.8 3.5v3.7h-3.7M6.2 20.5v-3.7h3.7"/>',
  theme: '<path d="M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5Z"/>',
  gear: '<circle cx="12" cy="12" r="3"/><path d="M12 2.8v2.4M12 18.8v2.4M4.2 7.5l2.1 1.2M17.7 15.3l2.1 1.2M4.2 16.5l2.1-1.2M17.7 8.7l2.1-1.2"/><circle cx="12" cy="12" r="7"/>',
  group: '<circle cx="9" cy="9" r="3"/><path d="M3.5 18.5c.6-2.8 2.8-4.5 5.5-4.5s4.9 1.7 5.5 4.5"/><path d="M15 6.2a3 3 0 0 1 0 5.6M17 14.3c1.8.6 3 2 3.5 4.2"/>',
  project: '<path d="M4 7.5A2.5 2.5 0 0 1 6.5 5H10l2 2h5.5A2.5 2.5 0 0 1 20 9.5v7a2.5 2.5 0 0 1-2.5 2.5h-11A2.5 2.5 0 0 1 4 16.5v-9Z"/>',
  pin: '<path d="M14.5 4.5 19.5 9.5l-2.5 1-3.5 3.5-.5 4-2.5-2.5-4 4M9 11.5 12.5 15"/><path d="m9.5 6 8.5 8.5"/>',
  msg: '<path d="M4 6.5A2.5 2.5 0 0 1 6.5 4h11A2.5 2.5 0 0 1 20 6.5v7a2.5 2.5 0 0 1-2.5 2.5h-6L7 19.5V16h-.5A2.5 2.5 0 0 1 4 13.5v-7Z"/>',
  send: '<path d="M4 12 20 4l-4 16-4-6.5L4 12Z"/><path d="M12 13.5 20 4"/>',
  people: '<circle cx="12" cy="8.5" r="3.5"/><path d="M5 19.5c.8-3.4 3.6-5.5 7-5.5s6.2 2.1 7 5.5"/>',
  at: '<circle cx="12" cy="12" r="3.5"/><path d="M15.5 12v1.3a2.2 2.2 0 0 0 4.4 0V12a7.9 7.9 0 1 0-3.2 6.3"/>',
  reply: '<path d="M9.5 6 4 11.5 9.5 17"/><path d="M4.5 11.5h9a6 6 0 0 1 6 6V19"/>',
  clock: '<circle cx="12" cy="12" r="8.5"/><path d="M12 7.5V12l3 2"/>',
  spark: '<path d="M12 3.5c.6 3.7 1.8 6 5.6 6.8v.4c-3.8.8-5 3.1-5.6 6.8h-.4c-.6-3.7-1.8-6-5.6-6.8v-.4c3.8-.8 5-3.1 5.6-6.8h.4Z"/><path d="M18.5 15.5c.2 1.4.7 2.2 2 2.5-1.3.3-1.8 1.1-2 2.5-.2-1.4-.7-2.2-2-2.5 1.3-.3 1.8-1.1 2-2.5Z"/>',
  flag: '<path d="M5.5 20.5V4.5M5.5 5h11l-2 4 2 4h-11"/>',
  check: '<path d="m5 12.5 4.5 4.5L19 7.5"/>',
  chevL: '<path d="m14.5 6-6 6 6 6"/>',
  chevR: '<path d="m9.5 6 6 6-6 6"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  trash: '<path d="M5 7h14M10 7V5h4v2M7 7l.8 12.5h8.4L17 7"/>',
  copy: '<rect x="8" y="8" width="11.5" height="11.5" rx="2"/><path d="M15.5 8V6.5a2 2 0 0 0-2-2h-7a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2H8"/>',
  room: '<path d="M12 20.5s6.5-5.4 6.5-10.5a6.5 6.5 0 0 0-13 0c0 5.1 6.5 10.5 6.5 10.5Z"/><circle cx="12" cy="10" r="2.3"/>',
  video: '<rect x="3.5" y="6.5" width="12" height="11" rx="2.5"/><path d="m15.5 10.5 5-3v9l-5-3"/>',
  file: '<path d="M7 3.5h7l4.5 4.5v12.5H7Z"/><path d="M14 3.5V8h4.5"/>',
  card: '<rect x="3.5" y="5" width="17" height="14" rx="2.5"/><path d="M3.5 9.5h17"/>',
  approval: '<path d="M8 4.5h8M9 3.5h6v3H9z"/><rect x="5" y="5" width="14" height="15.5" rx="2.5"/><path d="m9 13 2 2 4-4"/>',
  stop: '<rect x="7" y="7" width="10" height="10" rx="2"/>',
  link: '<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1"/><path d="M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>',
  close: '<path d="M6.5 6.5l11 11M17.5 6.5l-11 11"/>',
  image: '<rect x="3.5" y="4.5" width="17" height="15" rx="2.5"/><circle cx="9" cy="9.5" r="1.6"/><path d="m4 17 4.5-4.5 3.5 3.5 2.5-2.5L20 18"/>',
};
const icon = (name, cls = '') => `<svg class="${cls}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">${ICONS[name] || ''}</svg>`;

// ------------------------------------------------------------------ utils
const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];
const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const pad = n => String(n).padStart(2, '0');
const WEEK = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

async function api(path, opts = {}) {
  if (opts.json !== undefined) {
    opts = { ...opts, body: JSON.stringify(opts.json), headers: { 'Content-Type': 'application/json' } };
    delete opts.json;
  }
  const resp = await fetch(path, opts);
  const data = await resp.json().catch(() => ({}));
  if (!resp.ok) throw new Error(data.detail || `HTTP ${resp.status}`);
  return data;
}

function toast(msg, ms = 2200) {
  const t = $('#toast');
  t.textContent = msg;
  t.hidden = false;
  clearTimeout(toast._t);
  toast._t = setTimeout(() => (t.hidden = true), ms);
}

const ymd = d => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const hm = ts => { const d = new Date(ts); return `${pad(d.getHours())}:${pad(d.getMinutes())}`; };
const parseDay = s => { const [y, m, d] = s.split('-').map(Number); return new Date(y, m - 1, d); };
function sameDay(a, b) { return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate(); }

// Conversation list style time: today → HH:MM, yesterday, this week → 周X, else MM/DD
function shortTime(ts) {
  if (!ts) return '';
  const d = new Date(ts), now = new Date();
  if (sameDay(d, now)) return hm(ts);
  const y = new Date(now); y.setDate(now.getDate() - 1);
  if (sameDay(d, y)) return '昨天';
  if (now - d < 6 * 864e5) return WEEK[d.getDay()];
  if (d.getFullYear() === now.getFullYear()) return `${d.getMonth() + 1}/${d.getDate()}`;
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`;
}
function dayLabel(ts) {
  const d = new Date(ts), now = new Date();
  if (sameDay(d, now)) return '今天';
  const y = new Date(now); y.setDate(now.getDate() - 1);
  if (sameDay(d, y)) return '昨天';
  const s = `${d.getMonth() + 1}月${d.getDate()}日 ${WEEK[d.getDay()]}`;
  return d.getFullYear() === now.getFullYear() ? s : `${d.getFullYear()}年${s}`;
}
function fmtSize(b) {
  if (!b) return '';
  const u = ['B', 'KB', 'MB', 'GB']; let i = 0;
  while (b >= 1024 && i < 3) { b /= 1024; i++; }
  return `${b.toFixed(i ? 1 : 0)} ${u[i]}`;
}
function fmtMinutes(m) {
  if (!m) return '0 分钟';
  if (m < 60) return `${m} 分钟`;
  return `${Math.floor(m / 60)} 小时${m % 60 ? ` ${m % 60} 分` : ''}`;
}

// DingTalk-style avatar: last two characters of the name on a colour.
const AV_COLORS = ['#3d8bff', '#00b5c7', '#00b853', '#ff9200', '#8a5cf6', '#f25c8a', '#2fb5a3', '#5b7cf6'];
function hashStr(s) { let h = 0; for (const c of String(s)) h = (h * 31 + c.codePointAt(0)) | 0; return Math.abs(h); }
function avatar(name, { size = '', group = false, project = false, bot = false, seed } = {}) {
  const color = AV_COLORS[hashStr(seed ?? name) % AV_COLORS.length];
  const cls = `avatar ${size} ${group ? 'group' : ''} ${bot ? 'bot' : ''}`;
  if (group) return `<div class="${cls}" style="background:${project ? '#00b853' : color}">${icon(project ? 'project' : 'group')}</div>`;
  const clean = String(name || '?').replace(/[\s()（）\[\]【】]/g, '');
  const chars = [...clean];
  const isLatin = /^[a-z0-9._-]+$/i.test(clean);
  const text = isLatin ? clean.slice(0, 2).toUpperCase() : chars.slice(-2).join('');
  return `<div class="${cls}" style="background:${color}">${esc(text)}</div>`;
}

// Safe markdown subset for AI summaries.
function md(src) {
  const inline = t => esc(t)
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/(^|[^*])\*([^*\s][^*]*)\*/g, '$1<em>$2</em>')
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>');
  const out = []; let list = null, code = null, table = null, quote = false;
  const closeList = () => { if (list) { out.push(`</${list}>`); list = null; } };
  const closeTable = () => { if (table) { out.push('</tbody></table></div>'); table = null; } };
  const closeQuote = () => { if (quote) { out.push('</blockquote>'); quote = false; } };
  const closeAll = () => { closeList(); closeTable(); closeQuote(); };
  const cells = l => l.trim().replace(/^\||\|$/g, '').split('|').map(c => c.trim());
  const lines = String(src || '').split('\n');
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trimEnd(); let m;
    if (code !== null) {
      if (/^\s*```/.test(line)) { out.push(`<pre><code>${esc(code.join('\n'))}</code></pre>`); code = null; }
      else code.push(lines[i]);
      continue;
    }
    if (/^\s*```/.test(line)) { closeAll(); code = []; continue; }
    if (/^\s*\|.*\|\s*$/.test(line)) {
      if (!table) {
        const next = lines[i + 1] || '';
        if (/^\s*\|?\s*:?-{2,}/.test(next)) {
          closeList(); closeQuote();
          out.push(`<div class="md-table"><table><thead><tr>${cells(line).map(c => `<th>${inline(c)}</th>`).join('')}</tr></thead><tbody>`);
          table = true; i++; continue;
        }
      } else { out.push(`<tr>${cells(line).map(c => `<td>${inline(c)}</td>`).join('')}</tr>`); continue; }
    }
    closeTable();
    if ((m = line.match(/^>\s?(.*)$/))) { closeList(); if (!quote) { out.push('<blockquote>'); quote = true; } out.push(`<p>${inline(m[1])}</p>`); continue; }
    closeQuote();
    if (/^\s*(-{3,}|\*{3,})\s*$/.test(line)) { closeList(); out.push('<hr>'); }
    else if ((m = line.match(/^(#{1,4})\s+(.*)$/))) { closeList(); out.push(`<h${m[1].length + 1}>${inline(m[2])}</h${m[1].length + 1}>`); }
    else if ((m = line.match(/^\s*[-*]\s+(.*)$/))) { if (list !== 'ul') { closeList(); out.push('<ul>'); list = 'ul'; } out.push(`<li>${inline(m[1])}</li>`); }
    else if ((m = line.match(/^\s*\d+[.)]\s+(.*)$/))) { if (list !== 'ol') { closeList(); out.push('<ol>'); list = 'ol'; } out.push(`<li>${inline(m[1])}</li>`); }
    else if (!line.trim()) closeList();
    else { closeList(); out.push(`<p>${inline(line)}</p>`); }
  }
  if (code !== null) out.push(`<pre><code>${esc(code.join('\n'))}</code></pre>`);
  closeAll();
  return out.join('');
}

// ------------------------------------------------------------------ state
const S = {
  me: { uid: '', name: '' },
  convs: [],
  convMap: new Map(),
  convFilter: 'all',
  convSearch: '',
  convsLoadedAt: 0,
  chat: null, // { cid, offset, total, msgs, loading }
};

async function loadConvs(force = false) {
  if (!force && S.convs.length && Date.now() - S.convsLoadedAt < 60000) return S.convs;
  const d = await api('/api/conversations?limit=5000');
  S.convs = d.conversations;
  S.convMap = new Map(S.convs.map(c => [c.cid, c]));
  S.convsLoadedAt = Date.now();
  return S.convs;
}
const convOf = cid => S.convMap.get(cid);
function convAvatar(c, size = '') {
  if (!c) return avatar('?', { size });
  return c.type === 'group' ? avatar(c.title, { size, group: true, project: c.is_project, seed: c.cid })
    : avatar(c.title, { size, seed: c.other_uid || c.cid });
}

// ----------------------------------------------------------------- router
function parseRoute() {
  const h = location.hash.replace(/^#\/?/, '');
  const [pathPart, query = ''] = h.split('?');
  const parts = pathPart.split('/').filter(Boolean).map(decodeURIComponent);
  return { name: parts[0] || 'chat', arg: parts[1], params: new URLSearchParams(query) };
}
function go(hash) { if (location.hash !== hash) location.hash = hash; else render(); }
function setQuery(params) {
  const r = parseRoute();
  const q = new URLSearchParams(params).toString();
  history.replaceState(null, '', `#/${r.name}${r.arg ? '/' + encodeURIComponent(r.arg) : ''}${q ? '?' + q : ''}`);
}

const VIEWS = { chat: viewChat, ai: viewAI, report: viewReport, todos: viewTodos, calendar: viewCalendar, search: viewSearch, exports: viewExports, settings: viewSettings };
let renderSeq = 0;
async function render() {
  const r = parseRoute();
  const fn = VIEWS[r.name] || viewChat;
  $$('.rail-item[data-route]').forEach(a => a.classList.toggle('active', a.dataset.route === (VIEWS[r.name] ? r.name : 'chat')));
  const seq = ++renderSeq;
  try {
    await fn(r, () => seq === renderSeq);
  } catch (e) {
    if (seq !== renderSeq) return;
    $('#view').innerHTML = `<div class="page"><div class="page-inner"><div class="error-box">加载失败：${esc(e.message)}</div></div></div>`;
  }
}

// =================================================================== CHAT
async function viewChat(r, alive) {
  const view = $('#view');
  if (!$('.chat', view)) {
    view.innerHTML = `
      <div class="chat">
        <aside class="conv-pane">
          <div class="conv-top">
            <label class="search-input">${icon('search')}<input class="input" id="convSearch" placeholder="搜索会话" autocomplete="off"></label>
            <div class="seg" id="convSeg">
              <button data-f="all">全部</button><button data-f="single">单聊</button><button data-f="group">群聊</button><button data-f="project">项目</button>
            </div>
          </div>
          <div class="conv-list" id="convList"><div class="skeleton" style="height:56px;margin:6px"></div><div class="skeleton" style="height:56px;margin:6px"></div></div>
        </aside>
        <section class="chat-pane" id="chatPane"></section>
      </div>`;
    $('#convSearch').value = S.convSearch;
    $('#convSearch').addEventListener('input', e => { S.convSearch = e.target.value.trim(); renderConvList(); });
    $('#convSeg').addEventListener('click', e => {
      if (!e.target.dataset.f) return;
      S.convFilter = e.target.dataset.f; renderConvList();
    });
    $('#convList').addEventListener('click', e => {
      const el = e.target.closest('.conv');
      if (el) go(`#/chat/${encodeURIComponent(el.dataset.cid)}`);
    });
  }
  await loadConvs();
  if (!alive()) return;
  renderConvList();
  $('.chat').classList.toggle('has-current', !!r.arg);
  if (r.arg) {
    if (!S.chat || S.chat.cid !== r.arg) await openChat(r.arg);
  } else {
    S.chat = null;
    $('#chatPane').innerHTML = `<div class="chat-empty"><div><div class="big">${icon('chat')}</div>
      <h3>选择一个会话开始浏览</h3><div>共 ${S.convs.length} 个会话 · 数据仅保存在本机</div></div></div>`;
  }
  $$('.conv').forEach(el => el.classList.toggle('active', el.dataset.cid === r.arg));
}

function filteredConvs() {
  const kw = S.convSearch.toLowerCase();
  return S.convs.filter(c => {
    if (S.convFilter === 'single' && c.type !== 'single') return false;
    if (S.convFilter === 'group' && c.type !== 'group') return false;
    if (S.convFilter === 'project' && !c.is_project) return false;
    return !kw || c.title.toLowerCase().includes(kw);
  });
}

function renderConvList() {
  $$('#convSeg button').forEach(b => b.classList.toggle('active', b.dataset.f === S.convFilter));
  const list = filteredConvs();
  const cur = parseRoute().arg;
  const rows = list.slice(0, 400).map(c => {
    const lm = c.last_message;
    let preview = '';
    if (lm) preview = (lm.mine ? '我: ' : (c.type === 'group' && lm.sender ? lm.sender + ': ' : '')) + cleanPreview(lm.text);
    const t = lm ? lm.time : c.last_modify;
    return `<div class="conv ${c.cid === cur ? 'active' : ''}" data-cid="${esc(c.cid)}">
      ${convAvatar(c)}
      <div class="conv-main">
        <div class="conv-row"><div class="conv-title">${esc(c.title || c.cid)}</div><div class="conv-time">${shortTime(t)}</div></div>
        <div class="conv-row"><div class="conv-preview">${esc(preview) || '&nbsp;'}</div>${c.is_top ? icon('pin', 'conv-pin') : ''}</div>
      </div></div>`;
  }).join('');
  $('#convList').innerHTML = (rows || '<div class="empty">没有匹配的会话</div>') +
    (list.length > 400 ? `<div class="conv-count">仅显示前 400 个，共 ${list.length} 个，请用搜索缩小范围</div>` : '');
}

async function openChat(cid) {
  const c = convOf(cid);
  S.chat = { cid, offset: 0, total: 0, msgs: [], loading: false };
  const meta = c ? (c.type === 'group' ? `${c.member_count} 人${c.is_project ? ' · 项目群' : ''}` : '单聊') : '';
  $('#chatPane').innerHTML = `
    <header class="chat-head">
      <button class="btn btn-ghost btn-icon mobile-back" onclick="go('#/chat')" title="返回">${icon('chevL')}</button>
      ${convAvatar(c, 'sm')}
      <div class="grow"><h2>${esc(c ? c.title : cid)}</h2><div class="meta" id="chatMeta">${esc(meta)}</div></div>
      <button class="btn btn-ghost btn-sm" id="chatAskAI" title="基于这个${c && c.type === 'group' ? '群' : '人'}的聊天记录和 AI 对话">${icon('spark')}问 AI</button>
      <button class="btn btn-ghost btn-sm" id="chatSearchBtn">${icon('search')}在会话中搜索</button>
    </header>
    <div class="chat-body" id="chatBody"><div class="empty">加载中…</div></div>`;
  $('#chatSearchBtn').onclick = () => go(`#/search?cid=${encodeURIComponent(cid)}`);
  $('#chatAskAI').onclick = () => go(`#/ai?mention=${encodeURIComponent(cid)}`);
  $('#chatBody').addEventListener('click', e => {
    const img = e.target.closest('img.msg-img, img.inline-img');
    if (img) openViewer(img);
    if (e.target.closest('#loadOlder')) loadOlder();
  });
  await loadChatPage(true);
}

async function loadChatPage(first) {
  const ch = S.chat; if (!ch || ch.loading) return;
  ch.loading = true;
  const cid = ch.cid;
  try {
    const d = await api(`/api/conversations/${encodeURIComponent(cid)}/messages?limit=50&offset=${ch.offset}`);
    if (!S.chat || S.chat.cid !== cid) return;
    ch.total = d.total;
    ch.msgs = d.messages.concat(ch.msgs);
    const body = $('#chatBody');
    const prevH = body.scrollHeight, prevTop = body.scrollTop;
    renderMessages();
    if (first) body.scrollTop = body.scrollHeight;
    else body.scrollTop = body.scrollHeight - prevH + prevTop;
    $('#chatMeta').textContent = `${$('#chatMeta').textContent.split(' · 共')[0]} · 共 ${ch.total} 条消息`;
  } finally { ch.loading = false; }
}
function loadOlder() {
  const ch = S.chat;
  if (!ch || ch.msgs.length >= ch.total) return;
  ch.offset = ch.msgs.length;
  loadChatPage(false);
}

function renderMessages() {
  const ch = S.chat; const body = $('#chatBody');
  let html = ch.msgs.length < ch.total
    ? `<div class="load-older"><button class="btn btn-sm" id="loadOlder">${icon('clock')}查看更早的消息（还有 ${ch.total - ch.msgs.length} 条）</button></div>` : '';
  if (!ch.msgs.length) html = '<div class="empty">该会话没有消息</div>';
  let lastDay = '', prev = null;
  for (const m of ch.msgs) {
    const dk = ymd(new Date(m.created_at));
    if (dk !== lastDay) { html += `<div class="date-sep"><span>${dayLabel(m.created_at)}</span></div>`; lastDay = dk; prev = null; }
    if (m.is_system || m.content_type === 102 || m.content_type === 104) {
      html += `<div class="sys-msg">${esc(m.text || m.content_type_name)} · ${hm(m.created_at)}</div>`; prev = null; continue;
    }
    const cont = prev && prev.sender_id === m.sender_id && m.created_at - prev.created_at < 180000;
    const cls = ['msg', m.is_mine ? 'me' : '', cont ? 'cont' : '', m.mentions_me && !m.is_mine ? 'at-me' : ''].join(' ');
    const name = m.is_mine ? '' : esc(m.sender_name);
    const [content, media] = renderContent(m);
    html += `<div class="${cls}">
      ${m.is_mine ? avatar(S.me.name || '我', { size: 'sm', seed: S.me.uid }) : avatar(m.sender_name, { size: 'sm', seed: String(m.sender_id) })}
      <div class="msg-col">
        ${m.is_mine ? '' : `<div class="msg-name">${name}</div>`}
        <div class="bubble ${media ? 'media' : ''}">${content}</div>
      </div>
      <time class="msg-time-hover">${hm(m.created_at)}</time></div>`;
    prev = m;
  }
  body.innerHTML = html;
  $$('img', body).forEach(img => img.addEventListener('error', () => { img.outerHTML = '<span class="ph">图片加载失败</span>'; }, { once: true }));
}

// DingTalk rich text is markdown-ish: "> quote", "#### title", ![img](url), [text](url).
function cleanPreview(t) {
  // ")" may be missing when the server truncated the preview mid-link.
  return String(t || '').replace(/!\[[^\]]*\](\([^)]*\)?)?/g, '[图片]').replace(/\[([^\]]+)\]\((?:https?|dingtalk):[^)]*\)?/g, '$1')
    .replace(/(^|\s)(?:>|#{1,6})\s+/g, '$1').replace(/\s+/g, ' ').trim();
}
// Plain text with @uid replaced by names.
function resolveAt(m) {
  let t = String(m.text || '');
  for (const [uid, nm] of Object.entries(m.at_ids || {})) t = t.split('@' + uid).join('@' + nm);
  return t;
}
function richText(m) {
  const lines = String(m.text || '').replace(/!\[[^\]]*\]\([^)]*\)/g, '[图片]').split('\n');
  const out = [];
  for (let line of lines) {
    let quote = false;
    if (/^\s*>\s?/.test(line)) { quote = true; line = line.replace(/^\s*>\s?/, ''); }
    let h = esc(line.replace(/^\s*#{1,6}\s+/, ''))
      .replace(/\[([^\]]+)\]\((https?:[^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')
      .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
    if (!/<a /.test(h)) h = linkify(h);
    for (const [uid, nm] of Object.entries(m.at_ids || {})) h = h.split('@' + esc(uid)).join(`<span class="mention">@${esc(nm)}</span>`);
    if (/^\s*#{1,6}\s+/.test(line) && !quote) h = `<strong>${h}</strong>`;
    out.push(quote ? (line.trim() ? `<span class="quote-ref">${h}</span>` : '') : h);
  }
  return withImages(out.join('\n').replace(/\n{2,}/g, '\n').trim(), m);
}

function linkify(htmlEscaped) {
  return htmlEscaped.replace(/(https?:\/\/[^\s<]+)/g, '<a href="$1" target="_blank" rel="noopener noreferrer">$1</a>');
}
function textWithMentions(m) {
  let t = esc(m.text || '');
  for (const [uid, nm] of Object.entries(m.at_ids || {})) {
    t = t.split('@' + esc(uid)).join(`<span class="mention">@${esc(nm)}</span>`);
  }
  t = t.replace(/@所有人/g, '<span class="mention">@所有人</span>');
  return linkify(t);
}
function withImages(htmlText, m) {
  const imgs = (m.image_info && m.image_info.images) || [];
  let i = 0;
  return htmlText.replace(/\[图片\]/g, () => {
    const img = imgs[i++];
    return img && img.src ? imgTag(img, 'inline-img') : '<span class="ph">图片</span>';
  });
}
const FILE_COLORS = { pdf: '#f25c5c', doc: '#3d8bff', docx: '#3d8bff', xls: '#00b853', xlsx: '#00b853', ppt: '#ff9200', pptx: '#ff9200', zip: '#8a5cf6', rar: '#8a5cf6', '7z': '#8a5cf6', gz: '#8a5cf6', txt: '#7a8599', md: '#7a8599' };
function fileCard(a) {
  const ext = (a.filename.split('.').pop() || '').toLowerCase();
  const color = FILE_COLORS[ext] || '#5b7cf6';
  const inner = `<div class="file-icon" style="background:${color}">${esc(ext.slice(0, 4) || 'file')}</div>
    <div class="file-meta"><div class="file-name">${esc(a.filename)}</div>
    <div class="file-size">${fmtSize(a.file_size)}${a.local_available ? ' · 点击打开' : ' · 未下载到本地'}</div></div>`;
  return a.local_available && a.local_path
    ? `<a class="file-card" href="/api/local-file?path=${encodeURIComponent(a.local_path)}" target="_blank">${inner}</a>`
    : `<div class="file-card">${inner}</div>`;
}
function miniCard(title, iconName, text) {
  return `<div class="mini-card"><div class="mc-head">${icon(iconName)}${esc(title)}</div><div class="mc-body">${text}</div></div>`;
}

// Returns [html, isMedia]
function renderContent(m) {
  const ct = m.content_type;
  const files = (m.attachments || []).filter(a => a.filename);
  switch (ct) {
    case 1: return [textWithMentions(m), false];
    case 2: {
      const imgs = (m.image_info && m.image_info.images) || [];
      return imgs.length ? [imgs.map(i => imgTag(i, 'msg-img')).join(''), true] : ['<span class="ph">图片未缓存到本地</span>', false];
    }
    case 300: return ['🎙 语音消息', false];
    case 501: case 503: case 4:
      return files.length ? [files.map(fileCard).join(''), true] : ['[文件]', false];
    case 1101: return ['📞 通话记录', false];
    case 3100: {
      if (/^\s*>|!\[|^#{1,6}\s/m.test(m.text || '')) return [richText(m), false];
      const t = withImages(textWithMentions(m), m);
      const nl = t.indexOf('\n');
      return [nl > 0 && /^(.{1,40}?)[：:]/.test(m.text) ? `<span class="quote-ref">${t.slice(0, nl)}</span>${t.slice(nl + 1)}` : t, false];
    }
    case 1200: case 1201: case 1202:
      return [`<div class="rich">${m.text ? richText(m) : '[富文本]'}</div>`, false];
    case 1400: return [miniCard('审批', 'approval', esc(m.text || '审批消息')), true];
    case 1500: return [miniCard('任务', 'todo', esc(m.text || '任务')), true];
    case 1600: return [miniCard('日程', 'calendar', esc(m.text || '日程邀请')), true];
    case 2900: case 2950: return [miniCard(ct === 2950 ? '小程序' : '互动卡片', 'card', esc(m.text || '卡片消息')), true];
    case 500: return ['📍 ' + esc(m.text || '位置'), false];
    default:
      if (files.length) return [files.map(fileCard).join(''), true];
      return [m.text ? textWithMentions(m) : `<span class="ph">${esc(m.content_type_name || '消息')}</span>`, false];
  }
}

// Bubble shows the original (downscaled by the browser → sharp on HiDPI);
// a "缩略图" badge marks pictures whose original was never downloaded.
function imgTag(img, cls) {
  const tag = `<img class="${cls}" src="${esc(img.src)}" data-orig="${img.is_original ? 1 : 0}" alt="图片" loading="lazy" decoding="async">`;
  return img.is_original ? tag : `<span class="img-wrap">${tag}<span class="img-badge">缩略图</span></span>`;
}

// ---------------------------------------------------------------- viewer
const V = { list: [], idx: 0, scale: 1, fit: 1, x: 0, y: 0, nw: 0, nh: 0 };
function openViewer(imgEl) {
  V.list = $$('#chatBody img.msg-img, #chatBody img.inline-img');
  V.idx = Math.max(0, V.list.indexOf(imgEl));
  $('#viewer').hidden = false;
  showViewerImage();
}
function closeViewer() { $('#viewer').hidden = true; $('#vImg').removeAttribute('src'); }
function showViewerImage() {
  const el = V.list[V.idx]; if (!el) return;
  const img = $('#vImg');
  $('#vCount').textContent = V.list.length > 1 ? `${V.idx + 1} / ${V.list.length}` : '';
  $('#vTip').textContent = el.dataset.orig === '1' ? '' : '本机只缓存了缩略图（原图未下载）：在钉钉里点开这张图片，再同步一次即可看到原图';
  $('#vPrev').hidden = $('#vNext').hidden = V.list.length < 2;
  $('#vOpen').href = el.src; $('#vDown').href = el.src;
  img.style.visibility = 'hidden';
  img.onload = () => {
    V.nw = img.naturalWidth; V.nh = img.naturalHeight;
    const st = $('#vStage');
    V.fit = Math.min(1, (st.clientWidth - 80) / V.nw, (st.clientHeight - 140) / V.nh);
    // Thumbnails are small: let "fit" enlarge them up to the window as well.
    if (el.dataset.orig !== '1') V.fit = Math.min((st.clientWidth - 80) / V.nw, (st.clientHeight - 140) / V.nh, 2.5);
    setView(V.fit, 0, 0, false);
    img.style.visibility = '';
  };
  img.src = el.src;
}
function setView(scale, x, y, animate = true) {
  V.scale = Math.min(10, Math.max(0.05, scale)); V.x = x; V.y = y;
  const st = $('#vStage');
  st.classList.toggle('animate', animate);
  $('#vImg').style.width = V.nw + 'px';
  $('#vImg').style.transform = `translate(-50%, -50%) translate(${V.x}px, ${V.y}px) scale(${V.scale})`;
  $('#vPct').textContent = Math.round(V.scale * 100) + '%';
}
// Zoom keeping the point (cx, cy) — relative to the stage centre — fixed.
function zoomAt(factor, cx = 0, cy = 0, animate = true) {
  const ns = Math.min(10, Math.max(0.05, V.scale * factor));
  const r = ns / V.scale;
  setView(ns, cx - (cx - V.x) * r, cy - (cy - V.y) * r, animate);
}
function viewerStep(d) {
  if (V.list.length < 2) return;
  V.idx = (V.idx + d + V.list.length) % V.list.length;
  showViewerImage();
}
function initViewer() {
  $('#vPrev').innerHTML = icon('chevL'); $('#vNext').innerHTML = icon('chevR');
  const st = $('#vStage');
  const center = e => { const r = st.getBoundingClientRect(); return [e.clientX - r.left - r.width / 2, e.clientY - r.top - r.height / 2]; };
  st.addEventListener('wheel', e => { e.preventDefault(); const [cx, cy] = center(e); zoomAt(e.deltaY < 0 ? 1.15 : 1 / 1.15, cx, cy, false); }, { passive: false });
  st.addEventListener('dblclick', e => {
    const [cx, cy] = center(e);
    if (Math.abs(V.scale - V.fit) < 0.01) zoomAt((V.fit >= 1 ? 2 : 1) / V.scale, cx, cy);
    else setView(V.fit, 0, 0);
  });
  let drag = null;
  st.addEventListener('pointerdown', e => {
    if (e.button !== 0) return;
    // Remember what was pressed: with pointer capture, pointerup always targets the stage.
    drag = { sx: e.clientX, sy: e.clientY, x: V.x, y: V.y, moved: false, onBackdrop: e.target === st };
    st.setPointerCapture(e.pointerId); st.classList.add('dragging');
  });
  st.addEventListener('pointermove', e => {
    if (!drag) return;
    const dx = e.clientX - drag.sx, dy = e.clientY - drag.sy;
    if (Math.abs(dx) + Math.abs(dy) > 3) drag.moved = true;
    setView(V.scale, drag.x + dx, drag.y + dy, false);
  });
  st.addEventListener('pointerup', e => {
    st.classList.remove('dragging');
    // A plain click on the backdrop (not the picture) closes the viewer.
    if (drag && !drag.moved && drag.onBackdrop) closeViewer();
    drag = null;
  });
  $('#vIn').onclick = () => zoomAt(1.25);
  $('#vOut').onclick = () => zoomAt(1 / 1.25);
  $('#vFit').onclick = () => setView(V.fit, 0, 0);
  $('#v11').onclick = () => setView(1, 0, 0);
  $('#vClose').onclick = closeViewer;
  $('#vPrev').onclick = () => viewerStep(-1);
  $('#vNext').onclick = () => viewerStep(1);
  document.addEventListener('keydown', e => {
    if ($('#viewer').hidden) return;
    const k = { Escape: closeViewer, ArrowLeft: () => viewerStep(-1), ArrowRight: () => viewerStep(1),
      '+': () => zoomAt(1.25), '=': () => zoomAt(1.25), '-': () => zoomAt(1 / 1.25), '0': () => setView(1, 0, 0) }[e.key];
    if (k) { e.preventDefault(); k(); }
  });
}

// ============================================================== DASHBOARD
async function viewReport(r, alive) {
  const date = r.params.get('date') || ymd(new Date());
  const view = $('#view');
  view.innerHTML = `<div class="page"><div class="page-inner" id="reportPage">
    <div class="hero">
      <div><h1>${greeting()}，${esc(S.me.name || '')}</h1><div class="sub" id="heroSub">正在汇总 ${esc(date)} 的工作…</div></div>
      <div class="grow"></div>
      <div class="date-nav">
        <button class="btn btn-icon" id="dPrev" title="前一天">${icon('chevL')}</button>
        <input type="date" class="input" id="dPick" value="${esc(date)}">
        <button class="btn btn-icon" id="dNext" title="后一天">${icon('chevR')}</button>
        <button class="btn" id="dToday">今天</button>
      </div>
    </div>
    <div class="stats" id="stats">${'<div class="stat skeleton" style="height:72px"></div>'.repeat(6)}</div>
    <div id="dashBody"></div></div></div>`;
  const shift = n => { const d = parseDay(date); d.setDate(d.getDate() + n); go(`#/report?date=${ymd(d)}`); };
  $('#dPrev').onclick = () => shift(-1);
  $('#dNext').onclick = () => shift(1);
  $('#dToday').onclick = () => go('#/report');
  $('#dPick').onchange = e => e.target.value && go(`#/report?date=${e.target.value}`);

  const [rep, aiSt] = await Promise.all([api(`/api/agent/report?date=${encodeURIComponent(date)}`), loadAIStatus(), loadConvs()]);
  if (!alive()) return;
  S.ai = aiSt;
  renderDashboard(rep, date);
  if (aiSt && aiSt.running) pollAI();
}

function greeting() {
  const h = new Date().getHours();
  return h < 6 ? '夜深了' : h < 11 ? '早上好' : h < 13 ? '中午好' : h < 18 ? '下午好' : '晚上好';
}

function stat(iconName, color, num, label, sub = '') {
  return `<div class="stat"><div class="stat-icon i-${color}">${icon(iconName)}</div>
    <div><div class="stat-num">${esc(num)}</div><div class="stat-label">${esc(label)}</div>${sub ? `<div class="stat-sub">${esc(sub)}</div>` : ''}</div></div>`;
}

function renderDashboard(r, date) {
  const t = r.totals, cal = r.calendar || {};
  const unreplied = r.mentions_of_me.filter(m => !m.replied).length;
  $('#heroSub').textContent = `${r.period.label} · 数据截至 ${r.data_as_of.slice(5, 16)}`;
  $('#stats').innerHTML = [
    stat('msg', 'blue', t.messages, '消息', `发送 ${t.sent_by_me} · 收到 ${t.received}`),
    stat('people', 'cyan', r.people.count, '互动的人'),
    stat('group', 'green', r.groups.involved_count, '参与的群', `活跃 ${r.groups.active_count} 个`),
    stat('project', 'purple', r.projects.involved_count, '参与的项目', `活跃项目群 ${r.projects.active_count}`),
    cal.available ? stat('calendar', 'orange', cal.meeting_count, '会议', fmtMinutes(cal.meeting_minutes)) : '',
    stat('reply', 'red', r.awaiting_my_reply.length, '待我回复', unreplied ? `@我未回复 ${unreplied}` : `@我 ${r.mentions_of_me.length}`),
  ].join('');

  const s = r.saved_summary;
  const ai = S.ai || {};
  const focusHTML = f => f && f.length ? `<div class="focus"><h4>${icon('flag')}需要重点关注</h4><ul>${f.map(x => `<li>${esc(x)}</li>`).join('')}</ul></div>` : '';
  const summaryHTML = s ? `${focusHTML(s.focus)}<div class="md">${md(s.summary)}</div>`
    : ai.ready ? `<div class="empty">${esc(date)} 还没有日报<br>点右上角「生成日报」${ai.next_daily && ai.next_daily.startsWith(date) ? `，或等 ${esc(ai.next_daily.slice(11))} 自动生成` : ''}</div>`
    : `<div class="empty">还没有 AI 日报<br>在 <a href="#/settings">设置</a> 中接入 AI（DeepSeek、通义千问等），即可一键生成日报和周报<br>
      或在 Claude Code 中运行：<div class="cmd"><code id="cmdText">/mcp__dingtalk__daily_summary ${esc(date)}</code>
      <button class="btn btn-sm" id="copyCmd">${icon('copy')}复制</button></div></div>`;
  const aiButtons = ai.ready ? `
      <button class="btn btn-sm" data-ai-run="weekly" title="根据本周日报、事项和下周日程生成周报">${icon('spark')}生成周报</button>
      <button class="btn btn-sm btn-primary" data-ai-run="daily">${icon('spark')}${s ? '重新生成' : '生成日报'}</button>` : '';
  const w = r.weekly_summary;
  const weeklyHTML = w ? `
        <section class="card ai-card"><details class="weekly" ${[5, 6, 0].includes(new Date().getDay()) ? 'open' : ''}>
          <summary class="card-head"><span class="ai-badge">${icon('spark')}本周周报</span><span class="sub">${esc(w.date.replace('~', ' ~ '))}</span><div class="grow"></div>
          <span class="sub">${esc(`${w.author || 'agent'} · ${w.updated_at.slice(5, 16)}`)}</span>${icon('chevR', 'weekly-chev')}</summary>
          <div class="card-body">${focusHTML(w.focus)}<div class="md">${md(w.summary)}</div></div></details></section>` : '';

  const openTodos = r.open_todos || [];
  const todoHTML = openTodos.length ? openTodos.slice(0, 6).map(td => `
      <div class="list-item clickable" onclick="go('#/todos')">
        <div class="prio ${td.priority}"></div>
        <div class="li-main"><div class="li-title">${esc(td.title)}</div>
        <div class="li-sub">${td.kind === 'follow_up' ? '待跟踪' : '待办'}${td.due ? ' · 截止 ' + esc(td.due) : ''}${td.owner ? ' · ' + esc(td.owner) : ''}${td.ai_suggest ? ` <span class="tag tag-purple">AI：可能${td.ai_suggest === 'done' ? '已完成' : '不再需要'}</span>` : ''}</div></div>
      </div>`).join('') : '<div class="empty">没有未完成的事项 🎉</div>';

  const awaitingHTML = r.awaiting_my_reply.length ? r.awaiting_my_reply.map(p => {
    const c = convOf(p.cid);
    return `<div class="list-item clickable" data-cid="${esc(p.cid)}">${convAvatar(c || { title: p.conv_title, type: p.type, cid: p.cid }, 'sm')}
      <div class="li-main"><div class="li-title">${esc(p.conv_title)}<span class="tag ${p.type === 'group' ? 'tag-orange' : 'tag-blue'}">${p.type === 'group' ? '@我' : '单聊'}</span>
      <span class="muted" style="font-weight:400;font-size:12px">${esc(p.last_at.slice(11, 16))} · ${p.unanswered_count} 条</span></div>
      <div class="li-text">${esc(p.last_sender)}：${esc(p.last_text)}</div></div></div>`;
  }).join('') : '<div class="empty">没有等待回复的会话</div>';

  const mentionHTML = r.mentions_of_me.length ? r.mentions_of_me.map(m => `
      <div class="list-item clickable" data-cid="${esc(m.cid)}">${avatar(m.sender, { size: 'sm' })}
      <div class="li-main"><div class="li-title">${esc(m.sender)}<span class="muted" style="font-weight:400;font-size:12px">${esc(m.time.slice(11, 16))} · ${esc(m.conv_title)}</span>
      ${m.replied ? '<span class="tag tag-green">已回复</span>' : '<span class="tag tag-red">未回复</span>'}</div>
      <div class="li-text">${esc(m.text)}</div></div></div>`).join('') : '<div class="empty">没有@我的消息</div>';

  const maxScore = Math.max(1, ...r.people.list.map(p => p.score));
  const peopleHTML = r.people.list.length ? r.people.list.slice(0, 12).map(p => `
      <div class="person">${avatar(p.name, { size: 'sm', seed: String(p.uid) })}
      <div class="nm">${esc(p.name)}<small>${[p.direct_sent_by_me + p.direct_received ? `单聊 ${p.direct_sent_by_me + p.direct_received}` : '', p.mentioned_me ? `@我 ${p.mentioned_me}` : '', p.mentioned_by_me ? `我@ ${p.mentioned_by_me}` : '', ...(p.groups || [])].filter(Boolean).join(' · ')}</small></div>
      <div class="bar"><i style="width:${Math.round(p.score / maxScore * 100)}%"></i></div></div>`).join('') : '<div class="empty">无</div>';

  const groupHTML = r.groups.list.length ? r.groups.list.slice(0, 15).map(g => `
      <div class="group-row ${g.involved ? '' : 'dim'}" data-cid="${esc(g.cid)}">${avatar(g.title, { size: 'sm', group: true, project: g.is_project, seed: g.cid })}
      <div class="gr-main"><div class="gr-title">${esc(g.title)}</div><div class="gr-sub">${esc((g.top_speakers || []).join('  '))}</div></div>
      <div class="gr-nums"><span><b>${g.messages}</b> 条</span>${g.my_messages ? `<span class="tag tag-blue">我 ${g.my_messages}</span>` : ''}${g.mentions_me ? `<span class="tag tag-orange">@我 ${g.mentions_me}</span>` : ''}</div></div>`).join('') : '<div class="empty">无</div>';

  $('#dashBody').innerHTML = `
    <div class="dash">
      <div class="dash-col">
        <section class="card ai-card"><div class="card-head"><span class="ai-badge">${icon('spark')}AI 日报</span><div class="grow"></div>
          <span class="sub">${s ? esc(`${s.author || 'agent'} · ${s.updated_at.slice(5, 16)}`) : ''}</span>${aiButtons}</div>
          <div class="card-body"><div id="aiProgress"></div>${summaryHTML}</div></section>
        ${weeklyHTML}
        <section class="card"><div class="card-head"><h3>等待我回复</h3><span class="sub">${r.awaiting_my_reply.length || ''}</span></div><div class="card-body">${awaitingHTML}</div></section>
        <section class="card"><div class="card-head"><h3>@我的消息</h3><span class="sub">${r.mentions_of_me.length || ''}</span></div><div class="card-body">${mentionHTML}</div></section>
        <section class="card"><div class="card-head"><h3>群与项目</h3><span class="sub">参与 ${r.groups.involved_count} · 活跃 ${r.groups.active_count}</span></div><div class="card-body">${groupHTML}</div></section>
      </div>
      <div class="dash-col">
        ${cal.available ? `<section class="card"><div class="card-head"><h3>日程</h3><span class="sub">${cal.meeting_count} 个会议 · ${fmtMinutes(cal.meeting_minutes)}</span><div class="grow"></div>
          <a class="btn btn-ghost btn-sm" href="#/calendar?month=${date.slice(0, 7)}&day=${date}">月视图</a></div>
          <div class="card-body">${renderAgenda(cal)}</div></section>` : ''}
        <section class="card"><div class="card-head"><h3>待办与跟踪</h3><span class="sub">${openTodos.length} 项未完成</span><div class="grow"></div>
          <a class="btn btn-ghost btn-sm" href="#/todos">全部</a></div><div class="card-body">${todoHTML}</div></section>
        <section class="card"><div class="card-head"><h3>互动的人</h3><span class="sub">${r.people.count} 人</span></div><div class="card-body"><div class="people">${peopleHTML}</div></div></section>
      </div>
    </div>`;

  $('#dashBody').addEventListener('click', e => {
    const run = e.target.closest('[data-ai-run]');
    if (run) return startAI(run.dataset.aiRun, date);
    const el = e.target.closest('[data-cid]');
    if (el) go(`#/chat/${encodeURIComponent(el.dataset.cid)}`);
  });
  renderAIProgress(date);
  const cp = $('#copyCmd');
  if (cp) cp.onclick = () => navigator.clipboard.writeText($('#cmdText').textContent).then(() => toast('已复制'));
}

const KIND_TAG = { approval_todo: '<span class="tag tag-orange">审批</span>', leave: '<span class="tag">请假</span>', ai_minutes: '<span class="tag tag-purple">AI听记</span>', task: '<span class="tag tag-cyan">任务</span>' };
function eventTimeCell(e) {
  if (e.all_day) return '全天';
  return `${hm(e.start)}<small>${hm(e.end)}</small>`;
}
function eventItem(e) {
  const meta = [];
  if (e.meeting_room) meta.push(`${esc(e.meeting_room)}`); else if (e.location) meta.push(esc(e.location));
  if (e.has_video_conference) meta.push('视频会议');
  if (e.kind === 'meeting') meta.push(e.i_organized ? '我组织' : (e.organizer ? `组织者 ${esc(e.organizer)}` : ''));
  const tags = [KIND_TAG[e.kind] || ''];
  if (e.kind === 'meeting' && e.my_response === 'no_response' && !e.i_organized) tags.push('<span class="tag tag-red">未回复邀请</span>');
  if (e.status === 'cancelled') tags.push('<span class="tag">已取消</span>');
  if (e.status === 'done') tags.push('<span class="tag tag-green">已完成</span>');
  return `<div class="tl-item k-${e.kind} ${e.status === 'cancelled' || e.status === 'done' ? 'cancelled' : ''}">
    <div class="tl-time">${eventTimeCell(e)}</div><div class="tl-dot"></div>
    <div class="tl-body"><div class="tl-title">${esc(e.subject)} ${tags.join('')}</div>
    ${meta.filter(Boolean).length ? `<div class="tl-meta">${meta.filter(Boolean).join(' · ')}</div>` : ''}
    ${e.comment ? `<div class="tl-comment">${esc(e.comment)}</div>` : ''}</div></div>`;
}
function renderAgenda(c) {
  const today = [...c.meetings, ...c.other_entries, ...c.cancelled_meetings].sort((a, b) => a.start - b.start);
  let h = `<div class="timeline">${today.length ? today.map(eventItem).join('') : '<div class="empty">当天没有日程</div>'}</div>`;
  if (c.next_day_label) {
    h += `<div class="sub-title">明天 · ${esc(c.next_day_label.slice(5))}</div><div class="timeline">${c.next_day.length ? c.next_day.map(eventItem).join('') : '<div class="empty" style="padding:8px 0">没有日程</div>'}</div>`;
  }
  if (c.pending_approvals.length) {
    h += `<div class="sub-title">未处理的审批待办</div>` + c.pending_approvals.map(e => `
      <div class="list-item"><div class="li-main"><div class="li-title">${esc(e.subject)} ${e.status === 'in_progress' ? '<span class="tag tag-orange">处理中</span>' : ''}</div>
      <div class="li-sub">${esc(e.start_str.slice(5, 10))}</div></div></div>`).join('');
  }
  return h;
}

// ===================================================================== AI
async function loadAIStatus() {
  try { return await api('/api/ai/status'); } catch (_) { return null; }
}
async function startAI(kind, date) {
  try {
    S.ai = await api('/api/ai/run', { method: 'POST', json: { kind, date } });
    renderAIProgress(date);
    pollAI();
  } catch (e) { toast(e.message, 4000); }
}
let aiTimer = 0;
function pollAI() {
  clearTimeout(aiTimer);
  aiTimer = setTimeout(async () => {
    const st = await loadAIStatus();
    if (!st) return pollAI();
    const wasRunning = S.ai && S.ai.running;
    S.ai = st;
    if (st.running) { renderAIProgress(); return pollAI(); }
    if (wasRunning && st.last) {
      const what = st.last.kind === 'weekly' ? '周报' : '日报';
      if (st.last.ok) toast(`${what}已生成` + (st.last.kind === 'daily' ? `：新增 ${st.last.new_items} 项事项，AI 建议更新 ${st.last.suggestions} 项` : ''), 3500);
      else toast(`${what}生成失败：${st.last.error}`, 6000);
      updateTodoBadge();
      if (parseRoute().name === 'report') render();
    }
  }, 1500);
}
function renderAIProgress(date) {
  const box = $('#aiProgress');
  if (!box) return;
  const st = S.ai || {};
  $$('[data-ai-run]').forEach(b => (b.disabled = !!st.running));
  if (st.running) {
    const secs = Math.max(0, Math.round((Date.now() - new Date(st.started_at.replace(' ', 'T')).getTime()) / 1000));
    box.innerHTML = `<div class="ai-progress"><span class="spinner"></span>
      <b>正在生成${st.kind === 'weekly' ? '周报' : '日报'}</b><span class="muted">${esc(st.date_key.replace('~', ' ~ '))} · ${esc(st.stage)}${st.chars ? ` · 已生成 ${st.chars} 字` : ''} · ${secs} 秒</span></div>`;
    return;
  }
  const l = st.last;
  const d = date || (parseRoute().params.get('date') || ymd(new Date()));
  box.innerHTML = l && !l.ok && (l.date_key === d || l.date_key.includes('~'))
    ? `<div class="ai-progress error">${esc((l.kind === 'weekly' ? '周报' : '日报') + '生成失败：' + l.error)}<span class="muted">${esc(l.finished_at.slice(5, 16))}</span></div>` : '';
}

// ================================================================ AI CHAT
const AC = { session: 0, sessions: [], mentions: new Map(), images: [], streaming: null, pop: { items: [], idx: 0 } };
// Attached images are scaled down in the browser before sending: vision
// models downscale anyway, and smaller uploads keep requests fast and cheap.
const AI_IMG = { max: 4, side: 1568, quality: 0.85, maxFile: 20 << 20 };
// Prompts with a 「…」 placeholder are filled into the input with the
// placeholder selected, instead of being sent right away.
const AI_SUGGESTIONS = [
  ['我的工作', [
    ['总结今天的工作', '帮我总结一下今天的工作：做了什么、有哪些进展、还有什么没处理完'],
    ['答应别人的事', '最近一周我答应过别人、但还没完成的事情有哪些？'],
    ['需要回复的消息', '有哪些消息在等我回复？按紧急程度排序，并帮我起草回复'],
    ['明天的安排', '结合明天的日程和未完成的待办，帮我安排一下明天的工作'],
  ]],
  ['从聊天记录里找答案', [
    ['问题怎么解决的', '之前「问题或报错关键词」是怎么解决的？原因是什么、谁处理的？'],
    ['来龙去脉', '「项目或事情」的来龙去脉：什么时候开始的、有哪些关键决定、现在进展到哪了？'],
    ['谁熟悉这块', '谁比较熟悉「技术或业务」？之前都是谁在处理相关的问题？'],
    ['找文件或链接', '之前有人发过「文档、链接或安装包」吗？在哪个群、谁发的？'],
  ]],
];

async function viewAI(r, alive) {
  AC.session = +(r.arg || 0);
  const ready = await loadAIStatus();
  await loadConvs();
  if (!alive()) return;
  $('#view').innerHTML = `<div class="ai-page">
    <aside class="ai-side">
      <button class="btn btn-primary ai-new" id="aiNew">${icon('plus')}新对话</button>
      <div class="ai-sessions" id="aiSessions"></div>
    </aside>
    <section class="ai-main">
      <div class="ai-msgs" id="aiMsgs"></div>
      <div class="ai-composer" id="aiComposer">
        <div class="ai-attach" id="aiAttach" hidden></div>
        <div class="ai-input-wrap">
          <textarea id="aiInput" rows="1" placeholder="问问 AI，例如：之前「数据库连接超时」是怎么解决的？也可以 @ 某人或某群"></textarea>
          <div class="mention-pop" id="mentionPop" hidden></div>
        </div>
        <div class="ai-bar">
          <select class="input" id="aiDays" title="@ 的人或群附带多长时间的聊天记录">
            <option value="1">今天</option><option value="3">近 3 天</option><option value="7" selected>近 7 天</option><option value="30">近 30 天</option>
          </select>
          <button class="btn btn-ghost btn-icon" id="aiImgBtn" title="添加图片（也可以粘贴或拖入）">${icon('image')}</button>
          <input type="file" id="aiImgFile" accept="image/png,image/jpeg,image/webp,image/gif" multiple hidden>
          <span class="muted ai-hint">@ 的人或群会附上这段时间的聊天记录 · 可粘贴或拖入图片 · Enter 发送，Shift+Enter 换行</span>
          <div class="grow"></div>
          <button class="btn btn-primary" id="aiSend">${icon('send')}发送</button>
        </div>
      </div>
    </section></div>`;
  try { const d = localStorage.getItem('ai-days'); if (d) $('#aiDays').value = d; } catch (_) {}
  $('#aiDays').onchange = e => { try { localStorage.setItem('ai-days', e.target.value); } catch (_) {} };
  $('#aiNew').onclick = () => { if (!AC.streaming) { AC.mentions.clear(); AC.images = []; go('#/ai'); } };
  bindComposer(ready);
  await refreshSessions();
  if (AC.session) await loadSession(AC.session);
  else renderAIEmpty(ready);
  const mention = r.params.get('mention');
  if (mention) { const c = convOf(mention); if (c) insertMention(c, true); history.replaceState(null, '', '#/ai'); }
  $('#aiInput').focus();
}

function renderAIEmpty(st) {
  const box = $('#aiMsgs');
  if (!st || !st.ready) {
    box.innerHTML = `<div class="ai-welcome"><div class="ai-logo">${icon('spark')}</div><h2>AI 对话</h2>
      <p>基于你的钉钉聊天记录、日程和待办，和 AI 一起分析、讨论、处理工作。</p>
      <p class="muted">需要先在 <a href="#/settings">设置 → AI 助手</a> 中接入模型（DeepSeek、通义千问、Kimi 等）。</p></div>`;
    return;
  }
  box.innerHTML = `<div class="ai-welcome"><div class="ai-logo">${icon('spark')}</div><h2>有什么可以帮你？</h2>
    <p class="muted">AI 可以查阅你的聊天记录、日程和待办。输入 <b>@</b> 选择某人或某群，会附上与他们的聊天记录。</p>
    ${AI_SUGGESTIONS.map(([group, items]) => `<div class="ai-sg-group">${esc(group)}</div>
      <div class="ai-suggest">${items.map(([t, q]) => `<button class="ai-sg" data-q="${esc(q)}"><b>${esc(t)}</b><span>${esc(q)}</span></button>`).join('')}</div>`).join('')}
    <p class="muted ai-privacy">🔒 你提问时，相关的聊天内容和附上的图片会发送给你配置的模型服务商。</p></div>`;
  box.onclick = e => {
    const b = e.target.closest('.ai-sg'); if (!b) return;
    const q = b.dataset.q, input = $('#aiInput');
    input.value = q;
    const i = q.indexOf('「'), j = q.indexOf('」');
    if (i < 0 || j < i) { sendAI(); return; }
    input.focus();
    input.setSelectionRange(i + 1, j); // select the placeholder so typing replaces it
    input.dispatchEvent(new Event('input'));
  };
}

async function refreshSessions() {
  try { AC.sessions = (await api('/api/chat/sessions')).sessions; } catch (_) { AC.sessions = []; }
  renderSessions();
}
function renderSessions() {
  const box = $('#aiSessions'); if (!box) return;
  box.innerHTML = AC.sessions.length ? AC.sessions.map(x => `
    <div class="ai-sess ${x.id === AC.session ? 'active' : ''}" data-id="${x.id}">
      <div class="ai-sess-title">${esc(x.title || '新对话')}</div>
      <div class="ai-sess-time">${esc(x.updated_at.slice(5, 16))}</div>
      <button class="btn btn-ghost btn-icon btn-sm ai-sess-del" data-del="${x.id}" title="删除">${icon('trash')}</button>
    </div>`).join('') : '<div class="empty" style="padding:24px 8px">还没有对话</div>';
  box.onclick = async e => {
    const del = e.target.closest('[data-del]');
    if (del) {
      e.stopPropagation();
      if (!confirm('删除这个对话？')) return;
      await api(`/api/chat/sessions/${del.dataset.del}`, { method: 'DELETE' });
      if (+del.dataset.del === AC.session) go('#/ai'); else refreshSessions();
      return;
    }
    const it = e.target.closest('.ai-sess');
    if (it && !AC.streaming) go(`#/ai/${it.dataset.id}`);
  };
}

async function loadSession(id) {
  try {
    const d = await api(`/api/chat/sessions/${id}`);
    $('#aiMsgs').innerHTML = d.messages.map(m => m.role === 'user' ? userBubble(m) : aiBubble(m)).join('');
    scrollAI(true);
  } catch (e) { toast(e.message); go('#/ai'); }
}

function highlightMentions(text, mentions) {
  let h = esc(text);
  for (const m of (mentions || [])) {
    const t = esc('@' + m.title);
    const link = m.cid && !m.cid.startsWith('u:') ? ` data-cid="${esc(m.cid)}"` : '';
    h = h.split(t).join(`<span class="ai-at ${m.kind === 'group' ? 'grp' : ''}"${link}>${t}</span>`);
  }
  return h;
}
function userImages(srcs) {
  return srcs.length ? `<div class="ai-imgs">${srcs.map(u => `<a href="${esc(u)}" target="_blank" rel="noopener"><img src="${esc(u)}" alt="图片" loading="lazy"></a>`).join('')}</div>` : '';
}
function userBubble(m) {
  const meta = m.meta || {};
  const ctx = meta.mentions && meta.mentions.length ? `<div class="ai-ctx">${icon('link')}已附上 ${meta.mentions.map(x => esc(x.title)).join('、')} ${meta.days === 1 ? '今天' : `近 ${meta.days || 7} 天`}的聊天记录</div>` : '';
  const imgs = userImages((meta.images || []).map(n => `/api/chat/images/${m.session_id}/${encodeURIComponent(n)}`));
  return `<div class="ai-row user">${imgs}<div class="ai-bubble">${highlightMentions(m.content, meta.mentions).replace(/\n/g, '<br>')}</div>${ctx}</div>`;
}
function toolLine(t, running) {
  return `<div class="ai-tool ${t.error ? 'err' : ''}">${running ? '<span class="spinner"></span>' : icon(t.error ? 'close' : 'check')}
    <span>${esc(t.title || t.name)}</span>${t.summary && !running ? `<span class="muted">· ${esc(t.error ? t.error : t.summary)}</span>` : ''}</div>`;
}
function aiBubble(m, streaming = false) {
  const meta = m.meta || {};
  const tools = (meta.tools || []).map(t => toolLine(t, false)).join('');
  return `<div class="ai-row assistant" ${m.id ? `data-id="${m.id}"` : ''}>
    <div class="ai-avatar">${icon('spark')}</div>
    <div class="ai-answer">
      ${tools ? `<div class="ai-tools">${tools}</div>` : '<div class="ai-tools"></div>'}
      ${meta.note ? `<div class="ai-note">${esc(meta.note)}</div>` : ''}
      <div class="ai-content md">${m.content ? md(m.content) : streaming ? '<span class="ai-typing"><i></i><i></i><i></i></span>' : ''}</div>
      ${meta.error ? `<div class="ai-error">${esc(meta.error)}</div>` : ''}
      ${!streaming && m.content ? `<div class="ai-actions"><button class="btn btn-ghost btn-sm" data-copy-md="1">${icon('copy')}复制</button>
        ${meta.model ? `<span class="muted">${esc(meta.model)}</span>` : ''}</div>` : ''}
    </div></div>`;
}
function scrollAI(force) {
  const box = $('#aiMsgs'); if (!box) return;
  if (force || box.scrollHeight - box.scrollTop - box.clientHeight < 160) box.scrollTop = box.scrollHeight;
}

// ---- composer and @mentions
function bindComposer(st) {
  const input = $('#aiInput'), pop = $('#mentionPop');
  const autosize = () => { input.style.height = 'auto'; input.style.height = Math.min(input.scrollHeight, 220) + 'px'; };
  input.addEventListener('input', () => { autosize(); updateMentionPop(); });
  input.addEventListener('click', updateMentionPop);
  input.addEventListener('keydown', e => {
    if (!pop.hidden) {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); AC.pop.idx = (AC.pop.idx + (e.key === 'ArrowDown' ? 1 : -1) + AC.pop.items.length) % AC.pop.items.length; renderMentionPop(); return; }
      if (e.key === 'Enter' || e.key === 'Tab') { e.preventDefault(); insertMention(AC.pop.items[AC.pop.idx]); return; }
      if (e.key === 'Escape') { pop.hidden = true; return; }
    }
    if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) { e.preventDefault(); sendAI(); }
  });
  pop.addEventListener('mousedown', e => { const it = e.target.closest('[data-i]'); if (it) { e.preventDefault(); insertMention(AC.pop.items[+it.dataset.i]); } });
  $('#aiSend').onclick = () => AC.streaming ? stopAI() : sendAI();
  if (!st || !st.ready) { input.disabled = true; $('#aiSend').disabled = true; $('#aiImgBtn').disabled = true; input.placeholder = '请先在设置中接入 AI 模型'; }
  else bindImageInput();
  renderAttach();
  $('#aiMsgs').addEventListener('click', e => {
    const cp = e.target.closest('[data-copy-md]');
    if (cp) {
      const id = cp.closest('.ai-row').dataset.id;
      api(`/api/chat/sessions/${AC.session}`).then(d => {
        const m = d.messages.find(x => String(x.id) === id);
        if (m) navigator.clipboard.writeText(m.content).then(() => toast('已复制'));
      });
    }
    const at = e.target.closest('.ai-at[data-cid]');
    if (at) go(`#/chat/${encodeURIComponent(at.dataset.cid)}`);
  });
}
function mentionQuery() {
  const input = $('#aiInput');
  const before = input.value.slice(0, input.selectionStart);
  const m = before.match(/@([^\s@]{0,24})$/);
  return m ? { q: m[1], start: before.length - m[0].length } : null;
}
// The @ list: matching conversations (groups and people we chat with), plus
// contacts we only know from groups, fetched from the server as you type.
let peopleTimer = 0, peopleSeq = 0;
function updateMentionPop() {
  const pop = $('#mentionPop'), mq = mentionQuery();
  if (!mq) { pop.hidden = true; return; }
  const q = mq.q.toLowerCase();
  const list = S.convs.filter(c => !c.is_notification && !c.is_bot && c.title && (!q || c.title.toLowerCase().includes(q)));
  list.sort((a, b) => (b.title.toLowerCase().startsWith(q) - a.title.toLowerCase().startsWith(q)) || (b.last_modify - a.last_modify));
  const local = list.slice(0, q ? 6 : 8);
  showMentionItems(local);
  clearTimeout(peopleTimer);
  if (!q) return;
  const seq = ++peopleSeq;
  peopleTimer = setTimeout(async () => {
    let people = [];
    try { people = (await api(`/api/chat/people?q=${encodeURIComponent(mq.q)}&no_chat=1&limit=${10 - local.length}`)).people; } catch (_) {}
    const now = mentionQuery();
    if (seq !== peopleSeq || !now || now.q !== mq.q) return;
    const extra = people.filter(p => !p.cid).slice(0, 10 - local.length)
      .map(p => ({ kind: 'user', uid: String(p.uid), title: p.name }));
    if (extra.length) showMentionItems([...local, ...extra], true);
  }, 150);
}
function showMentionItems(items, keepIdx = false) {
  const pop = $('#mentionPop');
  AC.pop.items = items;
  if (!keepIdx || AC.pop.idx >= items.length) AC.pop.idx = 0;
  if (!items.length) { pop.hidden = true; return; }
  renderMentionPop(); pop.hidden = false;
}
function renderMentionPop() {
  $('#mentionPop').innerHTML = AC.pop.items.map((c, i) => {
    const av = c.kind === 'user' ? avatar(c.title, { size: 'xs', seed: c.uid }) : convAvatar(c, 'xs');
    const tag = c.kind === 'user' ? '<span class="tag" title="没有单聊，会附上他在群里的发言">人 · 群成员</span>'
      : `<span class="tag ${c.type === 'group' ? (c.is_project ? 'tag-purple' : 'tag-orange') : 'tag-blue'}">${c.type === 'group' ? (c.is_project ? '项目群' : '群') : '人'}</span>`;
    return `<div class="mp-item ${i === AC.pop.idx ? 'active' : ''}" data-i="${i}">${av}<span class="mp-title">${esc(c.title)}</span>${tag}</div>`;
  }).join('');
}
function insertMention(c, append = false) {
  if (!c) return;
  const input = $('#aiInput');
  const token = '@' + c.title + ' ';
  if (append) {
    input.value = (input.value && !input.value.endsWith(' ') ? input.value + ' ' : input.value) + token;
  } else {
    const mq = mentionQuery(); if (!mq) return;
    const pos = input.selectionStart;
    input.value = input.value.slice(0, mq.start) + token + input.value.slice(pos);
    const caret = mq.start + token.length;
    input.setSelectionRange(caret, caret);
  }
  AC.mentions.set(c.title, c.kind === 'user' ? 'u:' + c.uid : c.cid);
  $('#mentionPop').hidden = true;
  input.focus();
  input.dispatchEvent(new Event('input'));
}

// ---- image attachments: button, paste and drag-and-drop
function bindImageInput() {
  const file = $('#aiImgFile'), composer = $('#aiComposer');
  $('#aiImgBtn').onclick = () => file.click();
  file.onchange = () => { addImages([...file.files]); file.value = ''; };
  $('#aiInput').addEventListener('paste', e => {
    const files = [...(e.clipboardData?.items || [])].filter(it => it.kind === 'file' && it.type.startsWith('image/')).map(it => it.getAsFile()).filter(Boolean);
    if (files.length) { e.preventDefault(); addImages(files); }
  });
  composer.addEventListener('dragover', e => { if ([...(e.dataTransfer?.types || [])].includes('Files')) { e.preventDefault(); composer.classList.add('drop'); } });
  composer.addEventListener('dragleave', e => { if (!composer.contains(e.relatedTarget)) composer.classList.remove('drop'); });
  composer.addEventListener('drop', e => {
    composer.classList.remove('drop');
    if (!e.dataTransfer?.files?.length) return;
    e.preventDefault();
    addImages([...e.dataTransfer.files]);
  });
  $('#aiAttach').addEventListener('click', e => {
    const rm = e.target.closest('[data-rm]'); if (!rm) return;
    AC.images = AC.images.filter((_, i) => i !== +rm.dataset.rm);
    renderAttach();
  });
}
async function addImages(files) {
  for (const f of files) {
    if (AC.images.length >= AI_IMG.max) { toast(`一次最多发送 ${AI_IMG.max} 张图片`); break; }
    if (!/^image\/(png|jpeg|webp|gif)$/.test(f.type)) { toast(`不支持的文件：${f.name || f.type || '未知类型'}`); continue; }
    if (f.size > AI_IMG.maxFile) { toast(`图片太大：${f.name}`); continue; }
    try { AC.images = [...AC.images, await shrinkImage(f)]; } catch (_) { toast(`无法读取图片：${f.name}`); }
    renderAttach();
  }
  $('#aiInput').focus();
}
// shrinkImage scales an image so its long side is at most AI_IMG.side and
// re-encodes it as JPEG on white (transparent areas would turn black).
async function shrinkImage(file) {
  const bmp = await createImageBitmap(file);
  try {
    const k = Math.min(1, AI_IMG.side / Math.max(bmp.width, bmp.height));
    const w = Math.max(1, Math.round(bmp.width * k)), h = Math.max(1, Math.round(bmp.height * k));
    const cv = document.createElement('canvas');
    cv.width = w; cv.height = h;
    const g = cv.getContext('2d');
    g.fillStyle = '#fff'; g.fillRect(0, 0, w, h);
    g.drawImage(bmp, 0, 0, w, h);
    return cv.toDataURL('image/jpeg', AI_IMG.quality);
  } finally { bmp.close(); }
}
function renderAttach() {
  const box = $('#aiAttach'); if (!box) return;
  box.hidden = !AC.images.length;
  box.innerHTML = AC.images.map((u, i) => `<div class="ai-thumb"><img src="${u}" alt="图片 ${i + 1}">
    <button class="ai-thumb-rm" data-rm="${i}" title="移除">${icon('close')}</button></div>`).join('');
}

// ---- sending and streaming
async function sendAI() {
  const input = $('#aiInput');
  const text = input.value.trim();
  const images = AC.images;
  if ((!text && !images.length) || AC.streaming) return;
  const mentions = [...AC.mentions].filter(([t]) => text.includes('@' + t)).map(([, cid]) => cid);
  input.value = ''; input.style.height = 'auto';
  AC.images = []; renderAttach();
  const box = $('#aiMsgs');
  if (box.querySelector('.ai-welcome')) box.innerHTML = '';
  box.insertAdjacentHTML('beforeend', `<div class="ai-row user pending">${userImages(images)}<div class="ai-bubble">${esc(text || '请看图片').replace(/\n/g, '<br>')}</div></div>`);
  box.insertAdjacentHTML('beforeend', aiBubble({ content: '', meta: {} }, true).replace('class="ai-row assistant"', 'class="ai-row assistant live"'));
  scrollAI(true);
  const ctrl = new AbortController();
  AC.streaming = ctrl;
  setSendState(true);
  const live = () => $('#aiMsgs .ai-row.live');
  let text_ = '', raf = 0;
  const paint = () => { raf = 0; const el = live(); if (el) { el.querySelector('.ai-content').innerHTML = md(text_) || '<span class="ai-typing"><i></i><i></i><i></i></span>'; scrollAI(); } };
  const onEvent = ev => {
    const el = live();
    switch (ev.type) {
      case 'session':
        if (AC.session !== ev.session.id) { AC.session = ev.session.id; history.replaceState(null, '', `#/ai/${ev.session.id}`); }
        refreshSessions(); break;
      case 'user': {
        const p = $('#aiMsgs .ai-row.user.pending');
        if (p) p.outerHTML = userBubble(ev.message);
        break;
      }
      case 'tool_start':
        if (el) { text_ = ''; paint(); el.querySelector('.ai-tools').insertAdjacentHTML('beforeend', toolLine(ev.tool, true).replace('class="ai-tool', `data-ti="${ev.tool_index}" class="ai-tool`)); scrollAI(); }
        break;
      case 'tool_done':
        if (el) { const t = el.querySelector(`[data-ti="${ev.tool_index}"]`); if (t) t.outerHTML = toolLine(ev.tool, false); }
        break;
      case 'delta': text_ += ev.text; if (!raf) raf = requestAnimationFrame(paint); break;
      case 'note': if (el) el.querySelector('.ai-tools').insertAdjacentHTML('afterend', `<div class="ai-note">${esc(ev.text)}</div>`); break;
      case 'done': if (el) el.outerHTML = aiBubble(ev.message); scrollAI(); updateTodoBadge(); break;
      case 'error':
        if (el) { el.classList.remove('live'); el.querySelector('.ai-content').innerHTML = md(text_); el.querySelector('.ai-answer').insertAdjacentHTML('beforeend', `<div class="ai-error">${esc(ev.error)}</div>`); }
        break;
    }
  };
  try {
    const res = await fetch('/api/chat/send', { method: 'POST', headers: { 'Content-Type': 'application/json' }, signal: ctrl.signal,
      body: JSON.stringify({ session_id: AC.session, text, mentions, images, days: +$('#aiDays').value }) });
    if (!res.ok) { let msg = res.statusText; try { msg = (await res.json()).detail || msg; } catch (_) {} throw new Error(msg); }
    const reader = res.body.getReader(), dec = new TextDecoder();
    let buf = '';
    for (;;) {
      const { value, done } = await reader.read();
      if (done) break;
      buf += dec.decode(value, { stream: true });
      let i;
      while ((i = buf.indexOf('\n\n')) >= 0) {
        const chunk = buf.slice(0, i); buf = buf.slice(i + 2);
        for (const line of chunk.split('\n')) if (line.startsWith('data: ')) { try { onEvent(JSON.parse(line.slice(6))); } catch (_) {} }
      }
    }
  } catch (e) {
    if (e.name !== 'AbortError') onEvent({ type: 'error', error: e.message });
  }
  const el = live();
  if (el) { // stopped or failed before "done": reload what the server saved
    el.classList.remove('live');
    if (AC.session) setTimeout(() => { if (!AC.streaming) loadSession(AC.session); }, 400);
  }
  AC.streaming = null;
  setSendState(false);
  refreshSessions();
}
function stopAI() { if (AC.streaming) AC.streaming.abort(); }
function setSendState(busy) {
  const b = $('#aiSend'); if (!b) return;
  b.innerHTML = busy ? `${icon('stop')}停止` : `${icon('send')}发送`;
  b.classList.toggle('btn-danger', busy);
}

// ================================================================== TODOS
async function viewTodos(r, alive) {
  const status = r.params.get('status') || 'open';
  const kind = r.params.get('kind') || '';
  $('#view').innerHTML = `<div class="page"><div class="page-inner" style="max-width:920px">
    <div class="page-head"><h1>待办</h1><span class="muted" id="todoCount"></span><div class="grow"></div>
      <div class="seg" id="kindSeg"><button data-k="">全部</button><button data-k="todo">待办</button><button data-k="follow_up">待跟踪</button></div>
      <div class="seg" id="statusSeg"><button data-s="open">未完成</button><button data-s="done">已完成</button><button data-s="dropped">已放弃</button><button data-s="all">全部</button></div>
    </div>
    <form class="card todo-add" id="todoAdd">
      <select class="input" id="taKind"><option value="todo">待办</option><option value="follow_up">待跟踪</option></select>
      <input class="input title" id="taTitle" placeholder="添加事项，回车保存…" autocomplete="off">
      <input class="input" type="date" id="taDue" title="截止日期" style="width:140px">
      <select class="input" id="taPrio"><option value="high">高优先级</option><option value="medium" selected>中优先级</option><option value="low">低优先级</option></select>
      <button class="btn btn-primary" type="submit">${icon('plus')}添加</button>
    </form>
    <div id="todoList"><div class="card skeleton" style="height:180px"></div></div></div></div>`;
  $$('#statusSeg button').forEach(b => b.classList.toggle('active', b.dataset.s === status));
  $$('#kindSeg button').forEach(b => b.classList.toggle('active', b.dataset.k === kind));
  $('#statusSeg').onclick = e => e.target.dataset.s && go(`#/todos?status=${e.target.dataset.s}${kind ? '&kind=' + kind : ''}`);
  $('#kindSeg').onclick = e => e.target.dataset.k !== undefined && go(`#/todos?status=${status}${e.target.dataset.k ? '&kind=' + e.target.dataset.k : ''}`);
  $('#todoAdd').onsubmit = async e => {
    e.preventDefault();
    const title = $('#taTitle').value.trim(); if (!title) return;
    await api('/api/agent/todos', { method: 'POST', json: { items: [{ title, kind: $('#taKind').value, priority: $('#taPrio').value, due: $('#taDue').value, source_date: ymd(new Date()) }] } });
    $('#taTitle').value = ''; toast('已添加'); refreshTodos(status, kind); updateTodoBadge();
  };
  await loadConvs();
  if (!alive()) return;
  await refreshTodos(status, kind);
}

const PRIO = { high: ['高', 'tag-red'], medium: ['中', 'tag-orange'], low: ['低', ''] };
async function refreshTodos(status, kind) {
  const d = await api(`/api/agent/todos?status=${status}${kind ? '&kind=' + kind : ''}&limit=500`);
  $('#todoCount').textContent = `${d.todos.length} 项`;
  const groups = status === 'open'
    ? [['high', '高优先级'], ['medium', '中优先级'], ['low', '低优先级']].map(([p, label]) => [label, d.todos.filter(t => t.priority === p)])
    : [['', d.todos]];
  const todoRow = t => `
    <div class="todo ${t.status}" data-id="${t.id}">
      <button class="check" data-act="toggle" title="${t.status === 'done' ? '标记为未完成' : '标记完成'}">${icon('check')}</button>
      <div class="todo-main">
        <div class="todo-title">${esc(t.title)}</div>
        <div class="todo-meta">
          <span class="tag ${t.kind === 'follow_up' ? 'tag-purple' : 'tag-blue'}">${t.kind === 'follow_up' ? '待跟踪' : '待办'}</span>
          <span class="tag ${PRIO[t.priority][1]}">${PRIO[t.priority][0]}</span>
          ${t.due ? `<span class="tag ${t.status === 'open' && t.due < ymd(new Date()) ? 'tag-red' : ''}">${icon('clock').replace('<svg', '<svg width="11" height="11"')}${esc(t.due)}</span>` : ''}
          ${t.owner ? `<span>👤 ${esc(t.owner)}</span>` : ''}
          ${t.cid ? `<span class="src" data-cid="${esc(t.cid)}">💬 ${esc(t.conv_title || (convOf(t.cid) || {}).title || '来源会话')}</span>` : ''}
          ${t.source_date ? `<span>· ${esc(t.source_date)}</span>` : ''}
        </div>
        ${t.detail ? `<div class="todo-detail">${esc(t.detail)}</div>` : ''}
        ${t.ai_suggest && t.status === 'open' ? `<div class="todo-ai">${icon('spark')}<span><b>AI 认为${t.ai_suggest === 'done' ? '可能已完成' : '可能不再需要'}</b>${t.ai_note ? '：' + esc(t.ai_note) : ''}</span>
          <button class="btn btn-sm btn-primary" data-act="ai-accept" data-s="${esc(t.ai_suggest)}">${t.ai_suggest === 'done' ? '确认完成' : '确认放弃'}</button>
          <button class="btn btn-sm btn-ghost" data-act="ai-dismiss">忽略</button></div>` : ''}
      </div>
      <div class="todo-actions">
        <select class="input" data-act="prio" title="优先级">${['high', 'medium', 'low'].map(p => `<option value="${p}" ${p === t.priority ? 'selected' : ''}>${PRIO[p][0]}</option>`).join('')}</select>
        ${t.status === 'open' ? `<button class="btn btn-ghost btn-sm" data-act="drop">放弃</button>` : ''}
        <button class="btn btn-ghost btn-sm btn-icon" data-act="del" title="删除">${icon('trash')}</button>
      </div>
    </div>`;
  const html = groups.filter(([, l]) => l.length).map(([label, l]) =>
    `${label ? `<div class="todo-group-title">${esc(label)} · ${l.length}</div>` : ''}<div class="card">${l.map(todoRow).join('')}</div>`).join('');
  const list = $('#todoList');
  list.innerHTML = html || `<div class="card"><div class="empty">${status === 'open' ? '没有未完成的事项 🎉<br>运行每日总结后，AI 提取的待办和待跟踪事项会出现在这里' : '无'}</div></div>`;
  list.onclick = async e => {
    const src = e.target.closest('.src');
    if (src) return go(`#/chat/${encodeURIComponent(src.dataset.cid)}`);
    const row = e.target.closest('.todo'); const act = e.target.closest('[data-act]');
    if (!row || !act || act.tagName === 'SELECT') return;
    const id = row.dataset.id;
    try {
      if (act.dataset.act === 'toggle') await api(`/api/agent/todos/${id}`, { method: 'PATCH', json: { status: row.classList.contains('open') ? 'done' : 'open' } });
      if (act.dataset.act === 'drop') await api(`/api/agent/todos/${id}`, { method: 'PATCH', json: { status: 'dropped' } });
      if (act.dataset.act === 'ai-accept') await api(`/api/agent/todos/${id}`, { method: 'PATCH', json: { status: act.dataset.s } });
      if (act.dataset.act === 'ai-dismiss') await api(`/api/agent/todos/${id}`, { method: 'PATCH', json: { dismiss_ai: true } });
      if (act.dataset.act === 'del') { if (!confirm('删除这个事项？')) return; await api(`/api/agent/todos/${id}`, { method: 'DELETE' }); }
      refreshTodos(status, kind); updateTodoBadge();
    } catch (err) { toast(err.message); }
  };
  list.onchange = async e => {
    if (e.target.dataset.act !== 'prio') return;
    await api(`/api/agent/todos/${e.target.closest('.todo').dataset.id}`, { method: 'PATCH', json: { priority: e.target.value } });
    refreshTodos(status, kind);
  };
}

async function updateTodoBadge() {
  try {
    const d = await api('/api/agent/todos?status=open&limit=500');
    const b = $('#todoBadge');
    b.hidden = !d.todos.length;
    b.textContent = d.todos.length > 99 ? '99+' : d.todos.length;
  } catch (_) {}
}

// =============================================================== CALENDAR
async function viewCalendar(r, alive) {
  const today = new Date();
  const month = r.params.get('month') || `${today.getFullYear()}-${pad(today.getMonth() + 1)}`;
  const [y, m] = month.split('-').map(Number);
  const first = new Date(y, m - 1, 1);
  const gridStart = new Date(first); gridStart.setDate(1 - first.getDay());
  const weeks = Math.ceil((first.getDay() + new Date(y, m, 0).getDate()) / 7);
  const gridEnd = new Date(gridStart); gridEnd.setDate(gridStart.getDate() + weeks * 7 - 1);
  let selected = r.params.get('day') || (today.getFullYear() === y && today.getMonth() === m - 1 ? ymd(today) : ymd(first));

  $('#view').innerHTML = `<div class="cal-wrap">
    <div class="cal-main">
      <div class="cal-head">
        <button class="btn" id="cToday">今天</button>
        <button class="btn btn-ghost btn-icon" id="cPrev">${icon('chevL')}</button>
        <button class="btn btn-ghost btn-icon" id="cNext">${icon('chevR')}</button>
        <h1>${y}年${m}月</h1><div class="grow"></div>
        <div class="cal-legend"><span><i style="background:var(--success)"></i>会议</span><span><i style="background:var(--warning)"></i>审批待办</span><span><i style="background:var(--purple)"></i>AI听记</span><span><i style="background:var(--text-3)"></i>请假</span></div>
      </div>
      <div class="cal-week">${['周日', '周一', '周二', '周三', '周四', '周五', '周六'].map(w => `<div>${w}</div>`).join('')}</div>
      <div class="cal-grid" id="calGrid" style="grid-template-rows:repeat(${weeks},1fr)"></div>
    </div>
    <aside class="cal-side"><div class="cal-side-head"><h2 id="sideTitle"></h2><div class="sub" id="sideSub"></div></div><div class="cal-side-body" id="sideBody"></div></aside>
  </div>`;
  const nav = dm => { const d = new Date(y, m - 1 + dm, 1); go(`#/calendar?month=${d.getFullYear()}-${pad(d.getMonth() + 1)}`); };
  $('#cPrev').onclick = () => nav(-1);
  $('#cNext').onclick = () => nav(1);
  $('#cToday').onclick = () => go('#/calendar');

  const d = await api(`/api/agent/calendar?from=${ymd(gridStart)}&to=${ymd(gridEnd)}&include_closed=1`);
  if (!alive()) return;
  if (!d.available) { $('#calGrid').innerHTML = '<div class="empty" style="grid-column:1/-1">本机没有可用的钉钉日历数据</div>'; return; }
  const byDay = {};
  const push = (k, e) => (byDay[k] = byDay[k] || []).push(e);
  for (const e of d.events) {
    if (!e.all_day) { push(ymd(new Date(e.start)), e); continue; }
    for (const t = new Date(e.start); t.getTime() < e.end; t.setDate(t.getDate() + 1)) push(ymd(t), e); // multi-day all-day entries
  }
  const cells = [];
  for (let i = 0; i < weeks * 7; i++) {
    const day = new Date(gridStart); day.setDate(gridStart.getDate() + i);
    const k = ymd(day); const evs = byDay[k] || [];
    const shown = evs.slice(0, 4);
    cells.push(`<div class="cal-cell ${day.getMonth() !== m - 1 ? 'other' : ''} ${sameDay(day, today) ? 'today' : ''} ${k === selected ? 'selected' : ''}" data-day="${k}">
      <div class="cal-day">${day.getDate()}</div>
      ${shown.map(e => `<div class="cal-ev k-${e.kind} ${e.status === 'cancelled' || e.status === 'done' ? 'closed' : ''}" title="${esc(e.subject)}"><i></i>${e.all_day ? '' : `<b>${hm(e.start)}</b>`}<span>${esc(e.subject)}</span></div>`).join('')}
      ${evs.length > 4 ? `<div class="cal-more">还有 ${evs.length - 4} 项</div>` : ''}</div>`);
  }
  $('#calGrid').innerHTML = cells.join('');
  const showDay = k => {
    selected = k;
    $$('.cal-cell').forEach(c => c.classList.toggle('selected', c.dataset.day === k));
    const day = parseDay(k), evs = byDay[k] || [];
    const meetings = evs.filter(e => e.kind === 'meeting' && e.status !== 'cancelled');
    $('#sideTitle').textContent = `${day.getMonth() + 1}月${day.getDate()}日 ${WEEK[day.getDay()]}`;
    $('#sideSub').textContent = evs.length ? `${meetings.length} 个会议 · ${fmtMinutes(meetings.reduce((s, e) => s + e.duration_minutes, 0))}` : '';
    $('#sideBody').innerHTML = evs.length ? `<div class="timeline">${evs.map(eventItem).join('')}</div>` : '<div class="empty">这一天没有日程</div>';
    $('#sideBody').insertAdjacentHTML('beforeend', `<div style="margin-top:12px"><a class="btn btn-sm" href="#/report?date=${k}">${icon('dashboard')}查看当天工作台</a></div>`);
    setQuery({ month, day: k });
  };
  $('#calGrid').onclick = e => { const c = e.target.closest('.cal-cell'); if (c) showDay(c.dataset.day); };
  showDay(selected);
}

// ================================================================= SEARCH
async function viewSearch(r, alive) {
  const q = r.params.get('q') || '';
  const cid = r.params.get('cid') || '';
  const from = r.params.get('from') || '', to = r.params.get('to') || '';
  await loadConvs();
  if (!alive()) return;
  const scoped = cid ? convOf(cid) : null;
  $('#view').innerHTML = `<div class="page"><div class="page-inner"><div class="search-hero">
    <div class="page-head"><h1>搜索聊天记录</h1></div>
    <form class="search-big" id="sForm">
      <label class="search-input">${icon('search')}<input class="input" id="sQ" placeholder="输入关键字，如：部署、快照、IP 地址…" value="${esc(q)}" autocomplete="off" autofocus></label>
      <button class="btn btn-primary" type="submit">搜索</button>
    </form>
    <div class="search-opts">
      ${scoped ? `<span class="tag tag-blue">仅在「${esc(scoped.title)}」中 <a href="#/search?q=${encodeURIComponent(q)}" style="margin-left:4px">×</a></span>` : ''}
      <span>时间</span><input type="date" class="input" id="sFrom" value="${esc(from)}"><span>至</span><input type="date" class="input" id="sTo" value="${esc(to)}">
    </div>
    <div id="sResults"></div></div></div></div>`;
  $('#sForm').onsubmit = e => {
    e.preventDefault();
    const p = new URLSearchParams({ q: $('#sQ').value.trim() });
    if (cid) p.set('cid', cid);
    if ($('#sFrom').value) p.set('from', $('#sFrom').value);
    if ($('#sTo').value) p.set('to', $('#sTo').value);
    go(`#/search?${p}`);
  };
  if (!q) { $('#sResults').innerHTML = '<div class="card"><div class="empty">在所有单聊和群聊中搜索消息内容</div></div>'; return; }
  $('#sResults').innerHTML = '<div class="card skeleton" style="height:200px"></div>';
  const p = new URLSearchParams({ q, limit: 200 });
  if (cid) p.set('cid', cid); if (from) p.set('from', from); if (to) p.set('to', to);
  const d = await api(`/api/search?${p}`);
  if (!alive()) return;
  const re = new RegExp(q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi');
  const hl = s => esc(s).replace(re, mm => `<mark>${mm}</mark>`);
  $('#sResults').innerHTML = `<div class="muted" style="margin:0 4px 8px;font-size:12px">找到 ${d.total} 条${d.total >= 200 ? '（仅显示最近 200 条）' : ''}</div>` +
    (d.messages.length ? `<div class="card">${d.messages.map(m => {
      const c = convOf(m.cid);
      return `<div class="result" data-cid="${esc(m.cid)}">${avatar(m.sender_name, { size: 'sm', seed: String(m.sender_id) })}
        <div class="result-main"><div class="result-head"><b>${esc(m.sender_name)}</b><span class="conv-name">${c && c.type === 'group' ? '在 ' : ''}${esc(m.conv_title)}</span><time>${esc(m.created_at_str.slice(0, 16))}</time></div>
        <div class="result-text">${hl(cleanPreview(resolveAt(m)))}</div></div></div>`;
    }).join('')}</div>` : '<div class="card"><div class="empty">没有找到相关消息</div></div>');
  $('#sResults').onclick = e => { const el = e.target.closest('.result'); if (el) go(`#/chat/${encodeURIComponent(el.dataset.cid)}`); };
}

// ================================================================ EXPORTS
async function viewExports(r, alive) {
  await loadConvs();
  if (!alive()) return;
  const sel = new Set();
  let filter = 'all', kw = '';
  $('#view').innerHTML = `<div class="page"><div class="page-inner">
    <div class="page-head"><h1>导出</h1><span class="muted">导出为 JSON（含图片和文件附件），可直接交给 AI 分析</span></div>
    <div class="export-grid">
      <section class="card"><div class="card-head"><h3>选择会话</h3><span class="sub" id="selCount">已选 0 个</span></div>
        <div class="pick-tools">
          <label class="search-input" style="flex:1;min-width:160px">${icon('search')}<input class="input" id="pKw" placeholder="搜索会话"></label>
          <div class="seg" id="pSeg"><button data-f="all" class="active">全部</button><button data-f="single">单聊</button><button data-f="group">群聊</button></div>
          <button class="btn btn-sm" id="pAll">全选当前</button><button class="btn btn-sm btn-ghost" id="pNone">清空</button>
        </div>
        <div class="pick-list" id="pList"></div>
        <div class="export-foot">
          <span class="muted">时间范围</span>
          <select class="input" id="pRange"><option value="0">全部</option><option value="7">最近 7 天</option><option value="30" selected>最近 30 天</option><option value="90">最近 3 个月</option><option value="365">最近 1 年</option></select>
          <div style="flex:1"></div>
          <button class="btn" id="fullBtn" title="同步后导出全部会话">全量导出</button>
          <button class="btn btn-primary" id="expBtn" disabled>${icon('download')}导出选中</button>
        </div></section>
      <section class="card"><div class="card-head"><h3>导出记录</h3><div class="grow"></div><span class="sub" id="expStatus"></span></div>
        <div id="expList"></div></section>
    </div></div></div>`;
  const list = () => S.convs.filter(c => (filter === 'all' || c.type === filter) && (!kw || c.title.toLowerCase().includes(kw)));
  const drawPick = () => {
    $('#pList').innerHTML = list().slice(0, 500).map(c => `<label class="pick"><input type="checkbox" data-cid="${esc(c.cid)}" ${sel.has(c.cid) ? 'checked' : ''}>
      ${convAvatar(c, 'xs')}<span class="nm">${esc(c.title)}</span><span class="muted" style="font-size:11px">${shortTime(c.last_message ? c.last_message.time : c.last_modify)}</span></label>`).join('') || '<div class="empty">无</div>';
    $('#selCount').textContent = `已选 ${sel.size} 个`;
    $('#expBtn').disabled = !sel.size;
  };
  $('#pList').onchange = e => { const c = e.target.dataset.cid; if (!c) return; e.target.checked ? sel.add(c) : sel.delete(c); drawPick(); };
  $('#pKw').oninput = e => { kw = e.target.value.trim().toLowerCase(); drawPick(); };
  $('#pSeg').onclick = e => { if (!e.target.dataset.f) return; filter = e.target.dataset.f; $$('#pSeg button').forEach(b => b.classList.toggle('active', b === e.target)); drawPick(); };
  $('#pAll').onclick = () => { list().slice(0, 500).forEach(c => sel.add(c.cid)); drawPick(); };
  $('#pNone').onclick = () => { sel.clear(); drawPick(); };
  const waitIdle = async label => {
    $('#expStatus').textContent = label;
    for (let i = 0; i < 600; i++) {
      await new Promise(res => setTimeout(res, 1500));
      const s = await api('/api/sync/status');
      if (!s.is_syncing) { $('#expStatus').textContent = s.last_error ? '失败：' + s.last_error : '完成'; drawExports(); return; }
    }
  };
  $('#expBtn').onclick = async () => {
    const days = +$('#pRange').value;
    try {
      await api('/api/export/selected', { method: 'POST', json: { cids: [...sel], since_time: days ? Date.now() - days * 864e5 : null } });
      waitIdle('导出中…');
    } catch (e) { toast(e.message); }
  };
  $('#fullBtn').onclick = async () => {
    if (!confirm('同步最新数据并导出全部会话（含附件），可能需要几分钟。继续？')) return;
    try { await api('/api/sync/trigger?full=true', { method: 'POST' }); waitIdle('同步并全量导出中…'); } catch (e) { toast(e.message); }
  };
  const drawExports = async () => {
    const d = await api('/api/exports');
    $('#expList').innerHTML = d.exports.length ? d.exports.map(x => `<div class="export-item">
      <div class="stat-icon i-blue" style="width:34px;height:34px">${icon('file')}</div>
      <div class="nm">${esc(x.filename)}<small>${new Date(x.modified * 1000).toLocaleString('zh-CN', { hour12: false })} · ${fmtSize(x.size)}</small></div>
      <a class="btn btn-sm" href="${esc(x.download_url)}">${icon('download')}下载</a></div>`).join('') : '<div class="empty">暂无导出记录</div>';
  };
  drawPick(); drawExports();
}

// =============================================================== SETTINGS
const INTERVALS = [[0, '关闭'], [5, '5 分钟'], [10, '10 分钟'], [15, '15 分钟'], [30, '30 分钟'], [60, '1 小时'], [120, '2 小时'], [240, '4 小时']];
function intervalLabel(m) {
  const hit = INTERVALS.find(([v]) => v === m);
  if (hit) return hit[1];
  return m % 60 === 0 ? `${m / 60} 小时` : `${m} 分钟`;
}
// Presets only fill the endpoint; model names change often, so they are hints.
const AI_PRESETS = [
  ['', '自定义', '', '模型名'],
  ['deepseek', 'DeepSeek', 'https://api.deepseek.com/v1', '例如 deepseek-chat'],
  ['qwen', '通义千问（阿里云百炼）', 'https://dashscope.aliyuncs.com/compatible-mode/v1', '例如 qwen-plus'],
  ['kimi', 'Kimi（月之暗面）', 'https://api.moonshot.cn/v1', '填写 Kimi 控制台中的模型名'],
  ['zhipu', '智谱 GLM', 'https://open.bigmodel.cn/api/paas/v4', '填写智谱控制台中的模型名'],
  ['openai', 'OpenAI', 'https://api.openai.com/v1', '填写模型名'],
  ['ollama', 'Ollama（本机，数据不出电脑）', 'http://localhost:11434/v1', '本地已下载的模型，例如 qwen2.5:14b'],
];
function aiPresetOf(url) { return (AI_PRESETS.find(p => p[2] && p[2] === url) || AI_PRESETS[0])[0]; }

function aiSettingsCard(a, st) {
  const preset = aiPresetOf(a.base_url);
  const hint = (AI_PRESETS.find(p => p[0] === preset) || AI_PRESETS[0])[3];
  const last = st.last;
  const lastHTML = last ? `${last.ok ? '<span class="tag tag-green">成功</span>' : '<span class="tag tag-red">失败</span>'}
      ${esc(last.kind === 'weekly' ? '周报' : '日报')} ${esc(last.date_key.replace('~', ' ~ '))} · ${esc(last.finished_at.slice(5, 16))}
      <span class="muted">${last.trigger === 'schedule' ? '自动' : '手动'} · ${last.seconds.toFixed(0)} 秒${last.usage && last.usage.total_tokens ? ` · ${last.usage.total_tokens} tokens` : ''}${last.input_chars ? ` · 输入 ${last.input_chars} 字` : ''}</span>
      ${last.error ? `<div style="color:var(--danger);margin-top:4px">${esc(last.error)}</div>` : ''}` : '—';
  return `<section class="card ai-card"><div class="card-head"><span class="ai-badge">${icon('spark')}AI 助手</span><div class="grow"></div>
      <label class="switch" title="启用后可在工作台一键生成日报和周报"><input type="checkbox" id="aiEnabled" ${a.enabled ? 'checked' : ''}><i></i><span>${a.enabled ? '已启用' : '未启用'}</span></label></div>
    <div class="card-body">
      <div class="set-hint" style="margin-top:0">接入任何兼容 OpenAI 接口的大模型，每天自动整理日报、周报，提取待办和待跟踪事项，判断事项进展，并安排第二天的工作。</div>
      <div class="ai-form">
        <label>服务商</label>
        <select class="input" id="aiPreset">${AI_PRESETS.map(p => `<option value="${p[0]}" ${p[0] === preset ? 'selected' : ''}>${esc(p[1])}</option>`).join('')}</select>
        <label>接口地址</label>
        <input class="input" id="aiBase" value="${esc(a.base_url)}" placeholder="https://.../v1" autocomplete="off">
        <label>模型</label>
        <input class="input" id="aiModel" value="${esc(a.model)}" placeholder="${esc(hint)}" autocomplete="off">
        <label>API Key</label>
        <div class="ai-key"><input class="input" id="aiKey" type="password" autocomplete="new-password"
          placeholder="${a.key_from_env ? '已通过环境变量 DINGTALK_AI_API_KEY 提供' : a.has_key ? `已保存 ${esc(a.api_key)}，留空则不修改` : '本地 Ollama 可留空'}">
          ${a.has_key && !a.key_from_env ? '<button class="btn btn-ghost btn-sm" id="aiClearKey">清除</button>' : ''}</div>
        <label>自动生成</label>
        <div class="ai-sched">
          <label class="check-label"><input type="checkbox" id="aiDaily" ${a.daily_enabled ? 'checked' : ''}>每天</label>
          <input class="input" type="time" id="aiTime" value="${esc(a.daily_time || '18:30')}" style="width:110px">
          <span class="muted">生成当天日报</span>
          <label class="check-label"><input type="checkbox" id="aiWeekly" ${a.weekly_enabled ? 'checked' : ''}>每</label>
          <select class="input" id="aiWeekDay" style="width:80px">${[1, 2, 3, 4, 5, 6, 0].map(d => `<option value="${d}" ${d === a.weekly_day ? 'selected' : ''}>${WEEK[d]}</option>`).join('')}</select>
          <span class="muted">同一时间生成本周周报</span>
        </div>
      </div>
      <div class="ai-actions">
        <button class="btn" id="aiTest">${icon('sync')}测试连接</button>
        <button class="btn btn-primary" id="aiSave">保存</button>
        <span class="muted" id="aiTestResult"></span>
      </div>
      <div class="set-hint">🔒 隐私：生成日报时，只会把与你相关的聊天发给模型服务商：单聊、你发过言或被 @ 的群、项目群；其他群只发送统计数字。周报只使用已生成的日报和事项，不发送聊天原文。
        想让数据完全不离开电脑，请使用本机的 Ollama。API Key 保存在本地数据目录的 config.json 中。</div>
      ${a.ready ? `<div class="set-rows">
        ${a.daily_enabled ? `<div class="set-row"><div class="set-k">下次日报</div><div class="set-v">${esc(st.next_daily || '—')}</div></div>` : ''}
        ${a.weekly_enabled ? `<div class="set-row"><div class="set-k">下次周报</div><div class="set-v">${esc(st.next_weekly || '—')}</div></div>` : ''}
        <div class="set-row"><div class="set-k">上次运行</div><div class="set-v">${lastHTML}</div></div>
      </div>` : ''}
    </div></section>`;
}

function aiFormValues() {
  return {
    enabled: $('#aiEnabled').checked,
    base_url: $('#aiBase').value.trim(),
    model: $('#aiModel').value.trim(),
    api_key: $('#aiKey').value.trim(),
    daily_enabled: $('#aiDaily').checked,
    daily_time: $('#aiTime').value || '18:30',
    weekly_enabled: $('#aiWeekly').checked,
    weekly_day: +$('#aiWeekDay').value,
  };
}

function bindAISettings() {
  $('#aiPreset').onchange = e => {
    const p = AI_PRESETS.find(x => x[0] === e.target.value);
    if (p && p[2]) $('#aiBase').value = p[2];
    $('#aiModel').placeholder = p ? p[3] : '模型名';
    $('#aiModel').focus();
  };
  $('#aiBase').oninput = e => { $('#aiPreset').value = aiPresetOf(e.target.value.trim()); };
  const save = async (extra = {}) => {
    const v = { ...aiFormValues(), ...extra };
    if (v.enabled && (!v.base_url || !v.model)) { toast('请填写接口地址和模型'); return false; }
    try {
      await api('/api/ai/settings', { method: 'PUT', json: v });
      toast('AI 设置已保存');
      render();
      return true;
    } catch (e) { toast(e.message, 4000); return false; }
  };
  $('#aiSave').onclick = () => save();
  $('#aiEnabled').onchange = () => save();
  const clear = $('#aiClearKey');
  if (clear) clear.onclick = () => confirm('清除已保存的 API Key？') && save({ clear_key: true, api_key: '' });
  $('#aiTest').onclick = async () => {
    const out = $('#aiTestResult'), btn = $('#aiTest');
    const v = aiFormValues();
    if (!v.base_url || !v.model) { toast('请填写接口地址和模型'); return; }
    btn.disabled = true; out.textContent = '正在连接…'; out.style.color = '';
    try {
      const r = await api('/api/ai/test', { method: 'POST', json: v });
      out.textContent = r.ok ? `✓ 连接成功（${(r.ms / 1000).toFixed(1)} 秒）：${r.reply}` : `✗ ${r.error}`;
      out.style.color = r.ok ? 'var(--success)' : 'var(--danger)';
    } catch (e) { out.textContent = '✗ ' + e.message; out.style.color = 'var(--danger)'; }
    btn.disabled = false;
  };
}

function updateInfoText() {
  const u = U.st;
  if (!u) return '';
  if (u.check_error) return `<span style="color:var(--danger)">${esc(u.check_error)}</span>`;
  if (u.available && u.latest) return `<span style="color:var(--primary)">有新版本 ${esc(u.latest.tag_name)}</span>`;
  if (!u.supported && u.reason) return esc(u.reason);
  if (u.checked_at) return `已是最新版本 · ${esc(u.checked_at.slice(5, 16))} 检查`;
  return '';
}

async function viewSettings(r, alive) {
  const [st, ai] = await Promise.all([api('/api/settings'), api('/api/ai/settings'), loadUpdate()]);
  if (!alive()) return;
  const cur = st.sync_interval_minutes;
  const sync = st.sync || {};
  const row = (k, v, extra = '') => `<div class="set-row"><div class="set-k">${k}</div><div class="set-v">${v}</div>${extra}</div>`;
  const copyBtn = text => `<button class="btn btn-sm btn-ghost" data-copy="${esc(text)}">${icon('copy')}复制</button>`;
  const mcpCmd = `claude mcp add --transport http dingtalk ${st.mcp_url}`;
  $('#view').innerHTML = `<div class="page"><div class="page-inner" style="max-width:860px">
    <div class="page-head"><h1>设置</h1></div>

    ${aiSettingsCard(ai.settings, ai.status)}

    <section class="card" style="margin-top:16px"><div class="card-head"><h3>自动同步</h3><div class="grow"></div>
      <button class="btn btn-primary btn-sm" id="setSyncNow">${icon('sync')}立即同步</button></div>
      <div class="card-body">
        <div class="set-label">同步频率</div>
        <div class="interval-grid" id="intervals">
          ${INTERVALS.map(([v, l]) => `<button class="${v === cur ? 'active' : ''}" data-v="${v}">${l}</button>`).join('')}
          <span class="interval-custom"><input class="input" type="number" min="1" max="1440" id="customMin" placeholder="自定义" value="${INTERVALS.some(([v]) => v === cur) ? '' : cur}"><span class="muted">分钟</span></span>
        </div>
        <div class="set-hint">每次同步都会从钉钉客户端完整解密一次最新的聊天记录和日程（约 2 秒，不影响钉钉使用）。
          频率越高，工作台和 Agent 看到的数据越新。Agent 调用接口时，如果数据超过 5 分钟也会自动先同步。</div>
        <div class="set-rows" id="syncRows">
          ${row('当前设置', `<b>${cur ? '每 ' + intervalLabel(cur) : '已关闭自动同步'}</b>`)}
          ${row('上次同步', esc(sync.last_sync_time_str || '—') + (sync.last_duration_ms ? ` <span class="muted">（耗时 ${(sync.last_duration_ms / 1000).toFixed(1)} 秒）</span>` : ''))}
          ${row('下次同步', esc(cur ? (sync.next_sync_time || '—') : '—'))}
          ${row('最新消息时间', esc(sync.latest_message_str || '—'))}
          ${sync.last_error ? row('上次错误', `<span style="color:var(--danger)">${esc(sync.last_error)}</span>`) : ''}
        </div>
      </div></section>

    <section class="card" style="margin-top:16px"><div class="card-head"><h3>接入 Agent</h3></div>
      <div class="card-body"><div class="set-rows">
        ${row('MCP 地址', `<code>${esc(st.mcp_url)}</code>`, copyBtn(st.mcp_url))}
        ${row('Claude Code', `<code>${esc(mcpCmd)}</code>`, copyBtn(mcpCmd))}
        ${row('REST API', `<code>${esc(st.api_url)}</code>`, copyBtn(st.api_url))}
      </div></div></section>

    <section class="card" style="margin-top:16px"><div class="card-head"><h3>关于</h3></div>
      <div class="card-body"><div class="set-rows">
        ${row('版本', `${esc(st.version)} <span class="muted" id="updInfo">${updateInfoText()}</span>`,
          `<button class="btn btn-sm" id="updCheck">${icon('sync')}检查更新</button>`)}
        ${row('自动检查更新', `<label class="switch"><input type="checkbox" id="updAuto" ${st.auto_update_check ? 'checked' : ''}><i></i>
          <span>启动时和每 12 小时检查一次 GitHub 上的新版本</span></label>`)}
        ${row('钉钉账号', `${esc(S.me.name)} <span class="muted">UID ${esc(st.uid)}</span>`)}
        ${row('钉钉数据目录', `<code>${esc(st.dingtalk_dir)}</code>`)}
        ${row('本地数据目录', `<code>${esc(st.data_dir)}</code>`, copyBtn(st.data_dir))}
      </div></div></section>
  </div></div>`;

  const save = async minutes => {
    try {
      await api('/api/settings', { method: 'PUT', json: { sync_interval_minutes: minutes } });
      toast(minutes ? `已设为每 ${intervalLabel(minutes)}自动同步` : '已关闭自动同步');
      refreshSyncStatus();
      render();
    } catch (e) { toast(e.message, 3500); }
  };
  $('#intervals').onclick = e => { const v = e.target.dataset.v; if (v !== undefined) save(+v); };
  $('#customMin').onchange = e => { const v = Math.round(+e.target.value); if (v >= 1 && v <= 1440) save(v); else toast('请输入 1–1440 之间的分钟数'); };
  $('#setSyncNow').onclick = syncNow;
  bindAISettings();
  $('#updCheck').onclick = async () => {
    const b = $('#updCheck'); b.disabled = true; $('#updInfo').textContent = '正在检查…';
    const st = await loadUpdate(true);
    b.disabled = false;
    $('#updInfo').innerHTML = updateInfoText();
    if (st && st.available) { try { localStorage.removeItem('update-dismissed'); } catch (_) {} showUpdateCard(); }
  };
  $('#updAuto').onchange = async e => {
    try { await api('/api/settings', { method: 'PUT', json: { auto_update_check: e.target.checked } }); toast(e.target.checked ? '已开启自动检查更新' : '已关闭自动检查更新'); }
    catch (err) { toast(err.message); }
  };
  $('#view').addEventListener('click', e => {
    const b = e.target.closest('[data-copy]');
    if (b) navigator.clipboard.writeText(b.dataset.copy).then(() => toast('已复制'));
  });
}

// ================================================================== SYNC
async function refreshSyncStatus() {
  try {
    const s = await api('/api/sync/status');
    const btn = $('#syncBtn');
    btn.classList.toggle('spinning', s.is_syncing);
    btn.classList.toggle('error', !!s.last_error && !s.is_syncing);
    btn.title = s.is_syncing ? '同步中…' : s.last_error ? '同步失败：' + s.last_error
      : `最新消息 ${s.latest_message_str || '-'}\n上次同步 ${s.last_sync_time_str || '-'}${s.next_sync_time ? '\n下次自动同步 ' + s.next_sync_time : ''}\n点击立即同步`;
    $('#syncLabel').textContent = s.is_syncing ? '同步中' : '同步';
  } catch (_) {}
}
async function syncNow() {
  const btn = $('#syncBtn');
  if (btn.classList.contains('spinning')) return;
  btn.classList.add('spinning'); $('#syncLabel').textContent = '同步中';
  try {
    await api('/api/agent/sync', { method: 'POST' });
    S.convsLoadedAt = 0; S.chat = null;
    toast('已同步最新数据');
    render();
  } catch (e) { toast('同步失败：' + e.message, 4000); }
  refreshSyncStatus();
}

// ================================================================ UPDATE
const U = { st: null, autoCheck: true, timer: 0 };
function fmtMB(n) { return (n / 1048576).toFixed(1) + ' MB'; }
async function loadUpdate(force = false) {
  try {
    const d = await api(force ? '/api/update/check' : '/api/update/status', force ? { method: 'POST' } : {});
    U.st = d.status; U.autoCheck = d.auto_check;
  } catch (_) {}
  const settings = $('.rail-item[data-route="settings"]');
  if (settings) settings.classList.toggle('has-update', !!(U.st && U.st.available));
  return U.st;
}
function dismissedUpdate() { try { return localStorage.getItem('update-dismissed') || ''; } catch (_) { return ''; } }
async function maybeShowUpdate() {
  const st = await loadUpdate();
  if (st && st.available && st.latest && dismissedUpdate() !== st.latest.tag_name) showUpdateCard();
}
function showUpdateCard() {
  const st = U.st; if (!st || !st.latest) return;
  let card = $('#updateCard');
  if (!card) { card = document.createElement('div'); card.id = 'updateCard'; card.className = 'update-card'; document.body.appendChild(card); }
  const l = st.latest;
  card.innerHTML = `<div class="uc-head">${icon('spark')}<b>发现新版本 ${esc(l.tag_name)}</b><div class="grow"></div>
      <button class="btn btn-ghost btn-sm btn-icon" data-u="close" title="关闭">✕</button></div>
    <div class="uc-sub">当前版本 ${esc(st.current)}${l.published_at ? ` · 发布于 ${esc(l.published_at.slice(0, 10))}` : ''}
      · <a href="${esc(l.html_url)}" target="_blank" rel="noopener">更新说明</a></div>
    <div id="ucBody">${st.supported
      ? `<div class="uc-actions"><button class="btn btn-sm" data-u="later">稍后提醒</button><button class="btn btn-sm btn-primary" data-u="apply">立即升级</button></div>`
      : `<div class="uc-note">${esc(st.reason || '当前版本不支持自动升级')}</div><div class="uc-actions"><a class="btn btn-sm btn-primary" href="${esc(l.html_url)}" target="_blank" rel="noopener">前往下载</a></div>`}</div>`;
  card.onclick = e => {
    const b = e.target.closest('[data-u]'); if (!b) return;
    if (b.dataset.u === 'close' || b.dataset.u === 'later') {
      if (b.dataset.u === 'close') { try { localStorage.setItem('update-dismissed', l.tag_name); } catch (_) {} }
      card.remove();
    }
    if (b.dataset.u === 'apply') applyUpdate();
  };
}
async function applyUpdate() {
  const l = U.st && U.st.latest; if (!l) return;
  if (!confirm(`升级到 ${l.tag_name}？\n\n程序会自动下载新版本、校验并替换，然后重启。重启期间页面会短暂断开，完成后自动刷新。`)) return;
  if (!$('#updateCard')) showUpdateCard();
  try {
    const d = await api('/api/update/apply', { method: 'POST' });
    U.st = d.status;
  } catch (e) { updateProgress({ apply_error: e.message }); return; }
  pollUpdate(l.version || l.tag_name.replace(/^v/, ''), Date.now());
}
function updateProgress(st) {
  const body = $('#ucBody'); if (!body) return;
  if (st.apply_error) {
    body.innerHTML = `<div class="uc-note error">升级失败：${esc(st.apply_error)}</div>
      <div class="uc-actions"><a class="btn btn-sm" href="${esc((U.st && U.st.latest || {}).html_url || '#')}" target="_blank" rel="noopener">手动下载</a><button class="btn btn-sm btn-primary" data-u="apply">重试</button></div>`;
    return;
  }
  const pct = st.total ? Math.min(100, Math.round(st.downloaded / st.total * 100)) : 0;
  const text = st.restarting ? '正在重启，完成后页面会自动刷新…'
    : st.stage === '下载新版本' && st.total ? `正在下载 ${pct}%（${fmtMB(st.downloaded)} / ${fmtMB(st.total)}）` : (st.stage || '准备中') + '…';
  body.innerHTML = `<div class="uc-progress"><i style="width:${st.restarting ? 100 : st.stage === '下载新版本' ? pct : 8}%"></i></div><div class="uc-note">${esc(text)}</div>`;
}
function pollUpdate(target, started) {
  clearTimeout(U.timer);
  U.timer = setTimeout(async () => {
    if (Date.now() - started > 15 * 60 * 1000) { updateProgress({ apply_error: '升级超时' }); return; }
    let st = null;
    try { st = (await api('/api/update/status')).status; } catch (_) {}
    if (st && st.apply_error && !st.applying) { U.st = st; updateProgress(st); return; }
    if (st && !st.restarting) { updateProgress(st); return pollUpdate(target, started); }
    // Restarting (or the server is already down): wait for the new version.
    updateProgress({ restarting: true });
    waitForVersion(target, Date.now());
  }, 800);
}
function waitForVersion(target, started) {
  clearTimeout(U.timer);
  U.timer = setTimeout(async () => {
    try {
      const c = await api('/api/config');
      if (c.version === target) { toast(`已升级到 ${target}`); setTimeout(() => location.reload(), 600); return; }
    } catch (_) {}
    if (Date.now() - started > 90 * 1000) { updateProgress({ apply_error: '新版本已安装，但等待重启超时。请手动重新启动程序' }); return; }
    waitForVersion(target, started);
  }, 1000);
}

// ================================================================= THEME
function applyTheme(t) {
  if (t) document.documentElement.dataset.theme = t; else delete document.documentElement.dataset.theme;
}
function toggleTheme() {
  const cur = document.documentElement.dataset.theme || (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
  const next = cur === 'dark' ? 'light' : 'dark';
  applyTheme(next);
  try { localStorage.setItem('theme', next); } catch (_) {}
}

// ================================================================== BOOT
async function boot() {
  try { applyTheme(localStorage.getItem('theme')); } catch (_) {}
  $$('[data-icon]').forEach(el => (el.innerHTML = icon(el.dataset.icon)));
  $('#syncBtn').onclick = syncNow;
  $('#themeBtn').onclick = toggleTheme;
  initViewer();
  window.addEventListener('hashchange', render);
  try {
    const c = await api('/api/config');
    S.me = { uid: String(c.user_uid || ''), name: c.user_name || '' };
    $('#meAvatar').outerHTML = avatar(S.me.name || '我', { seed: S.me.uid }).replace('class="avatar', 'id="meAvatar" title="' + esc(S.me.name) + '" class="rail-avatar avatar');
  } catch (_) {}
  render();
  refreshSyncStatus(); updateTodoBadge();
  setInterval(refreshSyncStatus, 30000);
  setInterval(updateTodoBadge, 120000);
  setTimeout(maybeShowUpdate, 3000);
  setInterval(maybeShowUpdate, 30 * 60 * 1000);
}
window.go = go;
document.addEventListener('DOMContentLoaded', boot);
