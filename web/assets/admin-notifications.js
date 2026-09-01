(() => {
  const trigger = document.querySelector('.topbar-notification');
  if (!(trigger instanceof HTMLAnchorElement)) return;

  const host = document.createElement('div');
  host.className = 'notification-popover-host';
  host.hidden = true;
  trigger.insertAdjacentElement('afterend', host);
  let loaded = false;

  async function load() {
    if (loaded) return;
    host.innerHTML = '<div class="notification-popover-loading">Carregando notificações...</div>';
    try {
      const response = await fetch('/admin/notifications?preview=1', {
        credentials: 'same-origin',
        headers: { Accept: 'text/html' },
      });
      if (!response.ok) throw new Error('Não foi possível carregar as notificações.');
      host.innerHTML = await response.text();
      loaded = true;
    } catch (error) {
      host.innerHTML = `<div class="notification-popover-loading is-error">${error?.message || 'Não foi possível carregar as notificações.'}</div>`;
    }
  }

  async function open() {
    host.hidden = false;
    trigger.setAttribute('aria-expanded', 'true');
    await load();
  }

  function close() {
    host.hidden = true;
    trigger.setAttribute('aria-expanded', 'false');
  }

  trigger.setAttribute('aria-haspopup', 'dialog');
  trigger.setAttribute('aria-expanded', 'false');
  trigger.addEventListener('click', async (event) => {
    event.preventDefault();
    if (!host.hidden) {
      close();
      return;
    }
    await open();
  });

  document.addEventListener('click', (event) => {
    if (host.hidden) return;
    const target = event.target;
    if (target instanceof Node && (host.contains(target) || trigger.contains(target))) return;
    close();
  });
  document.addEventListener('keydown', (event) => {
    if (event.key !== 'Escape' || host.hidden) return;
    close();
    trigger.focus();
  });
})();
