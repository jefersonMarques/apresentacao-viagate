function safePresentationSocialUrl(value) {
  if (!value) {
    return '';
  }

  try {
    const url = new URL(value);
    return ['http:', 'https:'].includes(url.protocol) ? url.toString() : '';
  } catch {
    return '';
  }
}

function appendPresentationSocialStyles() {
  if (document.querySelector('style[data-presentation-social-links]')) {
    return;
  }

  const style = document.createElement('style');
  style.dataset.presentationSocialLinks = 'true';
  style.textContent = `
    .presentation-social-links{display:flex;gap:10px;flex-wrap:wrap;margin-top:14px}.presentation-social-links a{display:inline-flex;align-items:center;min-height:34px;padding:0 11px;border:1px solid rgba(255,255,255,.22);color:#fff;text-decoration:none;font-size:9px;font-weight:800;letter-spacing:.08em;text-transform:uppercase}.presentation-social-links a:hover{background:rgba(255,255,255,.08)}
  `;
  document.head.appendChild(style);
}

function initializePresentationSocialLinks() {
  const contact = window.presentationContact ?? {};
  const linkedin = safePresentationSocialUrl(contact.linkedin);
  const instagram = safePresentationSocialUrl(contact.instagram);
  const details = document.querySelector('.presentation-contact-details');

  if ((!linkedin && !instagram) || !details || details.parentElement?.querySelector('.presentation-social-links')) {
    return;
  }

  appendPresentationSocialStyles();

  const links = document.createElement('div');
  links.className = 'presentation-social-links';
  links.innerHTML = `
    ${linkedin ? `<a href="${linkedin}" target="_blank" rel="noopener noreferrer">LinkedIn</a>` : ''}
    ${instagram ? `<a href="${instagram}" target="_blank" rel="noopener noreferrer">Instagram</a>` : ''}
  `;

  details.insertAdjacentElement('afterend', links);
}

initializePresentationSocialLinks();
