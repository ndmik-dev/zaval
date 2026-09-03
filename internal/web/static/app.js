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
  palette.querySelector('input[name=q]').focus();
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
palette?.addEventListener('keydown', (e) => {
  if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
    e.preventDefault();
    palette.querySelector('input[name=force_now]').value = '1';
    palette.querySelector('form').requestSubmit();
  }
});
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
