import { supabase } from './supabase.js';

const state = {
  publishBypass: false,
  configured: false,
};

function wait(ms) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

function getProposalId() {
  return document.getElementById('proposalId')?.value?.trim() ?? '';
}

function getPasswordInput() {
  return document.getElementById('proposalAccessPassword');
}

function getRemoveInput() {
  return document.getElementById('proposalRemovePassword');
}

function getStatusElement() {
  return document.getElementById('proposalAccessStatus');
}

function setStatus(message, configured = state.configured) {
  state.configured = configured;
  const element = getStatusElement();

  if (!element) {
    return;
  }

  element.textContent = message;
  element.dataset.configured = configured ? 'true' : 'false';
}

function createAccessFields() {
  if (document.getElementById('proposalAccessPassword')) {
    return;
  }

  const validUntil = document.getElementById('validUntil')?.closest('.form-field');
  if (!validUntil) {
    return;
  }

  const wrapper = document.createElement('div');
  wrapper.className = 'form-field full proposal-access-field';
  wrapper.innerHTML = `
    <label for="proposalAccessPassword">Senha para abrir a proposta <span class="proposal-optional-text">opcional</span></label>
    <div class="proposal-access-input-row">
      <input id="proposalAccessPassword" type="password" minlength="4" maxlength="72" autocomplete="new-password" placeholder="Sem senha" />
      <label class="proposal-access-remove">
        <input id="proposalRemovePassword" type="checkbox" />
        <span>Remover senha atual</span>
      </label>
    </div>
    <small id="proposalAccessStatus" class="proposal-access-status">Sem senha configurada.</small>
  `;

  validUntil.insertAdjacentElement('afterend', wrapper);
}

async function loadAccessState(proposalId) {
  const passwordInput = getPasswordInput();
  const removeInput = getRemoveInput();

  if (passwordInput) {
    passwordInput.value = '';
  }

  if (removeInput) {
    removeInput.checked = false;
  }

  if (!proposalId) {
    setStatus('Sem senha configurada.', false);
    return;
  }

  const { data, error } = await supabase.rpc('get_proposal_access_settings', {
    target_proposal_id: proposalId,
  });

  if (error) {
    console.warn('Não foi possível carregar a configuração de acesso da proposta.', error);
    setStatus('Não foi possível consultar a senha desta proposta.', false);
    return;
  }

  const configured = Boolean(data?.requires_password);
  setStatus(
    configured
      ? 'Senha configurada. Deixe o campo vazio para manter a senha atual.'
      : 'Sem senha configurada.',
    configured,
  );
}

async function saveAccessState(proposalId) {
  if (!proposalId) {
    return;
  }

  const password = getPasswordInput()?.value?.trim() ?? '';
  const removePassword = Boolean(getRemoveInput()?.checked);

  if (!password && !removePassword) {
    return;
  }

  const { data, error } = await supabase.rpc('set_proposal_access_password', {
    target_proposal_id: proposalId,
    access_password: removePassword ? null : password,
  });

  if (error) {
    throw error;
  }

  if (getPasswordInput()) {
    getPasswordInput().value = '';
  }

  if (getRemoveInput()) {
    getRemoveInput().checked = false;
  }

  setStatus(
    data
      ? 'Senha configurada. Deixe o campo vazio para manter a senha atual.'
      : 'Sem senha configurada.',
    Boolean(data),
  );
}

async function waitForDraftSave(previousProposalId = '', timeoutMs = 8000) {
  const startedAt = Date.now();

  while (Date.now() - startedAt < timeoutMs) {
    const proposalId = getProposalId();
    const message = document.getElementById('proposalMessage');
    const success = Boolean(message && !message.hidden && message.classList.contains('success'));
    const error = Boolean(message && !message.hidden && message.classList.contains('error'));

    if (error) {
      return null;
    }

    if (proposalId && success) {
      return proposalId;
    }

    if (proposalId && proposalId === previousProposalId && success) {
      return proposalId;
    }

    await wait(80);
  }

  return null;
}

function bindOpenProposalState() {
  document.addEventListener('click', (event) => {
    const button = event.target.closest('[data-open-proposal]');
    if (!button) {
      return;
    }

    const expectedProposalId = button.dataset.openProposal;

    window.setTimeout(async () => {
      for (let attempt = 0; attempt < 60; attempt += 1) {
        if (getProposalId() === expectedProposalId) {
          await loadAccessState(expectedProposalId);
          return;
        }

        await wait(100);
      }
    }, 0);
  }, true);
}

function bindNewProposalState() {
  document.getElementById('newProposalButton')?.addEventListener('click', () => {
    window.setTimeout(() => loadAccessState(''), 0);
  });
}

function bindDraftSave() {
  document.getElementById('proposalForm')?.addEventListener('submit', () => {
    const previousProposalId = getProposalId();

    window.setTimeout(async () => {
      const proposalId = await waitForDraftSave(previousProposalId);
      if (!proposalId) {
        return;
      }

      try {
        await saveAccessState(proposalId);
      } catch (error) {
        console.error('Não foi possível salvar a senha da proposta.', error);
        setStatus('A proposta foi salva, mas a senha não pôde ser atualizada.', state.configured);
      }
    }, 0);
  });
}

function bindPublishProtection() {
  const publishButton = document.getElementById('publishProposalButton');
  const form = document.getElementById('proposalForm');

  publishButton?.addEventListener('click', async (event) => {
    if (state.publishBypass) {
      return;
    }

    const password = getPasswordInput()?.value?.trim() ?? '';
    const removePassword = Boolean(getRemoveInput()?.checked);

    if (!password && !removePassword) {
      return;
    }

    event.preventDefault();
    event.stopImmediatePropagation();

    const originalLabel = publishButton.textContent;
    publishButton.disabled = true;
    publishButton.textContent = 'Preparando...';

    const previousProposalId = getProposalId();
    form?.requestSubmit();

    const proposalId = await waitForDraftSave(previousProposalId);

    if (!proposalId) {
      publishButton.disabled = false;
      publishButton.textContent = originalLabel;
      return;
    }

    try {
      await saveAccessState(proposalId);
      state.publishBypass = true;
      publishButton.disabled = false;
      publishButton.textContent = originalLabel;
      publishButton.click();
    } catch (error) {
      console.error('Não foi possível configurar a senha antes da publicação.', error);
      publishButton.disabled = false;
      publishButton.textContent = originalLabel;
      setStatus('Não foi possível salvar a senha. A proposta não foi publicada.', state.configured);
    } finally {
      window.setTimeout(() => {
        state.publishBypass = false;
      }, 0);
    }
  }, true);
}

function appendStyles() {
  const style = document.createElement('style');
  style.textContent = `
    .proposal-access-field { margin-top: 4px; }
    .proposal-access-input-row { display:grid; grid-template-columns:minmax(0,1fr) auto; gap:12px; align-items:center; }
    .proposal-access-remove { min-height:44px; padding:0 12px; display:flex; align-items:center; gap:8px; border:1px solid #dbe3e8; background:#fff; color:#607583; font-size:11px; }
    .proposal-access-remove input { width:auto; }
    .proposal-access-status { display:block; margin-top:7px; color:#78858e; font-size:10px; }
    .proposal-access-status[data-configured="true"] { color:#178052; }
    .proposal-optional-text { color:#8b99a2; font-size:10px; font-weight:500; }
    @media (max-width:720px) { .proposal-access-input-row { grid-template-columns:1fr; } }
  `;
  document.head.appendChild(style);
}

function initialize() {
  createAccessFields();
  appendStyles();
  bindOpenProposalState();
  bindNewProposalState();
  bindDraftSave();
  bindPublishProtection();
}

initialize();
