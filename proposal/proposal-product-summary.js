const productIcons = [
  { pattern: /^Cargo Score\b/i, icon: 'shield-check' },
  { pattern: /^Consultas e autenticação\b/i, icon: 'scan-face' },
  { pattern: /^Cargo Truck\b/i, icon: 'route' },
  { pattern: /^Prevenção\b/i, icon: 'shield-alert' },
  { pattern: /^Monitoramento de veículos\b/i, icon: 'satellite' },
];

const iconMarkup = {
  'shield-check': '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 5 6v5c0 4.6 2.8 8 7 10 4.2-2 7-5.4 7-10V6l-7-3Z"/><path d="m9 12 2 2 4-4"/></svg>',
  'scan-face': '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 3H4a1 1 0 0 0-1 1v3M17 3h3a1 1 0 0 1 1 1v3M7 21H4a1 1 0 0 1-1-1v-3M17 21h3a1 1 0 0 0 1-1v-3"/><circle cx="12" cy="10" r="3"/><path d="M7.5 18c1-2 2.5-3 4.5-3s3.5 1 4.5 3"/></svg>',
  route: '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="6" cy="18" r="2"/><circle cx="18" cy="6" r="2"/><path d="M8 18h3a3 3 0 0 0 3-3V9a3 3 0 0 1 3-3"/></svg>',
  'shield-alert': '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 5 6v5c0 4.6 2.8 8 7 10 4.2-2 7-5.4 7-10V6l-7-3Z"/><path d="M12 8v5M12 16h.01"/></svg>',
  satellite: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M9 5l3 3-4 4-3-3 4-4ZM15 12l4 4-3 3-4-4 3-3ZM5 19c0-3.3 2.7-6 6-6M3 19c0-4.4 3.6-8 8-8"/></svg>',
  box: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m4 7 8-4 8 4-8 4-8-4Z"/><path d="m4 7 8 4 8-4v10l-8 4-8-4V7Z"/></svg>',
};

function resolveIcon(title) {
  const icon = productIcons.find((entry) => entry.pattern.test(title))?.icon ?? 'box';
  return iconMarkup[icon] ?? iconMarkup.box;
}

function appendStyles() {
  if (document.querySelector('style[data-proposal-product-summary]')) {
    return;
  }

  const style = document.createElement('style');
  style.dataset.proposalProductSummary = 'true';
  style.textContent = `
    .proposal-content-card.proposal-product-summary-card{min-height:210px;padding:26px;display:flex;flex-direction:column;justify-content:flex-start}.proposal-product-summary-icon{width:44px;height:44px;display:grid;place-items:center;background:#071827;color:#ff6b18}.dark .proposal-product-summary-icon{background:rgba(255,255,255,.08)}.proposal-product-summary-icon svg{width:20px;height:20px;fill:none;stroke:currentColor;stroke-width:1.7;stroke-linecap:round;stroke-linejoin:round}.proposal-content-card.proposal-product-summary-card h3{margin:24px 0 0;font-size:21px}.proposal-content-card.proposal-product-summary-card p{margin:10px 0 0;color:#687985;font-size:12px;line-height:1.65}.dark .proposal-content-card.proposal-product-summary-card p{color:rgba(255,255,255,.62)}
  `;
  document.head.appendChild(style);
}

function enhanceCards() {
  let enhancedCount = 0;

  document.querySelectorAll('.proposal-content-card h3').forEach((heading) => {
    const card = heading.closest('.proposal-content-card');
    const text = heading.textContent?.trim() ?? '';

    if (!card || card.dataset.productSummaryEnhanced === 'true' || !text.includes(' | ')) {
      return;
    }

    const [title, ...descriptionParts] = text.split(' | ');
    const description = descriptionParts.join(' | ').trim();

    if (!title || !description) {
      return;
    }

    card.dataset.productSummaryEnhanced = 'true';
    card.classList.add('proposal-product-summary-card');
    card.innerHTML = `
      <span class="proposal-product-summary-icon">${resolveIcon(title)}</span>
      <h3></h3>
      <p></p>
    `;
    card.querySelector('h3').textContent = title;
    card.querySelector('p').textContent = description;
    enhancedCount += 1;
  });

  return enhancedCount;
}

function initialize() {
  appendStyles();

  if (enhanceCards() > 0) {
    return;
  }

  const presentation = document.getElementById('proposalPresentation');
  if (!presentation) {
    return;
  }

  const observer = new MutationObserver(() => {
    if (enhanceCards() > 0 || presentation.querySelector('[data-proposal-slide]')) {
      observer.disconnect();
    }
  });

  observer.observe(presentation, {
    childList: true,
    subtree: true,
  });

  window.setTimeout(() => observer.disconnect(), 5000);
}

initialize();
