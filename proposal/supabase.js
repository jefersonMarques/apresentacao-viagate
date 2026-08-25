import { createClient } from 'https://cdn.jsdelivr.net/npm/@supabase/supabase-js@2/+esm';
import { proposalConfig } from './config.js';

export function hasSupabaseConfiguration() {
  return Boolean(proposalConfig.supabaseUrl && proposalConfig.supabaseAnonKey);
}

export const supabase = hasSupabaseConfiguration()
  ? createClient(proposalConfig.supabaseUrl, proposalConfig.supabaseAnonKey, {
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

export function buildPublicProposalUrl(token) {
  const url = new URL(proposalConfig.publicProposalUrl, window.location.origin);
  url.searchParams.set('token', token);
  return url.toString();
}
