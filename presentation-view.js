import { hasSupabaseConfiguration, supabase } from './proposal/supabase.js';

const token = new URLSearchParams(window.location.search).get('token');
const loadingElement = document.getElementById('presentationLoading');
const frame = document.getElementById('presentationFrame');
const startButton = document.getElementById('presentationStartButton');

const viewerState = {
  started: false,
  sessionId: null,
  content: null,
};

function getSessionId() {
  if (!token) return crypto.randomUUID();
  const key = `viagate:presentation:${token}:session`;
  let value = sessionStorage.getItem(key);
  if (!value) {
    value = crypto.randomUUID();
    sessionStorage.setItem(key, value);
  }
  return value;
}

async function track(eventName, slideNumber = null, slideTotal = null) {
  if (!token || !viewerState.sessionId || !supabase) return;

  try {
    await supabase.rpc('track_shared_document_event', {
      document_kind: 'presentation',
      document_token: token,
      event_name: eventName,
      viewer_session: viewerState.sessionId,
      slide_number: slideNumber,
      slide_total: slideTotal,
    });
  } catch {
  }
}

function showError(message) {
  loadingElement.innerHTML = `<div class="error">${message}</div>`;
}

function installPresentation(frameDocument, frameWindow) {
  if (!frameDocument || !frameWindow || frameDocument.querySelector('script[data-presentation-bootstrap]')) {
    return;
  }

  const content = viewerState.content ?? {};
  const salesperson = content.salesperson ?? {};

  frameWindow.presentationContact = Object.freeze({
    name: salesperson.name || 'ViaGate Comercial',
    role: salesperson.role || 'Equipe Comercial',
    email: salesperson.email || '',
    phone: salesperson.phone || '',
    whatsapp: salesperson.whatsapp || '',
    photoUrl: salesperson.photo_url || './assets/logo-viagate-color.svg',
  });

  frameWindow.presentationSettings = Object.freeze({
    showContactSlide: content.settings?.show_contact_slide !== false,
    showClientIdentity: Boolean(content.settings?.show_client_identity),
    client: content.client ?? {},
  });

  const bootstrapScript = frameDocument.createElement('script');
  bootstrapScript.src = './presentation-bootstrap.js?v=20260825-5';
  bootstrapScript.dataset.presentationBootstrap = 'true';
  frameDocument.body.appendChild(bootstrapScript);
}

function pausePresentation() {
  document.body.classList.add('is-paused');
  startButton.textContent = viewerState.started ? 'CONTINUAR APRESENTAÇÃO' : 'INICIAR APRESENTAÇÃO';
  frame.contentWindow?.hostPausePresentation?.();
}

async function waitForPresentationBridge(attempt = 0) {
  if (typeof frame.contentWindow?.hostStartPresentation === 'function') {
    return true;
  }

  if (attempt >= 60) {
    return false;
  }

  await new Promise((resolve) => window.setTimeout(resolve, 50));
  return waitForPresentationBridge(attempt + 1);
}

async function startPresentation() {
  try {
    if (!document.fullscreenElement) {
      await document.documentElement.requestFullscreen();
    }

    const bridgeReady = await waitForPresentationBridge();
    if (!bridgeReady) {
      return;
    }

    const firstStart = !viewerState.started;
    viewerState.started = true;
    document.body.classList.remove('is-paused');
    frame.contentWindow.hostStartPresentation(firstStart);
    await track('start');
  } catch {
  }
}

async function initializeViewer() {
  if (!hasSupabaseConfiguration() || !supabase) {
    showError('Apresentação indisponível: Supabase não configurado.');
    return;
  }

  if (!token) {
    showError('Link de apresentação inválido.');
    return;
  }

  viewerState.sessionId = getSessionId();

  const { data, error } = await supabase.rpc('get_public_presentation', {
    presentation_token: token,
  });

  if (error || !data) {
    showError('Apresentação não encontrada ou indisponível.');
    return;
  }

  viewerState.content = data.version?.content ?? {};
  const clientName = viewerState.content.client?.company_name;
  document.title = clientName ? `ViaGate — Apresentação para ${clientName}` : 'ViaGate — Apresentação';

  frame.addEventListener('load', () => {
    installPresentation(frame.contentDocument, frame.contentWindow);
  }, { once: true });

  frame.src = './presentation-content.html?v=20260825-5';
  frame.hidden = false;
  loadingElement.hidden = true;

  await track('open');
}

startButton.addEventListener('click', startPresentation);

document.addEventListener('fullscreenchange', () => {
  if (document.fullscreenElement) {
    document.body.classList.remove('is-paused');
    return;
  }

  if (viewerState.started) {
    pausePresentation();
  }
});

window.addEventListener('message', (event) => {
  if (event.origin !== window.location.origin || event.data?.source !== 'viagate-presentation') {
    return;
  }

  const slideNumber = Number(event.data.slideNumber || 0) || null;
  const slideTotal = Number(event.data.slideTotal || 0) || null;

  if (event.data.type === 'slide_view' && slideNumber && slideTotal) {
    track('slide_view', slideNumber, slideTotal);
  }

  if (event.data.type === 'complete' && slideNumber && slideTotal) {
    track('complete', slideNumber, slideTotal);
  }
});

initializeViewer();
