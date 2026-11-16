(() => {
  if (!window.htmx) {
    return;
  }

  window.htmx.config.historyCacheSize = 0;
  window.htmx.config.refreshOnHistoryMiss = true;
  window.htmx.config.selfRequestsOnly = true;

  document.body.addEventListener('htmx:beforeSwap', (event) => {
    const responseURL = event.detail.xhr.responseURL;

    if (!responseURL) {
      return;
    }

    const destination = new URL(responseURL, window.location.origin);

    if (destination.origin !== window.location.origin || destination.pathname !== '/login') {
      return;
    }

    event.detail.shouldSwap = false;
    window.location.assign(destination.href);
  });
})();
