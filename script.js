function loadEnhancementStyles() {
  if (document.querySelector('link[data-viagate-enhancements]')) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = './enhancements.css';
  link.dataset.viagateEnhancements = 'true';
  document.head.appendChild(link);
}

function loadInsurerStyles() {
  if (document.querySelector('link[data-viagate-insurers]')) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = './insurers.css';
  link.dataset.viagateInsurers = 'true';
  document.head.appendChild(link);
}

function loadHeroV2Styles() {
  if (document.querySelector('link[data-viagate-hero-v2]')) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = './hero-v2.css';
  link.dataset.viagateHeroV2 = 'true';
  document.head.appendChild(link);
}

function updateHeroSlide() {
  const slide = document.querySelector('#slide-01');
  const original = slide?.querySelector(':scope > .slide-inner');

  if (!slide || !original || slide.querySelector('.hero-v2-layout')) {
    return;
  }

  original.classList.add('hero-original', 'is-hidden');
  original.setAttribute('aria-hidden', 'true');

  const hero = document.createElement('div');
  hero.className = 'slide-inner hero-v2-layout';
  hero.innerHTML = `
    <div class="hero-v2-copy">
      <p class="kicker">VISÃO EXECUTIVA</p>
      <h1>Validação, análise de risco<br /><span>e operação logística.</span></h1>
      <p class="hero-v2-statement">Um único ecossistema para identificar quem está na operação, apoiar a decisão e acompanhar a viagem.</p>
      <p class="lead">Tecnologia própria para validar pessoas, consultar informações relevantes e tornar o processo mais seguro, rastreável e eficiente.</p>
    </div>

    <div class="hero-v2-panel">
      <div class="hero-v2-panel-head">
        <span>COMO A VIAGATE ATUA</span>
        <small>Três pilares da operação</small>
      </div>

      <div class="hero-v2-grid">
        <article class="hero-v2-card">
          <div class="hero-v2-card-head"><i data-lucide="scan-face"></i><span>01</span></div>
          <div>
            <h3>Validação</h3>
            <p>Biometria com prova de vida, identidade, CNH, ANTT e RNTRC.</p>
          </div>
        </article>
        <article class="hero-v2-card hero-v2-card-accent">
          <div class="hero-v2-card-head"><i data-lucide="shield-check"></i><span>02</span></div>
          <div>
            <h3>Análise de risco</h3>
            <p>Cargo Score, processos, veículo, proprietário e regras operacionais.</p>
          </div>
        </article>
        <article class="hero-v2-card">
          <div class="hero-v2-card-head"><i data-lucide="route"></i><span>03</span></div>
          <div>
            <h3>Operação</h3>
            <p>Cargo Truck, rastreamento, eventos e Plataforma Cargo durante a viagem.</p>
          </div>
        </article>
      </div>

      <div class="hero-v2-panel-footer">
        <div class="hero-v2-proof">
          <div><strong>300+</strong><span>clientes diretos</span></div>
          <div><strong>30+ anos</strong><span>de experiência acumulada</span></div>
          <div><strong>100%</strong><span>tecnologia própria</span></div>
        </div>
      </div>
    </div>
  `;

  slide.insertBefore(hero, original);
}

function updateAnalysisResultSlide() {
  const slide = document.querySelector('#slide-04');
  const content = slide?.querySelector('.slide-inner');

  if (!content) {
    return;
  }

  content.className = 'slide-inner analysis-result-layout';
  content.innerHTML = `
    <div class="section-copy analysis-result-heading">
      <p class="kicker dark-kicker">RESULTADO DA ANÁLISE · CARGO SCORE</p>
      <h2>Da coleta dos dados à <span>emissão do documento de liberação.</span></h2>
      <p class="lead dark-lead">O Cargo Score valida a identidade, consulta as informações necessárias e consolida o resultado em um fluxo rastreável.</p>
    </div>
    <div class="analysis-result-flow">
      <article class="analysis-result-step">
        <div class="analysis-result-top"><i data-lucide="file"></i><span>01</span></div>
        <h3>Motorista envia os dados</h3>
        <p>O processo começa com o envio das informações necessárias para a pesquisa.</p>
      </article>
      <article class="analysis-result-step">
        <div class="analysis-result-top"><i data-lucide="message-circle"></i><span>02</span></div>
        <h3>Sistema envia o link da biometria via WhatsApp</h3>
        <p>O motorista recebe um link e acessa a validação diretamente pelo navegador.</p>
      </article>
      <article class="analysis-result-step">
        <div class="analysis-result-top"><i data-lucide="scan-face"></i><span>03</span></div>
        <h3>Motorista realiza biometria com prova de vida</h3>
        <p>A identidade é confirmada antes do início das consultas.</p>
      </article>
      <article class="analysis-result-step">
        <div class="analysis-result-top"><i data-lucide="database"></i><span>04</span></div>
        <h3>Sistema analisa os dados cadastrais e de risco</h3>
        <p>As fontes consultadas formam o contexto principal da análise.</p>
        <strong>DADOS PESSOAIS · CNH · ANTT · RNTRC · PROCESSOS CRIMINAIS</strong>
      </article>
      <article class="analysis-result-step">
        <div class="analysis-result-top"><i data-lucide="car-front"></i><span>05</span></div>
        <h3>Sistema consulta os veículos</h3>
        <p>Os veículos vinculados à operação entram no processo de validação.</p>
      </article>
      <article class="analysis-result-step">
        <div class="analysis-result-top"><i data-lucide="user-round-check"></i><span>06</span></div>
        <h3>Sistema valida o proprietário do veículo</h3>
        <p>Os dados do proprietário são confrontados com as informações consultadas.</p>
      </article>
      <article class="analysis-result-step is-final">
        <div class="analysis-result-top"><i data-lucide="file-check-2"></i><span>07</span></div>
        <h3>Emissão do documento de liberação</h3>
        <p>O resultado final é consolidado em um documento para apoio à liberação da operação.</p>
        <strong>DOCUMENTO DE LIBERAÇÃO</strong>
      </article>
    </div>
  `;
}

function updateScoreStepsSlide() {
  const slide = document.querySelector('#slide-05');
  const flow = slide?.querySelector('.score-flow');
  const note = slide?.querySelector('.score-flow-note');

  if (!flow) {
    return;
  }

  flow.innerHTML = `
    <article>
      <div class="score-step-head"><i data-lucide="send"></i><span class="step-number">01</span></div>
      <strong>Início da pesquisa</strong>
      <p>A empresa inicia a análise e disponibiliza um link seguro ao pesquisado.</p>
    </article>
    <article>
      <div class="score-step-head"><i data-lucide="scan-face"></i><span class="step-number">02</span></div>
      <strong>Biometria</strong>
      <p>O motorista realiza a validação facial com prova de vida.</p>
    </article>
    <article>
      <div class="score-step-head"><i data-lucide="badge-check"></i><span class="step-number">03</span></div>
      <strong>Dados cadastrais</strong>
      <p>Identidade e CNH são confrontadas com as informações consultadas.</p>
    </article>
    <article>
      <div class="score-step-head"><i data-lucide="database"></i><span class="step-number">04</span></div>
      <strong>Consultas</strong>
      <p>ANTT, RNTRC, processos, veículo, proprietário e outros dados entram na análise.</p>
    </article>
    <article>
      <div class="score-step-head"><i data-lucide="settings-2"></i><span class="step-number">05</span></div>
      <strong>Regras de risco</strong>
      <p>Os critérios são parametrizados conforme o perfil operacional do cliente.</p>
    </article>
    <article class="flow-final">
      <div class="score-step-head"><i data-lucide="shield-check"></i><span class="step-number">06</span></div>
      <strong>Resultado</strong>
      <p>O sistema apresenta a decisão operacional e a documentação de liberação.</p>
    </article>
  `;

  if (note) {
    note.innerHTML = `
      <i data-lucide="timer"></i>
      <strong>Prazo de análise: 1 a 10 minutos</strong>
      <span>Prazo variável conforme a parametrização da operação.</span>
    `;
  }
}

function updateConsultationCopy() {
  const slide = document.querySelector('#slide-06');
  const heading = slide?.querySelector('h2');
  const lead = slide?.querySelector('.lead');
  const anttCard = Array.from(slide?.querySelectorAll('.data-grid article') ?? [])
    .find((card) => card.querySelector('h3')?.textContent?.trim() === 'ANTT');

  if (heading) {
    heading.innerHTML = 'O Cargo Score consulta <span>identidade, CNH, ANTT, RNTRC, processos, veículo e proprietário.</span>';
  }

  if (lead) {
    lead.textContent = 'As fontes consultadas formam a base da análise de risco e da liberação da operação.';
  }

  const anttDescription = anttCard?.querySelector('p');
  if (anttDescription) {
    anttDescription.textContent = 'Consulta de ANTT, RNTRC e informações relacionadas ao transporte rodoviário.';
  }
}

function updateProductPortfolioSlide() {
  const slide = document.querySelector('#slide-09');
  const heading = slide?.querySelector('h2');
  const lead = slide?.querySelector('.lead');
  const catalog = slide?.querySelector('.catalog-grid');

  if (heading) {
    heading.innerHTML = 'Produtos ViaGate <span>organizados por linha de solução.</span>';
  }

  if (lead) {
    lead.textContent = 'Portfólio de validação, consultas, operação logística e integração.';
  }

  if (!catalog) {
    return;
  }

  catalog.className = 'product-portfolio-grid';
  catalog.innerHTML = `
    <article class="product-portfolio-group">
      <header><span>ANÁLISE E VALIDAÇÃO</span><strong>Identidade e gerenciamento de risco</strong></header>
      <ul>
        <li>Cargo Score</li>
        <li>Biometria Facial</li>
        <li>Autenticador</li>
        <li>Pesquisa Cadastral</li>
      </ul>
    </article>
    <article class="product-portfolio-group">
      <header><span>CONSULTAS</span><strong>Dados para análise operacional</strong></header>
      <ul>
        <li>Validação de CNH</li>
        <li>Consulta ANTT</li>
        <li>Histórico Veicular</li>
        <li>Processos</li>
        <li>Antecedentes Criminais</li>
        <li>Vitimologia</li>
      </ul>
    </article>
    <article class="product-portfolio-group">
      <header><span>OPERAÇÃO</span><strong>Execução e acompanhamento logístico</strong></header>
      <ul>
        <li>Cargo Truck</li>
        <li>Plataforma Cargo</li>
        <li>Gestão Logística</li>
        <li>Gestão Securitária</li>
      </ul>
    </article>
    <article class="product-portfolio-group">
      <header><span>INTEGRAÇÃO</span><strong>Tecnologia dentro da operação do cliente</strong></header>
      <ul>
        <li>API</li>
        <li>White Label</li>
      </ul>
    </article>
  `;
}

function updateOperationalCopy() {
  const heading = document.querySelector('#slide-10 h2');
  if (heading) {
    heading.innerHTML = 'Depois da aprovação, <span>Cargo Truck e Plataforma Cargo acompanham a execução da viagem.</span>';
  }
}

function updateMarketProofCopy() {
  const slide = document.querySelector('#slide-13');
  const lead = slide?.querySelector('.proof-copy .lead');
  const experienceMetric = slide?.querySelector('.metric-success');
  const experienceValue = experienceMetric?.querySelector('strong');
  const experienceLabel = experienceMetric?.querySelector('span');
  const certification = slide?.querySelector('.metric-certification span');

  if (lead) {
    lead.textContent = 'A ViaGate já atende mais de 300 clientes diretos e possui reconhecimento no mercado securitário.';
  }

  if (experienceValue) {
    experienceValue.textContent = '30+ anos';
  }

  if (experienceLabel) {
    experienceLabel.textContent = 'de experiência acumulada na equipe.';
  }

  if (certification) {
    certification.textContent = 'Auditada pela FENSEG e reconhecida pelas principais seguradoras.';
  }
}

function insertInsurersSlide() {
  if (document.querySelector('#slide-insurers')) {
    return;
  }

  const teamSlide = document.querySelector('#slide-14');
  if (!teamSlide) {
    return;
  }

  loadInsurerStyles();

  const insurers = [
    { name: 'Allianz', src: './assets/insurers/color/allianz-logo.svg', className: 'logo-wide' },
    { name: 'AXA', src: './assets/insurers/color/axa-logo.svg', className: 'logo-tall' },
    { name: 'Chubb', src: './assets/insurers/color/chubb-logo.svg', className: 'logo-wide' },
    { name: 'HDI Seguros', src: './assets/insurers/color/hdi-seguros-logo.svg', className: '' },
    { name: 'Porto Seguro', src: './assets/insurers/color/porto-seguro-logo.svg', className: '' },
    { name: 'Mapfre', src: './assets/insurers/color/Mapfre_logo.svg', className: 'logo-tall' },
    { name: 'Liberty Seguros', src: './assets/insurers/color/liberty-seguros.svg', className: 'logo-tall' },
    { name: 'Grupo Sura', src: './assets/insurers/color/sura_logo.svg', className: '' },
    { name: 'Swiss Re', src: './assets/insurers/color/Swiss_Re_2013_logo.svg', className: 'logo-wide' },
    { name: 'Akad Seguros', src: './assets/insurers/color/akad-logo.svg', className: 'logo-wide' },
    { name: 'FF Seguros', src: './assets/insurers/color/ff-logo.svg', className: 'logo-wide' },
  ];

  const cards = insurers.map(({ name, src, className }) => `
    <article class="insurer-card ${className}">
      <img src="${src}" alt="${name}" loading="lazy" decoding="async" />
    </article>
  `).join('');

  const section = document.createElement('section');
  section.className = 'slide slide-light insurers-slide';
  section.id = 'slide-insurers';
  section.dataset.slide = '14';
  section.innerHTML = `
    <div class="slide-inner insurers-layout">
      <div class="section-copy insurers-heading">
        <p class="kicker dark-kicker">MERCADO SECURITÁRIO</p>
        <h2>Empresas do mercado securitário que <span>confiam na ViaGate.</span></h2>
        <p class="lead dark-lead">Relacionamentos construídos com empresas que exigem segurança, rastreabilidade e consistência operacional.</p>
      </div>
      <div class="insurers-grid" aria-label="Seguradoras e empresas do mercado securitário que confiam na ViaGate">
        ${cards}
      </div>
    </div>
    <div class="slide-footer dark-footer"><span>Mercado securitário</span><span></span></div>
  `;

  teamSlide.before(section);
}

function renumberSlides() {
  const allSlides = Array.from(document.querySelectorAll('[data-slide]'));
  const total = allSlides.length;

  allSlides.forEach((slide, index) => {
    const number = index + 1;
    slide.dataset.slide = String(number);
    slide.id = `slide-${String(number).padStart(2, '0')}`;

    const counter = slide.querySelector('.slide-footer span:last-child');
    if (counter) {
      counter.textContent = `${String(number).padStart(2, '0')} / ${String(total).padStart(2, '0')}`;
    }
  });
}

function updateTeamSlide() {
  const slide = document.querySelector('#slide-15');
  const people = Array.from(slide?.querySelectorAll('.person') ?? []);
  const photoByName = new Map([
    ['Antônio Santos', './assets/antonio-photo.svg'],
    ['Anésio Santos', './assets/anesio-photo.svg'],
    ['Mateus Insa', './assets/mateus-photo.svg'],
  ]);

  people.forEach((person) => {
    const name = person.querySelector('h3')?.textContent?.trim();
    const photo = person.querySelector('.person-photo');
    const src = photoByName.get(name);

    if (!photo || !src || !name) {
      return;
    }

    photo.classList.add('has-photo');
    photo.innerHTML = `<img src="${src}" alt="${name}" loading="lazy" decoding="async" />`;
  });
}

function initializeFadeUpAnimations() {
  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  const selectors = [
    '.hero-v2-copy',
    '.hero-v2-card',
    '.hero-v2-proof > div',
    '.hero-v2-next',
    '.hero-copy',
    '.hero-media',
    '.section-copy',
    '.market-grid article',
    '.benefit-card',
    '.phone-stage',
    '.analysis-result-step',
    '.score-flow article',
    '.data-grid article',
    '.catalog-item',
    '.fraud-block',
    '.product-portfolio-group',
    '.ecosystem-flow article',
    '.operation-media',
    '.operation-steps > div',
    '.integration-cards article',
    '.metric-wall article',
    '.insurer-card',
    '.person',
    '.closing-layout > *',
  ];

  const observer = reduceMotion
    ? null
    : new IntersectionObserver((entries) => {
        entries.forEach((entry) => {
          if (!entry.isIntersecting) {
            return;
          }

          entry.target.classList.add('is-visible');
          observer.unobserve(entry.target);
        });
      }, { threshold: 0.12, rootMargin: '0px 0px -6% 0px' });

  document.querySelectorAll('.slide').forEach((slide) => {
    const elements = Array.from(slide.querySelectorAll(selectors.join(',')));

    elements.forEach((element, index) => {
      element.classList.add('fade-up');
      element.style.setProperty('--fade-delay', `${Math.min(index, 5) * 55}ms`);

      if (reduceMotion) {
        element.classList.add('is-visible');
        return;
      }

      observer.observe(element);
    });
  });
}

function applyPresentationEnhancements() {
  loadEnhancementStyles();
  loadHeroV2Styles();
  updateHeroSlide();
  updateAnalysisResultSlide();
  updateScoreStepsSlide();
  updateConsultationCopy();
  updateProductPortfolioSlide();
  updateOperationalCopy();
  updateMarketProofCopy();
  insertInsurersSlide();
  renumberSlides();
  updateTeamSlide();
}

applyPresentationEnhancements();

const slides = Array.from(document.querySelectorAll('[data-slide]'));
const progressBar = document.getElementById('progressBar');
const nextButton = document.getElementById('nextSlide');
const header = document.querySelector('.presentation-header');

if (window.lucide) {
  window.lucide.createIcons();
}

initializeFadeUpAnimations();

function getCurrentSlideIndex() {
  const viewportCenter = window.scrollY + window.innerHeight / 2;
  let currentIndex = 0;

  slides.forEach((slide, index) => {
    if (viewportCenter >= slide.offsetTop) {
      currentIndex = index;
    }
  });

  return currentIndex;
}

function updateProgress() {
  const currentIndex = getCurrentSlideIndex();
  const percentage = ((currentIndex + 1) / slides.length) * 100;
  const currentSlide = slides[currentIndex];
  const isLight = currentSlide.classList.contains('slide-light');

  progressBar.style.width = `${percentage}%`;
  header.dataset.theme = isLight ? 'light' : 'dark';
  nextButton.style.opacity = currentIndex === slides.length - 1 ? '0' : '1';
  nextButton.style.pointerEvents = currentIndex === slides.length - 1 ? 'none' : 'auto';
}

function goToSlide(index) {
  if (index < 0 || index >= slides.length) {
    return;
  }

  slides[index].scrollIntoView({ behavior: 'smooth', block: 'start' });
}

nextButton.addEventListener('click', () => {
  goToSlide(getCurrentSlideIndex() + 1);
});

window.addEventListener('keydown', (event) => {
  if (['ArrowDown', 'PageDown', 'ArrowRight'].includes(event.key)) {
    event.preventDefault();
    goToSlide(getCurrentSlideIndex() + 1);
  }

  if (['ArrowUp', 'PageUp', 'ArrowLeft'].includes(event.key)) {
    event.preventDefault();
    goToSlide(getCurrentSlideIndex() - 1);
  }
});

window.addEventListener('scroll', updateProgress, { passive: true });
window.addEventListener('resize', updateProgress);
updateProgress();

const executiveScript = document.createElement('script');
executiveScript.src = './executive-v2.js';
executiveScript.defer = false;
document.body.appendChild(executiveScript);