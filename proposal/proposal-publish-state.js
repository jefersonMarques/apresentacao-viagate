const publishTargets = [
  {
    buttonId: 'publishProposalButton',
    messageId: 'proposalMessage',
    progressLabel: 'Publicando proposta...',
    terminalPatterns: [/publicad/i, /não foi possível publicar/i, /não foi possível salvar/i],
  },
  {
    buttonId: 'publishPresentationButton',
    messageId: 'presentationMessage',
    progressLabel: 'Publicando apresentação...',
    terminalPatterns: [/publicad/i, /não foi possível publicar/i, /não foi possível salvar/i],
  },
];

const state = new WeakMap();

function appendStyles() {
  if (document.querySelector('style[data-publish-state]')) {
    return;
  }

  const style = document.createElement('style');
  style.dataset.publishState = 'true';
  style.textContent = `
    .publish-processing {
      cursor: wait !important;
      opacity: .82;
    }

    .publish-processing-indicator {
      width: 14px;
      height: 14px;
      flex: 0 0 14px;
      border: 2px solid currentColor;
      border-right-color: transparent;
      border-radius: 50%;
      animation: publish-processing-spin .7s linear infinite;
    }

    .publish-processing-content {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
    }

    @keyframes publish-processing-spin {
      to { transform: rotate(360deg); }
    }
  `;
  document.head.appendChild(style);
}

function restoreButton(button) {
  const current = state.get(button);
  if (!current) {
    return;
  }

  window.clearTimeout(current.timeoutId);
  current.observer?.disconnect();
  button.innerHTML = current.originalHtml;
  button.disabled = current.originalDisabled;
  button.classList.remove('publish-processing');
  button.removeAttribute('aria-busy');
  state.delete(button);
}

function isTerminalMessage(message, patterns) {
  const text = message?.textContent?.trim() ?? '';
  return Boolean(text) && patterns.some((pattern) => pattern.test(text));
}

function startProcessing(target) {
  const button = document.getElementById(target.buttonId);
  const message = document.getElementById(target.messageId);

  if (!button || state.has(button)) {
    return;
  }

  const current = {
    originalHtml: button.innerHTML,
    originalDisabled: button.disabled,
    observer: null,
    timeoutId: null,
  };

  button.classList.add('publish-processing');
  button.setAttribute('aria-busy', 'true');
  button.disabled = true;
  button.innerHTML = `
    <span class="publish-processing-content">
      <span class="publish-processing-indicator" aria-hidden="true"></span>
      <span>${target.progressLabel}</span>
    </span>
  `;

  if (message) {
    current.observer = new MutationObserver(() => {
      if (isTerminalMessage(message, target.terminalPatterns)) {
        restoreButton(button);
      }
    });

    current.observer.observe(message, {
      attributes: true,
      childList: true,
      characterData: true,
      subtree: true,
      attributeFilter: ['hidden', 'class'],
    });
  }

  current.timeoutId = window.setTimeout(() => restoreButton(button), 30000);
  state.set(button, current);
}

function bindTarget(target) {
  const button = document.getElementById(target.buttonId);
  if (!button || button.dataset.publishStateBound === 'true') {
    return Boolean(button);
  }

  button.dataset.publishStateBound = 'true';
  button.addEventListener('click', () => {
    window.setTimeout(() => startProcessing(target), 0);
  });
  return true;
}

function initialize() {
  appendStyles();

  const bindAll = () => {
    publishTargets.forEach(bindTarget);
  };

  bindAll();

  const observer = new MutationObserver(bindAll);
  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });

  window.setTimeout(() => observer.disconnect(), 15000);
}

initialize();
