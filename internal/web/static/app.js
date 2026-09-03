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
