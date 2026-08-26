import { supabase } from './supabase.js';

const BUTTON_SELECTOR = '[data-duplicate-proposal]';

async function duplicateProposal(proposalId, button) {
  if (!supabase || !proposalId || button.disabled) {
    return;
  }

  const originalLabel = button.textContent;
  button.disabled = true;
  button.textContent = 'Duplicando...';

  try {
    const { data, error } = await supabase.rpc('duplicate_proposal', {
      source_proposal_id: proposalId,
    });

    if (error) {
      throw error;
    }

    button.textContent = 'Duplicada';
    document.getElementById('refreshButton')?.click();

    window.setTimeout(() => {
      const newProposalButton = document.querySelector(`[data-open-proposal="${data}"]`);
      newProposalButton?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }, 500);
  } catch (error) {
    console.error('Não foi possível duplicar a proposta.', error);
    button.textContent = 'Erro ao duplicar';

    window.setTimeout(() => {
      button.textContent = originalLabel;
      button.disabled = false;
    }, 1800);
    return;
  }

  window.setTimeout(() => {
    button.disabled = false;
    button.textContent = originalLabel;
  }, 1200);
}

function enhanceProposalRows() {
  document.querySelectorAll('[data-open-proposal]').forEach((openButton) => {
    const proposalId = openButton.dataset.openProposal;
    const actionCell = openButton.parentElement;

    if (!proposalId || !actionCell || actionCell.querySelector(`${BUTTON_SELECTOR}[data-proposal-id="${proposalId}"]`)) {
      return;
    }

    const duplicateButton = document.createElement('button');
    duplicateButton.className = 'link-button';
    duplicateButton.type = 'button';
    duplicateButton.dataset.duplicateProposal = 'true';
    duplicateButton.dataset.proposalId = proposalId;
    duplicateButton.textContent = 'Duplicar';
    duplicateButton.addEventListener('click', () => duplicateProposal(proposalId, duplicateButton));

    actionCell.insertBefore(duplicateButton, openButton);
  });
}

function initializeDuplicateProposal() {
  enhanceProposalRows();

  const proposalList = document.getElementById('proposalList');
  if (!proposalList) {
    return;
  }

  new MutationObserver(enhanceProposalRows).observe(proposalList, {
    childList: true,
    subtree: true,
  });
}

initializeDuplicateProposal();
