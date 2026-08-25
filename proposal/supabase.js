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
  const url = new URL(baseUrl, window.location.origin);
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
  import('./commercial-hub.js').catch((error) => {
    console.error('Não foi possível carregar o hub comercial.', error);
  });
}

if (document.getElementById('proposalPresentation')) {
  import('./analytics.js').catch((error) => {
    console.error('Não foi possível carregar as estatísticas da proposta.', error);
  });
}
