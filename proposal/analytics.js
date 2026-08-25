import { supabase } from './supabase.js';

const token = new URLSearchParams(window.location.search).get('token');
const analyticsState = {
  sessionId: null,
  lastSlide: 0,
  initialized: false,
  started: false,
};

function getSessionId() {
  if (!token) {
    return crypto.randomUUID();
  }

  const key = `viagate:proposal:${token}:session`;
  let value = sessionStorage.getItem(key);

  if (!value) {
    value = crypto.randomUUID();
    sessionStorage.setItem(key, value);
  }

  return value;
}

async function track(eventName, slideNumber = null, slideTotal = null) {
  if (!supabase || !token || !analyticsState.sessionId) {
    return;
  }

  try {
    await supabase.rpc('track_shared_document_event', {
      document_kind: 'proposal',
      document_token: token,
      event_name: eventName,
      viewer_session: analyticsState.sessionId,
      slide_number: slideNumber,
      slide_total: slideTotal,
    });
  } catch {
  }
}

function getSlides() {
  return Array.from(document.querySelectorAll('[data-proposal-slide]'));
}

function reportCurrentSlide() {
  if (!analyticsState.started) {
    return;
  }

  const slides = getSlides();
  if (!slides.length) {
    return;
  }

  const center = window.scrollY + window.innerHeight / 2;
  let currentIndex = 0;

  slides.forEach((slide, index) => {
    if (center >= slide.offsetTop) {
      currentIndex = index;
    }
  });

  const slideNumber = currentIndex + 1;
  if (slideNumber === analyticsState.lastSlide) {
    return;
  }

  analyticsState.lastSlide = slideNumber;
  track('slide_view', slideNumber, slides.length);

  if (slideNumber === slides.length) {
    track('complete', slideNumber, slides.length);
  }
}

function bindAnalytics() {
  if (analyticsState.initialized) {
    return;
  }

  analyticsState.initialized = true;
  analyticsState.sessionId = getSessionId();
  track('open');

  document.addEventListener('proposal:started', () => {
    analyticsState.started = true;
    track('start');
    window.setTimeout(reportCurrentSlide, 120);
  });

  document.addEventListener('proposal:restarted', () => {
    analyticsState.started = false;
    analyticsState.lastSlide = 0;
  });

  window.addEventListener('scroll', reportCurrentSlide, { passive: true });

  const observer = new MutationObserver(() => {
    if (analyticsState.started && getSlides().length) {
      reportCurrentSlide();
    }
  });

  observer.observe(document.getElementById('proposalPresentation') ?? document.body, {
    childList: true,
    subtree: true,
  });
}

bindAnalytics();
