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
    const dependencySatisfied = row.dataset.dependencySatisfied !== 'false';
    const active = Boolean(enabled?.checked) && dependencySatisfied;

    if (enabled instanceof HTMLInputElement) enabled.disabled = !dependencySatisfied;
    row.classList.toggle('is-dependency-blocked', !dependencySatisfied);
    row.dataset.productState = active ? (optional?.checked ? 'optional' : 'included') : 'off';
    if (status) status.value = active ? (optional?.checked ? 'optional' : 'included') : 'off';
    if (optional) optional.disabled = !active || !dependencySatisfied;
    if (price) {
      const priceLocked = !active || !canEditPrices || !dependencySatisfied;
      price.readOnly = priceLocked;
      price.classList.toggle('is-disabled', priceLocked);
    }

    const note = row.querySelector('[data-product-dependency-note]');
    if (note) note.dataset.state = dependencySatisfied ? 'ready' : 'blocked';
  }

  function initProposalEditor() {
    const form = document.querySelector('[data-proposal-editor]');
    if (!(form instanceof HTMLFormElement)) return;

    const permissions = currentPermissions();
    const canEditPrices = permissions.has('proposal.price.edit');
    const canEditConditions = permissions.has('proposal.conditions.edit');
    const products = Array.from(form.querySelectorAll('[data-proposal-product]'));
    const productByCode = new Map(
      products
        .map((row) => [row.getAttribute('data-product-code') || '', row])
        .filter(([code]) => code),
    );
    const categoryToggles = Array.from(form.querySelectorAll('[data-proposal-category]'));
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

    const isProductSelected = (code) => {
      const row = productByCode.get(code);
      if (!row || row.hidden) return false;
      const enabled = row.querySelector('[data-product-enabled]');
      return enabled instanceof HTMLInputElement && enabled.checked;
    };

    const dependenciesSatisfied = (row) => {
      const groups = Array.from(row.querySelectorAll('[data-product-dependency-group]'));
      if (groups.length === 0) return true;

      return groups.every((group) => {
        const mode = group.getAttribute('data-mode') || 'all';
        const requirements = Array.from(group.querySelectorAll('[data-required-code]'))
          .map((node) => node.getAttribute('data-required-code') || '')
          .filter(Boolean);
        if (requirements.length === 0) return true;
        if (mode === 'any') return requirements.some(isProductSelected);
        return requirements.every(isProductSelected);
      });
    };

    const firstMissingRequirementCode = (row) => {
      const groups = Array.from(row.querySelectorAll('[data-product-dependency-group]'));
      for (const group of groups) {
        const mode = group.getAttribute('data-mode') || 'all';
        const requirements = Array.from(group.querySelectorAll('[data-required-code]'))
          .map((node) => node.getAttribute('data-required-code') || '')
          .filter(Boolean);
        if (requirements.length === 0) continue;

        const satisfied = mode === 'any'
          ? requirements.some(isProductSelected)
          : requirements.every(isProductSelected);
        if (satisfied) continue;

        const missing = requirements.find((code) => !isProductSelected(code) && productByCode.has(code));
        if (missing) return missing;
      }
      return '';
    };

    const syncDependencyStates = () => {
      let changed = true;
      let iteration = 0;
      while (changed && iteration <= products.length + 1) {
        changed = false;
        iteration += 1;

        products.forEach((row) => {
          const satisfied = dependenciesSatisfied(row);
          const enabled = row.querySelector('[data-product-enabled]');
          const optional = row.querySelector('[data-product-optional]');
          const price = row.querySelector('[name="item_price"]');
          const status = row.querySelector('[name="item_status"]');

          row.dataset.dependencySatisfied = satisfied ? 'true' : 'false';
          if (!satisfied && enabled instanceof HTMLInputElement && enabled.checked) {
            enabled.checked = false;
            if (optional instanceof HTMLInputElement) optional.checked = false;
            if (canEditPrices && price instanceof HTMLInputElement) price.value = '';
            if (status instanceof HTMLInputElement) status.value = 'off';
            changed = true;
          }
          updateProduct(row, canEditPrices);
        });
      }
    };

    const syncCategoryVisibility = () => {
      const selected = new Set(
        categoryToggles
          .filter((toggle) => toggle instanceof HTMLInputElement && toggle.checked)
          .map((toggle) => toggle.value),
      );

      form.querySelectorAll('[data-proposal-category-choice]').forEach((choice) => {
        const toggle = choice.querySelector('[data-proposal-category]');
        choice.classList.toggle('is-selected', Boolean(toggle?.checked));
      });

      form.querySelectorAll('[data-catalog-group]').forEach((group) => {
        const groupCode = group.getAttribute('data-catalog-group') || '';
        const hidden = !selected.has(groupCode);
        group.hidden = hidden;

        group.querySelectorAll('[data-proposal-product]').forEach((row) => {
          row.hidden = hidden;
          if (!hidden) return;

          const enabled = row.querySelector('[data-product-enabled]');
          const optional = row.querySelector('[data-product-optional]');
          const price = row.querySelector('[name="item_price"]');
          const status = row.querySelector('[name="item_status"]');
          if (enabled instanceof HTMLInputElement) enabled.checked = false;
          if (optional instanceof HTMLInputElement) optional.checked = false;
          if (canEditPrices && price instanceof HTMLInputElement) price.value = '';
          if (status instanceof HTMLInputElement) status.value = 'off';
          updateProduct(row, canEditPrices);
        });
      });

      syncDependencyStates();
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

    const refreshEditor = () => {
      syncDependencyStates();
      refreshSummary();
    };

    products.forEach((row) => {
      const enabled = row.querySelector('[data-product-enabled]');
      const optional = row.querySelector('[data-product-optional]');
      const price = row.querySelector('[name="item_price"]');

      enabled?.addEventListener('change', () => {
        if (!enabled.checked) {
          if (canEditPrices && price) price.value = '';
          if (optional) optional.checked = false;
        }
        refreshEditor();
        if (enabled.checked && canEditPrices && price instanceof HTMLInputElement && !price.readOnly) {
          price.focus();
        }
      });
      optional?.addEventListener('change', refreshEditor);
      if (canEditPrices) {
        price?.addEventListener('input', () => {
          if (price.value.trim() !== '' && enabled && !enabled.checked && !enabled.disabled) enabled.checked = true;
          if (price.value.trim() === '' && enabled) {
            enabled.checked = false;
            if (optional) optional.checked = false;
          }
          refreshEditor();
        });
      }
      updateProduct(row, canEditPrices);
    });

    categoryToggles.forEach((toggle) => {
      toggle.addEventListener('change', () => {
        syncCategoryVisibility();
        refreshSummary();
      });
    });

    form.addEventListener('click', (event) => {
      const button = event.target.closest('[data-product-select-requirement]');
      if (!button) return;

      const sourceRow = button.closest('[data-proposal-product]');
      if (!(sourceRow instanceof HTMLElement)) return;
      const requiredCode = firstMissingRequirementCode(sourceRow);
      if (!requiredCode) return;

      const targetRow = productByCode.get(requiredCode);
      if (!(targetRow instanceof HTMLElement)) return;

      const categoryCode = targetRow.dataset.group || '';
      const categoryToggle = categoryToggles.find((toggle) =>
        toggle instanceof HTMLInputElement && toggle.value === categoryCode
      );
      if (categoryToggle instanceof HTMLInputElement && !categoryToggle.checked) {
        categoryToggle.checked = true;
        syncCategoryVisibility();
        refreshSummary();
      }

      window.requestAnimationFrame(() => {
        targetRow.scrollIntoView({ behavior: 'smooth', block: 'center' });
        targetRow.classList.add('is-requirement-focus');
        window.setTimeout(() => targetRow.classList.remove('is-requirement-focus'), 1800);
        const enabled = targetRow.querySelector('[data-product-enabled]');
        if (enabled instanceof HTMLInputElement && !enabled.disabled) enabled.focus();
      });
    });

    form.addEventListener('submit', () => {
      syncDependencyStates();
      products.forEach((row) => {
        const price = row.querySelector('[name="item_price"]');
        const enabled = row.querySelector('[data-product-enabled]');
        const optional = row.querySelector('[data-product-optional]');
        const status = row.querySelector('[name="item_status"]');
        if (!enabled || !status) return;
        if (row.hidden || row.dataset.dependencySatisfied === 'false') {
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

    syncCategoryVisibility();
    refreshEditor();
  }

  function initProposalShareDialog() {
    const dialog = document.querySelector('[data-proposal-share-dialog]');
    if (typeof HTMLDialogElement === 'undefined' || !(dialog instanceof HTMLDialogElement)) return;

    const status = dialog.querySelector('[data-proposal-share-status]');
    const templateSelect = dialog.querySelector('[data-proposal-email-template]');
    const setStatus = (message, state = '') => {
      if (!status) return;
      status.textContent = message;
      status.dataset.state = state;
    };

    const selectedTemplateID = () => (
      templateSelect instanceof HTMLSelectElement ? templateSelect.value : ''
    );

    const loadEmailDraft = async (button) => {
      const endpoint = button?.getAttribute('data-endpoint');
      if (!endpoint) throw new Error('Endpoint de e-mail indisponível.');

      const url = new URL(endpoint, window.location.origin);
      const templateID = selectedTemplateID();
      if (templateID) url.searchParams.set('template', templateID);

      const response = await fetch(url.toString(), {
        method: 'GET',
        credentials: 'same-origin',
        headers: {
          'Accept': 'application/json',
          'X-Requested-With': 'ViaGate-Proposal-Email-Draft',
        },
      });
      if (!response.ok) {
        const detail = (await response.text()).trim();
        throw new Error(detail || 'Não foi possível preparar o e-mail.');
      }
      return response.json();
    };

    dialog.querySelectorAll('[data-proposal-share-close]').forEach((button) => {
      button.addEventListener('click', () => dialog.close());
    });

    dialog.addEventListener('click', (event) => {
      if (event.target === dialog) dialog.close();
    });

    templateSelect?.addEventListener('change', () => setStatus(''));

    const copyButton = dialog.querySelector('[data-proposal-share-copy]');
    copyButton?.addEventListener('click', async () => {
      const url = copyButton.getAttribute('data-share-url');
      if (!url) return;
      try {
        const copied = await window.ViaGate?.copyText?.(url);
        if (!copied) throw new Error('clipboard unavailable');
        setStatus('Link público copiado.', 'success');
      } catch (_) {
        window.prompt('Copie o link:', url);
      }
    });

    const shareButton = dialog.querySelector('[data-proposal-share-native]');
    shareButton?.addEventListener('click', async () => {
      const url = shareButton.getAttribute('data-share-url');
      if (!url) return;
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
        const copied = await window.ViaGate?.copyText?.(url);
        if (!copied) throw new Error('clipboard unavailable');
        setStatus('Compartilhamento nativo indisponível. Link copiado.', 'success');
      } catch (_) {
        window.prompt('Copie o link:', url);
      }
    });

    const openEmailButton = dialog.querySelector('[data-proposal-share-email-open]');
    openEmailButton?.addEventListener('click', async () => {
      if (!(openEmailButton instanceof HTMLButtonElement) || openEmailButton.disabled) return;

      const original = openEmailButton.textContent;
      openEmailButton.disabled = true;
      openEmailButton.textContent = 'Preparando...';
      setStatus('');

      try {
        const draft = await loadEmailDraft(openEmailButton);
        const query = new URLSearchParams();
        if (draft.subject) query.set('subject', draft.subject);
        if (draft.text_body) query.set('body', draft.text_body);
        const recipient = String(draft.to || '').trim();
        const mailto = `mailto:${encodeURIComponent(recipient)}?${query.toString()}`;
        window.location.href = mailto;
        setStatus('Cliente de e-mail aberto. Para manter o layout da marca, use “Copiar e-mail em HTML” e cole no corpo da mensagem.', 'success');
      } catch (error) {
        setStatus(error?.message || 'Não foi possível preparar o e-mail.', 'error');
      } finally {
        openEmailButton.disabled = false;
        openEmailButton.textContent = original;
      }
    });

    const copyEmailButton = dialog.querySelector('[data-proposal-share-email-copy]');
    copyEmailButton?.addEventListener('click', async () => {
      if (!(copyEmailButton instanceof HTMLButtonElement) || copyEmailButton.disabled) return;

      const original = copyEmailButton.textContent;
      copyEmailButton.disabled = true;
      copyEmailButton.textContent = 'Preparando HTML...';
      setStatus('');

      try {
        const draft = await loadEmailDraft(copyEmailButton);
        const copiedFormat = await window.ViaGate?.copyRichHTML?.(draft.html_body || '', draft.text_body || '');
        if (copiedFormat === 'html') {
          setStatus('E-mail em HTML copiado. Abra uma nova mensagem no Outlook e cole no corpo do e-mail.', 'success');
        } else {
          setStatus('O navegador não permitiu HTML; a versão em texto foi copiada.', 'success');
        }
      } catch (error) {
        setStatus(error?.message || 'Não foi possível copiar o e-mail.', 'error');
      } finally {
        copyEmailButton.disabled = false;
        copyEmailButton.textContent = original;
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
