function escapePersonalizationText(value) {
  const element = document.createElement('span');
  element.textContent = String(value ?? '');
  return element.innerHTML;
}

function applyPresentationPersonalization() {
  const settings = window.presentationSettings;
  if (!settings) {
    return;
  }

  if (settings.showContactSlide === false) {
    document.querySelector('.presentation-contact-slide')?.remove();
    if (typeof renumberPresentationSlides === 'function') {
      renumberPresentationSlides();
    }
  }

  if (!settings.showClientIdentity || !settings.client?.company_name) {
    return;
  }

  const firstSlide = getPresentationSlides?.()[0];
  if (!firstSlide || firstSlide.querySelector('.presentation-client-context')) {
    return;
  }

  const companyName = escapePersonalizationText(settings.client.company_name);
  const contactName = escapePersonalizationText(settings.client.contact_name || '');
  const logoUrl = escapePersonalizationText(settings.client.logo_url || '');
  const context = document.createElement('div');
  context.className = 'presentation-client-context';
  context.innerHTML = `
    <small>APRESENTAÇÃO PREPARADA PARA</small>
    ${logoUrl ? `<img src="${logoUrl}" alt="${companyName}" />` : ''}
    <strong>${companyName}</strong>
    ${contactName ? `<span>${contactName}</span>` : ''}
  `;
  firstSlide.appendChild(context);

  if (!document.querySelector('style[data-presentation-client-context]')) {
    const style = document.createElement('style');
    style.dataset.presentationClientContext = 'true';
    style.textContent = `
      .presentation-client-context{position:absolute;right:38px;bottom:46px;z-index:8;max-width:260px;padding:14px 16px;border:1px solid rgba(255,255,255,.14);background:rgba(7,24,39,.72);backdrop-filter:blur(12px);text-align:right}.presentation-client-context small,.presentation-client-context strong,.presentation-client-context span{display:block}.presentation-client-context small{color:rgba(255,255,255,.48);font-size:8px;font-weight:800;letter-spacing:.12em}.presentation-client-context strong{margin-top:6px;color:#fff;font-size:14px}.presentation-client-context span{margin-top:3px;color:rgba(255,255,255,.62);font-size:10px}.presentation-client-context img{display:block;max-width:110px;max-height:32px;margin:8px 0 6px auto;object-fit:contain;filter:brightness(0) invert(1);opacity:.86}
    `;
    document.head.appendChild(style);
  }
}

applyPresentationPersonalization();
