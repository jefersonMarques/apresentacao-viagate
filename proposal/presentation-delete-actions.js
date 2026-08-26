import { supabase } from './supabase.js';

const trashIcon = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16"/><path d="M9 7V4h6v3"/><path d="m6 7 1 13h10l1-13"/><path d="M10 11v5M14 11v5"/></svg>';

const state = {
  processing: new Set(),
  observed: false,
};

function getPresentationName(row) {
  return row.querySelector('.hub-document-main strong, strong')?.textContent?.trim() || 'esta apresentação';
}

async function deletePresentation(presentationId, row, button) {
  if (!presentationId || state.processing.has(presentationId)) {
    return;
  }

  const confirmed = window.confirm(`Excluir definitivamente ${getPresentationName(row)}?`);
  if (!confirmed) {
    return;
  }

  state.processing.add(presentationId);
  button.disabled = true;

  try {
    const { data, error } = await supabase.rpc('delete_presentation', {
      target_presentation_id: presentationId,
    });

    if (error || data !== true) {
      throw error ?? new Error('Presentation was not deleted');
    }

    row.remove();
  } catch (error) {
    console.error(error);
    window.alert('Não foi possível excluir a apresentação. Verifique se a migration de exclusão foi aplicada no Supabase.');
    button.disabled = false;
  } finally {
    state.processing.delete(presentationId);
  }
}

function enhancePresentationActions() {
  const list = document.getElementById('presentationList');
  if (!list) {
    return false;
  }

  list.querySelectorAll('.hub-document-row, .hub-document-item, article').forEach((row) => {
    const editButton = row.querySelector('[data-open-presentation]');
    const presentationId = editButton?.dataset.openPresentation ?? '';
    const host = editButton?.closest('.hub-row-actions');

    if (!presentationId || !host || host.querySelector('[data-delete-presentation]')) {
      return;
    }

    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'proposal-list-action danger';
    button.dataset.deletePresentation = presentationId;
    button.title = 'Excluir apresentação';
    button.setAttribute('aria-label', 'Excluir apresentação');
    button.innerHTML = `${trashIcon}<span>Excluir</span>`;
    button.addEventListener('click', (event) => {
      event.preventDefault();
      event.stopPropagation();
      deletePresentation(presentationId, row, button);
    });

    host.appendChild(button);
  });

  if (!state.observed) {
    state.observed = true;
    const observer = new MutationObserver(enhancePresentationActions);
    observer.observe(list, {
      childList: true,
      subtree: true,
    });
  }

  return true;
}

function initialize() {
  if (enhancePresentationActions()) {
    return;
  }

  const observer = new MutationObserver(() => {
    if (enhancePresentationActions()) {
      observer.disconnect();
    }
  });

  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });

  window.setTimeout(() => observer.disconnect(), 15000);
}

initialize();
