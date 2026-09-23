// Lines. A line is read as rendered text and edited as its raw text in a
// textarea that grows with it; a backdrop behind the textarea colours the
// syntax (#tag, link, ? waiting, → command). Keyboard focus is a separate
// notion from editing: ↑/↓ walk lines from the moment the page opens, Enter
// opens the focused line, Esc steps back out.
function lineOf(el) { return el.closest('.ln'); }
function allLines() { return [...document.querySelectorAll('.ln:not(.hist)')]; }
function editable(line) { return !!line?.querySelector('.raw'); }
function size(area) { area.style.height = 'auto'; area.style.height = area.scrollHeight + 'px'; }

// ---- syntax colouring behind the textarea
const tagColors = {};
(document.getElementById('app')?.dataset.tags || '').split(' ').forEach((kv) => {
  const [slug, color] = kv.split(':');
  if (slug) tagColors[slug] = color;
});
function esc(t) { return t.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }
function paint(area) {
  const hl = area.previousElementSibling;
  if (!hl || !hl.classList.contains('hl')) return;
  const tokens = area.value.split(/(\s+)/);
  let waiting = false;
  let html = '';
  tokens.forEach((tok, i) => {
    if (!tok || /^\s+$/.test(tok)) { html += tok; return; }
    if (i === 0 && (tok === '/release' || tok === '/реліз')) { html += `<span class="r">${esc(tok)}</span>`; return; }
    if (/^https?:\/\//.test(tok)) { html += `<span class="u">${esc(tok)}</span>`; return; }
    if (tok.length > 1 && tok[0] === '#') {
      const slug = Object.keys(tagColors).find((k) => k.startsWith(tok.slice(1).toLowerCase()));
      const style = slug ? ` style="color:${tagColors[slug]}"` : '';
      html += `<span class="h"${style}>${esc(tok)}</span>`;
      waiting = false;
      return;
    }
    if (tok === '?') { waiting = true; html += `<span class="w">?</span>`; return; }
    if (tok === '→' || tok === '->') { html += `<span class="c">${esc(tok)}</span>`; waiting = false; return; }
    if (tokens[i - 2] === '→' || tokens[i - 2] === '->') { html += `<span class="c">${esc(tok)}</span>`; return; }
    html += waiting ? `<span class="w">${esc(tok)}</span>` : esc(tok);
  });
  hl.innerHTML = html + (area.value.endsWith('\n') ? ' ' : '');
}

// ---- editing
function titleEnd(form, area) {
  // The caret lands after the title, before "? waiting", "#tag" and links.
  const t = form.querySelector('.tx .t')?.textContent.trim();
  if (!t) return area.value.length;
  const at = area.value.indexOf(t);
  return at < 0 ? area.value.length : at + t.length;
}
function startEdit(form, text, caret) {
  if (!form) return;
  if (!editable(form)) { focusLine(form); return; }
  const area = form.querySelector('.raw');
  const ed = area.closest('.ed');
  if (!form.classList.contains('editing')) {
    form.classList.add('editing');
    ed.hidden = false;
    // The server's text, not what a morph may have kept in the box.
    area.value = text !== undefined ? text : area.defaultValue;
    area.dataset.orig = area.defaultValue;
    delete area.dataset.saving;
    if (caret === undefined && text === undefined) caret = titleEnd(form, area);
  }
  size(area);
  paint(area);
  area.focus({ preventScroll: true });
  const at = caret !== undefined ? caret : area.value.length;
  area.setSelectionRange(at, at);
  focusLine(form, false);
}
function stopEdit(form) {
  const area = form.querySelector('.raw');
  area.value = form.classList.contains('new') ? '' : area.defaultValue;
  form.classList.remove('editing');
  if (!form.classList.contains('new')) area.closest('.ed').hidden = true;
  area.style.height = '';
  paint(area);
}
function changed(area) { return area.value !== (area.dataset.orig ?? area.defaultValue); }
// Save now; `then` names the line to open once the page comes back.
function save(form, then) {
  const area = form.querySelector('.raw');
  area.dataset.saving = '1';
  window.__then = then === 'new' ? '#' + form.id : then;
  form.requestSubmit();
}
function findLine(key) {
  if (!key) return null;
  if (key.startsWith('#')) return document.getElementById(key.slice(1));
  return document.querySelector(`.ln[data-id="${key}"], .ln[data-item="${key}"]`);
}
function lineKey(form) {
  if (form.id) return '#' + form.id;
  return form.dataset.id || form.dataset.item;
}
function nextLine(form, step) {
  const all = allLines();
  return all[all.indexOf(form) + step];
}

// ---- keyboard focus on a line (no editing yet)
function focusedLine() { return document.querySelector('.ln.focused'); }
function focusLine(line, scroll = true) {
  allLines().forEach((l) => l.classList.toggle('focused', l === line));
  if (line && scroll) line.scrollIntoView({ block: 'nearest' });
}
function moveFocus(step) {
  const all = allLines();
  if (!all.length) return;
  const cur = focusedLine();
  const i = cur ? all.indexOf(cur) + step : (step > 0 ? 0 : all.length - 1);
  focusLine(all[Math.min(all.length - 1, Math.max(0, i))]);
}
// Enter on a focused line: edit it, or follow it when it is a link (a project row).
function activate(line) {
  if (!line) return;
  if (editable(line)) startEdit(line);
  else if (line.matches('a')) { window.__openRow = line.id; line.click(); }
}

document.addEventListener('click', (e) => {
  const tx = e.target.closest('.tx[data-edit]');
  if (!tx || e.target.closest('a, button')) return;
  startEdit(lineOf(tx));
});
// The project square selects the line (on a phone, that is how the words appear).
document.addEventListener('click', (e) => {
  const pm = e.target.closest('.ln .pm:not(.dashed)');
  if (!pm) return;
  const line = lineOf(pm);
  focusLine(line.classList.contains('focused') ? null : line, false);
});
document.addEventListener('click', (e) => {
  const row = e.target.closest('a.ln.pr');
  if (row && !row.classList.contains('open')) window.__openRow = row.id;
});
document.addEventListener('focusin', (e) => {
  const area = e.target instanceof Element && e.target.closest('.ln.new .raw');
  if (area) startEdit(lineOf(area));
});
document.addEventListener('input', (e) => {
  const area = e.target instanceof Element && e.target.closest('.ln .raw');
  if (area) { size(area); paint(area); }
});
document.addEventListener('focusout', (e) => {
  const area = e.target instanceof Element && e.target.closest('.ln .raw');
  if (!area) return;
  const form = lineOf(area);
  setTimeout(() => {
    // A morph in between (⌘⏎, ⌥↑) has already re-rendered the line: nothing to do.
    if (!form.isConnected || !form.classList.contains('editing') || area.dataset.saving) return;
    if (document.activeElement === area) return;
    if (form.classList.contains('new')) { if (!area.value.trim()) stopEdit(form); return; }
    if (changed(area)) save(form, null);
    else stopEdit(form);
  }, 0);
});
document.addEventListener('keydown', (e) => {
  const area = e.target instanceof Element && e.target.closest('.ln .raw');
  if (!area) return;
  const form = lineOf(area);
  const isNew = form.classList.contains('new');
  if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
    e.preventDefault();
    form.querySelector('.strike')?.click();
    return;
  }
  if (e.key === 'Enter') {
    e.preventDefault();
    if (isNew) { if (area.value.trim()) save(form, 'new'); return; }
    // Enter moves on like in a text editor: the next line, or the empty one.
    const next = nextLine(form, 1);
    const then = next ? lineKey(next) : null;
    if (changed(area)) save(form, then);
    else { stopEdit(form); startEdit(next); }
    return;
  }
  if (e.key === 'Escape') {
    e.preventDefault();
    stopEdit(form);
    area.blur();
    focusLine(form, false);
    return;
  }
  if (e.altKey && (e.key === 'ArrowUp' || e.key === 'ArrowDown')) {
    if (isNew) return;
    e.preventDefault();
    if (changed(area)) { save(form, lineKey(form)); return; }
    window.__then = lineKey(form); // keep editing the line after it moves
    form.querySelector(e.key === 'ArrowUp' ? '.mv.up' : '.mv.down')?.click();
    return;
  }
  if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
    const next = nextLine(form, e.key === 'ArrowDown' ? 1 : -1);
    if (!next) return;
    e.preventDefault();
    const then = lineKey(next);
    if (!isNew && changed(area)) { save(form, then); return; }
    if (isNew && area.value.trim()) { save(form, then); return; }
    stopEdit(form);
    area.blur();
    startEdit(next);
  }
});
// The project editor: Esc closes it, ↑/↓ walk the fields, ←/→ pick a kind or
// a colour, a save keeps the row focused.
function peditFields(pedit) {
  return [
    pedit.querySelector('input[name=name]'),
    pedit.querySelector('input[name=slug]'),
    pedit.querySelector('.kind input:checked') || pedit.querySelector('.kind input'),
    pedit.querySelector('.sw.on') || pedit.querySelector('.sw'),
    pedit.querySelector('.pact .act'),
  ].filter(Boolean);
}
// Which field of the project editor has focus, so an autosave can hand it back.
function peditFocus() {
  const el = document.activeElement;
  const pedit = el?.closest?.('.pedit');
  if (!pedit) return null;
  if (el.matches('.sw')) return { sel: '.sw.on' };
  if (el.name) return { sel: `[name="${el.name}"]${el.type === 'radio' ? ':checked' : ''}`, caret: el.selectionStart };
  return null;
}
document.addEventListener('keydown', (e) => {
  const pedit = e.target instanceof Element && e.target.closest('.pedit');
  if (!pedit) return;
  if (e.key === 'Escape') { e.preventDefault(); document.getElementById(pedit.dataset.row)?.click(); return; }
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault();
    const fields = peditFields(pedit);
    const at = fields.findIndex((f) => f === e.target || f.contains(e.target));
    const next = fields[Math.min(fields.length - 1, Math.max(0, at + (e.key === 'ArrowDown' ? 1 : -1)))];
    next?.focus();
    if (next?.matches('input[type=text], input:not([type])')) next.setSelectionRange(next.value.length, next.value.length);
    return;
  }
  if ((e.key === 'ArrowLeft' || e.key === 'ArrowRight') && e.target.matches('.sw')) {
    e.preventDefault();
    const sws = [...pedit.querySelectorAll('.sw[data-color]')];
    const i = sws.indexOf(e.target);
    const next = sws[Math.min(sws.length - 1, Math.max(0, i + (e.key === 'ArrowRight' ? 1 : -1)))];
    if (next) { next.click(); next.focus(); }
  }
});
document.addEventListener('submit', (e) => {
  const pedit = e.target.closest?.('.pedit');
  if (pedit) { window.__focusKey = '#' + pedit.dataset.row; window.__peditFocus = peditFocus(); }
});

// The page is morphed after every save. Remember what was being edited (and
// the caret) and what was focused, and put both back — a blur-save must not
// swallow the click that started editing another line.
document.addEventListener('htmx:before:swap', () => {
  // Two lines can be "editing" at once: the one being saved and the one just clicked.
  const form = [...document.querySelectorAll('.ln.editing')].find((f) => !f.querySelector('.raw').dataset.saving);
  const area = form?.querySelector('.raw');
  window.__editing = form ? { key: lineKey(form), text: area.value, caret: area.selectionStart } : null;
  const f = focusedLine();
  if (f && !window.__focusKey) window.__focusKey = lineKey(f);
  if (!window.__peditFocus) window.__peditFocus = peditFocus();
});
document.addEventListener('htmx:after:swap', () => {
  document.querySelectorAll('.ln.new .raw').forEach((a) => { if (!a.matches(':focus')) { a.value = ''; paint(a); } }); // a morph keeps typed text; the line is saved
  const then = window.__then;
  const was = window.__editing;
  const focus = window.__focusKey;
  const opened = window.__openRow;
  window.__then = null;
  window.__editing = null;
  window.__focusKey = null;
  window.__openRow = null;
  if (then) { startEdit(findLine(then)); return; }
  if (was) {
    const form = findLine(was.key);
    if (form) { startEdit(form, was.text, was.caret); return; }
  }
  if (opened) {
    const name = document.querySelector('.pedit input[name=name]');
    if (name) { name.focus(); name.setSelectionRange(name.value.length, name.value.length); }
    focusLine(document.getElementById(opened), false);
    return;
  }
  if (focus) focusLine(findLine(focus), false);
  const pf = window.__peditFocus;
  window.__peditFocus = null;
  if (pf) {
    const el = document.querySelector('.pedit ' + pf.sel);
    if (el) { el.focus({ preventScroll: true }); if (pf.caret !== undefined && el.setSelectionRange) el.setSelectionRange(pf.caret, pf.caret); }
  }
});
// A mouse click on an action word must not leave a focus ring on it.
document.addEventListener('click', (e) => {
  const b = e.target.closest('.ln-act button, .ln-act a, .box, .act, .strike, .sw');
  if (b && e.detail > 0) setTimeout(() => b.blur(), 0);
});

// The undo toast: ⌘Z presses its button; it leaves on its own after five seconds.
document.addEventListener('keydown', (e) => {
  if ((e.metaKey || e.ctrlKey) && e.code === 'KeyZ' && !e.shiftKey) {
    const undo = document.querySelector('#toast .undo');
    if (undo && !typing(e)) { e.preventDefault(); undo.click(); }
  }
});
// After an undo the line it brought back is the one to stand on.
document.addEventListener('click', (e) => {
  const undo = e.target.closest('#toast .undo');
  if (undo) window.__focusKey = undo.closest('#toast').dataset.id;
});
function armToast() {
  const toast = document.getElementById('toast');
  if (!toast || toast.dataset.armed) return;
  toast.dataset.armed = '1';
  setTimeout(() => toast.remove(), 5000);
}
armToast();
document.addEventListener('htmx:after:swap', armToast);

// Keys outside inputs.
const help = document.getElementById('help');
function typing(e) { return e.target instanceof Element && e.target.closest('input, textarea, select, [contenteditable]'); }
document.addEventListener('click', (e) => {
  if (e.target.closest('[data-open-help]')) help?.showModal();
  if (e.target.closest('[data-close-help]') || e.target === help) help?.close();
});
// Keys are read by position (e.code), so they work under the Ukrainian layout too.
function newLineForm() {
  const cur = focusedLine();
  return cur?.closest('.chk')?.querySelector('.ln.new')
    || document.getElementById('new-line') || document.getElementById('new-project') || document.querySelector('.ln.new');
}
function switchProject(step) {
  const words = [...document.querySelectorAll('.pwords a, .pwords .on')];
  const i = words.findIndex((w) => w.classList.contains('on'));
  const next = words[i + step];
  if (next?.matches('a')) next.click();
}
document.addEventListener('keydown', (e) => {
  if (typing(e) || e.altKey) return;
  if (e.target instanceof Element && e.target.closest('.pedit')) return; // the project editor has its own keys
  if (document.querySelector('dialog[open]') && e.code !== 'Slash') return;
  const cur = focusedLine();
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') { e.preventDefault(); cur?.querySelector('.strike')?.click(); return; }
  if (e.metaKey || e.ctrlKey) return;
  switch (e.code) {
    case 'Slash': e.preventDefault(); if (help) help.open ? help.close() : help.showModal(); break;
    case 'ArrowDown': case 'KeyJ': e.preventDefault(); moveFocus(1); break;
    case 'ArrowUp': case 'KeyK': e.preventDefault(); moveFocus(-1); break;
    case 'ArrowLeft': switchProject(-1); break;
    case 'ArrowRight': switchProject(1); break;
    case 'Enter': e.preventDefault(); activate(cur); break;
    case 'Escape': focusLine(null); break;
    case 'KeyX': cur?.querySelector('.strike')?.click(); break;
    case 'KeyB': cur?.querySelector('.ln-act .send')?.click(); break;
    case 'Backspace': case 'Delete': if (cur) { e.preventDefault(); cur.querySelector('.ln-act .del')?.click(); } break;
    case 'KeyN': e.preventDefault(); startEdit(newLineForm()); break;
    case 'Digit1': location.href = '/'; break;
    case 'Digit2': location.href = '/backlog'; break;
    case 'Digit3': location.href = '/releases'; break;
    case 'Digit4': location.href = '/projects'; break;
  }
});
document.querySelectorAll('.ln.new .raw').forEach(paint);

// Drag ordering within a page. The order sent is every line of that state in
// document order, so the backlog's project bands share one sequence.
const sortables = new WeakSet(); // a morph erases attributes, so remember instances by element
function initSortable() {
  document.querySelectorAll('.sortable').forEach((list) => {
    if (sortables.has(list)) return;
    sortables.add(list);
    new Sortable(list, {
      group: 'lines-' + list.dataset.state,
      draggable: '.ln:not(.new)',
      handle: '.pm',
      animation: 120,
      ghostClass: 'drag-ghost',
      onEnd: () => sendOrder(list.dataset.state),
    });
  });
}
function sendOrder(state) {
  const form = document.getElementById('order');
  if (!form) return;
  form.innerHTML = '';
  const add = (name, value) => {
    const input = document.createElement('input');
    input.type = 'hidden';
    input.name = name;
    input.value = value;
    form.appendChild(input);
  };
  add('state', state);
  document.querySelectorAll(`.sortable[data-state="${state}"] .ln[data-id]`).forEach((row) => add('id', row.dataset.id));
  form.dispatchEvent(new CustomEvent('order', { bubbles: true }));
}
initSortable();
document.addEventListener('htmx:after:swap', initSortable);

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js');
}

// A shadow under the sticky header once the column is scrolled.
document.addEventListener('scroll', (e) => {
  const pane = e.target instanceof Element && e.target.closest('.col');
  if (pane) pane.classList.toggle('scrolled', pane.scrollTop > 2);
}, true);

// Confirmation modal for [data-confirm]: the first click is held back and shown
// in a dialog; accepting re-clicks the element with a one-shot pass flag.
const confirmDialog = document.getElementById('confirm');
let confirmTarget = null;
document.addEventListener('click', (e) => {
  const el = e.target.closest('[data-confirm]');
  if (!el || !confirmDialog || el.dataset.confirmPass) return;
  e.stopPropagation();
  e.preventDefault();
  confirmTarget = el;
  confirmDialog.querySelector('.confirm-text').textContent = el.dataset.confirm;
  const ok = confirmDialog.querySelector('[data-confirm-accept]');
  ok.textContent = el.dataset.confirmOk || 'OK';
  ok.classList.toggle('danger', el.hasAttribute('data-confirm-danger'));
  confirmDialog.showModal();
  ok.focus();
}, true);
confirmDialog?.addEventListener('click', (e) => {
  if (e.target.closest('[data-confirm-accept]')) {
    const el = confirmTarget;
    confirmDialog.close();
    if (!el) return;
    el.dataset.confirmPass = '1';
    el.click();
    delete el.dataset.confirmPass;
  } else if (e.target.closest('[data-confirm-cancel]') || e.target === confirmDialog) {
    confirmDialog.close();
  }
});
confirmDialog?.addEventListener('close', () => { confirmTarget = null; });

// Color swatches in project settings: a preset click sets the hidden color input.
document.addEventListener('click', (e) => {
  const sw = e.target.closest('.sw[data-color]');
  if (!sw) return;
  const wrap = sw.closest('.swatches');
  const color = wrap.querySelector('input[type=color]');
  color.value = sw.dataset.color;
  color.dispatchEvent(new Event('change', { bubbles: true }));
  wrap.querySelectorAll('.sw').forEach((s) => s.classList.toggle('on', s === sw));
  const form = sw.closest('form');
  if (form) form.style.setProperty('--project', sw.dataset.color);
});
