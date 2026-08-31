(() => {
  function normalize(value) {
    return String(value || '')
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .toLowerCase()
      .trim();
  }

  function initList(root) {
    const rows = Array.from(root.querySelectorAll('[data-admin-list-row]'));
    const search = root.querySelector('[data-admin-list-search]');
    const filters = Array.from(root.querySelectorAll('[data-admin-list-filter]'));
    const count = root.querySelector('[data-admin-list-count]');
    const empty = root.querySelector('[data-admin-list-empty]');
    const clear = root.querySelector('[data-admin-list-clear]');

    const apply = () => {
      const query = normalize(search?.value);
      let visible = 0;
      let hasActiveFilter = Boolean(query);

      rows.forEach((row) => {
        let show = !query || normalize(row.getAttribute('data-search')).includes(query);
        filters.forEach((filter) => {
          const selected = filter.value || '';
          if (!selected) return;
          hasActiveFilter = true;
          const key = filter.getAttribute('data-admin-list-filter');
          const value = key ? row.getAttribute(`data-filter-${key}`) || '' : '';
          if (value !== selected) show = false;
        });
        row.hidden = !show;
        if (show) visible += 1;
      });

      if (count) count.textContent = String(visible);
      if (empty) empty.hidden = visible !== 0;
      if (clear) clear.hidden = !hasActiveFilter;
    };

    search?.addEventListener('input', apply);
    filters.forEach((filter) => filter.addEventListener('change', apply));
    clear?.addEventListener('click', () => {
      if (search) search.value = '';
      filters.forEach((filter) => { filter.value = ''; });
      apply();
      search?.focus();
    });

    apply();
  }

  function initActionMenus() {
    document.addEventListener('click', (event) => {
      const active = event.target.closest('.admin-action-menu');
      document.querySelectorAll('.admin-action-menu[open]').forEach((menu) => {
        if (menu !== active) menu.removeAttribute('open');
      });
    });

    document.addEventListener('keydown', (event) => {
      if (event.key !== 'Escape') return;
      document.querySelectorAll('.admin-action-menu[open]').forEach((menu) => menu.removeAttribute('open'));
    });
  }

  document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-admin-list]').forEach(initList);
    initActionMenus();
  });
})();
