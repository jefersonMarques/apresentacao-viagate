(() => {
  function dialogByKey(key) {
    return Array.from(document.querySelectorAll('[data-catalog-dialog]'))
      .find((dialog) => dialog.dataset.catalogDialog === key);
  }

  function openDialog(dialog) {
    if (typeof HTMLDialogElement === 'undefined' || !(dialog instanceof HTMLDialogElement)) return;
    if (typeof dialog.showModal === 'function') {
      dialog.showModal();
    } else {
      dialog.setAttribute('open', '');
    }
    window.requestAnimationFrame(() => {
      dialog.querySelector('input:not([type="hidden"]), select, textarea, button')?.focus();
    });
  }

  function closeDialog(dialog) {
    if (typeof HTMLDialogElement === 'undefined' || !(dialog instanceof HTMLDialogElement)) return;
    if (typeof dialog.close === 'function') {
      dialog.close();
    } else {
      dialog.removeAttribute('open');
    }
  }

  function dependencyEditor(node) {
    return node?.closest('[data-dependency-editor]');
  }

  function dependencyIndex(group) {
    return Number(group?.dataset.dependencyIndex || '0');
  }

  function dependencyMode(group) {
    const checked = group?.querySelector('[data-dependency-mode]:checked');
    return checked instanceof HTMLInputElement ? checked.value : 'any';
  }

  function dependencyProducts(group) {
    return Array.from(group?.querySelectorAll('[data-dependency-chip]') || []).map((chip) => ({
      id: chip.getAttribute('data-product-id') || '',
      name: chip.getAttribute('data-product-name') || '',
    })).filter((product) => product.id);
  }

  function setDependencyInputNames(group, index) {
    if (!(group instanceof HTMLElement)) return;
    group.dataset.dependencyIndex = String(index);

    const conditionNumber = group.querySelector('[data-condition-number]');
    if (conditionNumber) conditionNumber.textContent = `Condição ${index + 1}`;

    group.querySelectorAll('[data-dependency-mode]').forEach((input) => {
      if (!(input instanceof HTMLInputElement)) return;
      input.name = `dependency_mode_${index}`;
      input.removeAttribute('data-dependency-mode-template');
    });

    group.querySelectorAll('[data-dependency-chip] input[type="hidden"]').forEach((input) => {
      if (!(input instanceof HTMLInputElement)) return;
      input.name = `dependency_product_${index}`;
    });
  }

  function createDependencyChip(group, product) {
    if (!(group instanceof HTMLElement) || !product?.id) return;
    if (group.querySelector(`[data-dependency-chip][data-product-id="${CSS.escape(product.id)}"]`)) return;

    const chip = document.createElement('div');
    chip.className = 'catalog-product-chip';
    chip.dataset.dependencyChip = '';
    chip.dataset.productId = product.id;
    chip.dataset.productName = product.name || '';
    chip.innerHTML = `
      <span class="catalog-product-chip-icon" aria-hidden="true">
        <svg viewBox="0 0 24 24"><path d="M4 7h16v12H4z"/><path d="M8 7V5h8v2"/></svg>
      </span>
      <span class="catalog-product-chip-copy">
        <strong></strong>
        <small></small>
      </span>
      <button type="button" data-remove-dependency-product aria-label="Remover produto">
        <svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6 6 18"/></svg>
      </button>
      <input type="hidden"/>
    `;

    chip.querySelector('strong').textContent = product.name || '';
    chip.querySelector('small').textContent = product.category || '';
    const removeButton = chip.querySelector('[data-remove-dependency-product]');
    if (removeButton) removeButton.setAttribute('aria-label', `Remover ${product.name || 'produto'}`);
    const hidden = chip.querySelector('input[type="hidden"]');
    if (hidden instanceof HTMLInputElement) {
      hidden.name = `dependency_product_${dependencyIndex(group)}`;
      hidden.value = product.id;
    }

    const selected = group.querySelector('[data-dependency-selected]');
    const empty = group.querySelector('[data-dependency-empty]');
    if (empty) selected?.insertBefore(chip, empty);
    else selected?.appendChild(chip);
  }

  function refreshDependencyPicker(group) {
    if (!(group instanceof HTMLElement)) return;
    const search = group.querySelector('[data-dependency-search]');
    const query = search instanceof HTMLInputElement
      ? search.value.trim().toLocaleLowerCase('pt-BR')
      : '';
    const selected = new Set(dependencyProducts(group).map((product) => product.id));
    let visible = 0;

    group.querySelectorAll('[data-dependency-option]').forEach((option) => {
      const id = option.getAttribute('data-product-id') || '';
      const haystack = [
        option.getAttribute('data-product-name') || '',
        option.getAttribute('data-category-name') || '',
      ].join(' ').toLocaleLowerCase('pt-BR');
      const hidden = selected.has(id) || (query !== '' && !haystack.includes(query));
      option.hidden = hidden;
      if (!hidden) visible += 1;
    });

    const noResults = group.querySelector('[data-dependency-no-results]');
    if (noResults) noResults.hidden = visible > 0;
  }

  function updateDependencyGroup(group) {
    if (!(group instanceof HTMLElement)) return;
    const products = dependencyProducts(group);
    const empty = group.querySelector('[data-dependency-empty]');
    if (empty) empty.hidden = products.length > 0;

    const connector = dependencyMode(group) === 'all' ? ' e ' : ' ou ';
    const prefix = dependencyMode(group) === 'all' ? 'Todos: ' : 'Qualquer um: ';
    const summary = group.querySelector('[data-dependency-group-summary]');
    if (summary) {
      summary.textContent = products.length > 0
        ? prefix + products.map((product) => product.name).join(connector)
        : 'Adicione pelo menos um produto.';
    }

    refreshDependencyPicker(group);
    updateDependencyEditor(dependencyEditor(group));
  }

  function updateDependencyEditor(editor) {
    if (!(editor instanceof HTMLElement)) return;
    const groups = Array.from(editor.querySelectorAll('[data-dependency-group]'));
    const isDependent = editor.dataset.dependencyAvailability === 'dependent';
    const builder = editor.querySelector('[data-dependency-builder]');

    editor.querySelectorAll('[data-dependency-availability-option]').forEach((button) => {
      button.classList.toggle('is-active', button.dataset.dependencyAvailabilityOption === editor.dataset.dependencyAvailability);
    });

    if (builder instanceof HTMLElement) builder.hidden = !isDependent;

    groups.forEach((group) => {
      group.querySelectorAll('[name^="dependency_"]').forEach((control) => {
        if ('disabled' in control) control.disabled = !isDependent;
      });
    });

    const summaries = groups.map((group) => {
      const products = dependencyProducts(group);
      if (products.length === 0) return '';
      const connector = dependencyMode(group) === 'all' ? ' e ' : ' ou ';
      return `(${products.map((product) => product.name).join(connector)})`;
    }).filter(Boolean);

    const finalSummary = editor.querySelector('[data-dependency-rule-summary]');
    if (finalSummary) {
      finalSummary.textContent = !isDependent
        ? 'Sempre disponível.'
        : summaries.length > 0
          ? summaries.join(' E ')
          : 'Adicione uma condição para definir a disponibilidade.';
    }

    const details = editor.closest('.catalog-dependencies');
    const headerSummary = details?.querySelector('summary small');
    if (headerSummary) {
      headerSummary.textContent = !isDependent
        ? 'Este produto pode ser selecionado sem pré-requisitos.'
        : summaries.length > 0
          ? summaries.join(' E ')
          : 'Dependência ativada sem produtos selecionados.';
    }
  }

  function addDependencyGroup(editor) {
    if (!(editor instanceof HTMLElement)) return null;
    const groups = editor.querySelector('[data-dependency-groups]');
    const template = editor.querySelector('[data-dependency-group-template]');
    if (!(groups instanceof HTMLElement) || !(template instanceof HTMLTemplateElement)) return null;

    const index = Number(groups.dataset.nextIndex || '0');
    const fragment = template.content.cloneNode(true);
    const group = fragment.querySelector('[data-dependency-group]');
    if (!(group instanceof HTMLElement)) return null;

    setDependencyInputNames(group, index);
    groups.appendChild(fragment);
    groups.dataset.nextIndex = String(index + 1);
    updateDependencyGroup(group);
    return group;
  }

  function setDependencyAvailability(editor, state) {
    if (!(editor instanceof HTMLElement)) return;
    editor.dataset.dependencyAvailability = state;
    if (state === 'dependent' && editor.querySelectorAll('[data-dependency-group]').length === 0) {
      addDependencyGroup(editor);
    }
    updateDependencyEditor(editor);
  }

  function initializeDependencyEditors() {
    document.querySelectorAll('[data-dependency-editor]').forEach((editor) => {
      const groups = editor.querySelectorAll('[data-dependency-group]');
      editor.dataset.dependencyAvailability = groups.length > 0 ? 'dependent' : 'always';
      groups.forEach((group, index) => {
        setDependencyInputNames(group, dependencyIndex(group) || index);
        updateDependencyGroup(group);
      });
      updateDependencyEditor(editor);
    });
  }

  document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-catalog-dialog-open]').forEach((button) => {
      button.addEventListener('click', () => {
        const key = button.getAttribute('data-catalog-dialog-open');
        if (!key) return;
        openDialog(dialogByKey(key));
      });
    });

    document.querySelectorAll('[data-catalog-new-product-category]').forEach((button) => {
      button.addEventListener('click', () => {
        const dialog = dialogByKey('new-product');
        if (typeof HTMLDialogElement === 'undefined' || !(dialog instanceof HTMLDialogElement)) return;
        const category = button.getAttribute('data-catalog-new-product-category');
        const select = dialog.querySelector('[data-catalog-product-category]');
        if (select instanceof HTMLSelectElement && category) select.value = category;
        openDialog(dialog);
      });
    });

    document.querySelectorAll('[data-catalog-dialog]').forEach((dialog) => {
      dialog.querySelectorAll('[data-catalog-dialog-close]').forEach((button) => {
        button.addEventListener('click', () => closeDialog(dialog));
      });
      dialog.addEventListener('click', (event) => {
        if (event.target === dialog) closeDialog(dialog);
      });
    });

    initializeDependencyEditors();

    document.addEventListener('click', (event) => {
      const availability = event.target.closest('[data-dependency-availability-option]');
      if (availability) {
        setDependencyAvailability(
          dependencyEditor(availability),
          availability.getAttribute('data-dependency-availability-option') || 'always',
        );
        return;
      }

      const addCondition = event.target.closest('[data-add-dependency-group]');
      if (addCondition) {
        const editor = dependencyEditor(addCondition);
        setDependencyAvailability(editor, 'dependent');
        const group = addDependencyGroup(editor);
        group?.querySelector('[data-dependency-search]')?.focus();
        return;
      }

      const removeGroup = event.target.closest('[data-remove-dependency-group]');
      if (removeGroup) {
        const editor = dependencyEditor(removeGroup);
        removeGroup.closest('[data-dependency-group]')?.remove();
        if (editor?.querySelectorAll('[data-dependency-group]').length === 0) {
          setDependencyAvailability(editor, 'always');
        } else {
          updateDependencyEditor(editor);
        }
        return;
      }

      const option = event.target.closest('[data-dependency-option]');
      if (option) {
        const group = option.closest('[data-dependency-group]');
        if (!(group instanceof HTMLElement)) return;
        createDependencyChip(group, {
          id: option.getAttribute('data-product-id') || '',
          name: option.getAttribute('data-product-name') || '',
          category: option.getAttribute('data-category-name') || '',
        });
        const search = group.querySelector('[data-dependency-search]');
        if (search instanceof HTMLInputElement) search.value = '';
        const options = group.querySelector('[data-dependency-options]');
        if (options instanceof HTMLElement) options.hidden = true;
        updateDependencyGroup(group);
        return;
      }

      const removeProduct = event.target.closest('[data-remove-dependency-product]');
      if (removeProduct) {
        const group = removeProduct.closest('[data-dependency-group]');
        removeProduct.closest('[data-dependency-chip]')?.remove();
        updateDependencyGroup(group);
        return;
      }

      if (!event.target.closest('[data-dependency-picker]')) {
        document.querySelectorAll('[data-dependency-options]').forEach((options) => {
          options.hidden = true;
        });
      }
    });

    document.addEventListener('change', (event) => {
      if (!event.target.matches('[data-dependency-mode]')) return;
      updateDependencyGroup(event.target.closest('[data-dependency-group]'));
    });

    document.addEventListener('focusin', (event) => {
      if (!event.target.matches('[data-dependency-search]')) return;
      const group = event.target.closest('[data-dependency-group]');
      const options = group?.querySelector('[data-dependency-options]');
      if (options instanceof HTMLElement) options.hidden = false;
      refreshDependencyPicker(group);
    });

    document.addEventListener('input', (event) => {
      if (!event.target.matches('[data-dependency-search]')) return;
      const group = event.target.closest('[data-dependency-group]');
      const options = group?.querySelector('[data-dependency-options]');
      if (options instanceof HTMLElement) options.hidden = false;
      refreshDependencyPicker(group);
    });

    const deleteDialog = document.querySelector('[data-catalog-delete-dialog]');
    let pendingDeleteForm = null;

    if (typeof HTMLDialogElement !== 'undefined' && deleteDialog instanceof HTMLDialogElement) {
      const title = deleteDialog.querySelector('[data-catalog-delete-title]');
      const name = deleteDialog.querySelector('[data-catalog-delete-name]');
      const message = deleteDialog.querySelector('[data-catalog-delete-message]');
      const confirmButton = deleteDialog.querySelector('[data-catalog-delete-confirm]');

      document.querySelectorAll('[data-catalog-delete-open]').forEach((button) => {
        button.addEventListener('click', () => {
          const form = button.closest('[data-catalog-delete-form]');
          if (!(form instanceof HTMLFormElement)) return;

          pendingDeleteForm = form;
          if (title) title.textContent = form.dataset.deleteTitle || 'Excluir item';
          if (name) name.textContent = form.dataset.deleteName || '';
          if (message) message.textContent = form.dataset.deleteMessage || 'Confirme a exclusão deste item do catálogo.';
          if (confirmButton instanceof HTMLButtonElement) {
            confirmButton.disabled = false;
            confirmButton.textContent = 'Excluir';
          }
          openDialog(deleteDialog);
        });
      });

      deleteDialog.querySelectorAll('[data-catalog-delete-close]').forEach((button) => {
        button.addEventListener('click', () => {
          pendingDeleteForm = null;
          closeDialog(deleteDialog);
        });
      });

      deleteDialog.addEventListener('click', (event) => {
        if (event.target !== deleteDialog) return;
        pendingDeleteForm = null;
        closeDialog(deleteDialog);
      });

      deleteDialog.addEventListener('close', () => {
        if (!deleteDialog.returnValue) pendingDeleteForm = null;
      });

      confirmButton?.addEventListener('click', () => {
        if (!(pendingDeleteForm instanceof HTMLFormElement)) return;
        if (confirmButton instanceof HTMLButtonElement) {
          confirmButton.disabled = true;
          confirmButton.textContent = 'Excluindo...';
        }
        const form = pendingDeleteForm;
        pendingDeleteForm = null;
        closeDialog(deleteDialog);
        form.requestSubmit();
      });
    }
  });
})();
