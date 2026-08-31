(() => {
  let openActionMenu = null;

  function normalize(value) {
    return String(value || '')
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .toLowerCase()
      .trim();
  }

  function updateNoun(root, total) {
    const noun = root.querySelector('[data-admin-list-noun]');
    if (!noun) return;
    noun.textContent = total === 1 ? (noun.dataset.singular || '') : (noun.dataset.plural || '');
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
      updateNoun(root, visible);
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

  function resetPopoverPosition(popover) {
    popover.style.position = '';
    popover.style.left = '';
    popover.style.right = '';
    popover.style.top = '';
    popover.style.bottom = '';
    popover.style.width = '';
  }

  function closeActionMenu({ restoreFocus = false } = {}) {
    if (!openActionMenu) return;
    const { root, trigger, popover, placeholder } = openActionMenu;
    popover.hidden = true;
    resetPopoverPosition(popover);
    if (placeholder.parentNode) placeholder.parentNode.insertBefore(popover, placeholder);
    placeholder.remove();
    trigger.setAttribute('aria-expanded', 'false');
    root.classList.remove('is-open');
    openActionMenu = null;
    if (restoreFocus) trigger.focus();
  }

  function positionActionMenu(trigger, popover) {
    const margin = 8;
    const gap = 6;
    const rect = trigger.getBoundingClientRect();

    popover.hidden = false;
    popover.style.position = 'fixed';
    popover.style.left = '0';
    popover.style.top = '0';

    const measured = popover.getBoundingClientRect();
    const width = Math.min(measured.width || 220, window.innerWidth - margin * 2);
    const height = measured.height;
    const left = Math.max(margin, Math.min(rect.right - width, window.innerWidth - width - margin));
    const below = rect.bottom + gap;
    const above = rect.top - height - gap;
    const top = below + height <= window.innerHeight - margin || above < margin ? below : above;

    popover.style.width = `${width}px`;
    popover.style.left = `${left}px`;
    popover.style.top = `${Math.max(margin, top)}px`;
  }

  function openMenu(root) {
    const trigger = root.querySelector('[data-admin-action-trigger]');
    const popover = root.querySelector('[data-admin-action-popover]');
    if (!trigger || !popover) return;

    if (openActionMenu?.root === root) {
      closeActionMenu({ restoreFocus: true });
      return;
    }

    closeActionMenu();
    const placeholder = document.createComment('admin-action-menu');
    popover.parentNode.insertBefore(placeholder, popover);
    document.body.appendChild(popover);
    root.classList.add('is-open');
    trigger.setAttribute('aria-expanded', 'true');
    openActionMenu = { root, trigger, popover, placeholder };
    positionActionMenu(trigger, popover);

    const firstAction = popover.querySelector('a,button,[tabindex]:not([tabindex="-1"])');
    window.requestAnimationFrame(() => firstAction?.focus());
  }

  function initActionMenus() {
    document.addEventListener('click', (event) => {
      const trigger = event.target.closest('[data-admin-action-trigger]');
      if (trigger) {
        event.preventDefault();
        const root = trigger.closest('[data-admin-action-menu]');
        if (root) openMenu(root);
        return;
      }

      if (!openActionMenu) return;
      if (openActionMenu.popover.contains(event.target)) {
        if (event.target.closest('a,button')) closeActionMenu();
        return;
      }
      closeActionMenu();
    });

    document.addEventListener('keydown', (event) => {
      if (!openActionMenu) return;
      if (event.key === 'Escape') {
        event.preventDefault();
        closeActionMenu({ restoreFocus: true });
        return;
      }
      if (event.key !== 'Tab') return;
      const focusable = Array.from(openActionMenu.popover.querySelectorAll('a,button,[tabindex]:not([tabindex="-1"])'))
        .filter((element) => !element.hasAttribute('disabled'));
      if (focusable.length === 0) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    });

    window.addEventListener('resize', () => {
      if (openActionMenu) positionActionMenu(openActionMenu.trigger, openActionMenu.popover);
    });
    window.addEventListener('scroll', () => closeActionMenu(), true);
  }

  document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-admin-list]').forEach(initList);
    initActionMenus();
  });
})();
