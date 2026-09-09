(() => {
  const goodsRoot = document.querySelector('[data-activation-goods]');
  const usersRoot = document.querySelector('[data-activation-users]');
  const goodsPersisted = goodsRoot?.getAttribute('data-persisted-complete') === 'true';
  const usersPersisted = usersRoot?.getAttribute('data-persisted-complete') === 'true';
  let goodsDirty = false;
  let usersDirty = false;

  function goodsFormReady() {
    if (!goodsRoot) return false;
    const inputs = Array.from(goodsRoot.querySelectorAll('input[name="goods"]'));
    return inputs.length > 0 && inputs.every((input) => input.value.trim() !== '');
  }

  function usersFormReady() {
    if (!usersRoot) return false;
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

  function syncSectionSaveButton(root, ready) {
    const form = root?.closest('form');
    const button = form?.querySelector('[data-activation-section-save]');
    if (!(button instanceof HTMLButtonElement)) return;
    button.disabled = !ready;
    button.setAttribute('aria-disabled', ready ? 'false' : 'true');
  }

  function syncSectionSaveButtons() {
    syncSectionSaveButton(goodsRoot, goodsFormReady());
    syncSectionSaveButton(usersRoot, usersFormReady());
  }

  const bindRemove = (root, markDirty) => {
    if (!root) return;
    root.addEventListener('click', (event) => {
      const button = event.target.closest('.activation-remove');
      if (!button) return;
      const row = button.closest('.activation-repeat-row, .activation-user-row');
      if (!row) return;

      const rows = root.querySelectorAll('.activation-repeat-row, .activation-user-row');
      if (rows.length === 1) {
        row.querySelectorAll('input').forEach((input) => { input.value = ''; });
      } else {
        row.remove();
      }

      markDirty();
      syncSectionSaveButtons();
      renderProgress();
    });
  };

  bindRemove(goodsRoot, () => { goodsDirty = true; });
  bindRemove(usersRoot, () => { usersDirty = true; });

  const addGood = document.querySelector('[data-add-activation-good]');
  if (addGood && goodsRoot) {
    addGood.addEventListener('click', () => {
      const row = document.createElement('div');
      row.className = 'activation-repeat-row';
      row.innerHTML = '<input name="goods" placeholder="Ex.: Alimentos refrigerados" required><button type="button" class="btn ghost activation-remove" aria-label="Remover">×</button>';
      goodsRoot.appendChild(row);
      goodsDirty = true;
      syncSectionSaveButtons();
      renderProgress();
      row.querySelector('input')?.focus();
    });
  }

  const addUser = document.querySelector('[data-add-activation-user]');
  if (addUser && usersRoot) {
    addUser.addEventListener('click', () => {
      const row = document.createElement('div');
      row.className = 'activation-user-row';
      row.innerHTML = '<input name="system_user_name" placeholder="Nome" required><input name="system_user_phone" placeholder="Telefone"><input type="email" name="system_user_email" placeholder="E-mail" required><button type="button" class="btn ghost activation-remove" aria-label="Remover">×</button>';
      usersRoot.appendChild(row);
      usersDirty = true;
      syncSectionSaveButtons();
      renderProgress();
      row.querySelector('input')?.focus();
    });
  }

  goodsRoot?.addEventListener('input', () => {
    goodsDirty = true;
    syncSectionSaveButtons();
    renderProgress();
  });
  usersRoot?.addEventListener('input', () => {
    usersDirty = true;
    syncSectionSaveButtons();
    renderProgress();
  });

  const finance = document.querySelector('#financeiro');
  const goods = document.querySelector('#mercadorias');
  const users = document.querySelector('#usuarios');
  const submitBox = document.querySelector('.activation-submit-box');
  const completeBox = document.querySelector('.activation-complete-box');
  const progress = Array.from(document.querySelectorAll('.activation-progress-item'));
  if (!finance || !goods || !users || !progress.length) return;

  const panels = { finance, goods, users, review: submitBox || completeBox };
  const params = new URL(window.location.href).searchParams;
  let activeStep = '';

  function hasValue(selector) {
    const input = document.querySelector(selector);
    return input instanceof HTMLInputElement && input.value.trim() !== '';
  }

  function financeComplete() {
    return hasValue('#financeiro input[name="finance_name"]') || Boolean(document.querySelector('#financeiro .activation-answer strong'));
  }

  function goodsComplete() {
    if (document.querySelector('#mercadorias .activation-tags span')) return true;
    return goodsPersisted && !goodsDirty;
  }

  function usersComplete() {
    if (document.querySelector('#usuarios .activation-user-list strong')) return true;
    return usersPersisted && !usersDirty;
  }

  function completion() {
    return { finance: financeComplete(), goods: goodsComplete(), users: usersComplete() };
  }

  function initialStep() {
    if (completeBox) return 'review';
    const saved = params.get('saved');
    if (saved === 'finance') return 'goods';
    if (saved === 'goods') return 'users';
    if (saved === 'users') return 'review';

    const state = completion();
    if (!state.finance) return 'finance';
    if (!state.goods) return 'goods';
    if (!state.users) return 'users';
    return 'review';
  }

  function resolveStep(step) {
    const state = completion();
    if (step === 'goods' && !state.finance) return 'finance';
    if (step === 'users' && (!state.finance || !state.goods)) return !state.finance ? 'finance' : 'goods';
    if (step === 'review' && (!state.finance || !state.goods || !state.users) && !completeBox) {
      return !state.finance ? 'finance' : !state.goods ? 'goods' : 'users';
    }
    return step;
  }

  function renderProgress() {
    const state = completion();
    progress.forEach((item, index) => {
      const name = ['finance', 'goods', 'users'][index];
      item.classList.toggle('is-current', name === activeStep);
      item.classList.toggle('is-complete', Boolean(state[name]));
      item.setAttribute('role', 'button');
      item.tabIndex = 0;
      item.setAttribute('aria-current', name === activeStep ? 'step' : 'false');
    });
  }

  function setStep(step) {
    step = resolveStep(step);
    activeStep = step;

    Object.entries(panels).forEach(([name, panel]) => {
      if (panel instanceof HTMLElement) panel.hidden = name !== step;
    });

    renderProgress();

    const url = new URL(window.location.href);
    url.searchParams.set('step', step);
    window.history.replaceState({}, '', url);
  }

  progress.forEach((item, index) => {
    const step = ['finance', 'goods', 'users'][index];
    const activate = () => setStep(step);
    item.addEventListener('click', activate);
    item.addEventListener('keydown', (event) => {
      if (event.key !== 'Enter' && event.key !== ' ') return;
      event.preventDefault();
      activate();
    });
  });

  syncSectionSaveButtons();
  setStep(params.get('step') || initialStep());
})();
