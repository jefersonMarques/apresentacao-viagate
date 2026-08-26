import {
  buildPublicPresentationUrl,
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
  observedTargets: new WeakSet(),
};

function appendStyles() {
  if (document.querySelector('style[data-proposal-list-actions]')) {
    return;
  }

  const style = document.createElement('style');
  style.dataset.proposalListActions = 'true';
  style.textContent = `
    .proposal-list-actions,
    .hub-row-actions.proposal-actions-enhanced,
    .hub-tracking-actions {
      display: flex;
      flex-wrap: wrap;
      justify-content: flex-end;
      gap: 6px;
    }

    .proposal-list-action,
    .hub-row-actions.proposal-actions-enhanced > button,
    .hub-tracking-actions > button {
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
      text-decoration: none;
    }

    .proposal-list-action:hover,
    .hub-row-actions.proposal-actions-enhanced > button:hover,
    .hub-tracking-actions > button:hover {
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

    .proposal-list-action svg,
    .hub-row-actions.proposal-actions-enhanced svg,
    .hub-tracking-actions svg {
      width: 15px;
      height: 15px;
      flex: 0 0 15px;
      fill: none;
      stroke: currentColor;
      stroke-width: 1.8;
      stroke-linecap: round;
      stroke-linejoin: round;
    }

    .proposal-list-action.copy-success,
    .copy-success {
      border-color: #14804a !important;
      color: #14804a !important;
      background: #f3fbf6 !important;
    }

    .proposal-original-edit,
    .proposal-row [data-copy-proposal-tracking] {
      display: none !important;
    }

    @media (max-width: 720px) {
      .proposal-list-actions,
      .hub-row-actions.proposal-actions-enhanced,
      .hub-tracking-actions {
        justify-content: flex-start;
      }

      .proposal-list-action,
      .hub-row-actions.proposal-actions-enhanced > button,
      .hub-tracking-actions > button {
        min-height: 38px;
      }

      .proposal-list-action span,
      .hub-row-actions.proposal-actions-enhanced > button span,
      .hub-tracking-actions > button span {
        display: none;
      }
    }
  `;
  document.head.appendChild(style);
}

function buildPublicUrl(kind, token) {
  if (!token) {
    return '';
  }

  return kind === 'presentation'
    ? buildPublicPresentationUrl(token)
    : buildPublicProposalUrl(token);
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
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch {
      // Usa o fallback abaixo quando a permissão do Clipboard API for bloqueada.
    }
  }

  const textarea = document.createElement('textarea');
  textarea.value = value;
  textarea.setAttribute('readonly', '');
  textarea.style.position = 'fixed';
  textarea.style.top = '0';
  textarea.style.left = '-9999px';
  textarea.style.opacity = '0';
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  textarea.setSelectionRange(0, textarea.value.length);

  const copied = document.execCommand('copy');
  textarea.remove();

  if (!copied) {
    throw new Error('Clipboard unavailable');
  }
}

function flashCopyButton(button) {
  const original = button.innerHTML;
  button.classList.add('copy-success');
  button.innerHTML = `${iconMarkup.copy}<span>Copiado</span>`;

  window.setTimeout(() => {
    button.classList.remove('copy-success');
    button.innerHTML = original;
  }, 1400);
}

async function copyPublicUrl(kind, token, button) {
  const url = buildPublicUrl(kind, token);
  if (!url) {
    return;
  }

  try {
    await copyText(url);
    flashCopyButton(button);
  } catch {
    window.prompt('Copie o link:', url);
  }
}

function openPublicUrl(kind, token) {
  const url = buildPublicUrl(kind, token);
  if (!url) {
    return;
  }

  const opened = window.open(url, '_blank');
  if (opened) {
    opened.opener = null;
    return;
  }

  window.location.href = url;
}

async function copyProposalLink(proposalId, button) {
  const version = await getLatestPublishedVersion(proposalId);
  if (!version?.public_token) {
    window.alert('Esta proposta ainda não possui uma versão publicada.');
    return;
  }

  await copyPublicUrl('proposal', version.public_token, button);
}

async function openPublicProposal(proposalId) {
  const version = await getLatestPublishedVersion(proposalId);
  if (!version?.public_token) {
    window.alert('Esta proposta ainda não possui uma versão publicada.');
    return;
  }

  openPublicUrl('proposal', version.public_token);
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
    const { data, error } = await supabase.rpc('delete_proposal', {
      target_proposal_id: proposalId,
    });

    if (error || data !== true) {
      throw error ?? new Error('Proposal was not deleted');
    }

    row.remove();
  } catch (error) {
    console.error(error);
    window.alert('Não foi possível excluir a proposta. Verifique se a migration de exclusão foi aplicada no Supabase.');
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
  const originalEditButton = row.querySelector('[data-open-proposal]');
  const proposalId = originalEditButton?.dataset.openProposal ?? '';
  if (!originalEditButton || !proposalId) {
    return;
  }

  const host = originalEditButton.parentElement;
  if (!host) {
    return;
  }

  originalEditButton.classList.add('proposal-original-edit');

  let actions = host.querySelector('.proposal-list-actions');
  if (!actions) {
    actions = document.createElement('div');
    actions.className = 'proposal-list-actions';
    host.appendChild(actions);
  }

  if (actions.dataset.ready === 'true') {
    return;
  }

  actions.dataset.ready = 'true';
  actions.append(
    createActionButton({
      icon: 'edit',
      label: 'Editar',
      title: 'Editar proposta',
      handler: () => originalEditButton.click(),
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
}

function enhanceTrackingTable() {
  document.querySelectorAll('#hubTrackingTable [data-copy-tracking-token]').forEach((copyButton) => {
    const token = copyButton.dataset.copyTrackingToken ?? '';
    const kind = copyButton.dataset.copyTrackingKind ?? 'proposal';
    const host = copyButton.parentElement;

    if (!host || !token) {
      return;
    }

    let actions = host.querySelector('.hub-tracking-actions');
    if (!actions) {
      actions = document.createElement('div');
      actions.className = 'hub-tracking-actions';
      host.appendChild(actions);
    }

    copyButton.remove();

    if (actions.dataset.ready === 'true') {
      return;
    }

    actions.dataset.ready = 'true';
    actions.append(
      createActionButton({
        icon: 'external',
        label: 'Abrir',
        title: kind === 'presentation' ? 'Abrir apresentação' : 'Abrir proposta',
        handler: () => openPublicUrl(kind, token),
      }),
      createActionButton({
        icon: 'copy',
        label: 'Copiar link',
        title: 'Copiar link público',
        handler: (button) => copyPublicUrl(kind, token, button),
      }),
    );
  });
}

function enhancePresentationList() {
  document.querySelectorAll('#presentationList .hub-row-actions').forEach((host) => {
    const copyButton = host.querySelector('[data-copy-presentation]');
    const editButton = host.querySelector('[data-open-presentation]');
    const token = copyButton?.dataset.copyPresentation ?? '';

    host.classList.add('proposal-actions-enhanced');

    if (editButton && !editButton.dataset.iconEnhanced) {
      editButton.dataset.iconEnhanced = 'true';
      editButton.innerHTML = `${iconMarkup.edit}<span>Editar</span>`;
      editButton.title = 'Editar apresentação';
    }

    if (!token || host.querySelector('[data-open-published-presentation]')) {
      return;
    }

    if (copyButton) {
      copyButton.innerHTML = `${iconMarkup.copy}<span>Copiar link</span>`;
      copyButton.title = 'Copiar link público';
    }

    const openButton = createActionButton({
      icon: 'external',
      label: 'Abrir',
      title: 'Abrir apresentação publicada',
      handler: () => openPublicUrl('presentation', token),
    });
    openButton.dataset.openPublishedPresentation = 'true';
    host.insertBefore(openButton, copyButton ?? editButton ?? null);
  });
}

function enhanceAllLists() {
  document.querySelectorAll('#proposalList .proposal-row').forEach(enhanceProposalRow);
  enhanceTrackingTable();
  enhancePresentationList();
}

function observeAvailableTargets() {
  const targets = [
    document.getElementById('proposalList'),
    document.getElementById('hubTrackingTable'),
    document.getElementById('presentationList'),
  ].filter(Boolean);

  targets.forEach((target) => {
    if (state.observedTargets.has(target)) {
      return;
    }

    state.observedTargets.add(target);
    const observer = new MutationObserver(() => window.setTimeout(enhanceAllLists, 0));
    observer.observe(target, { childList: true, subtree: true });
  });

  return targets.length;
}

function interceptLegacyCopy(event) {
  const button = event.target.closest(
    '[data-copy-tracking-token], [data-copy-presentation], [data-copy-proposal-tracking]',
  );

  if (!button) {
    return;
  }

  let kind = 'proposal';
  let token = '';

  if (button.dataset.copyTrackingToken) {
    kind = button.dataset.copyTrackingKind || 'proposal';
    token = button.dataset.copyTrackingToken;
  } else if (button.dataset.copyPresentation) {
    kind = 'presentation';
    token = button.dataset.copyPresentation;
  } else if (button.dataset.copyProposalTracking) {
    kind = 'proposal';
    token = button.dataset.copyProposalTracking;
  }

  if (!token) {
    return;
  }

  event.preventDefault();
  event.stopImmediatePropagation();
  copyPublicUrl(kind, token, button).catch((error) => {
    console.error(error);
  });
}

function initialize() {
  appendStyles();
  enhanceAllLists();
  observeAvailableTargets();
  document.addEventListener('click', interceptLegacyCopy, true);

  document.querySelectorAll('[data-hub-tab]').forEach((button) => {
    button.addEventListener('click', () => window.setTimeout(() => {
      observeAvailableTargets();
      enhanceAllLists();
    }, 0));
  });

  const bootstrapObserver = new MutationObserver(() => {
    observeAvailableTargets();
    enhanceAllLists();
  });

  bootstrapObserver.observe(document.body, { childList: true, subtree: true });
  window.setTimeout(() => bootstrapObserver.disconnect(), 10000);
}

initialize();
