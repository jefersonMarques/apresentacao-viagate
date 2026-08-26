import {
  buildPublicProposalUrl,
  supabase,
} from './supabase.js';

const iconMarkup = {
  edit: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20h4l11-11-4-4L4 16v4Z"/><path d="m13.5 6.5 4 4"/></svg>',
  external: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M14 4h6v6"/><path d="m20 4-9 9"/><path d="M20 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1h5"/></svg>',
  copy: '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="9" y="9" width="11" height="11" rx="1"/><path d="M15 9V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v9a1 1 0 0 0 1 1h4"/></svg>',
  trash: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16"/><path d="M9 7V4h6v3"/><path d="m6 7 1 13h10l1-13"/><path d="M10 11v5M14 11v5"/></svg>',
};

const state = {
  processingDelete: new Set(),
};

function appendStyles() {
  if (document.querySelector('style[data-proposal-list-actions]')) {
    return;
  }

  const style = document.createElement('style');
  style.dataset.proposalListActions = 'true';
  style.textContent = `
    .proposal-list-actions {
      display: flex;
      flex-wrap: wrap;
      justify-content: flex-end;
      gap: 6px;
    }

    .proposal-list-action {
      min-width: 36px;
      min-height: 34px;
      padding: 0 9px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 6px;
      border: 1px solid #dbe3e8;
      background: #fff;
      color: #173044;
      cursor: pointer;
      font: inherit;
      font-size: 9px;
      font-weight: 800;
      line-height: 1;
      white-space: nowrap;
    }

    .proposal-list-action:hover {
      border-color: #ff6b18;
      color: #d85000;
      background: #fff8f3;
    }

    .proposal-list-action.danger:hover {
      border-color: #b42318;
      color: #b42318;
      background: #fff5f4;
    }

    .proposal-list-action:disabled {
      cursor: wait;
      opacity: .55;
    }

    .proposal-list-action svg {
      width: 15px;
      height: 15px;
      flex: 0 0 15px;
      fill: none;
      stroke: currentColor;
      stroke-width: 1.8;
      stroke-linecap: round;
      stroke-linejoin: round;
    }

    .proposal-list-action.copy-success {
      border-color: #14804a;
      color: #14804a;
      background: #f3fbf6;
    }

    @media (max-width: 720px) {
      .proposal-list-actions {
        justify-content: flex-start;
      }

      .proposal-list-action {
        min-height: 38px;
      }

      .proposal-list-action span {
        display: none;
      }
    }
  `;
  document.head.appendChild(style);
}

function getProposalId(row) {
  return row.querySelector('[data-open-proposal]')?.dataset.openProposal ?? '';
}

async function getLatestPublishedVersion(proposalId) {
  const { data, error } = await supabase
    .from('proposal_versions')
    .select('public_token,published_at,version_number')
    .eq('proposal_id', proposalId)
    .not('published_at', 'is', null)
    .order('version_number', { ascending: false })
    .limit(1)
    .maybeSingle();

  if (error) {
    throw error;
  }

  return data ?? null;
}

async function copyText(value) {
  if (navigator.clipboard?.writeText && window.isSecureContext) {
    await navigator.clipboard.writeText(value);
    return;
  }

  const textarea = document.createElement('textarea');
  textarea.value = value;
  textarea.setAttribute('readonly', '');
  textarea.style.position = 'fixed';
  textarea.style.opacity = '0';
  document.body.appendChild(textarea);
  textarea.select();
  textarea.setSelectionRange(0, textarea.value.length);

  const copied = document.execCommand('copy');
  textarea.remove();

  if (!copied) {
    throw new Error('Clipboard unavailable');
  }
}

function flashButton(button, label = 'Copiado') {
  const original = button.innerHTML;
  button.classList.add('copy-success');
  button.innerHTML = `${iconMarkup.copy}<span>${label}</span>`;

  window.setTimeout(() => {
    button.classList.remove('copy-success');
    button.innerHTML = original;
  }, 1400);
}

async function copyProposalLink(proposalId, button) {
  const version = await getLatestPublishedVersion(proposalId);
  if (!version?.public_token) {
    window.alert('Esta proposta ainda não possui uma versão publicada.');
    return;
  }

  const url = buildPublicProposalUrl(version.public_token);

  try {
    await copyText(url);
    flashButton(button);
  } catch {
    window.prompt('Copie o link da proposta:', url);
  }
}

async function openPublicProposal(proposalId) {
  const version = await getLatestPublishedVersion(proposalId);
  if (!version?.public_token) {
    window.alert('Esta proposta ainda não possui uma versão publicada.');
    return;
  }

  window.open(buildPublicProposalUrl(version.public_token), '_blank', 'noopener,noreferrer');
}

async function deleteProposal(proposalId, row, button) {
  if (!proposalId || state.processingDelete.has(proposalId)) {
    return;
  }

  const clientName = row.querySelector('strong')?.textContent?.trim() || 'esta proposta';
  const confirmed = window.confirm(`Excluir definitivamente a proposta de ${clientName}?`);

  if (!confirmed) {
    return;
  }

  state.processingDelete.add(proposalId);
  button.disabled = true;

  try {
    const { error } = await supabase
      .from('proposals')
      .delete()
      .eq('id', proposalId);

    if (error) {
      throw error;
    }

    row.remove();
  } catch (error) {
    console.error(error);
    window.alert('Não foi possível excluir a proposta. Verifique as permissões e vínculos existentes.');
    button.disabled = false;
  } finally {
    state.processingDelete.delete(proposalId);
  }
}

function createActionButton({ icon, label, title, className = '', handler }) {
  const button = document.createElement('button');
  button.type = 'button';
  button.className = `proposal-list-action${className ? ` ${className}` : ''}`;
  button.title = title;
  button.setAttribute('aria-label', title);
  button.innerHTML = `${iconMarkup[icon]}<span>${label}</span>`;

  button.addEventListener('click', async (event) => {
    event.preventDefault();
    event.stopPropagation();

    try {
      await handler(button);
    } catch (error) {
      console.error(error);
      window.alert('Não foi possível concluir esta ação.');
    }
  });

  return button;
}

function enhanceProposalRow(row) {
  if (row.dataset.actionsEnhanced === 'true') {
    return;
  }

  const originalEditButton = row.querySelector('[data-open-proposal]');
  const proposalId = getProposalId(row);
  if (!originalEditButton || !proposalId) {
    return;
  }

  const host = originalEditButton.parentElement;
  if (!host) {
    return;
  }

  const editHandler = () => originalEditButton.click();

  row.dataset.actionsEnhanced = 'true';
  host.innerHTML = '';

  const actions = document.createElement('div');
  actions.className = 'proposal-list-actions';

  actions.append(
    createActionButton({
      icon: 'edit',
      label: 'Editar',
      title: 'Editar proposta',
      handler: editHandler,
    }),
    createActionButton({
      icon: 'external',
      label: 'Abrir',
      title: 'Abrir proposta publicada',
      handler: () => openPublicProposal(proposalId),
    }),
    createActionButton({
      icon: 'copy',
      label: 'Copiar link',
      title: 'Copiar link público',
      handler: (button) => copyProposalLink(proposalId, button),
    }),
    createActionButton({
      icon: 'trash',
      label: 'Excluir',
      title: 'Excluir proposta',
      className: 'danger',
      handler: (button) => deleteProposal(proposalId, row, button),
    }),
  );

  host.appendChild(actions);
}

function enhanceProposalList() {
  document.querySelectorAll('#proposalList .proposal-row').forEach(enhanceProposalRow);
}

function initialize() {
  appendStyles();
  enhanceProposalList();

  const list = document.getElementById('proposalList');
  if (!list) {
    return;
  }

  const observer = new MutationObserver(enhanceProposalList);
  observer.observe(list, { childList: true });
}

initialize();
