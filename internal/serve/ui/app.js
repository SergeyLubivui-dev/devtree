// devtree — the local editor.
//
// No framework and no build step: the page talks to a handful of JSON
// endpoints and redraws the part that changed. Everything that moves is driven
// by a class or a data attribute, so the motion lives in app.css and this file
// only decides when.

const state = {
  plan: null,
  glyphs: {},
  section: 'plan',            // plan | docs
  view: 'tree',               // tree | board | page | mermaid | yaml
  scope: 'all',               // all | open
  search: '',
  selected: null,
  // Dark is the default. The reader's own setting wins if they have expressed
  // one, and the choice is remembered from then on.
  theme: localStorage.getItem('devtree-theme')
    || (matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'),
};

const $ = (id) => document.getElementById(id);
const el = (tag, cls) => { const n = document.createElement(tag); if (cls) n.className = cls; return n; };

// ---------------------------------------------------------------------------
// talking to devtree
// ---------------------------------------------------------------------------

// api returns the parsed body, and throws with the message devtree wrote —
// which is the same wording its command line uses, because the person reading
// it is the same person.
async function api(path, options = {}) {
  const res = await fetch(path, options);
  const text = await res.text();
  let body = null;
  try { body = text ? JSON.parse(text) : null; } catch { /* not json */ }
  if (!res.ok) throw new Error((body && body.error) || text || res.statusText);
  return body;
}

async function loadPlan(saved) {
  const plan = saved || await api('/api/plan');
  state.plan = plan;
  paint();
  return plan;
}

// ---------------------------------------------------------------------------
// glyphs
// ---------------------------------------------------------------------------

// The status marks come from the binary, not from this file. The first version
// drew its own little circles here, and the editor's idea of "blocked" drifted
// from the drawing's within a week.
function glyph(name, color) {
  const body = state.glyphs[name];
  if (!body) return '';
  const fill = color ? ` style="color:${color}"` : '';
  return `<svg viewBox="0 0 24 24" fill="currentColor"${fill} aria-hidden="true">${body}</svg>`;
}

const statusGlyph = {
  todo: 'clock-circle',
  in_progress: 'circle-half-dotted-check',
  blocked: 'lock-circle',
  done: 'check-circle',
  dropped: 'close-circle',
};

function statusMark(status) {
  return glyph(statusGlyph[status] || 'clock-circle', `var(--${status})`);
}

function fillGlyphs(scope = document) {
  scope.querySelectorAll('[data-glyph]').forEach((node) => {
    node.innerHTML = glyph(node.dataset.glyph);
  });
}

// ---------------------------------------------------------------------------
// the panel
// ---------------------------------------------------------------------------

function paint() {
  const plan = state.plan;
  if (!plan) return;

  const name = $('crumb-name');
  name.textContent = plan.project || 'devtree';
  name.classList.remove('shimmer');
  $('side-title').classList.remove('shimmer');
  $('side-title').textContent = state.section === 'docs' ? 'Documents' : 'Plan';

  // All/Open is a question about work. A document is neither open nor done, so
  // the control goes away rather than sitting there doing nothing.
  $('scope').hidden = state.section === 'docs';
  $('search').placeholder = state.section === 'docs' ? 'Search outputs...' : 'Search...';
  document.title = plan.project ? `${plan.project} — devtree` : 'devtree';

  $('rollup').textContent = plan.totals ? `${plan.totals[0]} / ${plan.totals[1]}` : '';

  if (state.section === 'docs') paintDocs(); else paintTasks();
  paintFoot();
}

// visible decides what survives the search box and the All/Open switch.
// A parent stays if any of its children do, so filtering never orphans work.
function visible(plan) {
  const term = state.search.trim().toLowerCase();
  const wanted = new Set();

  const matches = (n) => {
    if (state.scope === 'open' && (n.status === 'done' || n.status === 'dropped')) return false;
    if (!term) return true;
    return [n.title, n.id, n.owner, n.branch, (n.tags || []).join(' ')]
      .filter(Boolean).some((s) => s.toLowerCase().includes(term));
  };

  const byID = new Map(plan.nodes.map((n) => [n.id, n]));
  for (const n of plan.nodes) {
    if (!matches(n)) continue;
    wanted.add(n.id);
    let p = n.parent && byID.get(n.parent);
    while (p) { wanted.add(p.id); p = p.parent && byID.get(p.parent); }
  }
  return wanted;
}

function paintTasks() {
  const list = $('list');
  const plan = state.plan;
  list.innerHTML = '';

  if (!plan.nodes.length) {
    list.append(hint('No tasks yet. The + above starts the plan.'));
    return;
  }

  const keep = visible(plan);
  const kids = new Map();
  for (const n of plan.nodes) {
    if (!keep.has(n.id)) continue;
    const bucket = kids.get(n.parent || '') || [];
    bucket.push(n);
    kids.set(n.parent || '', bucket);
  }

  if (!keep.size) {
    list.append(hint('Nothing matches.'));
    return;
  }

  const build = (parent) => {
    const ul = el('ul');
    for (const node of kids.get(parent) || []) {
      const li = el('li');
      li.append(taskRow(node));
      const below = build(node.id);
      if (below.childElementCount) li.append(below);
      ul.append(li);
    }
    return ul;
  };

  list.append(build(''));
}

function taskRow(node) {
  const row = el('button', 'row');
  row.type = 'button';
  row.dataset.id = node.id;
  row.dataset.status = node.status;
  if (state.selected === node.id) row.setAttribute('aria-current', 'true');

  const mark = el('span', 'glyph');
  mark.innerHTML = statusMark(node.status);

  const label = el('span', 'label');
  label.textContent = node.title || node.id;

  row.append(mark, label);

  if (node.total > 0) {
    const count = el('span', 'count');
    count.textContent = `${node.done}/${node.total}`;
    if (node.done === node.total) count.dataset.full = 'true';
    row.append(count);
  }

  row.addEventListener('click', () => editTask(node.id));
  return row;
}

function paintDocs() {
  const list = $('list');
  list.innerHTML = '';

  const term = state.search.trim().toLowerCase();
  const all = state.plan.docs || [];
  const docs = term ? all.filter((d) => d.path.toLowerCase().includes(term)) : all;

  if (!all.length) {
    list.append(hint('No outputs yet. The + above names the first one.'));
    return;
  }
  if (!docs.length) {
    list.append(hint('Nothing matches.'));
    return;
  }

  const ul = el('ul');
  for (const doc of docs) {
    const li = el('li');
    const row = el('button', 'row doc-row');
    row.type = 'button';
    row.dataset.exists = String(doc.exists);
    row.title = doc.exists ? `${doc.bytes} bytes on disk` : 'not written yet';

    const mark = el('span', 'glyph');
    mark.innerHTML = glyph(doc.kind === 'page' ? 'monitor' : doc.kind === 'markdown' ? 'note' : 'nodes');

    const label = el('span', 'label');
    label.textContent = doc.path;

    const kind = el('span', 'kind');
    kind.textContent = doc.theme === 'dark' ? `${doc.kind} · dark` : doc.kind;

    row.append(mark, label, kind);
    row.addEventListener('click', () => openDocument(doc));
    li.append(row);
    ul.append(li);
  }
  list.append(ul);
}

function hint(text) {
  const node = el('p', 'empty');
  node.textContent = text;
  return node;
}

function paintFoot() {
  const foot = $('side-foot');
  foot.innerHTML = '';
  const warnings = state.plan.warnings || [];
  if (warnings.length) {
    const warn = el('span', 'warn');
    warn.textContent = warnings[0];
    warn.title = warnings.join('\n');
    foot.append(warn);
    return;
  }
  const count = state.section === 'docs'
    ? `${(state.plan.docs || []).length} outputs`
    : `${state.plan.nodes.length} tasks`;
  foot.textContent = count;
}

// ---------------------------------------------------------------------------
// the stage
// ---------------------------------------------------------------------------

let stageToken = 0;

async function paintStage() {
  const stage = $('stage');
  const token = ++stageToken;

  stage.innerHTML = '<div class="skeleton"><i></i><i></i><i></i><i></i></div>';
  hideNodeAdd();

  const params = new URLSearchParams({ theme: state.theme });
  try {
    const res = await fetch(`/api/view/${state.view}?${params}`);
    const text = await res.text();
    if (token !== stageToken) return;           // a newer view won the race
    if (!res.ok) throw new Error(text);

    if (state.view === 'tree' || state.view === 'board') {
      stage.innerHTML = text;
      // The drawing keeps the size it was drawn at. Stripping width and height
      // would let it stretch to whatever the pane happens to be, which is how
      // a diagram ends up with cards the size of paragraphs.
      const svg = stage.querySelector('svg');
      if (svg) svg.classList.add('plan', 'reveal');
      wireCards();
    } else if (state.view === 'page') {
      const frame = el('iframe');
      frame.className = 'reveal';
      frame.title = 'The HTML export';
      frame.srcdoc = text;
      stage.innerHTML = '';
      stage.append(frame);
    } else {
      const pre = el('pre', 'reveal');
      pre.textContent = text;
      stage.innerHTML = '';
      stage.append(pre);
    }
  } catch (err) {
    if (token !== stageToken) return;
    stage.innerHTML = '';
    stage.append(hint(String(err.message || err)));
  }
}

// wireCards makes the drawing itself the editing surface: click a card to open
// it, hover it to get a + that adds work underneath it. The cards are found by
// the identity the renderer writes onto them, so the page never has to guess
// which shape is which task.
function wireCards() {
  const stage = $('stage');
  const svg = stage.querySelector('svg');
  if (!svg) return;

  svg.querySelectorAll('[data-node]').forEach((group) => {
    const id = group.dataset.node;
    group.addEventListener('click', (event) => { event.stopPropagation(); editTask(id); });
    group.addEventListener('mouseenter', () => showNodeAdd(group, id));
  });

  stage.addEventListener('mouseleave', hideNodeAdd, { once: true });
}

function showNodeAdd(group, id) {
  const button = $('node-add');
  const stage = $('stage');
  const box = group.getBoundingClientRect();
  const frame = stage.getBoundingClientRect();

  // The stage scrolls, so position against the main region and let the
  // button ride along rather than re-measuring on every scroll tick.
  button.hidden = false;
  button.style.left = `${box.right - frame.left + stage.offsetLeft - 12}px`;
  button.style.top = `${box.top - frame.top + stage.offsetTop + box.height / 2 - 12}px`;
  button.dataset.show = '';
  button.dataset.parent = id;
}

function hideNodeAdd() {
  const button = $('node-add');
  delete button.dataset.show;
  button.hidden = true;
}

// ---------------------------------------------------------------------------
// the drawer
// ---------------------------------------------------------------------------

const drawer = {
  node: null,
  lastFocus: null,
  onSubmit: null,
};

function openDrawer({ title, body, actions, onSubmit }) {
  drawer.lastFocus = document.activeElement;
  drawer.onSubmit = onSubmit || null;

  $('drawer-title').textContent = title;

  const wrap = el('div', 'morph');
  const inner = el('div');
  inner.append(body);
  wrap.append(inner);
  $('drawer-body').innerHTML = '';
  $('drawer-body').append(wrap);

  const foot = $('drawer-foot');
  foot.innerHTML = '';
  for (const action of actions || []) foot.append(action);

  const node = $('drawer');
  const scrim = $('scrim');
  node.hidden = false;
  scrim.hidden = false;
  // A frame between "in the document" and "open" is what gives the transition
  // a start state to travel from.
  requestAnimationFrame(() => {
    node.dataset.open = '';
    scrim.dataset.open = '';
  });
  node.setAttribute('aria-hidden', 'false');

  const first = $('drawer-body').querySelector('input, textarea, button');
  if (first) setTimeout(() => first.focus(), 60);
}

function closeDrawer() {
  const node = $('drawer');
  const scrim = $('scrim');
  if (node.hidden) return;

  delete node.dataset.open;
  delete scrim.dataset.open;
  node.setAttribute('aria-hidden', 'true');
  drawer.onSubmit = null;

  // Kept in the document until it has finished leaving, then taken out of the
  // tab order entirely — a panel off-screen is still a panel you can tab into.
  setTimeout(() => { node.hidden = true; scrim.hidden = true; }, 340);

  if (drawer.lastFocus && document.contains(drawer.lastFocus)) drawer.lastFocus.focus();
}

// ask is what the editor uses instead of a browser dialog. A native one blocks
// the page, cannot be styled, and appears somewhere the interface has no say
// over; this asks in the same panel everything else is asked in.
function ask({ title, message, confirm = 'Confirm', danger = false }) {
  return new Promise((resolve) => {
    const body = el('div', 'note');
    body.innerHTML = message;

    const cancel = el('button', 'btn');
    cancel.type = 'button';
    cancel.textContent = 'Cancel';
    cancel.addEventListener('click', () => { closeDrawer(); resolve(false); });

    const go = el('button', danger ? 'btn btn-danger' : 'btn btn-primary');
    go.type = 'button';
    go.textContent = confirm;
    go.addEventListener('click', () => { closeDrawer(); resolve(true); });

    openDrawer({ title, body, actions: [cancel, spacer(), go] });
  });
}

function spacer() { return el('span', 'grow'); }

// ---------------------------------------------------------------------------
// forms
// ---------------------------------------------------------------------------

function field(label, name, value = '', { placeholder = '', hint = '', area = false } = {}) {
  const wrap = el('div', 'field');
  const id = `f-${name}`;

  const tag = el('label');
  tag.setAttribute('for', id);
  tag.textContent = label;

  const input = el(area ? 'textarea' : 'input');
  if (!area) input.type = 'text';
  input.id = id;
  input.name = name;
  input.value = value || '';
  input.placeholder = placeholder;
  input.autocomplete = 'off';

  wrap.append(tag, input);
  if (hint) {
    const note = el('p', 'hint');
    note.innerHTML = hint;
    wrap.append(note);
  }
  return wrap;
}

// statusPicker is the menu-drop snippet doing a real job: five options, opened
// from the control that shows the current one.
function statusPicker(current, note = '') {
  const wrap = el('div', 'field');
  const tag = el('label');
  tag.textContent = 'Status';

  const menuWrap = el('div', 'menu-wrap');
  const button = el('button', 'menu-btn');
  button.type = 'button';
  button.setAttribute('aria-haspopup', 'menu');

  const mark = el('span', 'glyph');
  const text = el('span');
  const chev = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  chev.setAttribute('viewBox', '0 0 24 24');
  chev.setAttribute('class', 'ui chev');
  chev.innerHTML = '<path d="m7 10 5 5 5-5"/>';

  let value = current;
  const show = () => {
    const status = (state.plan.statuses || []).find((s) => s.name === value);
    mark.innerHTML = statusMark(value);
    text.textContent = status ? status.label : value;
  };

  button.append(mark, text, chev);

  const menu = el('div', 'menu');
  menu.setAttribute('role', 'menu');
  for (const status of state.plan.statuses || []) {
    const item = el('button');
    item.type = 'button';
    item.setAttribute('role', 'menuitem');
    item.innerHTML = `${statusMark(status.name)}<span>${status.label}</span>`;
    item.addEventListener('click', () => {
      value = status.name;
      show();
      delete menu.dataset.open;
    });
    menu.append(item);
  }

  button.addEventListener('click', (event) => {
    event.stopPropagation();
    if (menu.dataset.open === undefined) menu.dataset.open = ''; else delete menu.dataset.open;
  });
  document.addEventListener('click', () => { delete menu.dataset.open; });

  show();
  menuWrap.append(button, menu);
  wrap.append(tag, menuWrap);
  if (note) {
    const line = el('p', 'hint');
    line.innerHTML = note;
    wrap.append(line);
  }
  wrap.read = () => value;
  return wrap;
}

function readFields(body) {
  const out = {};
  body.querySelectorAll('input[name], textarea[name]').forEach((input) => {
    out[input.name] = input.value.trim();
  });
  return out;
}

// ---------------------------------------------------------------------------
// editing
// ---------------------------------------------------------------------------

function editTask(id) {
  const node = (state.plan.nodes || []).find((n) => n.id === id);
  if (!node) return;

  state.selected = id;
  paint();

  // Every field says what it is for. These are not decoration: the fields do
  // real work elsewhere in devtree — a branch closes a task when it merges, a
  // pull request outranks an issue for the link — and a form that keeps that
  // to itself is a form people fill in wrong.
  const kids = (state.plan.nodes || []).filter((n) => n.parent === node.id).length;

  const body = el('div');
  const title = field('Title', 'title', node.title, {
    hint: 'What the card says, in the drawing and in every export.',
  });
  const status = statusPicker(node.status, kids
    ? `This one has ${kids} task${kids > 1 ? 's' : ''} under it, so its progress is counted from them,
       not from this. The status still colours the card.`
    : 'Colours the card and decides which column it lands in on the board.');

  const pair = el('div', 'pair');
  pair.append(
    field('Owner', 'owner', node.owner, { placeholder: 'ann', hint: 'Who has it. <code>@</code> is trimmed.' }),
    field('Branch', 'branch', node.branch, {
      placeholder: 'feat/auth',
      hint: '<code>devtree sync</code> closes this task when that branch merges.',
    }));

  const pair2 = el('div', 'pair');
  pair2.append(
    field('Issue', 'issue', node.issue, { placeholder: '12', hint: 'Number only.' }),
    field('Pull request', 'pr', node.pr, { placeholder: '44', hint: 'Wins over the issue for the link.' }));

  const tags = field('Tags', 'tags', (node.tags || []).join(', '), {
    placeholder: 'backend, api',
    hint: 'Comma separated. <code>devtree ls --tag backend</code> filters on these.',
  });
  const note = field('Note', 'note', node.note, {
    area: true,
    hint: 'A line of context for whoever opens the plan next — why it is blocked, what it is waiting on.',
  });

  body.append(title, status, pair, pair2, tags, note);

  const remove = el('button', 'btn btn-danger');
  remove.type = 'button';
  remove.textContent = 'Delete';
  remove.addEventListener('click', () => removeTask(node));

  const save = el('button', 'btn btn-primary');
  save.type = 'button';
  save.textContent = 'Save';

  const submit = async () => {
    const values = readFields(body);
    save.disabled = true;
    try {
      const plan = await api(`/api/task/${encodeURIComponent(node.id)}`, {
        method: 'PATCH',
        body: JSON.stringify({
          title: values.title,
          status: status.read(),
          owner: values.owner,
          branch: values.branch,
          issue: values.issue,
          pr: values.pr,
          note: values.note,
          tags: values.tags ? values.tags.split(',').map((t) => t.trim()).filter(Boolean) : [],
        }),
      });
      closeDrawer();
      await loadPlan(plan);
      paintStage();
      toast('Saved');
    } catch (err) {
      toast(err.message, 'bad');
    } finally {
      save.disabled = false;
    }
  };

  save.addEventListener('click', submit);
  drawer.onSubmit = submit;

  openDrawer({ title: node.title || node.id, body, actions: [remove, spacer(), save], onSubmit: submit });
}

function newTask(parent) {
  const body = el('div');
  const under = parent && (state.plan.nodes || []).find((n) => n.id === parent);

  const title = field('Title', 'title', '', {
    placeholder: 'Search filters',
    hint: 'What the card will say. Everything else can be filled in later.',
  });
  const status = statusPicker('todo', 'Where the work stands today.');
  const id = field('ID', 'id', '', {
    placeholder: 'left blank, devtree makes one from the title',
    hint: 'What <code>parent</code> points at, so it is worth keeping short. Cyrillic is transliterated.',
  });

  body.append(title, status, id);
  if (under) {
    const where = el('p', 'note');
    where.innerHTML = `It goes under <strong>${escapeHTML(under.title || under.id)}</strong>.`;
    body.prepend(where);
  }

  const create = el('button', 'btn btn-primary');
  create.type = 'button';
  create.textContent = 'Create';

  const submit = async () => {
    const values = readFields(body);
    if (!values.title) { toast('A task needs a title', 'bad'); return; }
    create.disabled = true;
    try {
      const plan = await api('/api/task', {
        method: 'POST',
        body: JSON.stringify({ title: values.title, id: values.id, parent: parent || '', status: status.read() }),
      });
      closeDrawer();
      await loadPlan(plan);
      paintStage();
      toast('Added');
    } catch (err) {
      toast(err.message, 'bad');
    } finally {
      create.disabled = false;
    }
  };

  create.addEventListener('click', submit);
  openDrawer({ title: under ? 'New task' : 'New top-level task', body, actions: [spacer(), create], onSubmit: submit });
}

async function removeTask(node) {
  const kids = (state.plan.nodes || []).filter((n) => n.parent === node.id);
  const message = kids.length
    ? `<strong>${escapeHTML(node.title || node.id)}</strong> has ${kids.length} task${kids.length > 1 ? 's' : ''} under it.
       They move up to its parent rather than disappearing.`
    : `<strong>${escapeHTML(node.title || node.id)}</strong> will be removed from the plan.`;

  if (!await ask({ title: 'Delete this task?', message, confirm: 'Delete', danger: true })) return;

  try {
    const plan = await api(`/api/task/${encodeURIComponent(node.id)}`, { method: 'DELETE' });
    state.selected = null;
    await loadPlan(plan);
    paintStage();
    toast('Deleted');
  } catch (err) {
    toast(err.message, 'bad');
  }
}

// ---------------------------------------------------------------------------
// documents
// ---------------------------------------------------------------------------

function openDocument(doc) {
  // A document is a destination, and its name already says which drawing it
  // holds — so opening one switches the stage to exactly that.
  const view = doc.kind === 'markdown' ? 'mermaid' : doc.kind === 'page' ? 'page' : doc.kind;
  if (doc.theme && doc.theme !== state.theme) setTheme(doc.theme);
  setView(view);
}

function newDocument() {
  const body = el('div');
  const path = field('Path', 'path', '', {
    placeholder: 'docs/assets/board-dark.svg',
    hint: 'The name decides the drawing: <code>.svg</code> is the tree, '
        + '<code>board.svg</code> the board, <code>-dark.svg</code> the dark palette, '
        + '<code>.html</code> the page, anything else the Mermaid block.',
  });
  body.append(path);

  const create = el('button', 'btn btn-primary');
  create.type = 'button';
  create.textContent = 'Add';

  const submit = async () => {
    const values = readFields(body);
    create.disabled = true;
    try {
      const plan = await api('/api/document', { method: 'POST', body: JSON.stringify({ path: values.path }) });
      closeDrawer();
      await loadPlan(plan);
      toast('Added — write the outputs to put it on disk');
    } catch (err) {
      toast(err.message, 'bad');
    } finally {
      create.disabled = false;
    }
  };

  create.addEventListener('click', submit);
  openDrawer({ title: 'New document', body, actions: [spacer(), create], onSubmit: submit });
}

// ---------------------------------------------------------------------------
// chrome
// ---------------------------------------------------------------------------

// setSection points the panel at a list. Clicking the section already showing
// folds the panel away, and clicking any section brings it back — the collapse
// button lives inside the panel, so on its own it would be a one-way door.
function setSection(section) {
  if (collapsed()) {
    setCollapsed(false);
  } else if (section === state.section) {
    setCollapsed(true);
    return;
  }

  state.section = section;
  document.querySelectorAll('.rail-btn[data-section]').forEach((btn) => {
    btn.setAttribute('aria-pressed', String(btn.dataset.section === section));
  });
  paint();
  requestAnimationFrame(() => slidePill($('scope')));
}

function collapsed() { return document.querySelector('.app').dataset.collapsed !== undefined; }

function setCollapsed(shut) {
  const app = document.querySelector('.app');
  const side = document.querySelector('.side');
  if (shut) app.dataset.collapsed = ''; else delete app.dataset.collapsed;

  // A panel folded to nothing must not still be tabbable, and it must not
  // answer a screen reader either.
  side.inert = shut;
  side.setAttribute('aria-hidden', String(shut));
  $('side-collapse').title = shut ? 'Show the panel' : 'Collapse the panel';
}

const views = ['tree', 'board', 'page', 'mermaid', 'yaml'];

// setView also writes the view into the address, so a reload comes back to the
// same one and a view can be handed to somebody as a link.
function setView(view) {
  if (!views.includes(view)) view = 'tree';
  state.view = view;

  const seg = $('views');
  seg.querySelectorAll('button').forEach((btn) => {
    btn.setAttribute('aria-selected', String(btn.dataset.view === view));
  });
  slidePill(seg);

  if (location.hash.slice(1) !== view) history.replaceState(null, '', `#${view}`);
  paintStage();
}

function setScope(scope) {
  state.scope = scope;
  const seg = $('scope');
  seg.querySelectorAll('button').forEach((btn) => {
    btn.setAttribute('aria-selected', String(btn.dataset.scope === scope));
  });
  slidePill(seg);
  paint();
}

// slidePill moves the pill to sit exactly under the selected button. Measured
// rather than assumed, because the labels are words and words are not equal.
function slidePill(seg) {
  const active = seg.querySelector('[aria-selected="true"]');
  const pill = seg.querySelector('.seg-pill');
  if (!active || !pill) return;
  pill.style.setProperty('--x', `${active.offsetLeft - 3}px`);
  pill.style.setProperty('--w', `${active.offsetWidth}px`);
  requestAnimationFrame(() => seg.dataset.ready = '1');
}

function setTheme(theme) {
  state.theme = theme;
  document.documentElement.dataset.theme = theme;
  localStorage.setItem('devtree-theme', theme);
  paintStage();
}

let toastTimer = null;
function toast(message, tone) {
  const node = $('toast');
  node.textContent = message;
  if (tone) node.dataset.tone = tone; else delete node.dataset.tone;
  node.dataset.show = '';
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => delete node.dataset.show, tone === 'bad' ? 4200 : 1900);
}

async function writeOutputs() {
  const button = $('write-btn');
  try {
    const result = await api('/api/render', { method: 'POST' });
    button.dataset.saved = '';
    setTimeout(() => delete button.dataset.saved, 1100);
    const written = (result && result.written) || [];
    toast(written.length ? `Wrote ${written.length} file${written.length > 1 ? 's' : ''}` : 'Nothing to write');
    await loadPlan();
  } catch (err) {
    toast(err.message, 'bad');
  }
}

// isTyping keeps a shortcut from firing while somebody is filling in a field.
function isTyping(node) {
  return !!node && (node.tagName === 'INPUT' || node.tagName === 'TEXTAREA');
}

function escapeHTML(text) {
  return String(text).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
}

// ---------------------------------------------------------------------------
// wiring
// ---------------------------------------------------------------------------

function wire() {
  document.querySelectorAll('.rail-btn[data-section]').forEach((btn) => {
    btn.addEventListener('click', () => setSection(btn.dataset.section));
  });

  $('views').querySelectorAll('button').forEach((btn) => {
    btn.addEventListener('click', () => setView(btn.dataset.view));
  });
  $('scope').querySelectorAll('button').forEach((btn) => {
    btn.addEventListener('click', () => setScope(btn.dataset.scope));
  });

  $('theme-btn').addEventListener('click', () => setTheme(state.theme === 'dark' ? 'light' : 'dark'));
  $('write-btn').addEventListener('click', writeOutputs);

  $('side-add').addEventListener('click', () => {
    if (state.section === 'docs') newDocument(); else newTask('');
  });
  $('side-collapse').addEventListener('click', () => setCollapsed(!collapsed()));

  $('node-add').addEventListener('click', (event) => {
    event.stopPropagation();
    newTask($('node-add').dataset.parent || '');
  });

  $('search').addEventListener('input', (event) => { state.search = event.target.value; paint(); });

  $('crumb').addEventListener('click', () => {
    const plan = state.plan || {};
    const body = el('div', 'note');
    body.innerHTML = `
      <p><strong>${escapeHTML(plan.project || 'devtree')}</strong> — ${plan.totals ? `${plan.totals[0]} of ${plan.totals[1]} tasks done` : 'no tasks yet'}.</p>
      <p>The plan lives in <code>.devtree/tree.yaml</code>. Everything here writes to that one file, so a
         terminal open beside this page stays in step.</p>
      ${plan.repo ? `<p>Repository: <code>${escapeHTML(plan.repo)}</code></p>` : ''}
      <p>${(plan.docs || []).length} output${(plan.docs || []).length === 1 ? '' : 's'} named. Previews write nothing —
         the button on the rail is what puts them on disk.</p>`;
    openDrawer({ title: 'This plan', body, actions: [] });
  });

  $('drawer-close').addEventListener('click', closeDrawer);
  $('scrim').addEventListener('click', closeDrawer);

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') { closeDrawer(); return; }
    // Enter submits from a single-line field, the way a form would.
    if (event.key === 'Enter' && !event.shiftKey && drawer.onSubmit) {
      const target = event.target;
      if (target && target.tagName === 'INPUT') { event.preventDefault(); drawer.onSubmit(); }
    }
    if (event.key === '/' && document.activeElement !== $('search')) {
      event.preventDefault();
      if (collapsed()) setCollapsed(false);
      $('search').focus();
    }
    // The same key folds and unfolds the panel, which is the only way back
    // once the button has gone with it.
    if (event.key === '[' && !isTyping(event.target)) {
      event.preventDefault();
      setCollapsed(!collapsed());
    }
  });

  // A panel that has left the screen must not still be tabbable, and one that
  // is on screen must keep the focus inside it.
  document.addEventListener('focusin', (event) => {
    const node = $('drawer');
    if (node.hidden || node.contains(event.target)) return;
    const first = node.querySelector('button, input, textarea');
    if (first) first.focus();
  });

  addEventListener('resize', () => { slidePill($('views')); slidePill($('scope')); });
}

// live tells the page when the plan changed underneath it — an edit made in a
// terminal shows up here within a second. ?live=0 turns it off, for a proxy
// that will not hold a connection open and for tools that wait for a page to
// go quiet before capturing it.
function live() {
  if (new URLSearchParams(location.search).get('live') === '0') return;
  const source = new EventSource('/api/events');
  source.addEventListener('changed', async () => {
    await loadPlan();
    paintStage();
  });
}

async function boot() {
  document.documentElement.dataset.theme = state.theme;
  wire();

  try {
    state.glyphs = await api('/api/glyphs');
  } catch { /* the page still works, just without marks */ }
  fillGlyphs();

  try {
    await loadPlan();
  } catch (err) {
    $('list').append(hint(String(err.message || err)));
  }

  slidePill($('scope'));
  setView(location.hash.slice(1) || 'tree');
  live();

  // Back and forward through the views, since they are in the address now.
  addEventListener('hashchange', () => setView(location.hash.slice(1) || 'tree'));
}

boot();
