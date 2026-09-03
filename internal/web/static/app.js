// Escape closes the drawer; everything else is htmx.
document.addEventListener('keydown', (e) => {
  if (e.key !== 'Escape') return;
  const inField = e.target instanceof Element && e.target.closest('input, textarea');
  const close = document.querySelector('.drawer-close');
  if (close && !inField) close.click();
});

// ⌘K / Ctrl+K opens the palette; ⌘Enter inside it creates straight into "Зараз".
const palette = document.getElementById('palette');
function openPalette() {
  if (!palette || palette.open) return;
  palette.showModal();
  palette.querySelector('input[name=text]').focus();
}
document.addEventListener('keydown', (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault();
    openPalette();
  }
});
document.addEventListener('click', (e) => {
  if (e.target.closest('[data-open-palette]')) openPalette();
  if (e.target === palette) palette.close();
});
// ↑/↓ move between the create row and the matches; Enter activates the highlighted one.
function paletteRows() { return [...palette.querySelectorAll('.prow')]; }
function highlight(i) {
  const all = paletteRows();
  all.forEach((r, k) => r.classList.toggle('hl', k === i));
}
palette?.addEventListener('keydown', (e) => {
  if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
    e.preventDefault();
    palette.querySelector('input[name=force_now]').value = '1';
    palette.querySelector('form').requestSubmit();
    return;
  }
  const all = paletteRows();
  const cur = all.findIndex((r) => r.classList.contains('hl'));
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault();
    if (!all.length) return;
    highlight(Math.min(all.length - 1, Math.max(0, cur + (e.key === 'ArrowDown' ? 1 : -1))));
  } else if (e.key === 'Enter' && cur > 0) {
    e.preventDefault();
    all[cur].click();
    palette.close();
  }
});
palette?.addEventListener('htmx:after:swap', () => { if (paletteRows().length) highlight(0); });
palette?.addEventListener('close', () => {
  palette.querySelector('input[name=force_now]').value = '';
});
// Close only after the form's own POST — the search input's GETs bubble the same event.
palette?.addEventListener('htmx:after:request', (e) => {
  const form = palette.querySelector('form');
  if (e.target === form) {
    form.reset();
    palette.close();
  }
});

// Drag ordering. Lists share a group, so dragging across them changes state.
// Morph keeps the list containers, so one init per container is enough.
function initSortable() {
  document.querySelectorAll('.sortable:not([data-sortable])').forEach((list) => {
    list.dataset.sortable = '1';
    new Sortable(list, {
      group: 'tasks',
      draggable: '.task',
      animation: 120,
      ghostClass: 'drag-ghost',
      onEnd: sendOrder,
    });
  });
}
function sendOrder() {
  const form = document.getElementById('reorder');
  if (!form) return;
  form.querySelectorAll('input[name=now], input[name=backlog]').forEach((i) => i.remove());
  document.querySelectorAll('.sortable').forEach((list) => {
    list.querySelectorAll('.task').forEach((row) => {
      const input = document.createElement('input');
      input.type = 'hidden';
      input.name = list.dataset.state;
      input.value = row.id.replace('task-', '');
      form.appendChild(input);
    });
  });
  form.dispatchEvent(new CustomEvent('reorder', { bubbles: true }));
}
initSortable();
document.addEventListener('htmx:after:swap', initSortable);

// Copy-to-clipboard buttons: data-copy holds the selector of the text source.
document.addEventListener('click', async (e) => {
  const btn = e.target.closest('[data-copy]');
  if (!btn) return;
  const src = document.querySelector(btn.dataset.copy);
  if (!src) return;
  await navigator.clipboard.writeText(src.textContent);
  const label = btn.textContent;
  btn.textContent = 'Скопійовано';
  setTimeout(() => { btn.textContent = label; }, 1200);
});

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js');
}

// Projects: a click anywhere outside the expanded card collapses it.
document.addEventListener('click', (e) => {
  const open = document.querySelector('.pcard.open');
  const link = document.getElementById('collapse-projects');
  if (!open || !link || e.target.closest('.pcard.open') || e.target.closest('.pcard.new') || e.target.closest('a, button')) return;
  link.click();
});

// Custom dropdown (.dd): hidden input + button + menu. Choosing an option
// updates the input and fires a change event so hx-trigger="change" forms react.
function closeDropdowns(except) {
  document.querySelectorAll('.dd.open').forEach((dd) => {
    if (dd === except) return;
    dd.classList.remove('open');
    dd.querySelector('.dd-menu').hidden = true;
    dd.querySelector('.dd-btn').setAttribute('aria-expanded', 'false');
  });
}
document.addEventListener('click', (e) => {
  const btn = e.target.closest('.dd-btn');
  const opt = e.target.closest('.dd-menu [data-value]');
  if (btn) {
    const dd = btn.closest('.dd');
    const open = !dd.classList.contains('open');
    closeDropdowns(dd);
    dd.classList.toggle('open', open);
    dd.querySelector('.dd-menu').hidden = !open;
    btn.setAttribute('aria-expanded', String(open));
    return;
  }
  if (opt) {
    const dd = opt.closest('.dd');
    const input = dd.querySelector('input[type=hidden]');
    input.value = opt.dataset.value;
    dd.querySelector('.dd-label').textContent = opt.textContent.trim();
    const dot = dd.querySelector('.dd-btn .dot');
    if (dot && opt.dataset.color) dot.style.background = opt.dataset.color;
    dd.querySelectorAll('[data-value]').forEach((o) => o.toggleAttribute('aria-selected', o === opt));
    closeDropdowns();
    input.dispatchEvent(new Event('change', { bubbles: true }));
    return;
  }
  if (!e.target.closest('.dd')) closeDropdowns();
});
document.addEventListener('keydown', (e) => { if (e.key === 'Escape') closeDropdowns(); });

// A click anywhere on a task row opens its card; controls inside keep their own behaviour.
document.addEventListener('click', (e) => {
  const row = e.target.closest('.task[data-open]');
  if (!row || e.target.closest('a, button, input, .acts')) return;
  const title = row.querySelector('a.title');
  if (title) title.click();
});

// Keyboard: j/k walk the rows, the rest act on the focused row. Off while typing.
const help = document.getElementById('help');
function typing(e) { return e.target instanceof Element && e.target.closest('input, textarea, select, [contenteditable]'); }
function rows() { return [...document.querySelectorAll('.task[data-open]')]; }
function focusedRow() { return document.querySelector('.task.focused'); }
function focusRow(row) {
  rows().forEach((r) => r.classList.remove('focused'));
  if (!row) return;
  row.classList.add('focused');
  row.scrollIntoView({ block: 'nearest' });
}
function moveFocus(step) {
  const all = rows();
  if (!all.length) return;
  const i = all.indexOf(focusedRow());
  focusRow(all[Math.min(all.length - 1, Math.max(0, i + step))]);
}
document.addEventListener('click', (e) => {
  if (e.target.closest('[data-open-help]')) help?.showModal();
  if (e.target.closest('[data-close-help]') || e.target === help) help?.close();
});
document.addEventListener('keydown', (e) => {
  if (typing(e) || e.metaKey || e.ctrlKey || e.altKey) return;
  if (document.querySelector('dialog[open]') && e.key !== 'Escape' && e.key !== '?') return;
  const row = focusedRow();
  const act = (sel) => { const b = row && row.querySelector(sel); if (b) b.click(); };
  switch (e.key) {
    case '?': e.preventDefault(); if (help) help.open ? help.close() : help.showModal(); break;
    case '/': e.preventDefault(); document.getElementById('q')?.focus(); break;
    case 'j': moveFocus(1); break;
    case 'k': moveFocus(-1); break;
    case 'Enter': if (row) { e.preventDefault(); row.querySelector('a.title')?.click(); } break;
    case 'x': act('.tick'); break;
    case 'f': act('.acts button:not(.del)'); break;
    case 'w': if (row) { row.querySelector('a.title')?.click(); setTimeout(() => document.querySelector('.drawer-wait input')?.focus(), 500); } break;
    case 'l': if (row) { row.querySelector('a.title')?.click(); setTimeout(() => document.querySelector('.inline-add input[name=url]')?.focus(), 500); } break;
    case 'd': document.querySelector('.head-actions .btn')?.click(); break;
    case '1': location.href = '/'; break;
    case '2': location.href = '/releases'; break;
    case '3': location.href = '/journal'; break;
  }
});
// Keep the focus ring on the same task after a morph.
document.addEventListener('htmx:before:swap', () => { const r = focusedRow(); if (r) window.__focusedTask = r.id; });
document.addEventListener('htmx:after:swap', () => { if (window.__focusedTask) { const r = document.getElementById(window.__focusedTask); if (r && !r.classList.contains('focused')) focusRow(r); } });

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
