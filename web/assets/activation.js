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
})();
