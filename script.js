const slides = Array.from(document.querySelectorAll('[data-slide]'));
const progressBar = document.getElementById('progressBar');
const nextButton = document.getElementById('nextSlide');
const header = document.querySelector('.presentation-header');

if (window.lucide) {
  window.lucide.createIcons();
}

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
