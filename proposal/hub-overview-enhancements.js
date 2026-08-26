function getKindFromRow(row) {
  const meta = row.querySelector('small')?.textContent ?? '';

  if (/^Apresentação\b/i.test(meta)) {
    return 'presentation';
  }

  if (/^Proposta\b/i.test(meta)) {
    return 'proposal';
  }

  return '';
}

function getKindLabel(kind) {
  return kind === 'proposal' ? 'PROPOSTA' : 'APRESENTAÇÃO';
}

function enhanceTrackingRows() {
  document.querySelectorAll('.hub-tracking-row').forEach((row) => {
    if (row.dataset.kindEnhanced === 'true') {
      return;
    }

    const kind = getKindFromRow(row);
    if (!kind) {
      return;
    }

    row.dataset.kindEnhanced = 'true';
    row.dataset.documentKind = kind;

    const main = row.firstElementChild;
    if (main) {
      const badge = document.createElement('span');
      badge.className = `hub-kind-badge ${kind}`;
      badge.textContent = getKindLabel(kind);
      main.insertBefore(badge, main.firstChild);
    }

    const copyButton = row.querySelector('[data-copy-tracking-token]');
    if (copyButton) {
      copyButton.textContent = kind === 'proposal' ? 'Copiar proposta' : 'Copiar apresentação';
      copyButton.setAttribute(
        'aria-label',
        kind === 'proposal' ? 'Copiar link da proposta' : 'Copiar link da apresentação',
      );
    }
  });
}

function appendStyles() {
  if (document.querySelector('style[data-hub-kind-styles]')) {
    return;
  }

  const style = document.createElement('style');
  style.dataset.hubKindStyles = 'true';
  style.textContent = `
    .hub-kind-badge {
      width:max-content;
      min-height:22px;
      margin:0 0 7px !important;
      padding:0 7px;
      display:inline-flex !important;
      align-items:center;
      border:1px solid rgba(7,24,39,.15);
      font-size:8px !important;
      font-weight:900;
      letter-spacing:.12em;
      line-height:1;
    }
    .hub-kind-badge.presentation {
      border-color:rgba(48,91,126,.26);
      background:rgba(48,91,126,.08);
      color:#305b7e;
    }
    .hub-kind-badge.proposal {
      border-color:rgba(255,107,24,.34);
      background:rgba(255,107,24,.09);
      color:#c94e09;
    }
    .hub-tracking-row[data-document-kind="proposal"] { box-shadow:inset 3px 0 0 rgba(255,107,24,.72); }
    .hub-tracking-row[data-document-kind="presentation"] { box-shadow:inset 3px 0 0 rgba(48,91,126,.72); }
  `;
  document.head.appendChild(style);
}

function initialize() {
  appendStyles();
  enhanceTrackingRows();

  const table = document.getElementById('hubTrackingTable');
  if (!table) {
    return;
  }

  new MutationObserver(enhanceTrackingRows).observe(table, {
    childList: true,
    subtree: true,
  });
}

initialize();
