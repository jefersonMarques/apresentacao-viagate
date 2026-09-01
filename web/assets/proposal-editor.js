(() => {
  function parseMoney(value) {
    const normalized = String(value || '')
      .replace(/[^0-9,.-]/g, '')
      .replace(/\./g, '')
      .replace(',', '.');
    const parsed = Number(normalized);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  function updateProduct(row) {
    const enabled = row.querySelector('[data-product-enabled]');
    const optional = row.querySelector('[data-product-optional]');
    const price = row.querySelector('[name="item_price"]');
    const status = row.querySelector('[name="item_status"]');
    const active = Boolean(enabled?.checked);

    row.dataset.productState = active ? (optional?.checked ? 'optional' : 'included') : 'off';
    if (status) status.value = active ? (optional?.checked ? 'optional' : 'included') : 'off';
    if (optional) optional.disabled = !active;
    if (price) price.classList.toggle('is-disabled', !active);
  }

  function initProposalEditor() {
    const form = document.querySelector('[data-proposal-editor]');
    if (!(form instanceof HTMLFormElement)) return;

    const products = Array.from(form.querySelectorAll('[data-proposal-product]'));
    const pricingRadios = Array.from(form.querySelectorAll('[name="pricing_model"]'));
    const summaryCount = form.querySelector('[data-proposal-summary-count]');
    const summaryOptional = form.querySelector('[data-proposal-summary-optional]');
    const summaryTotal = form.querySelector('[data-proposal-summary-total]');

    const syncModelVisibility = () => {
      const model = form.querySelector('[name="pricing_model"]:checked')?.value || 'per_item';
      products.forEach((row) => {
        const models = String(row.dataset.models || '').split(',').filter(Boolean);
        row.hidden = models.length > 0 && !models.includes(model);
      });
    };

    const refreshSummary = () => {
      let included = 0;
      let optional = 0;
      let total = 0;
      products.forEach((row) => {
        updateProduct(row);
        const price = parseMoney(row.querySelector('[name="item_price"]')?.value);
        if (row.dataset.productState === 'included') {
          included += 1;
          total += price;
        }
        if (row.dataset.productState === 'optional') optional += 1;
      });
      if (summaryCount) summaryCount.textContent = String(included);
      if (summaryOptional) summaryOptional.textContent = String(optional);
      if (summaryTotal) summaryTotal.textContent = total.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' });
    };

    products.forEach((row) => {
      const enabled = row.querySelector('[data-product-enabled]');
      const optional = row.querySelector('[data-product-optional]');
      const price = row.querySelector('[name="item_price"]');

      enabled?.addEventListener('change', () => {
        if (!enabled.checked) {
          if (price) price.value = '';
          if (optional) optional.checked = false;
        } else if (price) {
          price.focus();
        }
        refreshSummary();
      });
      optional?.addEventListener('change', refreshSummary);
      price?.addEventListener('input', () => {
        if (price.value.trim() !== '' && enabled && !enabled.checked) enabled.checked = true;
        if (price.value.trim() === '' && enabled) {
          enabled.checked = false;
          if (optional) optional.checked = false;
        }
        refreshSummary();
      });
      updateProduct(row);
    });

    pricingRadios.forEach((radio) => {
      radio.addEventListener('change', () => {
        syncModelVisibility();
        refreshSummary();
      });
    });

    form.querySelectorAll('[data-proposal-preset]').forEach((button) => {
      button.addEventListener('click', () => {
        const model = button.dataset.model || 'per_item';
        const groups = new Set((button.dataset.groups || '').split(',').filter(Boolean));
        const optionalGroups = new Set((button.dataset.optionalGroups || '').split(',').filter(Boolean));
        const radio = form.querySelector(`[name="pricing_model"][value="${CSS.escape(model)}"]`);
        if (radio instanceof HTMLInputElement) {
          radio.checked = true;
          radio.dispatchEvent(new Event('change', { bubbles: true }));
        }

        products.forEach((row) => {
          const group = row.dataset.group || '';
          const enabled = row.querySelector('[data-product-enabled]');
          const optional = row.querySelector('[data-product-optional]');
          if (!(enabled instanceof HTMLInputElement)) return;

          if (!groups.has(group) && !optionalGroups.has(group)) {
            enabled.checked = false;
            const price = row.querySelector('[name="item_price"]');
            if (price) price.value = '';
            if (optional) optional.checked = false;
          } else if (optional instanceof HTMLInputElement && optionalGroups.has(group) && enabled.checked) {
            optional.checked = true;
          }
          updateProduct(row);
        });

        syncModelVisibility();
        refreshSummary();
      });
    });

    form.addEventListener('submit', () => {
      products.forEach((row) => {
        const price = row.querySelector('[name="item_price"]');
        const enabled = row.querySelector('[data-product-enabled]');
        const optional = row.querySelector('[data-product-optional]');
        const status = row.querySelector('[name="item_status"]');
        const hasPrice = Boolean(price?.value?.trim());
        if (enabled) enabled.checked = hasPrice;
        if (status) status.value = hasPrice ? (optional?.checked ? 'optional' : 'included') : 'off';
      });
    }, { capture: true });

    syncModelVisibility();
    refreshSummary();
  }

  document.addEventListener('DOMContentLoaded', initProposalEditor);
})();
