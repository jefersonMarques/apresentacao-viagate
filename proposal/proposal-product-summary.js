const productIcons = [
  { pattern: /^Cargo Score\b/i, icon: 'shield-check' },
  { pattern: /^Consultas e autenticação\b/i, icon: 'scan-face' },
  { pattern: /^Cargo Truck\b/i, icon: 'route' },
  { pattern: /^Prevenção\b/i, icon: 'shield-alert' },
  { pattern: /^Monitoramento de veículos\b/i, icon: 'satellite' },
];

function resolveIcon(title) {
  return productIcons.find((entry) => entry.pattern.test(title))?.icon ?? 'box';
}

function appendStyles() {
  if (document.querySelector('style[data-proposal-product-summary]')) {
    return;
  }

  const style = document.createElement('style');
  style.dataset.proposalProductSummary = 'true';
  style.textContent = `
    .proposal-content-card.proposal-product-summary-card{min-height:210px;padding:26px;display:flex;flex-direction:column;justify-content:flex-start}.proposal-product-summary-icon{width:44px;height:44px;display:grid;place-items:center;background:#071827;color:#ff6b18}.dark .proposal-product-summary-icon{background:rgba(255,255,255,.08)}.proposal-product-summary-icon svg{width:20px;height:20px}.proposal-content-card.proposal-product-summary-card h3{margin:24px 0 0;font-size:21px}.proposal-content-card.proposal-product-summary-card p{margin:10px 0 0;color:#687985;font-size:12px;line-height:1.65}.dark .proposal-content-card.proposal-product-summary-card p{color:rgba(255,255,255,.62)}
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
      <span class="proposal-product-summary-icon"><i data-lucide="${resolveIcon(title)}"></i></span>
      <h3></h3>
      <p></p>
    `;
    card.querySelector('h3').textContent = title;
    card.querySelector('p').textContent = description;
    enhancedCount += 1;
  });

  if (enhancedCount > 0) {
    window.lucide?.createIcons();
  }

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
