import { hasSupabaseConfiguration, supabase } from './proposal/supabase.js';

const token = new URLSearchParams(window.location.search).get('token');
const loadingElement = document.getElementById('presentationLoading');
const frame = document.getElementById('presentationFrame');
const startButton = document.getElementById('presentationStartButton');
const restartButton = document.getElementById('presentationRestartButton');
const navigation = document.getElementById('presentationNavigation');
const previousButton = document.getElementById('presentationPreviousButton');
const nextButton = document.getElementById('presentationNextButton');
const counter = document.getElementById('presentationCounter');

const viewerState = {
  started: false,
  sessionId: null,
  content: null,
  frameLoaded: false,
  bridgeReady: false,
  slideNumber: 1,
  slideTotal: 1,
};

function getSessionId() {
  if (!token) {
    return crypto.randomUUID();
  }

  const key = `viagate:presentation:${token}:session`;
  let value = sessionStorage.getItem(key);

  if (!value) {
    value = crypto.randomUUID();
    sessionStorage.setItem(key, value);
  }

  return value;
}

async function track(eventName, slideNumber = null, slideTotal = null) {
  if (!token || !viewerState.sessionId || !supabase) {
    return;
  }

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
  loadingElement.hidden = false;
  loadingElement.innerHTML = `<div class="error">${message}</div>`;
}

function updateNavigation(slideNumber = viewerState.slideNumber, slideTotal = viewerState.slideTotal) {
  viewerState.slideNumber = Math.max(1, Number(slideNumber) || 1);
  viewerState.slideTotal = Math.max(1, Number(slideTotal) || 1);

  counter.textContent = `${String(viewerState.slideNumber).padStart(2, '0')} / ${String(viewerState.slideTotal).padStart(2, '0')}`;
  previousButton.disabled = viewerState.slideNumber <= 1;
  nextButton.disabled = viewerState.slideNumber >= viewerState.slideTotal;
}

function installPresentation(frameDocument, frameWindow) {
  if (!frameDocument || !frameWindow || frameDocument.querySelector('script[data-presentation-bootstrap]')) {
    return;
  }

  const content = viewerState.content ?? {};
  const salesperson = content.salesperson ?? {};

  frameWindow.presentationContact = Object.freeze({
    name: salesperson.name || 'ViaGate Comercial',
    role: salesperson.role || '',
    email: salesperson.email || '',
    phone: salesperson.phone || '',
    whatsapp: salesperson.whatsapp || '',
    linkedin: salesperson.linkedin || '',
    instagram: salesperson.instagram || '',
    photoUrl: salesperson.photo_url || './assets/logo-viagate-color.svg',
  });

  frameWindow.presentationSettings = Object.freeze({
    showContactSlide: content.settings?.show_contact_slide !== false,
    showClientIdentity: Boolean(content.settings?.show_client_identity),
    client: content.client ?? {},
  });

  const bootstrapScript = frameDocument.createElement('script');
  bootstrapScript.src = './presentation-bootstrap.js?v=20260825-7';
  bootstrapScript.dataset.presentationBootstrap = 'true';
  frameDocument.body.appendChild(bootstrapScript);
}

async function waitForBridge(timeoutMs = 8000) {
  const startedAt = Date.now();

  while (Date.now() - startedAt < timeoutMs) {
    if (typeof frame.contentWindow?.hostStartPresentation === 'function') {
      viewerState.bridgeReady = true;
      return true;
    }

    await new Promise((resolve) => window.setTimeout(resolve, 50));
  }

  return false;
}

function syncNavigationFromFrame() {
  const state = frame.contentWindow?.hostGetPresentationState?.();

  if (state) {
    updateNavigation(state.slideNumber, state.slideTotal);
  }
}

function showGate(mode) {
  document.body.classList.add('is-paused');
  navigation.hidden = true;

  const continuing = mode === 'continue';
  startButton.textContent = continuing ? 'CONTINUAR APRESENTAÇÃO' : 'INICIAR APRESENTAÇÃO';
  startButton.disabled = !viewerState.bridgeReady;
  restartButton.hidden = !continuing;

  frame.contentWindow?.hostPausePresentation?.();
}

function revealPresentation() {
  document.body.classList.remove('is-paused');
  navigation.hidden = false;
  restartButton.hidden = true;
  syncNavigationFromFrame();
}

async function preparePresentationStart() {
  const bridgeReady = viewerState.bridgeReady || await waitForBridge();

  if (!bridgeReady) {
    showError('Não foi possível inicializar os controles da apresentação. Atualize a página e tente novamente.');
    return false;
  }

  loadingElement.hidden = true;
  showGate('start');
  syncNavigationFromFrame();
  return true;
}

async function startPresentation() {
  if (!viewerState.frameLoaded || !viewerState.bridgeReady) {
    return;
  }

  try {
    if (!document.fullscreenElement) {
      await document.documentElement.requestFullscreen();
    }

    const firstStart = !viewerState.started;
    viewerState.started = true;
    frame.contentWindow.hostStartPresentation(firstStart);
    revealPresentation();
    syncNavigationFromFrame();
    await track('start');
  } catch {
    showGate(viewerState.started ? 'continue' : 'start');
  }
}

function restartPresentation() {
  frame.contentWindow?.hostRestartPresentation?.();
  viewerState.started = false;
  updateNavigation(1, viewerState.slideTotal);
  showGate('start');
}

function navigatePresentation(direction) {
  if (!viewerState.started || !document.fullscreenElement) {
    return;
  }

  if (direction < 0) {
    frame.contentWindow?.hostPreviousSlide?.();
  } else {
    frame.contentWindow?.hostNextSlide?.();
  }

  window.setTimeout(syncNavigationFromFrame, 80);
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

  frame.addEventListener('load', async () => {
    viewerState.frameLoaded = true;
    installPresentation(frame.contentDocument, frame.contentWindow);
    await preparePresentationStart();
  }, { once: true });

  frame.src = './presentation-content.html?v=20260825-7';
  frame.hidden = false;
  await track('open');
}

startButton.disabled = true;
startButton.addEventListener('click', startPresentation);
restartButton.addEventListener('click', restartPresentation);
previousButton.addEventListener('click', () => navigatePresentation(-1));
nextButton.addEventListener('click', () => navigatePresentation(1));

document.addEventListener('fullscreenchange', () => {
  if (document.fullscreenElement) {
    if (viewerState.started) {
      revealPresentation();
    }
    return;
  }

  if (viewerState.started) {
    showGate('continue');
  }
});

window.addEventListener('message', (event) => {
  if (event.origin !== window.location.origin || event.data?.source !== 'viagate-presentation') {
    return;
  }

  const slideNumber = Number(event.data.slideNumber || 0) || null;
  const slideTotal = Number(event.data.slideTotal || 0) || null;

  if (event.data.type === 'bridge_ready') {
    viewerState.bridgeReady = true;
    startButton.disabled = false;
  }

  if (slideNumber && slideTotal) {
    updateNavigation(slideNumber, slideTotal);
  }

  if (event.data.type === 'slide_view' && slideNumber && slideTotal) {
    track('slide_view', slideNumber, slideTotal);
  }

  if (event.data.type === 'complete' && slideNumber && slideTotal) {
    track('complete', slideNumber, slideTotal);
  }
});

updateNavigation();
initializeViewer();
