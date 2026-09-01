(() => {
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

  window.ViaGate = Object.assign(window.ViaGate || {}, { copyText });

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
