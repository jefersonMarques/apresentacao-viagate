(() => {
  const linkField = document.querySelector('[data-post-sale-share-url]');
  const copyButton = document.querySelector('[data-post-sale-copy]');
  const shareButton = document.querySelector('[data-post-sale-share]');
  const status = document.querySelector('[data-post-sale-share-status]');

  if (!(linkField instanceof HTMLInputElement)) {
    return;
  }

  function showStatus(message) {
    if (status) {
      status.textContent = message;
    }
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(linkField.value);
      showStatus('Link copiado.');
    } catch (_) {
      linkField.focus();
      linkField.select();
      showStatus('Selecione e copie o link exibido.');
    }
  }

  async function shareLink() {
    if (typeof navigator.share !== 'function') {
      await copyLink();
      return;
    }

    try {
      await navigator.share({
        title: 'Dados para implantação ViaGate',
        text: 'Preencha os dados necessários à implantação.',
        url: linkField.value,
      });
    } catch (_) {
      showStatus('Compartilhamento não concluído.');
    }
  }

  copyButton?.addEventListener('click', copyLink);
  shareButton?.addEventListener('click', shareLink);
})();
