(() => {
  const modal = document.querySelector('[data-condition-modal]');
  const form = modal?.querySelector('[data-condition-form]');
  const title = modal?.querySelector('[data-condition-modal-title]');
  const submit = modal?.querySelector('[data-condition-submit]');
  const newButton = document.querySelector('[data-condition-new]');

  if (!(modal instanceof HTMLElement) || !(form instanceof HTMLFormElement)) return;

  const idField = form.elements.namedItem('id');
  const textField = form.elements.namedItem('condition_text');
  const sortField = form.elements.namedItem('sort_order');
  const activeField = form.elements.namedItem('is_active');
  const groupFields = () => Array.from(form.querySelectorAll('input[name="group_code"]'));

  function nextSortOrder() {
    const values = Array.from(document.querySelectorAll('[data-condition-edit]'))
      .map((button) => Number.parseInt(button.dataset.conditionSort || '0', 10))
      .filter((value) => Number.isFinite(value));
    return values.length === 0 ? 10 : Math.max(...values) + 10;
  }

  function setGroups(groups) {
    const selected = new Set(groups);
    groupFields().forEach((field) => {
      field.checked = selected.has(field.value);
    });
  }

  function openModal(condition = null) {
    form.reset();

    if (condition) {
      if (idField instanceof HTMLInputElement) idField.value = condition.id || '';
      if (textField instanceof HTMLTextAreaElement) textField.value = condition.text || '';
      if (sortField instanceof HTMLInputElement) sortField.value = condition.sort || '';
      if (activeField instanceof HTMLInputElement) activeField.checked = condition.active === 'true';
      setGroups((condition.groups || '').split(',').map((value) => value.trim()).filter(Boolean));
      if (title) title.textContent = 'Editar condição';
      if (submit) submit.textContent = 'Salvar alteração';
    } else {
      if (idField instanceof HTMLInputElement) idField.value = '';
      if (sortField instanceof HTMLInputElement) sortField.value = String(nextSortOrder());
      if (activeField instanceof HTMLInputElement) activeField.checked = true;
      setGroups([]);
      if (title) title.textContent = 'Nova condição';
      if (submit) submit.textContent = 'Adicionar condição';
    }

    modal.classList.add('is-open');
    modal.setAttribute('aria-hidden', 'false');
    document.body.classList.add('proposal-condition-modal-open');
    window.setTimeout(() => textField?.focus(), 0);
  }

  function closeModal() {
    modal.classList.remove('is-open');
    modal.setAttribute('aria-hidden', 'true');
    document.body.classList.remove('proposal-condition-modal-open');
    newButton?.focus();
  }

  newButton?.addEventListener('click', () => openModal());

  document.querySelectorAll('[data-condition-edit]').forEach((button) => {
    button.addEventListener('click', () => {
      openModal({
        id: button.dataset.conditionId || '',
        text: button.dataset.conditionText || '',
        sort: button.dataset.conditionSort || '',
        active: button.dataset.conditionActive || 'false',
        groups: button.dataset.conditionGroups || '',
      });
    });
  });

  modal.querySelectorAll('[data-condition-modal-close]').forEach((button) => {
    button.addEventListener('click', closeModal);
  });

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && modal.classList.contains('is-open')) {
      closeModal();
    }
  });

  form.addEventListener('submit', () => {
    if (submit instanceof HTMLButtonElement) {
      submit.disabled = true;
      submit.textContent = 'Salvando...';
    }
  });
})();