function loadPresentationMode(attempt = 0) {
  const executiveReady = Boolean(document.querySelector('.exec-analysis-metrics-slide'));
  const reachedTimeout = attempt >= 100;

  if (!executiveReady && !reachedTimeout) {
    window.setTimeout(() => loadPresentationMode(attempt + 1), 50);
    return;
  }

  if (!document.querySelector('link[data-presentation-mode]')) {
    const styleLink = document.createElement('link');
    styleLink.rel = 'stylesheet';
    styleLink.href = './presentation-mode.css';
    styleLink.dataset.presentationMode = 'true';
    document.head.appendChild(styleLink);
  }

  if (document.querySelector('script[data-presentation-mode]')) {
    return;
  }

  const presentationScript = document.createElement('script');
  presentationScript.src = './presentation-mode.js';
  presentationScript.dataset.presentationMode = 'true';
  presentationScript.addEventListener('load', () => {
    if (typeof window.initializePresentationMode === 'function') {
      window.initializePresentationMode();
    }
  });
  document.body.appendChild(presentationScript);
}

loadPresentationMode();
