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

  appendStylesheet('./presentation-story.css', 'data-presentation-story');
  await appendScript('./presentation-story.js', 'data-presentation-story');
  await appendScript('./presentation-contact.js', 'data-presentation-contact');

  appendStylesheet('./presentation-mode.css', 'data-presentation-mode');
  await appendScript('./presentation-mode.js', 'data-presentation-mode');

  if (typeof window.initializePresentationMode === 'function') {
    window.initializePresentationMode();
  }
}

loadPresentationMode();
