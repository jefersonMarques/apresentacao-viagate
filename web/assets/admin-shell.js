(() => {
  const sidebarStorageKey = 'viagate.admin.sidebar.collapsed';

  function initAdminSidebar() {
    const shell = document.querySelector('[data-admin-shell]');
    if (!shell) return;

    try {
      if (localStorage.getItem(sidebarStorageKey) === '1') shell.classList.add('sidebar-collapsed');
    } catch (_) {}

    document.querySelectorAll('[data-sidebar-toggle]').forEach((button) => {
      button.addEventListener('click', () => {
        shell.classList.toggle('sidebar-collapsed');
        try {
          localStorage.setItem(sidebarStorageKey, shell.classList.contains('sidebar-collapsed') ? '1' : '0');
        } catch (_) {}
      });
    });

    const closeMobile = () => shell.classList.remove('sidebar-mobile-open');
    document.querySelectorAll('[data-sidebar-mobile-toggle]').forEach((button) => {
      button.addEventListener('click', () => shell.classList.add('sidebar-mobile-open'));
    });
    document.querySelectorAll('[data-sidebar-backdrop]').forEach((button) => {
      button.addEventListener('click', closeMobile);
    });
    document.addEventListener('keydown', (event) => {
      if (event.key === 'Escape') closeMobile();
    });

    const path = window.location.pathname.replace(/\/$/, '') || '/';
    document.querySelectorAll('[data-admin-nav]').forEach((link) => {
      const href = new URL(link.href, window.location.origin).pathname.replace(/\/$/, '') || '/';
      const active = href === '/admin' ? path === href : path === href || path.startsWith(`${href}/`);
      link.classList.toggle('is-active', active);
    });
  }

  document.addEventListener('DOMContentLoaded', initAdminSidebar);
})();
