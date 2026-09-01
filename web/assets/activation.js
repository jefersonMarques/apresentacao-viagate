(() => {
  const goodsRoot = document.querySelector('[data-activation-goods]');
  const usersRoot = document.querySelector('[data-activation-users]');

  const bindRemove = (root) => {
    if (!root) return;
    root.addEventListener('click', (event) => {
      const button = event.target.closest('.activation-remove');
      if (!button) return;
      const row = button.closest('.activation-repeat-row, .activation-user-row');
      if (!row) return;
      const rows = root.querySelectorAll('.activation-repeat-row, .activation-user-row');
      if (rows.length === 1) {
        row.querySelectorAll('input').forEach((input) => { input.value = ''; });
        return;
      }
      row.remove();
    });
  };

  bindRemove(goodsRoot);
  bindRemove(usersRoot);

  const addGood = document.querySelector('[data-add-activation-good]');
  if (addGood && goodsRoot) {
    addGood.addEventListener('click', () => {
      const row = document.createElement('div');
      row.className = 'activation-repeat-row';
      row.innerHTML = '<input name="goods" placeholder="Ex.: Alimentos refrigerados" required><button type="button" class="btn ghost activation-remove" aria-label="Remover">×</button>';
      goodsRoot.appendChild(row);
      row.querySelector('input').focus();
    });
  }

  const addUser = document.querySelector('[data-add-activation-user]');
  if (addUser && usersRoot) {
    addUser.addEventListener('click', () => {
      const row = document.createElement('div');
      row.className = 'activation-user-row';
      row.innerHTML = '<input name="system_user_name" placeholder="Nome" required><input name="system_user_phone" placeholder="Telefone"><input type="email" name="system_user_email" placeholder="E-mail" required><button type="button" class="btn ghost activation-remove" aria-label="Remover">×</button>';
      usersRoot.appendChild(row);
      row.querySelector('input').focus();
    });
  }

  const finance = document.querySelector('#financeiro');
  const goods = document.querySelector('#mercadorias');
  const users = document.querySelector('#usuarios');
  const submitBox = document.querySelector('.activation-submit-box');
  const completeBox = document.querySelector('.activation-complete-box');
  const progress = Array.from(document.querySelectorAll('.activation-progress-item'));
  if (!finance || !goods || !users || !progress.length) return;

  const panels = { finance, goods, users, review: submitBox || completeBox };
  const params = new URL(window.location.href).searchParams;

  function hasValue(selector) {
    const input = document.querySelector(selector);
    return input instanceof HTMLInputElement && input.value.trim() !== '';
  }

  function financeComplete() {
    return hasValue('#financeiro input[name="finance_name"]') || Boolean(document.querySelector('#financeiro .activation-answer strong'));
  }

  function goodsComplete() {
    const editable = Array.from(document.querySelectorAll('#mercadorias input[name="goods"]')).some((input) => input.value.trim());
    return editable || Boolean(document.querySelector('#mercadorias .activation-tags span'));
  }

  function usersComplete() {
    const editable = Array.from(document.querySelectorAll('#usuarios input[name="system_user_name"]')).some((input) => input.value.trim());
    return editable || Boolean(document.querySelector('#usuarios .activation-user-list strong'));
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

  function setButtonLabels(step) {
    const financeButton = finance.querySelector('button[type="submit"]');
    const goodsButtons = Array.from(goods.querySelectorAll('button[type="submit"]'));
    const usersButtons = Array.from(users.querySelectorAll('button[type="submit"]'));
    if (financeButton && step === 'finance') financeButton.textContent = 'Salvar e continuar';
    if (goodsButtons.length && step === 'goods') goodsButtons.at(-1).textContent = 'Salvar e continuar';
    if (usersButtons.length && step === 'users') usersButtons.at(-1).textContent = 'Salvar e revisar';
  }

  function setStep(step) {
    const state = completion();
    if (step === 'goods' && !state.finance) step = 'finance';
    if (step === 'users' && (!state.finance || !state.goods)) step = !state.finance ? 'finance' : 'goods';
    if (step === 'review' && (!state.finance || !state.goods || !state.users) && !completeBox) {
      step = !state.finance ? 'finance' : !state.goods ? 'goods' : 'users';
    }

    Object.entries(panels).forEach(([name, panel]) => {
      if (panel instanceof HTMLElement) panel.hidden = name !== step;
    });
    progress.forEach((item, index) => {
      const name = ['finance', 'goods', 'users'][index];
      item.classList.toggle('is-current', name === step);
      item.classList.toggle('is-complete', Boolean(state[name]));
      item.setAttribute('role', 'button');
      item.tabIndex = 0;
      item.setAttribute('aria-current', name === step ? 'step' : 'false');
    });
    setButtonLabels(step);
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

  setStep(params.get('step') || initialStep());
})();
