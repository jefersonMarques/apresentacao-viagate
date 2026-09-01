(() => {
  function initNotificationPopover() {
    const trigger = document.querySelector('.topbar-notification');
    const host = document.querySelector('#notificationPopover');
    if (!(trigger instanceof HTMLAnchorElement) || !(host instanceof HTMLElement)) return;
    if (trigger.dataset.notificationReady === 'true') return;
    trigger.dataset.notificationReady = 'true';

    function close() {
      host.hidden = true;
      trigger.setAttribute('aria-expanded', 'false');
    }

    function open() {
      host.hidden = false;
      trigger.setAttribute('aria-expanded', 'true');
      if (host.dataset.loaded !== 'true') {
        host.innerHTML = '<div class="notification-popover-loading">Carregando notificações...</div>';
        if (window.htmx) {
          window.htmx.trigger(trigger, 'notification:load');
        }
      }
    }

    trigger.setAttribute('aria-haspopup', 'dialog');
    trigger.setAttribute('aria-expanded', 'false');
    trigger.addEventListener('click', (event) => {
      event.preventDefault();
      if (!host.hidden) {
        close();
        return;
      }
      open();
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

    document.addEventListener('htmx:afterSwap', (event) => {
      if (event.detail?.target !== host) return;
      host.dataset.loaded = 'true';
    });

    document.addEventListener('htmx:responseError', (event) => {
      if (event.detail?.target !== host) return;
      host.innerHTML = '<div class="notification-popover-loading is-error">Não foi possível carregar as notificações.</div>';
    });
  }

  document.addEventListener('DOMContentLoaded', initNotificationPopover);
})();
