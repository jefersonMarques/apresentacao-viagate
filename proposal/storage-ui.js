import { proposalConfig } from './config.js';
import { supabase } from './supabase.js';

const MAX_FILE_SIZE = 2 * 1024 * 1024;
const ALLOWED_TYPES = new Set([
  'image/jpeg',
  'image/png',
  'image/webp',
  'image/svg+xml',
]);

const FIELD_CONFIG = [
  {
    sourceId: 'salesPhotoUrl',
    fileId: 'salesPhotoFile',
    category: 'salespeople',
    label: 'Foto do comercial',
    accept: 'image/jpeg,image/png,image/webp,image/svg+xml',
  },
  {
    sourceId: 'clientLogoUrl',
    fileId: 'clientLogoFile',
    category: 'clients',
    label: 'Logo do cliente',
    accept: 'image/jpeg,image/png,image/webp,image/svg+xml',
  },
  {
    sourceId: 'presentationClientLogo',
    fileId: 'presentationClientLogoFile',
    category: 'clients',
    label: 'Logo do cliente',
    accept: 'image/jpeg,image/png,image/webp,image/svg+xml',
  },
];

function getFileExtension(file) {
  const mimeExtensions = {
    'image/jpeg': 'jpg',
    'image/png': 'png',
    'image/webp': 'webp',
    'image/svg+xml': 'svg',
  };

  if (mimeExtensions[file.type]) {
    return mimeExtensions[file.type];
  }

  const extension = file.name.split('.').pop()?.toLowerCase();
  return extension?.replace(/[^a-z0-9]/g, '') || 'bin';
}

function validateFile(file) {
  if (!ALLOWED_TYPES.has(file.type)) {
    throw new Error('Formato inválido. Use PNG, JPG, WEBP ou SVG.');
  }

  if (file.size > MAX_FILE_SIZE) {
    throw new Error('A imagem deve ter no máximo 2 MB.');
  }
}

function getPublicUrl(path) {
  const { data } = supabase.storage
    .from(proposalConfig.assetBucket)
    .getPublicUrl(path);

  return data.publicUrl;
}

async function uploadFile(file, category) {
  validateFile(file);

  const { data: userData, error: userError } = await supabase.auth.getUser();
  const user = userData?.user;

  if (userError || !user) {
    throw new Error('Faça login novamente antes de enviar a imagem.');
  }

  const extension = getFileExtension(file);
  const path = `${category}/${user.id}/${crypto.randomUUID()}.${extension}`;
  const { error } = await supabase.storage
    .from(proposalConfig.assetBucket)
    .upload(path, file, {
      cacheControl: '3600',
      contentType: file.type,
      upsert: false,
    });

  if (error) {
    throw new Error(error.message || 'Não foi possível enviar a imagem.');
  }

  return {
    path,
    publicUrl: getPublicUrl(path),
  };
}

function setStatus(wrapper, message, isError = false) {
  const status = wrapper.querySelector('[data-storage-status]');
  if (!status) return;

  status.textContent = message;
  status.classList.toggle('error', isError);
}

function updatePreview(sourceInput, wrapper) {
  const preview = wrapper.querySelector('[data-storage-preview]');
  const image = wrapper.querySelector('[data-storage-image]');
  const removeButton = wrapper.querySelector('[data-storage-remove]');
  const url = sourceInput.value.trim();

  if (!preview || !image || !removeButton) return;

  if (!url) {
    preview.hidden = true;
    image.removeAttribute('src');
    removeButton.hidden = true;
    return;
  }

  if (image.getAttribute('src') !== url) {
    image.src = url;
  }

  preview.hidden = false;
  removeButton.hidden = false;
}

function enhanceStorageField(config) {
  const sourceInput = document.getElementById(config.sourceId);
  if (!sourceInput || sourceInput.dataset.storageEnhanced === 'true') {
    return false;
  }

  const field = sourceInput.closest('.form-field');
  if (!field) {
    return false;
  }

  sourceInput.dataset.storageEnhanced = 'true';
  sourceInput.type = 'hidden';

  const originalLabel = field.querySelector(`label[for="${config.sourceId}"]`);
  if (originalLabel) {
    originalLabel.htmlFor = config.fileId;
    originalLabel.textContent = config.label;
  }

  const wrapper = document.createElement('div');
  wrapper.className = 'storage-upload';
  wrapper.innerHTML = `
    <div class="storage-upload-row">
      <label class="storage-upload-button" for="${config.fileId}">Selecionar imagem</label>
      <input id="${config.fileId}" type="file" accept="${config.accept}" hidden />
      <button class="link-button storage-remove" type="button" data-storage-remove hidden>Remover</button>
    </div>
    <div class="storage-preview" data-storage-preview hidden>
      <img data-storage-image alt="Pré-visualização da imagem" />
    </div>
    <small class="storage-help">PNG, JPG, WEBP ou SVG · máximo 2 MB</small>
    <small class="storage-status" data-storage-status></small>
  `;

  sourceInput.insertAdjacentElement('afterend', wrapper);

  const fileInput = wrapper.querySelector(`#${config.fileId}`);
  const removeButton = wrapper.querySelector('[data-storage-remove]');

  fileInput?.addEventListener('change', async () => {
    const file = fileInput.files?.[0];
    if (!file) return;

    setStatus(wrapper, 'Enviando...');

    try {
      const uploaded = await uploadFile(file, config.category);
      sourceInput.value = uploaded.publicUrl;
      sourceInput.dataset.storagePath = uploaded.path;
      updatePreview(sourceInput, wrapper);
      setStatus(wrapper, 'Imagem enviada.');
    } catch (error) {
      setStatus(wrapper, error.message || 'Não foi possível enviar a imagem.', true);
    } finally {
      fileInput.value = '';
    }
  });

  removeButton?.addEventListener('click', () => {
    sourceInput.value = '';
    delete sourceInput.dataset.storagePath;
    updatePreview(sourceInput, wrapper);
    setStatus(wrapper, 'Imagem removida deste material.');
  });

  updatePreview(sourceInput, wrapper);
  return true;
}

function enhanceAvailableFields() {
  FIELD_CONFIG.forEach(enhanceStorageField);
}

function syncPreviews() {
  FIELD_CONFIG.forEach((config) => {
    const sourceInput = document.getElementById(config.sourceId);
    const wrapper = sourceInput?.parentElement?.querySelector('.storage-upload');
    if (sourceInput && wrapper) {
      updatePreview(sourceInput, wrapper);
    }
  });
}

function appendStyles() {
  if (document.querySelector('style[data-storage-ui]')) return;

  const style = document.createElement('style');
  style.dataset.storageUi = 'true';
  style.textContent = `
    .storage-upload{margin-top:2px}.storage-upload-row{display:flex;align-items:center;gap:12px;flex-wrap:wrap}.storage-upload-button{min-height:42px;padding:0 16px;border:1px solid rgba(7,24,39,.18);display:inline-flex;align-items:center;justify-content:center;background:#fff;color:#071827;cursor:pointer;font-size:12px;font-weight:700;letter-spacing:.04em}.storage-preview{margin-top:14px;width:160px;height:96px;border:1px solid rgba(7,24,39,.12);background:#f3f5f6;padding:10px}.storage-preview img{width:100%;height:100%;object-fit:contain;display:block}.storage-help,.storage-status{display:block;margin-top:8px;color:#70808d;font-size:11px;line-height:1.4}.storage-status{min-height:15px;color:#39735a}.storage-status.error{color:#b42318}.storage-remove{padding:0}.auth-shell .storage-upload-button,.admin-shell .storage-upload-button{box-sizing:border-box}
  `;
  document.head.appendChild(style);
}

function initializeStorageUi() {
  if (!supabase || !proposalConfig.assetBucket) return;

  appendStyles();
  enhanceAvailableFields();

  const observer = new MutationObserver(() => {
    enhanceAvailableFields();
    syncPreviews();
  });

  observer.observe(document.body, { childList: true, subtree: true });

  document.addEventListener('click', () => {
    window.setTimeout(syncPreviews, 100);
    window.setTimeout(syncPreviews, 500);
  }, { passive: true });

  supabase.auth.onAuthStateChange(() => {
    window.setTimeout(syncPreviews, 100);
    window.setTimeout(syncPreviews, 600);
  });
}

initializeStorageUi();
