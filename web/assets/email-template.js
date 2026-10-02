(() => {
  function initEmailTemplateVariables() {
    let activeEditor = document.querySelector('[data-email-template-html]');

    document.querySelectorAll('[data-email-template-editor]').forEach((editor) => {
      editor.addEventListener('focus', () => {
        activeEditor = editor;
      });
    });

    document.querySelectorAll('[data-email-template-variable]').forEach((button) => {
      button.addEventListener('click', () => {
        const value = button.getAttribute('data-email-template-variable') || '';
        const editor = activeEditor;
        if (!(editor instanceof HTMLInputElement) && !(editor instanceof HTMLTextAreaElement)) return;

        const start = Number.isInteger(editor.selectionStart) ? editor.selectionStart : editor.value.length;
        const end = Number.isInteger(editor.selectionEnd) ? editor.selectionEnd : start;
        editor.setRangeText(value, start, end, 'end');
        editor.focus();
      });
    });
  }

  document.addEventListener('DOMContentLoaded', initEmailTemplateVariables);
})();
