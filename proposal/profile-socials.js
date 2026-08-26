import { supabase } from './supabase.js';

const SOCIAL_FIELDS = [
  {
    id: 'salesLinkedIn',
    label: 'LinkedIn',
    column: 'linkedin_url',
    placeholder: 'https://www.linkedin.com/in/...',
  },
  {
    id: 'salesInstagram',
    label: 'Instagram',
    column: 'instagram_url',
    placeholder: 'https://www.instagram.com/...',
  },
];

function createSocialFields() {
  const photoField = document.getElementById('salesPhotoUrl')?.closest('.form-field');

  if (!photoField || document.getElementById('salesLinkedIn')) {
    return;
  }

  SOCIAL_FIELDS.forEach((field) => {
    const wrapper = document.createElement('div');
    wrapper.className = 'form-field full';
    wrapper.innerHTML = `
      <label for="${field.id}">${field.label}</label>
      <input id="${field.id}" type="url" placeholder="${field.placeholder}" autocomplete="url" />
    `;
    photoField.insertAdjacentElement('beforebegin', wrapper);
  });
}

async function getAuthenticatedUser() {
  const { data } = await supabase.auth.getUser();
  return data?.user ?? null;
}

async function loadSocialProfile() {
  const user = await getAuthenticatedUser();
  if (!user) {
    return;
  }

  const { data, error } = await supabase
    .from('salespeople')
    .select('linkedin_url,instagram_url')
    .eq('auth_user_id', user.id)
    .maybeSingle();

  if (error) {
    console.warn('Campos de redes sociais ainda não estão disponíveis no Supabase.', error.message);
    return;
  }

  document.getElementById('salesLinkedIn').value = data?.linkedin_url ?? '';
  document.getElementById('salesInstagram').value = data?.instagram_url ?? '';
}

async function waitForSalesProfile(userId, attempts = 20) {
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    const { data, error } = await supabase
      .from('salespeople')
      .select('id')
      .eq('auth_user_id', userId)
      .maybeSingle();

    if (!error && data?.id) {
      return true;
    }

    await new Promise((resolve) => window.setTimeout(resolve, 150));
  }

  return false;
}

async function saveSocialProfile() {
  const user = await getAuthenticatedUser();
  if (!user) {
    return;
  }

  const profileReady = await waitForSalesProfile(user.id);
  if (!profileReady) {
    return;
  }

  const payload = {
    linkedin_url: document.getElementById('salesLinkedIn')?.value?.trim() || null,
    instagram_url: document.getElementById('salesInstagram')?.value?.trim() || null,
  };

  const { error } = await supabase
    .from('salespeople')
    .update(payload)
    .eq('auth_user_id', user.id);

  if (error) {
    console.error('Não foi possível salvar as redes sociais do perfil.', error);
  }
}

function initializeProfileSocials() {
  createSocialFields();
  loadSocialProfile().catch(console.error);

  document.getElementById('salesProfileForm')?.addEventListener('submit', () => {
    window.setTimeout(() => saveSocialProfile().catch(console.error), 0);
  });

  supabase.auth.onAuthStateChange((_event, session) => {
    if (session?.user) {
      window.setTimeout(() => loadSocialProfile().catch(console.error), 100);
    }
  });
}

initializeProfileSocials();
