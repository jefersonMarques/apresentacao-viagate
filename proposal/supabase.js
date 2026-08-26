import { createClient } from 'https://cdn.jsdelivr.net/npm/@supabase/supabase-js@2/+esm';
import { proposalConfig } from './config.js';

export function hasSupabaseConfiguration() {
  return Boolean(proposalConfig.supabaseUrl && proposalConfig.supabasePublishableKey);
}

export const supabase = hasSupabaseConfiguration()
  ? createClient(proposalConfig.supabaseUrl, proposalConfig.supabasePublishableKey, {
      auth: {
        persistSession: true,
        autoRefreshToken: true,
        detectSessionInUrl: true,
      },
    })
  : null;

export function formatCurrency(value) {
  const number = Number(value ?? 0);

  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
    minimumFractionDigits: 2,
  }).format(Number.isFinite(number) ? number : 0);
}

export function formatDate(value) {
  if (!value) {
    return '—';
  }

  const date = new Date(`${value}T12:00:00`);
  return new Intl.DateTimeFormat('pt-BR').format(date);
}

function buildPublicUrl(baseUrl, token) {
  const moduleUrl = new URL(import.meta.url);
  const url = new URL(baseUrl, moduleUrl);
  url.searchParams.set('token', token);
  return url.toString();
}

export function buildPublicProposalUrl(token) {
  return buildPublicUrl(proposalConfig.publicProposalUrl, token);
}

export function buildPublicPresentationUrl(token) {
  return buildPublicUrl(proposalConfig.publicPresentationUrl, token);
}

if (document.getElementById('adminView')) {
  document.title = 'ViaGate — Hub Comercial';

  const brandEyebrow = document.querySelector('.auth-brand-copy small');
  const brandTitle = document.querySelector('.auth-brand-copy h1');
  const brandCopy = document.querySelector('.auth-brand-copy p');
  const brandFooter = document.querySelector('.auth-brand-footer');

  if (brandEyebrow) brandEyebrow.textContent = 'HUB COMERCIAL';
  if (brandTitle) brandTitle.textContent = 'Apresentações e propostas em um único lugar.';
  if (brandCopy) brandCopy.textContent = 'Gere materiais personalizados, publique links individuais e acompanhe abertura e leitura.';
  if (brandFooter) brandFooter.textContent = 'ViaGate · Hub Comercial · 2026';

  import('./commercial-hub.js?v=20260825-8').catch((error) => {
    console.error('Não foi possível carregar o hub comercial.', error);
  });

  import('./storage-ui.js?v=20260825-7').catch((error) => {
    console.error('Não foi possível carregar os uploads do Storage.', error);
  });

  import('./profile-socials.js?v=20260825-7').catch((error) => {
    console.error('Não foi possível carregar as redes sociais do perfil.', error);
  });

  import('./duplicate-proposal.js?v=20260825-7').catch((error) => {
    console.error('Não foi possível carregar a duplicação de propostas.', error);
  });

  import('./proposal-access-admin.js?v=20260825-1').catch((error) => {
    console.error('Não foi possível carregar o controle de acesso das propostas.', error);
  });

  import('./hub-overview-enhancements.js?v=20260825-1').catch((error) => {
    console.error('Não foi possível identificar os tipos de materiais no painel.', error);
  });
}

if (document.getElementById('proposalPresentation')) {
  import('./analytics.js?v=20260825-8').catch((error) => {
    console.error('Não foi possível carregar as estatísticas da proposta.', error);
  });

  import('./social-links.js?v=20260825-7').catch((error) => {
    console.error('Não foi possível carregar as redes sociais do comercial.', error);
  });
}
