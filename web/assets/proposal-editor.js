(() => {
  function currentPermissions() {
    const node = document.querySelector('[data-user-permissions]');
    return new Set(String(node?.dataset.userPermissions || '').split(',').map((value) => value.trim()).filter(Boolean));
  }

  function parseMoney(value) {
    const normalized = String(value || '')
      .replace(/[^0-9,.-]/g, '')
      .replace(/\./g, '')
      .replace(',', '.');
    const parsed = Number(normalized);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  function updateProduct(row, canEditPrices) {
    const enabled = row.querySelector('[data-product-enabled]');
    const optional = row.querySelector('[data-product-optional]');
    const price = row.querySelector('[name="item_price"]');
    const status = row.querySelector('[name="item_status"]');
    const active = Boolean(enabled?.checked);

    row.dataset.productState = active ? (optional?.checked ? 'optional' : 'included') : 'off';
    if (status) status.value = active ? (optional?.checked ? 'optional' : 'included') : 'off';
    if (optional) optional.disabled = !active;
    if (price) {
      price.readOnly = !canEditPrices;
      price.classList.toggle('is-disabled', !active || !canEditPrices);
    }
  }

  function initProposalEditor() {
    const form = document.querySelector('[data-proposal-editor]');
    if (!(form instanceof HTMLFormElement)) return;

    const permissions = currentPermissions();
    const canEditPrices = permissions.has('proposal.price.edit');
    const canEditConditions = permissions.has('proposal.conditions.edit');
    const products = Array.from(form.querySelectorAll('[data-proposal-product]'));
    const pricingRadios = Array.from(form.querySelectorAll('[name="pricing_model"]'));
    const summaryCount = form.querySelector('[data-proposal-summary-count]');
    const summaryOptional = form.querySelector('[data-proposal-summary-optional]');
    const summaryTotal = form.querySelector('[data-proposal-summary-total]');

    form.querySelectorAll('[name="minimum_invoice"], [name="setup_fee"]').forEach((field) => {
      if (field instanceof HTMLInputElement) {
        field.readOnly = !canEditPrices;
        field.classList.toggle('is-disabled', !canEditPrices);
      }
    });

    if (!canEditConditions) {
      form.querySelectorAll('[name="condition"], [name="custom_conditions"]').forEach((field) => {
        field.disabled = true;
        field.classList.add('is-disabled');
      });
    }

    const syncModelVisibility = () => {
      const model = form.querySelector('[name="pricing_model"]:checked')?.value || 'per_item';
      products.forEach((row) => {
        const models = String(row.dataset.models || '').split(',').filter(Boolean);
        const hidden = models.length > 0 && !models.includes(model);
        row.hidden = hidden;

        if (hidden) {
          const enabled = row.querySelector('[data-product-enabled]');
          const optional = row.querySelector('[data-product-optional]');
          const price = row.querySelector('[name="item_price"]');
          const status = row.querySelector('[name="item_status"]');
          if (enabled instanceof HTMLInputElement) enabled.checked = false;
          if (optional instanceof HTMLInputElement) optional.checked = false;
          if (canEditPrices && price instanceof HTMLInputElement) price.value = '';
          if (status instanceof HTMLInputElement) status.value = 'off';
          updateProduct(row, canEditPrices);
        }
      });

      form.querySelectorAll('[data-catalog-group]').forEach((group) => {
        const visibleProducts = Array.from(group.querySelectorAll('[data-proposal-product]')).some((row) => !row.hidden);
        group.hidden = !visibleProducts;
      });
    };

    const refreshSummary = () => {
      let included = 0;
      let optional = 0;
      let total = 0;
      products.forEach((row) => {
        if (row.hidden) return;
        updateProduct(row, canEditPrices);
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
          if (canEditPrices && price) price.value = '';
          if (optional) optional.checked = false;
        } else if (canEditPrices && price) {
          price.focus();
        }
        refreshSummary();
      });
      optional?.addEventListener('change', refreshSummary);
      if (canEditPrices) {
        price?.addEventListener('input', () => {
          if (price.value.trim() !== '' && enabled && !enabled.checked) enabled.checked = true;
          if (price.value.trim() === '' && enabled) {
            enabled.checked = false;
            if (optional) optional.checked = false;
          }
          refreshSummary();
        });
      }
      updateProduct(row, canEditPrices);
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
            if (canEditPrices && price) price.value = '';
            if (optional) optional.checked = false;
          } else if (optional instanceof HTMLInputElement && optionalGroups.has(group) && enabled.checked) {
            optional.checked = true;
          }
          updateProduct(row, canEditPrices);
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
        if (!enabled || !status) return;
        if (row.hidden) {
          status.value = 'off';
          return;
        }
        if (canEditPrices) {
          const hasPrice = Boolean(price?.value?.trim());
          enabled.checked = hasPrice;
          status.value = hasPrice ? (optional?.checked ? 'optional' : 'included') : 'off';
          return;
        }
        status.value = enabled.checked ? (optional?.checked ? 'optional' : 'included') : 'off';
      });
    }, { capture: true });

    syncModelVisibility();
    refreshSummary();
  }

  function initProposalShareDialog() {
    const dialog = document.querySelector('[data-proposal-share-dialog]');
    if (!(dialog instanceof HTMLDialogElement)) return;

    const status = dialog.querySelector('[data-proposal-share-status]');
    const setStatus = (message, state = '') => {
      if (!status) return;
      status.textContent = message;
      status.dataset.state = state;
    };

    const fullURL = (path) => new URL(path, window.location.origin).toString();

    dialog.querySelectorAll('[data-proposal-share-close]').forEach((button) => {
      button.addEventListener('click', () => dialog.close());
    });

    dialog.addEventListener('click', (event) => {
      if (event.target === dialog) dialog.close();
    });

    const copyButton = dialog.querySelector('[data-proposal-share-copy]');
    copyButton?.addEventListener('click', async () => {
      const path = copyButton.getAttribute('data-share-path');
      if (!path) return;
      try {
        await window.ViaGate?.copyText?.(fullURL(path));
        setStatus('Link copiado.', 'success');
      } catch (_) {
        setStatus('Não foi possível copiar o link.', 'error');
      }
    });

    const shareButton = dialog.querySelector('[data-proposal-share-native]');
    shareButton?.addEventListener('click', async () => {
      const path = shareButton.getAttribute('data-share-path');
      if (!path) return;
      const url = fullURL(path);
      const title = shareButton.getAttribute('data-share-title') || 'Proposta ViaGate';

      if (navigator.share) {
        try {
          await navigator.share({
            title,
            text: 'Acesse a proposta comercial da ViaGate.',
            url,
          });
          setStatus('Compartilhamento aberto.', 'success');
          return;
        } catch (error) {
          if (error?.name === 'AbortError') return;
        }
      }

      try {
        await window.ViaGate?.copyText?.(url);
        setStatus('Compartilhamento nativo indisponível. Link copiado.', 'success');
      } catch (_) {
        setStatus('Não foi possível compartilhar o link.', 'error');
      }
    });

    const emailButton = dialog.querySelector('[data-proposal-share-email]');
    emailButton?.addEventListener('click', async () => {
      if (!(emailButton instanceof HTMLButtonElement) || emailButton.disabled) return;
      const endpoint = emailButton.dataset.endpoint;
      if (!endpoint) return;

      const original = emailButton.textContent;
      emailButton.disabled = true;
      emailButton.textContent = 'Agendando envio...';
      setStatus('');

      try {
        const response = await fetch(endpoint, {
          method: 'POST',
          credentials: 'same-origin',
          headers: {
            'Accept': 'application/json',
            'X-Requested-With': 'ViaGate-Proposal-Share',
          },
        });
        if (!response.ok) {
          const detail = (await response.text()).trim();
          throw new Error(detail || 'Não foi possível enviar o e-mail.');
        }
        const payload = await response.json();
        emailButton.textContent = 'E-mail na fila';
        setStatus(payload.message || 'E-mail adicionado à fila de envio.', 'success');
      } catch (error) {
        emailButton.disabled = false;
        emailButton.textContent = original;
        setStatus(error?.message || 'Não foi possível agendar o e-mail.', 'error');
      }
    });

    if (dialog.dataset.autoOpen === 'true' && typeof dialog.showModal === 'function') {
      window.requestAnimationFrame(() => dialog.showModal());
    }
  }

  document.addEventListener('DOMContentLoaded', () => {
    initProposalEditor();
    initProposalShareDialog();
  });
})();
