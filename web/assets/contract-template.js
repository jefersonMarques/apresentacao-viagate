(() => {
  async function copyText(value) {
    if (window.ViaGate?.copyText) return window.ViaGate.copyText(value);
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(value);
      return true;
    }
    return false;
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

  document.addEventListener('DOMContentLoaded', initContractVariables);
})();
