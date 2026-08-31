(() => {
  const root = document.querySelector('[data-contract-verification]');
  if (!root) return;

  const input = root.querySelector('[data-verification-file]');
  const result = root.querySelector('[data-verification-result]');
  const expectedHash = (root.dataset.expectedHash || '').toLowerCase();
  if (!input || !result || expectedHash.length !== 64) return;

  const toHex = (buffer) => Array.from(new Uint8Array(buffer), (byte) => byte.toString(16).padStart(2, '0')).join('');

  const showResult = (message, ok) => {
    result.hidden = false;
    result.classList.toggle('error', !ok);
    result.textContent = message;
  };

  input.addEventListener('change', async () => {
    const file = input.files && input.files[0];
    if (!file) {
      result.hidden = true;
      result.textContent = '';
      result.classList.remove('error');
      return;
    }

    if (!file.name.toLowerCase().endsWith('.pdf')) {
      showResult('Selecione um arquivo PDF para fazer a comparação.', false);
      return;
    }

    try {
      result.hidden = false;
      result.classList.remove('error');
      result.textContent = 'Comparando o arquivo...';

      const content = await file.arrayBuffer();
      const digest = await crypto.subtle.digest('SHA-256', content);
      const actualHash = toHex(digest);

      if (actualHash === expectedHash) {
        showResult('Arquivo confirmado: este PDF é exatamente o mesmo documento registrado pela ViaGate.', true);
      } else {
        showResult('Arquivo diferente: este PDF não corresponde exatamente ao documento registrado pela ViaGate.', false);
      }
    } catch (_) {
      showResult('Não foi possível conferir este arquivo no navegador.', false);
    }
  });
})();
