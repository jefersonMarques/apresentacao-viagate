function appendStylesheet(href, dataAttribute) {
  if (document.querySelector(`link[${dataAttribute}]`)) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = href;
  link.setAttribute(dataAttribute, 'true');
  document.head.appendChild(link);
}

function appendScript(src, dataAttribute) {
  return new Promise((resolve) => {
    if (document.querySelector(`script[${dataAttribute}]`)) {
      resolve();
      return;
    }

    const script = document.createElement('script');
    script.src = src;
    script.setAttribute(dataAttribute, 'true');
    script.addEventListener('load', resolve, { once: true });
    document.body.appendChild(script);
  });
}

async function loadPresentationMode(attempt = 0) {
  const executiveReady = Boolean(document.querySelector('.exec-analysis-metrics-slide'));
  const reachedTimeout = attempt >= 100;

  if (!executiveReady && !reachedTimeout) {
    window.setTimeout(() => loadPresentationMode(attempt + 1), 50);
    return;
  }

  appendStylesheet('./presentation-story.css?v=20260825-4', 'data-presentation-story');
  await appendScript('./presentation-story.js?v=20260825-4', 'data-presentation-story');
  await appendScript('./presentation-contact.js?v=20260825-4', 'data-presentation-contact');

  appendStylesheet('./presentation-mode.css?v=20260825-4', 'data-presentation-mode');
  await appendScript('./presentation-mode.js?v=20260825-4', 'data-presentation-mode');

  if (typeof window.initializePresentationMode === 'function') {
    window.initializePresentationMode();
  }

  appendStylesheet('./presentation-navigation.css?v=20260825-4', 'data-presentation-navigation-style');
  await appendScript('./presentation-navigation.js?v=20260825-4', 'data-presentation-navigation');

  await appendScript('./presentation-host-bridge.js?v=20260825-4', 'data-presentation-host-bridge');
  document.querySelector('.presentation-gate')?.remove();
}

loadPresentationMode();
