function appendResponsiveStylesheet() {
  if (document.querySelector('link[data-hub-responsive]')) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = './hub-responsive.css?v=20260825-1';
  link.dataset.hubResponsive = 'true';

  const hubStylesheet = document.querySelector('link[data-commercial-hub]');

  if (hubStylesheet) {
    hubStylesheet.insertAdjacentElement('afterend', link);
    return;
  }

  document.head.appendChild(link);
}

function initializeResponsiveLayer() {
  if (document.querySelector('link[data-commercial-hub]')) {
    appendResponsiveStylesheet();
    return;
  }

  const observer = new MutationObserver(() => {
    if (!document.querySelector('link[data-commercial-hub]')) {
      return;
    }

    observer.disconnect();
    appendResponsiveStylesheet();
  });

  observer.observe(document.head, {
    childList: true,
  });
}

initializeResponsiveLayer();
