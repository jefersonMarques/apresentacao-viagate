(() => {
  const maskTypes = new Set(['cpf', 'cnpj', 'phone', 'postal-code']);

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
    if (raw.startsWith('55') && raw.length >= 12) {
      return `+55 ${formatLocalPhone(raw.slice(2))}`;
    }
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
    masked.forEach((input) => {
      input.value = rawMaskedValue(input.dataset.mask || '', input.value);
    });
    return masked;
  }

  function restoreFormattedFields(inputs) {
    inputs.forEach((input) => {
      if (!document.contains(input)) return;
      input.value = formatMaskedValue(input.dataset.mask || '', input.value);
    });
  }

  document.addEventListener('DOMContentLoaded', () => configureMaskedInputs());

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
    form.querySelectorAll('button[type="submit"]').forEach((button) => {
      button.disabled = true;
      button.setAttribute('aria-disabled', 'true');
    });
    submitter.dataset.originalText = submitter.textContent || '';
    submitter.textContent = label;
    submitter.classList.add('is-processing');
  });
})();
