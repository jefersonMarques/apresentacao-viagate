(() => {
  const brl = new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });

  function centsFromValue(value) {
    const digits = String(value || '').replace(/\D/g, '');
    if (!digits) return null;
    return Number(digits) / 100;
  }

  function formatInput(input) {
    const amount = centsFromValue(input.value);
    input.value = amount === null ? '' : brl.format(amount);
    if (input.value) {
      try { input.setSelectionRange(input.value.length, input.value.length); } catch (_) {}
    }
  }

  function proposalSummary(form) {
    const rows = Array.from(form.querySelectorAll('[data-proposal-product]'));
    let included = 0;
    let optional = 0;
    let total = 0;

    rows.forEach((row) => {
      if (row.hidden) return;
      const enabled = row.querySelector('[data-product-enabled]');
      const optionalInput = row.querySelector('[data-product-optional]');
      const priceInput = row.querySelector('[name="item_price"]');
      const amount = centsFromValue(priceInput?.value || '');
      const active = Boolean(enabled?.checked && amount !== null);
      if (!active) return;
      if (optionalInput?.checked) optional += 1;
      else {
        included += 1;
        total += amount;
      }
    });

    const count = form.querySelector('[data-proposal-summary-count]');
    const optionalCount = form.querySelector('[data-proposal-summary-optional]');
    const totalElement = form.querySelector('[data-proposal-summary-total]');
    if (count) count.textContent = String(included);
    if (optionalCount) optionalCount.textContent = String(optional);
    if (totalElement) totalElement.textContent = brl.format(total);
  }

  function initMoneyFields() {
    const fields = Array.from(document.querySelectorAll('input[data-money]'));
    if (!fields.length) return;

    fields.forEach((input) => {
      if (!(input instanceof HTMLInputElement) || input.dataset.moneyBound === '1') return;
      input.dataset.moneyBound = '1';
      input.inputMode = 'numeric';
      formatInput(input);
      input.addEventListener('input', () => {
        formatInput(input);
        const form = input.closest('[data-proposal-editor]');
        if (form instanceof HTMLFormElement) {
          queueMicrotask(() => proposalSummary(form));
        }
      });
    });

    document.querySelectorAll('[data-proposal-editor]').forEach((form) => {
      if (!(form instanceof HTMLFormElement)) return;
      form.addEventListener('change', () => queueMicrotask(() => proposalSummary(form)));
      form.addEventListener('click', (event) => {
        if (!event.target.closest('[data-proposal-preset]')) return;
        window.setTimeout(() => {
          fields.forEach((input) => {
            if (input instanceof HTMLInputElement && input.closest('form') === form) formatInput(input);
          });
          proposalSummary(form);
        }, 0);
      });
      queueMicrotask(() => proposalSummary(form));
    });
  }

  document.addEventListener('DOMContentLoaded', initMoneyFields);
})();
