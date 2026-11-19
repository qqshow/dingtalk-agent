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
  const inline = s => esc(s).replace(/`([^`]+)`/g, '<code>$1</code>').replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  const out = []; let list = null;
  const close = () => { if (list) { out.push(`</${list}>`); list = null; } };
  for (const raw of String(src || '').split('\n')) {
    const line = raw.trimEnd(); let m;
    if ((m = line.match(/^(#{1,4})\s+(.*)$/))) { close(); out.push(`<h${m[1].length + 1}>${inline(m[2])}</h${m[1].length + 1}>`); }
    else if ((m = line.match(/^\s*[-*]\s+(.*)$/))) { if (list !== 'ul') { close(); out.push('<ul>'); list = 'ul'; } out.push(`<li>${inline(m[1])}</li>`); }
    else if ((m = line.match(/^\s*\d+[.)]\s+(.*)$/))) { if (list !== 'ol') { close(); out.push('<ol>'); list = 'ol'; } out.push(`<li>${inline(m[1])}</li>`); }
    else if (!line.trim()) close();
    else { close(); out.push(`<p>${inline(line)}</p>`); }
  }
  close();
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

const VIEWS = { chat: viewChat, report: viewReport, todos: viewTodos, calendar: viewCalendar, search: viewSearch, exports: viewExports, settings: viewSettings };
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
      <button class="btn btn-ghost btn-sm" id="chatSearchBtn">${icon('search')}在会话中搜索</button>
    </header>
    <div class="chat-body" id="chatBody"><div class="empty">加载中…</div></div>`;
  $('#chatSearchBtn').onclick = () => go(`#/search?cid=${encodeURIComponent(cid)}`);
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

  const [rep] = await Promise.all([api(`/api/agent/report?date=${encodeURIComponent(date)}`), loadConvs()]);
  if (!alive()) return;
  renderDashboard(rep, date);
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
  const summaryHTML = s ? `
      ${s.focus && s.focus.length ? `<div class="focus"><h4>${icon('flag')}需要重点关注</h4><ul>${s.focus.map(f => `<li>${esc(f)}</li>`).join('')}</ul></div>` : ''}
      <div class="md">${md(s.summary)}</div>`
    : `<div class="empty">今天还没有 AI 总结<br>在 Claude Code 中运行下面的命令，生成日报并自动整理待办
      <div class="cmd"><code id="cmdText">/mcp__dingtalk__daily_summary ${esc(date)}</code>
      <button class="btn btn-sm" id="copyCmd">${icon('copy')}复制</button></div></div>`;

  const openTodos = r.open_todos || [];
  const todoHTML = openTodos.length ? openTodos.slice(0, 6).map(td => `
      <div class="list-item clickable" onclick="go('#/todos')">
        <div class="prio ${td.priority}"></div>
        <div class="li-main"><div class="li-title">${esc(td.title)}</div>
        <div class="li-sub">${td.kind === 'follow_up' ? '待跟踪' : '待办'}${td.due ? ' · 截止 ' + esc(td.due) : ''}${td.owner ? ' · ' + esc(td.owner) : ''}</div></div>
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
          <span class="sub">${s ? esc(`${s.author || 'agent'} · ${s.updated_at.slice(5, 16)}`) : ''}</span></div>
          <div class="card-body">${summaryHTML}</div></section>
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
    const el = e.target.closest('[data-cid]');
    if (el) go(`#/chat/${encodeURIComponent(el.dataset.cid)}`);
  });
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
async function viewSettings(r, alive) {
  const st = await api('/api/settings');
  if (!alive()) return;
  const cur = st.sync_interval_minutes;
  const sync = st.sync || {};
  const row = (k, v, extra = '') => `<div class="set-row"><div class="set-k">${k}</div><div class="set-v">${v}</div>${extra}</div>`;
  const copyBtn = text => `<button class="btn btn-sm btn-ghost" data-copy="${esc(text)}">${icon('copy')}复制</button>`;
  const mcpCmd = `claude mcp add --transport http dingtalk ${st.mcp_url}`;
  $('#view').innerHTML = `<div class="page"><div class="page-inner" style="max-width:860px">
    <div class="page-head"><h1>设置</h1></div>

    <section class="card"><div class="card-head"><h3>自动同步</h3><div class="grow"></div>
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
        ${row('版本', esc(st.version))}
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
}
window.go = go;
document.addEventListener('DOMContentLoaded', boot);
