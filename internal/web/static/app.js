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
