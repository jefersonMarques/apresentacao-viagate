(() => {
  if (!window.htmx) return;

  // Admin history snapshots may contain customer/commercial data. Keep browser
  // history navigation, but always restore the current server representation.
  window.htmx.config.historyCacheSize = 0;
  window.htmx.config.refreshOnHistoryMiss = true;
  window.htmx.config.selfRequestsOnly = true;
})();
