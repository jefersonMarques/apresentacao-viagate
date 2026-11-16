(() => {
  const page = document.querySelector('[data-activation-wizard]');
  if (!(page instanceof HTMLElement)) {
    return;
  }

  const stepButtons = Array.from(page.querySelectorAll('[data-activation-step-link]'));
  const panels = new Map(
    Array.from(page.querySelectorAll('[data-activation-step]')).map((panel) => [
      panel.dataset.activationStep,
      panel,
    ]),
  );
  if (stepButtons.length === 0 || panels.size === 0) {
    return;
  }

  const steps = stepButtons.map((button) => button.dataset.activationStepLink);
  const accessSection = page.dataset.activationAccessSection;
  const params = new URL(window.location.href).searchParams;
  const stepCount = page.querySelector('[data-activation-step-count]');
  const stepLabel = page.querySelector('[data-activation-step-label]');
  const goodsRoot = page.querySelector('[data-activation-goods]');
  const usersRoot = page.querySelector('[data-activation-users]');
  const policyInput = page.querySelector('[data-activation-policy-input]');
  const policyUpload = page.querySelector('[data-activation-policy-upload]');
  const policyFileName = page.querySelector('[data-activation-policy-filename]');
  const policyFeedback = page.querySelector('[data-activation-policy-feedback]');
  const policySubmit = page.querySelector('[data-activation-policy-submit]');
  const maxPolicyBytes = 15 * 1024 * 1024;
  let activeStep = '';

  function currentPanel() {
    return panels.get(activeStep);
  }

  function currentForm() {
    return currentPanel()?.querySelector('[data-activation-step-form]');
  }

  function hasUnsavedChanges() {
    return currentForm()?.dataset.activationDirty === 'true';
  }

  function confirmNavigation() {
    if (!hasUnsavedChanges()) {
      return true;
    }

    return window.confirm('Existem alterações não salvas nesta etapa. Deseja sair sem salvar?');
  }

  function findFirstIncomplete() {
    const nextIncomplete = stepButtons.find((button) => (
      button.dataset.activationStepLink !== 'review' &&
      button.dataset.activationComplete !== 'true'
    ));

    if (nextIncomplete) {
      return nextIncomplete.dataset.activationStepLink;
    }

    return panels.has('review') ? 'review' : steps[0];
  }

  function nextStep(step) {
    const index = steps.indexOf(step);
    if (index < 0 || index === steps.length - 1) {
      return step;
    }

    return steps[index + 1];
  }

  function previousStep(step) {
    const index = steps.indexOf(step);
    if (index <= 0) {
      return step;
    }

    return steps[index - 1];
  }

  function resolveInitialStep() {
    const requested = params.get('step');
    if (panels.has(requested)) {
      return requested;
    }

    if (accessSection !== 'all') {
      return steps[0];
    }

    const saved = params.get('saved');
    if (panels.has(saved)) {
      return nextStep(saved);
    }

    if (saved === 'delegated') {
      return 'review';
    }

    return findFirstIncomplete();
  }

  function renderCurrentStep(step) {
    const index = steps.indexOf(step);

    panels.forEach((panel, key) => {
      const selected = key === step;
      panel.classList.toggle('is-active', selected);
      panel.hidden = !selected;
    });

    stepButtons.forEach((button) => {
      const selected = button.dataset.activationStepLink === step;
      button.classList.toggle('is-current', selected);

      if (selected) {
        button.setAttribute('aria-current', 'step');
      } else {
        button.removeAttribute('aria-current');
      }
    });

    if (stepCount) {
      stepCount.textContent = `Etapa ${index + 1} de ${steps.length}`;
    }
    if (stepLabel) {
      stepLabel.textContent = stepButtons[index]?.querySelector('strong')?.textContent || '';
    }

    const panel = panels.get(step);
    panel?.querySelectorAll('[data-activation-previous]').forEach((button) => {
      button.hidden = index === 0;
      button.style.display = index === 0 ? 'none' : '';
    });

    panel?.querySelectorAll('[data-activation-next]').forEach((button) => {
      button.hidden = index === steps.length - 1;
      button.style.display = index === steps.length - 1 ? 'none' : '';
    });
  }

  function goToStep(step, updateURL = true) {
    if (!panels.has(step) || step === activeStep) {
      return;
    }
    if (activeStep && !confirmNavigation()) {
      return;
    }

    activeStep = step;
    renderCurrentStep(step);

    if (updateURL) {
      const url = new URL(window.location.href);
      url.searchParams.set('step', step);
      url.searchParams.delete('saved');
      window.history.replaceState({}, '', url);
      page.querySelector('.activation-step-summary')?.scrollIntoView({
        block: 'start',
        behavior: 'smooth',
      });
    }
  }

  function updatePolicyUpload() {
    if (!(policyInput instanceof HTMLInputElement) ||
        !(policySubmit instanceof HTMLButtonElement)) {
      return;
    }

    const selectedFile = policyInput.files?.[0];

    if (!selectedFile) {
      policyInput.setCustomValidity('');
      policySubmit.disabled = true;
      policyUpload?.classList.remove('is-invalid');
      policyFeedback?.classList.remove('is-error', 'is-valid');

      if (policyFileName) {
        policyFileName.textContent = 'Selecione um arquivo';
      }
      if (policyFeedback) {
        policyFeedback.textContent = '';
      }
      return;
    }

    const supportedExtension = /\.(pdf|jpe?g|png)$/i.test(selectedFile.name);
    const supportedMimeTypes = new Set([
      'application/pdf',
      'image/jpeg',
      'image/png',
    ]);
    const supportedMime = !selectedFile.type ||
      selectedFile.type === 'application/octet-stream' ||
      supportedMimeTypes.has(selectedFile.type);
    const invalidFormat = !supportedExtension || !supportedMime;
    const invalidSize = selectedFile.size > maxPolicyBytes;
    const invalidEmptyFile = selectedFile.size === 0;

    let errorMessage = '';
    if (invalidFormat) {
      errorMessage = 'Formato não permitido. Use PDF, JPG ou PNG.';
    } else if (invalidSize) {
      errorMessage = 'O arquivo ultrapassa o limite de 15 MB.';
    } else if (invalidEmptyFile) {
      errorMessage = 'O arquivo está vazio. Selecione outro.';
    }

    policyInput.setCustomValidity(errorMessage);
    policySubmit.disabled = errorMessage !== '';

    if (policyFileName) {
      policyFileName.textContent = selectedFile.name;
    }
    if (policyFeedback) {
      const sizeInMB = (selectedFile.size / (1024 * 1024)).toFixed(1);
      policyFeedback.textContent = errorMessage ||
        `Arquivo selecionado · ${sizeInMB} MB · pronto para enviar`;
      policyFeedback.classList.toggle('is-error', errorMessage !== '');
      policyFeedback.classList.toggle('is-valid', errorMessage === '');
    }

    policyUpload?.classList.toggle('is-invalid', errorMessage !== '');
  }

  function goodsFormReady() {
    if (!goodsRoot) {
      return false;
    }

    const inputs = Array.from(goodsRoot.querySelectorAll('input[name="goods"]'));
    return inputs.length > 0 && inputs.every((input) => input.value.trim() !== '');
  }

  function usersFormReady() {
    if (!usersRoot) {
      return false;
    }

    const rows = Array.from(usersRoot.querySelectorAll('.activation-user-row'));
    return rows.length > 0 && rows.every((row) => {
      const name = row.querySelector('input[name="system_user_name"]');
      const email = row.querySelector('input[name="system_user_email"]');

      return name instanceof HTMLInputElement &&
        email instanceof HTMLInputElement &&
        name.value.trim() !== '' &&
        email.value.trim() !== '' &&
        email.validity.valid;
    });
  }

  function syncSaveButton(root, valid) {
    const button = root?.closest('form')?.querySelector('[data-activation-section-save]');
    if (button instanceof HTMLButtonElement) {
      button.disabled = !valid;
    }
  }

  function syncSaveButtons() {
    syncSaveButton(goodsRoot, goodsFormReady());
    syncSaveButton(usersRoot, usersFormReady());
  }

  function markDirty(root) {
    const form = root?.closest('form');
    if (form instanceof HTMLFormElement) {
      form.dataset.activationDirty = 'true';
    }
  }

  function removeRow(event, root) {
    const removeButton = event.target.closest('.activation-remove');
    if (!removeButton) {
      return;
    }

    const row = removeButton.closest('.activation-repeat-row, .activation-user-row');
    if (!row) {
      return;
    }

    const rows = root.querySelectorAll('.activation-repeat-row, .activation-user-row');
    if (rows.length === 1) {
      row.querySelectorAll('input').forEach((input) => {
        input.value = '';
      });
    } else {
      row.remove();
    }

    markDirty(root);
    syncSaveButtons();
  }

  function appendRow(root, markup, inputSelector) {
    if (!root) {
      return;
    }

    const row = document.createElement('div');
    row.innerHTML = markup.trim();
    const content = row.firstElementChild;
    if (!(content instanceof HTMLElement)) {
      return;
    }

    root.appendChild(content);
    markDirty(root);
    syncSaveButtons();
    content.querySelector(inputSelector)?.focus();
  }

  goodsRoot?.addEventListener('click', (event) => {
    removeRow(event, goodsRoot);
  });

  usersRoot?.addEventListener('click', (event) => {
    removeRow(event, usersRoot);
  });

  page.querySelector('[data-add-activation-good]')?.addEventListener('click', () => {
    appendRow(goodsRoot, `
      <div class="activation-repeat-row">
        <input name="goods" placeholder="Ex.: Alimentos refrigerados" required>
        <button class="btn ghost activation-remove" type="button" aria-label="Remover mercadoria">×</button>
      </div>
    `, 'input[name="goods"]');
  });

  page.querySelector('[data-add-activation-user]')?.addEventListener('click', () => {
    appendRow(usersRoot, `
      <div class="activation-user-row">
        <label class="field activation-repeat-field">
          Nome
          <input name="system_user_name" placeholder="Nome completo" required>
        </label>
        <label class="field activation-repeat-field">
          Telefone (opcional)
          <input name="system_user_phone" inputmode="tel" placeholder="(00) 00000-0000">
        </label>
        <label class="field activation-repeat-field">
          E-mail
          <input name="system_user_email" placeholder="nome@empresa.com.br" type="email" required>
        </label>
        <button class="btn ghost activation-remove" type="button" aria-label="Remover usuário">×</button>
      </div>
    `, 'input[name="system_user_name"]');
  });

  policyInput?.addEventListener('change', updatePolicyUpload);

  page.querySelectorAll('[data-activation-step-form]').forEach((form) => {
    form.addEventListener('input', () => {
      markDirty(form);
      syncSaveButtons();
    });

    form.addEventListener('change', () => {
      markDirty(form);
      syncSaveButtons();
    });
  });

  page.addEventListener('click', (event) => {
    const navigation = event.target.closest(
      '[data-activation-step-link], [data-activation-edit], ' +
      '[data-activation-next], [data-activation-previous]',
    );

    if (!(navigation instanceof HTMLElement)) {
      return;
    }

    const requested = navigation.dataset.activationStepLink ||
      navigation.dataset.activationEdit;

    if (requested) {
      goToStep(requested);
      return;
    }

    if (navigation.hasAttribute('data-activation-next')) {
      goToStep(nextStep(activeStep));
    }

    if (navigation.hasAttribute('data-activation-previous')) {
      goToStep(previousStep(activeStep));
    }
  });

  page.dataset.activationWizardReady = 'true';
  syncSaveButtons();
  updatePolicyUpload();
  goToStep(resolveInitialStep(), false);
})();
