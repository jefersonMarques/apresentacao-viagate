(() => {
  const maskTypes = new Set(['cpf', 'cnpj', 'phone', 'postal-code']);
  const sidebarStorageKey = 'viagate.admin.sidebar.collapsed';

  function fallbackCopy(value) {
    const textarea = document.createElement('textarea');
    textarea.value = value;
    textarea.setAttribute('readonly', '');
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    const copied = document.execCommand('copy');
    textarea.remove();
    return copied;
  }

  async function copyText(value) {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(value);
      return true;
    }
    return fallbackCopy(value);
  }

  function digits(value, limit) {
    return String(value || '').replace(/\D/g, '').slice(0, limit);
  }

  function cnpjCharacters(value) {
    return String(value || '').toUpperCase().replace(/[^0-9A-Z]/g, '').slice(0, 14);
  }

  function formatCPF(value) {
    const raw = digits(value, 11);
    if (raw.length <= 3) return raw;
    if (raw.length <= 6) return `${raw.slice(0, 3)}.${raw.slice(3)}`;
    if (raw.length <= 9) return `${raw.slice(0, 3)}.${raw.slice(3, 6)}.${raw.slice(6)}`;
    return `${raw.slice(0, 3)}.${raw.slice(3, 6)}.${raw.slice(6, 9)}-${raw.slice(9)}`;
  }

  function formatCNPJ(value) {
    const raw = cnpjCharacters(value);
    if (raw.length <= 2) return raw;
    if (raw.length <= 5) return `${raw.slice(0, 2)}.${raw.slice(2)}`;
    if (raw.length <= 8) return `${raw.slice(0, 2)}.${raw.slice(2, 5)}.${raw.slice(5)}`;
    if (raw.length <= 12) return `${raw.slice(0, 2)}.${raw.slice(2, 5)}.${raw.slice(5, 8)}/${raw.slice(8)}`;
    return `${raw.slice(0, 2)}.${raw.slice(2, 5)}.${raw.slice(5, 8)}/${raw.slice(8, 12)}-${raw.slice(12)}`;
  }

  function formatLocalPhone(raw) {
    if (raw.length === 0) return '';
    if (raw.length <= 2) return `(${raw}`;
    const areaCode = raw.slice(0, 2);
    const number = raw.slice(2);
    if (number.length <= 4) return `(${areaCode}) ${number}`;
    if (raw.length <= 10) return `(${areaCode}) ${number.slice(0, 4)}-${number.slice(4)}`;
    return `(${areaCode}) ${number.slice(0, 5)}-${number.slice(5, 9)}`;
  }

  function formatPhone(value) {
    const raw = digits(value, 13);
    if (raw.startsWith('55') && raw.length >= 12) return `+55 ${formatLocalPhone(raw.slice(2))}`;
    return formatLocalPhone(raw.slice(0, 11));
  }

  function formatPostalCode(value) {
    const raw = digits(value, 8);
    if (raw.length <= 5) return raw;
    return `${raw.slice(0, 5)}-${raw.slice(5)}`;
  }

  function inferMaskType(input) {
    const explicit = input.getAttribute('data-mask');
    if (maskTypes.has(explicit)) return explicit;
    const name = String(input.name || '').toLowerCase();
    if (name === 'cpf' || name.endsWith('_cpf')) return 'cpf';
    if (name === 'cnpj' || name.endsWith('_cnpj')) return 'cnpj';
    if (name === 'phone' || name.endsWith('_phone')) return 'phone';
    if (name === 'cep' || name.endsWith('_cep') || name === 'postal_code' || name.endsWith('_postal_code')) return 'postal-code';
    return '';
  }

  function formatMaskedValue(type, value) {
    switch (type) {
      case 'cpf': return formatCPF(value);
      case 'cnpj': return formatCNPJ(value);
      case 'phone': return formatPhone(value);
      case 'postal-code': return formatPostalCode(value);
      default: return value;
    }
  }

  function rawMaskedValue(type, value) {
    switch (type) {
      case 'cpf': return digits(value, 11);
      case 'cnpj': return cnpjCharacters(value);
      case 'phone': return digits(value, 13);
      case 'postal-code': return digits(value, 8);
      default: return value;
    }
  }

  function configureMaskedInput(input) {
    if (!(input instanceof HTMLInputElement)) return;
    const type = inferMaskType(input);
    if (!type) return;
    input.dataset.mask = type;
    if (type === 'cpf') {
      input.maxLength = 14;
      input.inputMode = 'numeric';
    } else if (type === 'cnpj') {
      input.maxLength = 18;
      input.inputMode = 'text';
      input.autocapitalize = 'characters';
      input.spellcheck = false;
    } else if (type === 'phone') {
      input.maxLength = 19;
      input.inputMode = 'tel';
    } else if (type === 'postal-code') {
      input.maxLength = 9;
      input.inputMode = 'numeric';
    }
    input.value = formatMaskedValue(type, input.value);
  }

  function configureMaskedInputs(root = document) {
    root.querySelectorAll('input[name], input[data-mask]').forEach(configureMaskedInput);
  }

  function normalizeFormFields(form) {
    const masked = Array.from(form.querySelectorAll('input[data-mask]'));
    masked.forEach((input) => { input.value = rawMaskedValue(input.dataset.mask || '', input.value); });
    return masked;
  }

  function restoreFormattedFields(inputs) {
    inputs.forEach((input) => {
      if (!document.contains(input)) return;
      input.value = formatMaskedValue(input.dataset.mask || '', input.value);
    });
  }

  function initAdminSidebar() {
    const shell = document.querySelector('[data-admin-shell]');
    if (!shell) return;
    try {
      if (localStorage.getItem(sidebarStorageKey) === '1') shell.classList.add('sidebar-collapsed');
    } catch (_) {}
    document.querySelectorAll('[data-sidebar-toggle]').forEach((button) => {
      button.addEventListener('click', () => {
        shell.classList.toggle('sidebar-collapsed');
        try { localStorage.setItem(sidebarStorageKey, shell.classList.contains('sidebar-collapsed') ? '1' : '0'); } catch (_) {}
      });
    });
    const closeMobile = () => shell.classList.remove('sidebar-mobile-open');
    document.querySelectorAll('[data-sidebar-mobile-toggle]').forEach((button) => button.addEventListener('click', () => shell.classList.add('sidebar-mobile-open')));
    document.querySelectorAll('[data-sidebar-backdrop]').forEach((button) => button.addEventListener('click', closeMobile));
    document.addEventListener('keydown', (event) => { if (event.key === 'Escape') closeMobile(); });
    const path = window.location.pathname.replace(/\/$/, '') || '/';
    document.querySelectorAll('[data-admin-nav]').forEach((link) => {
      const href = new URL(link.href, window.location.origin).pathname.replace(/\/$/, '') || '/';
      const active = href === '/admin' ? path === href : path === href || path.startsWith(`${href}/`);
      link.classList.toggle('is-active', active);
    });
  }

  async function uploadCommercialImage(root, file) {
    if (!file) return;
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) throw new Error('Use JPG, PNG ou WEBP.');
    if (file.size > 2 * 1024 * 1024) throw new Error('A imagem deve ter no máximo 2 MB.');
    const category = root.dataset.imageUpload;
    const body = new FormData();
    body.append('image', file);
    const response = await fetch(`/admin/assets?category=${encodeURIComponent(category)}`, { method: 'POST', body, credentials: 'same-origin' });
    if (!response.ok) throw new Error((await response.text()).trim() || 'Não foi possível enviar a imagem.');
    return response.json();
  }

  function syncImageUpload(root) {
    const value = root.querySelector('[data-image-value]');
    const preview = root.querySelector('[data-image-preview]');
    const empty = root.querySelector('[data-image-empty]');
    const remove = root.querySelector('[data-image-remove]');
    const url = value?.value?.trim() || '';
    if (preview instanceof HTMLImageElement) {
      if (url) preview.src = url;
      else preview.removeAttribute('src');
      preview.hidden = !url;
    }
    if (empty) empty.hidden = Boolean(url);
    if (remove) remove.hidden = !url;
  }

  function initImageUploads() {
    document.querySelectorAll('[data-image-upload]').forEach((root) => {
      const file = root.querySelector('[data-image-file]');
      const value = root.querySelector('[data-image-value]');
      const status = root.querySelector('[data-image-status]');
      const remove = root.querySelector('[data-image-remove]');
      syncImageUpload(root);
      file?.addEventListener('change', async () => {
        const selected = file.files?.[0];
        if (!selected) return;
        if (status) { status.textContent = 'Enviando imagem...'; status.classList.remove('error'); }
        try {
          const result = await uploadCommercialImage(root, selected);
          if (value) value.value = result.url || '';
          syncImageUpload(root);
          if (status) status.textContent = 'Imagem enviada.';
          value?.dispatchEvent(new Event('change', { bubbles: true }));
        } catch (error) {
          if (status) { status.textContent = error.message || 'Falha no upload.'; status.classList.add('error'); }
        } finally {
          file.value = '';
        }
      });
      remove?.addEventListener('click', () => {
        if (value) value.value = '';
        syncImageUpload(root);
        if (status) { status.textContent = 'Imagem removida deste material.'; status.classList.remove('error'); }
        value?.dispatchEvent(new Event('change', { bubbles: true }));
      });
    });
  }

  function fillInput(form, name, value) {
    if (!value) return;
    const input = form.elements.namedItem(name);
    if (!(input instanceof HTMLInputElement) && !(input instanceof HTMLTextAreaElement)) return;
    input.value = value;
    configureMaskedInput(input);
    input.dispatchEvent(new Event('input', { bubbles: true }));
  }

  function initCNPJLookups() {
    document.querySelectorAll('[data-cnpj-lookup]').forEach((button) => {
      button.addEventListener('click', async () => {
        const form = button.closest('form');
        if (!form) return;
        const cnpjField = form.elements.namedItem(button.dataset.cnpjField || 'client_cnpj');
        if (!(cnpjField instanceof HTMLInputElement)) return;
        const raw = cnpjCharacters(cnpjField.value);
        const status = button.parentElement?.querySelector('[data-cnpj-status]') || form.querySelector('[data-cnpj-status]');
        if (raw.length !== 14) {
          if (status) { status.textContent = 'Informe um CNPJ válido.'; status.classList.add('error'); }
          cnpjField.focus();
          return;
        }
        button.disabled = true;
        if (status) { status.textContent = 'Buscando dados na Receita...'; status.classList.remove('error'); }
        try {
          const endpoint = button.dataset.cnpjEndpoint || '/admin/api/cnpj/';
          const response = await fetch(`${endpoint}${encodeURIComponent(raw)}`, { credentials: 'same-origin' });
          if (!response.ok) throw new Error((await response.text()).trim() || 'CNPJ não encontrado.');
          const data = await response.json();
          const prefix = button.hasAttribute('data-cnpj-prefix') ? (button.dataset.cnpjPrefix || '') : 'client_';
          fillInput(form, `${prefix}legal_name`, data.legal_name);
          fillInput(form, `${prefix}trade_name`, data.trade_name);
          fillInput(form, `${prefix}email`, data.email);
          fillInput(form, `${prefix}phone`, data.phone);
          fillInput(form, `${prefix}street`, data.street);
          fillInput(form, `${prefix}street_number`, data.number);
          fillInput(form, `${prefix}complement`, data.complement);
          fillInput(form, `${prefix}district`, data.district);
          fillInput(form, `${prefix}city`, data.city);
          fillInput(form, `${prefix}state`, data.state);
          fillInput(form, `${prefix}postal_code`, data.postal_code);
          if (status) status.textContent = 'Dados encontrados. Revise antes de salvar.';
        } catch (error) {
          if (status) { status.textContent = error.message || 'Não foi possível consultar o CNPJ.'; status.classList.add('error'); }
        } finally {
          button.disabled = false;
        }
      });
    });
  }

  function updateProposalProduct(row) {
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
        updateProposalProduct(row);
        const price = Number(String(row.querySelector('[name="item_price"]')?.value || '').replace(',', '.'));
        if (row.dataset.productState === 'included') { included += 1; if (Number.isFinite(price)) total += price; }
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
        if (price.value.trim() === '' && enabled) { enabled.checked = false; if (optional) optional.checked = false; }
        refreshSummary();
      });
      updateProposalProduct(row);
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
        if (radio instanceof HTMLInputElement) { radio.checked = true; radio.dispatchEvent(new Event('change', { bubbles: true })); }
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
          updateProposalProduct(row);
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

  function initContractVariables() {
    document.querySelectorAll('[data-contract-variable]').forEach((button) => {
      button.addEventListener('click', async () => {
        const value = button.dataset.contractVariable || '';
        const editor = document.querySelector('[data-contract-markdown]');
        if (editor instanceof HTMLTextAreaElement) {
          const start = editor.selectionStart;
          const end = editor.selectionEnd;
          editor.setRangeText(value, start, end, 'end');
          editor.focus();
          return;
        }
        try { await copyText(value); } catch (_) {}
      });
    });
  }

  document.addEventListener('DOMContentLoaded', () => {
    configureMaskedInputs();
    initAdminSidebar();
    initImageUploads();
    initCNPJLookups();
    initProposalEditor();
    initContractVariables();
  });

  document.addEventListener('input', (event) => {
    const input = event.target;
    if (!(input instanceof HTMLInputElement)) return;
    const type = inferMaskType(input);
    if (!type) return;
    configureMaskedInput(input);
  });

  document.addEventListener('click', async (event) => {
    const button = event.target.closest('[data-copy-path]');
    if (!button) return;
    const path = button.getAttribute('data-copy-path');
    if (!path) return;
    const value = new URL(path, window.location.origin).toString();
    const original = button.innerHTML;
    try {
      const copied = await copyText(value);
      if (!copied) throw new Error('clipboard unavailable');
      button.textContent = 'Copiado';
    } catch (_) {
      window.prompt('Copie o link:', value);
    } finally {
      window.setTimeout(() => { button.innerHTML = original; }, 1400);
    }
  });

  document.addEventListener('submit', (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement)) return;
    const normalizedInputs = normalizeFormFields(form);
    window.setTimeout(() => restoreFormattedFields(normalizedInputs), 0);
    const submitter = event.submitter;
    if (!(submitter instanceof HTMLButtonElement)) return;
    const label = submitter.getAttribute('data-processing');
    if (!label) return;
    window.setTimeout(() => {
      if (!document.contains(form)) return;
      submitter.dataset.originalText = submitter.textContent || '';
      submitter.textContent = label;
      submitter.classList.add('is-processing');
      form.querySelectorAll('button[type="submit"]').forEach((button) => {
        button.disabled = true;
        button.setAttribute('aria-disabled', 'true');
      });
    }, 0);
  });
})();
