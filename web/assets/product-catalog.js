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
  });
})();
