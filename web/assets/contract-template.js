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
        const rawValue = button.dataset.contractVariable || '';
        const editor = document.querySelector('[data-contract-markdown]');
        if (editor instanceof HTMLTextAreaElement) {
          const start = editor.selectionStart;
          const end = editor.selectionEnd;
          const booleanMatch = rawValue.match(/^\{(products\.[a-zA-Z0-9_.-]+)\}$/);
          if (booleanMatch) {
            const prefix = `{% if ${booleanMatch[1]} %}\n`;
            const suffix = '\n{% endif %}';
            const selected = editor.value.slice(start, end);
            const content = prefix + selected + suffix;
            editor.setRangeText(content, start, end, 'end');
            const cursor = start + prefix.length;
            editor.setSelectionRange(cursor, cursor + selected.length);
          } else {
            editor.setRangeText(rawValue, start, end, 'end');
          }
          editor.focus();
          return;
        }
        try { await copyText(value); } catch (_) {}
      });
    });
  }

  document.addEventListener('DOMContentLoaded', initContractVariables);
})();
