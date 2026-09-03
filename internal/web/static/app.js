// Escape closes the drawer; everything else is htmx.
document.addEventListener('keydown', (e) => {
  if (e.key !== 'Escape') return;
  const inField = e.target instanceof Element && e.target.closest('input, textarea');
  const close = document.querySelector('.drawer-close');
  if (close && !inField) close.click();
});
