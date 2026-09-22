(() => {
  function dialogByKey(key) {
    return Array.from(document.querySelectorAll('[data-catalog-dialog]'))
      .find((dialog) => dialog.dataset.catalogDialog === key);
  }

  function openDialog(dialog) {
    if (typeof HTMLDialogElement === 'undefined' || !(dialog instanceof HTMLDialogElement)) return;
    if (typeof dialog.showModal === 'function') {
      dialog.showModal();
    } else {
      dialog.setAttribute('open', '');
    }
    window.requestAnimationFrame(() => {
      dialog.querySelector('input:not([type="hidden"]), select, textarea')?.focus();
    });
  }

  function closeDialog(dialog) {
    if (typeof HTMLDialogElement === 'undefined' || !(dialog instanceof HTMLDialogElement)) return;
    if (typeof dialog.close === 'function') {
      dialog.close();
    } else {
      dialog.removeAttribute('open');
    }
  }

  function addDependencyGroup(button) {
    const container = button.closest('.catalog-dependencies-body');
    const groups = container?.querySelector('[data-dependency-groups]');
    const template = container?.querySelector('[data-dependency-group-template]');
    if (!(groups instanceof HTMLElement) || !(template instanceof HTMLTemplateElement)) return;

    const index = Number(groups.dataset.nextIndex || '0');
    const fragment = template.content.cloneNode(true);
    const mode = fragment.querySelector('[data-dependency-mode-template]');
    const products = fragment.querySelector('[data-dependency-products-template]');
    if (mode instanceof HTMLSelectElement) {
      mode.name = `dependency_mode_${index}`;
      mode.removeAttribute('data-dependency-mode-template');
    }
    if (products instanceof HTMLSelectElement) {
      products.name = `dependency_product_${index}`;
      products.removeAttribute('data-dependency-products-template');
    }
    groups.appendChild(fragment);
    groups.dataset.nextIndex = String(index + 1);
  }

  document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-catalog-dialog-open]').forEach((button) => {
      button.addEventListener('click', () => {
        const key = button.getAttribute('data-catalog-dialog-open');
        if (!key) return;
        openDialog(dialogByKey(key));
      });
    });

    document.querySelectorAll('[data-catalog-new-product-category]').forEach((button) => {
      button.addEventListener('click', () => {
        const dialog = dialogByKey('new-product');
        if (typeof HTMLDialogElement === 'undefined' || !(dialog instanceof HTMLDialogElement)) return;
        const category = button.getAttribute('data-catalog-new-product-category');
        const select = dialog.querySelector('[data-catalog-product-category]');
        if (select instanceof HTMLSelectElement && category) {
          select.value = category;
        }
        openDialog(dialog);
      });
    });

    document.querySelectorAll('[data-catalog-dialog]').forEach((dialog) => {
      dialog.querySelectorAll('[data-catalog-dialog-close]').forEach((button) => {
        button.addEventListener('click', () => closeDialog(dialog));
      });

      dialog.addEventListener('click', (event) => {
        if (event.target === dialog) closeDialog(dialog);
      });
    });

    document.querySelectorAll('[data-add-dependency-group]').forEach((button) => {
      button.addEventListener('click', () => addDependencyGroup(button));
    });

    document.addEventListener('click', (event) => {
      const remove = event.target.closest('[data-remove-dependency-group]');
      if (!remove) return;
      remove.closest('[data-dependency-group]')?.remove();
    });

    document.querySelectorAll('[data-catalog-confirm]').forEach((form) => {
      form.addEventListener('submit', (event) => {
        const message = form.getAttribute('data-catalog-confirm') || 'Confirmar esta ação?';
        if (!window.confirm(message)) event.preventDefault();
      });
    });
  });
})();
