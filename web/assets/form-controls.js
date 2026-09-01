(() => {
  const maskTypes = new Set(['cpf', 'cnpj', 'phone', 'postal-code']);

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

  async function uploadCommercialImage(root, file) {
    if (!file) return null;
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) throw new Error('Use JPG, PNG ou WEBP.');
    if (file.size > 2 * 1024 * 1024) throw new Error('A imagem deve ter no máximo 2 MB.');

    const body = new FormData();
    body.append('image', file);
    const category = root.dataset.imageUpload;
    const response = await fetch(`/admin/assets?category=${encodeURIComponent(category)}`, {
      method: 'POST',
      body,
      credentials: 'same-origin',
    });
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
        if (status) {
          status.textContent = 'Enviando imagem...';
          status.classList.remove('error');
        }
        try {
          const result = await uploadCommercialImage(root, selected);
          if (value) value.value = result?.url || '';
          syncImageUpload(root);
          if (status) status.textContent = 'Imagem enviada.';
          value?.dispatchEvent(new Event('change', { bubbles: true }));
        } catch (error) {
          if (status) {
            status.textContent = error.message || 'Falha no upload.';
            status.classList.add('error');
          }
        } finally {
          file.value = '';
        }
      });

      remove?.addEventListener('click', () => {
        if (value) value.value = '';
        syncImageUpload(root);
        if (status) {
          status.textContent = 'Imagem removida deste material.';
          status.classList.remove('error');
        }
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
          if (status) {
            status.textContent = 'Informe um CNPJ válido.';
            status.classList.add('error');
          }
          cnpjField.focus();
          return;
        }

        button.disabled = true;
        if (status) {
          status.textContent = 'Buscando dados na Receita...';
          status.classList.remove('error');
        }
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
          if (status) {
            status.textContent = error.message || 'Não foi possível consultar o CNPJ.';
            status.classList.add('error');
          }
        } finally {
          button.disabled = false;
        }
      });
    });
  }

  document.addEventListener('DOMContentLoaded', () => {
    configureMaskedInputs();
    initImageUploads();
    initCNPJLookups();
  });

  document.addEventListener('input', (event) => {
    const input = event.target;
    if (!(input instanceof HTMLInputElement)) return;
    const type = inferMaskType(input);
    if (!type) return;
    configureMaskedInput(input);
  });

  document.addEventListener('submit', (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement)) return;
    const normalizedInputs = normalizeFormFields(form);
    window.setTimeout(() => restoreFormattedFields(normalizedInputs), 0);
  });
})();
