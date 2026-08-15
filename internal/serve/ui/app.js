// devtree — the local editor.
//
// No framework and no build step: the page talks to a handful of JSON
// endpoints and redraws the parts that changed. Everything that moves is
// driven by a class or a data attribute, so the motion lives in app.css and
// this file only decides when.

const state = {
  plan: null,
  selected: null,
  view: 'tree',
  theme: localStorage.getItem('devtree-theme') || 'light',
};

const $ = (selector) => document.querySelector(selector);
const el = (tag, className) => {
  const node = document.createElement(tag);
  if (className) node.className = className;
  return node;
};

// ── glyphs ──────────────────────────────────────────────────────────────────
//
// Fetched from the binary, not drawn here: the editor and the pictures share
// one icon set, so "blocked" cannot come to mean two different shapes in two
// places.

let GLYPHS = {};

const STATUS_GLYPH = {
  todo: 'clock-circle',
  in_progress: 'circle-half-dotted-check',
  blocked: 'lock-circle',
  done: 'check-circle',
  dropped: 'close-circle',
};

// paint fills every <svg data-glyph="name"> that is on the page or was just
// created. The markup already carries the viewBox, so the body drops straight
// in and inherits currentColor.
function paint(root = document) {
  for (const svg of root.querySelectorAll('svg[data-glyph]')) {
    const body = GLYPHS[svg.dataset.glyph];
    if (body && svg.innerHTML.trim() === '') svg.innerHTML = body;
  }
}

// ── talking to the server ───────────────────────────────────────────────────

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  const text = await response.text();
  const body = text ? JSON.parse(text) : {};
  if (!response.ok) throw new Error(body.error || response.statusText);
  return body;
}

async function loadPlan() {
  state.plan = await api('/api/plan');
  drawHeader();
  drawTasks();
  drawWarnings();
  refreshPreview();
}

// ── header ──────────────────────────────────────────────────────────────────

function drawHeader() {
  const { project, totals } = state.plan;
  const name = $('.project');
  name.textContent = project || 'devtree';
  name.dataset.text = project || 'devtree';
  // The shimmer says "still loading". Once there is a name, it has nothing
  // left to say.
  name.classList.add('is-settled');

  const [done, total] = totals;
  $('.bar-fill').style.width = total ? `${(done / total) * 100}%` : '0%';
  $('.count').textContent = `${done} / ${total}`;
}

// ── the task list ───────────────────────────────────────────────────────────

function drawTasks() {
  const list = $('#tasks');
  list.textContent = '';

  for (const node of state.plan.nodes) {
    const item = el('button', 'task');
    item.type = 'button';
    item.dataset.id = node.id;
    item.dataset.status = node.status;
    item.style.marginLeft = `${node.depth * 16}px`;
    item.setAttribute('role', 'treeitem');
    item.setAttribute('aria-selected', String(node.id === state.selected));

    const icon = statusIcon(node.status);
    if (icon) item.appendChild(icon);

    const body = el('span', 'task-body');
    const title = el('span', 'task-title');
    title.textContent = node.title || node.id;
    body.appendChild(title);

    const meta = [node.branch, node.issue && `#${node.issue}`, node.pr && `!${node.pr}`,
                  node.owner && `@${node.owner}`, (node.tags || []).join(', ')]
      .filter(Boolean).join('  ');
    if (meta) {
      const line = el('span', 'task-meta');
      line.textContent = meta;
      body.appendChild(line);
    }
    item.appendChild(body);

    if (node.total > 0) {
      const ratio = el('span', 'task-ratio');
      ratio.textContent = `${node.done}/${node.total}`;
      item.appendChild(ratio);
    }

    item.addEventListener('click', () => select(node.id));
    list.appendChild(item);
  }
}

function statusIcon(status) {
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('class', 'glyph');
  svg.setAttribute('viewBox', '0 0 24 24');
  svg.dataset.glyph = STATUS_GLYPH[status] || 'clock-circle';
  svg.style.color = `var(--st-${status.replace('_', '-')})`;
  svg.innerHTML = GLYPHS[svg.dataset.glyph] || '';
  return svg;
}

function drawWarnings() {
  const box = $('#warnings');
  const list = state.plan.warnings || [];
  box.hidden = list.length === 0;
  box.textContent = '';
  if (!list.length) return;

  const title = el('b');
  title.textContent = list.length === 1 ? '1 thing to look at' : `${list.length} things to look at`;
  box.appendChild(title);
  for (const warning of list) {
    const line = el('div');
    line.textContent = warning;
    box.appendChild(line);
  }
}

// ── preview ─────────────────────────────────────────────────────────────────

let previewToken = 0;

async function refreshPreview() {
  const token = ++previewToken;
  const panel = $('#preview');
  const content = $('#preview-content');

  // Back to the skeleton while the next drawing is fetched. The reset class
  // kills the reverse transition so it snaps rather than fading backwards.
  panel.classList.add('is-resetting');
  panel.classList.remove('is-revealed');
  void panel.offsetWidth; // force the reflow the reset depends on
  panel.classList.remove('is-resetting');

  const theme = state.theme === 'dark' ? 'dark' : 'light';
  const url = `/api/view/${state.view}?theme=${theme}`;

  try {
    const response = await fetch(url, { headers: { 'Cache-Control': 'no-store' } });
    const text = await response.text();
    if (token !== previewToken) return; // a newer request won

    content.textContent = '';
    if (state.view === 'tree' || state.view === 'board') {
      content.innerHTML = text;
    } else if (state.view === 'page') {
      const frame = el('iframe');
      frame.setAttribute('title', 'The exported page');
      frame.srcdoc = text;
      content.appendChild(frame);
    } else {
      const pre = el('pre');
      pre.textContent = text;
      content.appendChild(pre);
    }
    panel.classList.add('is-revealed');
  } catch (err) {
    toast(err.message, 'error');
  }
}

// ── the editor ──────────────────────────────────────────────────────────────

function select(id) {
  state.selected = id;
  drawTasks();

  const node = state.plan.nodes.find((n) => n.id === id);
  if (!node) return;

  const form = $('#editor-form');
  form.title.value = node.title || '';
  form.branch.value = node.branch || '';
  form.owner.value = node.owner || '';
  form.issue.value = node.issue || '';
  form.pr.value = node.pr || '';
  form.tags.value = (node.tags || []).join(', ');
  form.note.value = node.note || '';
  setStatus(node.status);

  $('#editor-title').textContent = node.id;
  $('#editor').hidden = false;
}

function setStatus(status) {
  $('#status-label').textContent = status;
  $('#status-button').dataset.value = status;
}

function buildStatusMenu() {
  const menu = $('#status-menu');
  menu.textContent = '';
  for (const status of state.plan.statuses) {
    const option = el('button');
    option.type = 'button';
    option.setAttribute('role', 'menuitem');
    option.appendChild(statusIcon(status.name));
    option.appendChild(document.createTextNode(status.label));
    option.addEventListener('click', () => {
      setStatus(status.name);
      closeMenu();
    });
    menu.appendChild(option);
  }
}

// The dropdown closes in two steps: swap is-open for is-closing so the shorter
// leave transition runs, then remove the class once it has.
let closing = null;
function openMenu() {
  const menu = $('#status-menu');
  clearTimeout(closing);
  menu.classList.remove('is-closing');
  menu.classList.add('is-open');
  $('#status-button').setAttribute('aria-expanded', 'true');
}
function closeMenu() {
  const menu = $('#status-menu');
  if (!menu.classList.contains('is-open')) return;
  menu.classList.remove('is-open');
  menu.classList.add('is-closing');
  $('#status-button').setAttribute('aria-expanded', 'false');
  closing = setTimeout(() => menu.classList.remove('is-closing'), 150);
}

async function saveTask(event) {
  event.preventDefault();
  if (!state.selected) return;

  const form = event.target;
  const body = {
    title: form.title.value,
    status: $('#status-button').dataset.value,
    branch: form.branch.value,
    owner: form.owner.value,
    issue: form.issue.value,
    pr: form.pr.value,
    note: form.note.value,
    tags: form.tags.value.split(',').map((t) => t.trim()).filter(Boolean),
  };

  try {
    state.plan = await api(`/api/task/${encodeURIComponent(state.selected)}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    });
    drawHeader(); drawTasks(); drawWarnings(); refreshPreview();
    celebrate();
  } catch (err) {
    toast(err.message, 'error');
  }
}

// celebrate draws the check once. Restarting it means clearing the state and
// forcing a reflow, or the browser sees no change and skips the animation.
function celebrate() {
  const check = document.querySelector('.t-success-check');
  check.dataset.state = 'out';
  void check.offsetWidth;
  check.dataset.state = 'in';
}

async function addTask(parent) {
  const title = prompt(parent ? `New task under ${parent}` : 'New task');
  if (!title) return;
  try {
    state.plan = await api('/api/task', {
      method: 'POST',
      body: JSON.stringify({ title, parent: parent || '' }),
    });
    state.selected = state.plan.saved || state.selected;
    drawHeader(); drawTasks(); drawWarnings(); refreshPreview();
    if (state.selected) select(state.selected);
    celebrate();
  } catch (err) {
    toast(err.message, 'error');
  }
}

async function deleteTask() {
  if (!state.selected) return;
  if (!confirm(`Delete ${state.selected}? Its children move up to its parent.`)) return;
  try {
    state.plan = await api(`/api/task/${encodeURIComponent(state.selected)}`, { method: 'DELETE' });
    state.selected = null;
    $('#editor').hidden = true;
    drawHeader(); drawTasks(); drawWarnings(); refreshPreview();
  } catch (err) {
    toast(err.message, 'error');
  }
}

// ── tabs ────────────────────────────────────────────────────────────────────

// The pill is positioned from the measured tab. On the first paint and on
// resize the values are written with the transition suspended, so it appears
// where it belongs instead of sliding in from the left edge.
function movePill(animate) {
  const tabs = $('#views');
  const pill = tabs.querySelector('.t-tabs-pill');
  const active = tabs.querySelector('[aria-selected="true"]');
  if (!active) return;

  if (!animate) {
    pill.style.transition = 'none';
  }
  pill.style.transform = `translateX(${active.offsetLeft - 3}px)`;
  pill.style.width = `${active.offsetWidth}px`;
  if (!animate) {
    void pill.offsetWidth;
    pill.style.transition = '';
  }
}

function selectView(view) {
  state.view = view;
  for (const tab of document.querySelectorAll('.t-tab')) {
    tab.setAttribute('aria-selected', String(tab.dataset.view === view));
  }
  movePill(true);
  refreshPreview();
}

// ── theme ───────────────────────────────────────────────────────────────────

function applyTheme() {
  document.documentElement.dataset.theme = state.theme;
  document.querySelector('.t-icon-swap').dataset.state = state.theme === 'dark' ? 'b' : 'a';
  localStorage.setItem('devtree-theme', state.theme);
}

// ── the divider ─────────────────────────────────────────────────────────────

function draggableDivider() {
  const divider = $('#divider');
  const left = $('#left');
  let dragging = false;

  const width = (px) => {
    left.style.width = `${Math.min(Math.max(px, 260), 720)}px`;
  };

  divider.addEventListener('pointerdown', (event) => {
    dragging = true;
    left.classList.add('is-dragging'); // track the cursor exactly while dragging
    divider.setPointerCapture(event.pointerId);
  });
  divider.addEventListener('pointermove', (event) => {
    if (dragging) width(event.clientX);
  });
  divider.addEventListener('pointerup', () => {
    dragging = false;
    left.classList.remove('is-dragging');
  });

  // The keyboard moves it in steps, and those *should* tween.
  divider.addEventListener('keydown', (event) => {
    const current = left.getBoundingClientRect().width;
    if (event.key === 'ArrowLeft') width(current - 40);
    if (event.key === 'ArrowRight') width(current + 40);
  });
}

// ── odds and ends ───────────────────────────────────────────────────────────

let toastTimer = null;
function toast(message, kind) {
  const box = $('#toast');
  box.textContent = message;
  box.dataset.kind = kind || 'info';
  box.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { box.hidden = true; }, 4000);
}

async function loadGlyphs() {
  GLYPHS = await api('/api/glyphs');
  const mark = document.querySelector('.mark');
  mark.innerHTML = GLYPHS['tree'] || '';
  paint();
}

// ── wiring ──────────────────────────────────────────────────────────────────

async function start() {
  await loadGlyphs();
  applyTheme();
  draggableDivider();

  $('#views').addEventListener('click', (event) => {
    const tab = event.target.closest('.t-tab');
    if (tab) selectView(tab.dataset.view);
  });

  $('#theme').addEventListener('click', () => {
    state.theme = state.theme === 'dark' ? 'light' : 'dark';
    applyTheme();
    refreshPreview();
  });

  $('#write').addEventListener('click', async () => {
    try {
      const result = await api('/api/render', { method: 'POST' });
      celebrate();
      toast(`Wrote ${result.written.join(', ')}`);
    } catch (err) {
      toast(err.message, 'error');
    }
  });

  $('#add-root').addEventListener('click', () => addTask(''));
  $('#add-child').addEventListener('click', () => addTask(state.selected));
  $('#delete').addEventListener('click', deleteTask);
  $('#editor-form').addEventListener('submit', saveTask);
  $('#editor-close').addEventListener('click', () => { $('#editor').hidden = true; });

  $('#status-button').addEventListener('click', (event) => {
    event.stopPropagation();
    const open = $('#status-menu').classList.contains('is-open');
    open ? closeMenu() : openMenu();
  });
  document.addEventListener('click', closeMenu);
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') { closeMenu(); }
  });

  await loadPlan();
  buildStatusMenu();
  movePill(false);
  window.addEventListener('resize', () => movePill(false));

  // An edit made in a terminal shows up here within a second. ?live=0 turns
  // the stream off — for a proxy that will not hold a connection open, and for
  // any tool that waits for the page to go quiet before capturing it.
  if (new URLSearchParams(location.search).get('live') !== '0') {
    const events = new EventSource('/api/events');
    events.addEventListener('changed', () => loadPlan());
  }
}

start();
