(() => {
  const form = document.querySelector('[data-signature-otp-form]');
  if (!(form instanceof HTMLFormElement)) return;
  const button = form.querySelector('[data-signature-otp-button]');
  const status = document.querySelector('[data-signature-otp-status]');
  let timer = 0;

  function setStatus(message, type = 'success') {
    if (!(status instanceof HTMLElement)) return;
    status.textContent = message || '';
    status.hidden = !message;
    status.classList.toggle('is-error', type === 'error');
    status.classList.toggle('is-success', type === 'success');
  }

  function startCountdown(seconds) {
    if (!(button instanceof HTMLButtonElement)) return;
    window.clearInterval(timer);
    let remaining = Math.max(1, Number(seconds) || 60);
    button.disabled = true;
    const tick = () => {
      const minutes = String(Math.floor(remaining / 60)).padStart(2, '0');
      const secs = String(remaining % 60).padStart(2, '0');
      button.textContent = `Reenviar em ${minutes}:${secs}`;
      remaining -= 1;
      if (remaining < 0) {
        window.clearInterval(timer);
        button.disabled = false;
        button.textContent = 'Reenviar código';
      }
    };
    tick();
    timer = window.setInterval(tick, 1000);
  }

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    if (!(button instanceof HTMLButtonElement) || button.disabled) return;
    button.disabled = true;
    button.textContent = 'Enviando código...';
    setStatus('');
    try {
      const response = await fetch(form.action, {
        method: 'POST',
        credentials: 'same-origin',
        headers: { Accept: 'application/json' },
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Não foi possível enviar o código.');
      if (data.signed) {
        window.location.reload();
        return;
      }
      setStatus(`✓ ${data.message || 'Código enviado.'}`);
      startCountdown(data.retry_after_seconds || 60);
      const otp = document.querySelector('input[name="otp"]');
      if (otp instanceof HTMLInputElement) otp.focus();
    } catch (error) {
      setStatus(error?.message || 'Não foi possível enviar o código.', 'error');
      button.disabled = false;
      button.textContent = 'Enviar código por e-mail';
    }
  });
})();
