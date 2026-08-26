import { supabase } from './supabase.js';

function safeExternalUrl(value) {
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

function appendStyles() {
  if (document.querySelector('style[data-proposal-social-links]')) {
    return;
  }

  const style = document.createElement('style');
  style.dataset.proposalSocialLinks = 'true';
  style.textContent = `
    .proposal-social-links{display:flex;gap:10px;flex-wrap:wrap;margin-top:14px}.proposal-social-links a{display:inline-flex;align-items:center;min-height:34px;padding:0 11px;border:1px solid rgba(255,255,255,.25);color:#fff;text-decoration:none;font-size:10px;font-weight:800;letter-spacing:.08em;text-transform:uppercase}.proposal-social-links a:hover{background:rgba(255,255,255,.1)}
  `;
  document.head.appendChild(style);
}

async function waitForContactCard(attempts = 40) {
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    const card = document.querySelector('.proposal-contact-data');
    if (card) {
      return card;
    }

    await new Promise((resolve) => window.setTimeout(resolve, 100));
  }

  return null;
}

async function initializeProposalSocialLinks() {
  const token = new URLSearchParams(window.location.search).get('token');
  if (!token || !supabase) {
    return;
  }

  const { data, error } = await supabase.rpc('get_public_proposal', {
    proposal_token: token,
  });

  if (error || !data) {
    return;
  }

  const salesperson = data.version?.content?.salesperson ?? {};
  const linkedin = safeExternalUrl(salesperson.linkedin);
  const instagram = safeExternalUrl(salesperson.instagram);

  if (!linkedin && !instagram) {
    return;
  }

  const card = await waitForContactCard();
  if (!card || card.querySelector('.proposal-social-links')) {
    return;
  }

  appendStyles();

  const links = document.createElement('div');
  links.className = 'proposal-social-links';
  links.innerHTML = `
    ${linkedin ? `<a href="${linkedin}" target="_blank" rel="noopener noreferrer">LinkedIn</a>` : ''}
    ${instagram ? `<a href="${instagram}" target="_blank" rel="noopener noreferrer">Instagram</a>` : ''}
  `;

  card.appendChild(links);
}

initializeProposalSocialLinks().catch(console.error);
