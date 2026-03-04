(function () {
  const body = document.body;
  const video = document.getElementById('mediaVideo');
  const playBtn = document.getElementById('playBtn');
  const stopBtn = document.getElementById('stopBtn');
  const volumeBtn = document.getElementById('volumeBtn');
  const volumeSlider = document.getElementById('volumeSlider');
  const fullscreenBtn = document.getElementById('fullscreenBtn');
  const audioBtn = document.getElementById('audioBtn');
  const videoContainer = document.getElementById('videoContainer');
  const videoOverlay = document.getElementById('videoOverlay');
  const progressBar = document.getElementById('progressBar');
  const progressFill = document.getElementById('progressFill');
  const bufferFill = document.getElementById('bufferFill');
  const currentTimeEl = document.getElementById('currentTime');
  const durationEl = document.getElementById('duration');
  const mediaAudio = document.getElementById('mediaAudio');
  const audioContainer = document.getElementById('audioContainer');
  const playlistItems = document.querySelectorAll('.playlist-item');
  const themeBtn = document.getElementById('themeBtn');

  let playing = false;
  let progressInterval = null;
  const THEME_KEY = 'player-theme';

  function formatTime(t) {
    if (isNaN(t) || t === Infinity) return '00:00';
    const m = Math.floor(t / 60);
    const s = Math.floor(t % 60);
    return (m < 10 ? '0' : '') + m + ':' + (s < 10 ? '0' : '') + s;
  }

  function setPlayIcon(isPlaying) {
    if (!playBtn) return;
    playBtn.innerHTML = isPlaying ? '<i class="fas fa-pause"></i>' : '<i class="fas fa-play"></i>';
  }

  function updateProgress() {
    if (!video) return;
    const d = video.duration || 0;
    const c = video.currentTime || 0;
    const pct = d > 0 ? (c / d) * 100 : 0;
    progressFill.style.width = pct + '%';
    if (bufferFill && video.buffered && video.buffered.length) {
      try {
        const bufferedEnd = video.buffered.end(video.buffered.length - 1);
        const buffPct = d > 0 ? (bufferedEnd / d) * 100 : 0;
        bufferFill.style.width = buffPct + '%';
      } catch (e) {
        // ignore
      }
    }
    if (currentTimeEl) currentTimeEl.textContent = formatTime(c);
    if (video.ended) {
      playing = false;
      setPlayIcon(false);
      if (progressInterval) { clearInterval(progressInterval); progressInterval = null; }
    }
  }

  function playMedia() {
    if (!video) return;
    video.play();
    playing = true;
    setPlayIcon(true);
    if (!progressInterval) progressInterval = setInterval(updateProgress, 500);
  }

  function pauseMedia() {
    if (!video) return;
    video.pause();
    playing = false;
    setPlayIcon(false);
    if (progressInterval) {
      clearInterval(progressInterval);
      progressInterval = null;
    }
  }

  function togglePlayPause() {
    if (!video) return;
    if (video.paused || video.ended) {
      playMedia();
    } else {
      pauseMedia();
    }
  }

  function stopMedia() {
    if (!video) return;
    video.pause();
    video.currentTime = 0;
    updateProgress();
    playing = false;
    setPlayIcon(false);
    if (progressInterval) {
      clearInterval(progressInterval);
      progressInterval = null;
    }
  }

  function onProgressClick(e) {
    if (!video) return;
    const rect = progressBar.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const w = rect.width;
    const ratio = Math.max(0, Math.min(1, x / w));
    video.currentTime = ratio * (video.duration || 0);
    updateProgress();
  }

  function setVolumeFromSlider(val) {
    const v = parseFloat(val);
    if (video) video.volume = v;
    if (volumeBtn) {
      if (v <= 0.01) volumeBtn.innerHTML = '<i class="fas fa-volume-mute"></i>';
      else volumeBtn.innerHTML = '<i class="fas fa-volume-up"></i>';
    }
  }

  function toggleFullscreen() {
    const el = videoContainer || document.documentElement;
    if (!document.fullscreenElement) {
      if (el.requestFullscreen) el.requestFullscreen();
      else if (el.webkitRequestFullscreen) el.webkitRequestFullscreen();
      else if (el.msRequestFullscreen) el.msRequestFullscreen();
    } else {
      if (document.exitFullscreen) document.exitFullscreen();
    }
  }

  function loadMediaFromPlaylist(item) {
    const src = item.getAttribute('data-src');
    const type = item.getAttribute('data-type');
    if (type === 'video') {
      if (video) {
        video.src = src;
        video.load();
        video.play();
        videoContainer.style.display = 'block';
        audioContainer.style.display = 'none';
        if (videoOverlay) videoOverlay.style.display = 'flex';
        if (progressInterval) { clearInterval(progressInterval); progressInterval = null; }
        playing = true;
        setPlayIcon(true);
        progressInterval = setInterval(updateProgress, 500);
      }
    } else if (type === 'audio') {
      if (video) video.pause();
      videoContainer.style.display = 'none';
      audioContainer.style.display = 'block';
      if (mediaAudio) {
        mediaAudio.src = src;
        mediaAudio.play();
      }
      if (progressInterval) {
        clearInterval(progressInterval);
        progressInterval = null;
      }
      currentTimeEl.textContent = '00:00';
      durationEl.textContent = '00:00';
      progressFill.style.width = '0%';
      if (bufferFill) bufferFill.style.width = '0%';
    }
    // set active item styling
    playlistItems.forEach(i => i.classList.remove('active'));
    item.classList.add('active');
  }

  // Attach listeners
  if (typeof playBtn !== 'undefined' && playBtn) playBtn.addEventListener('click', togglePlayPause);
  if (typeof stopBtn !== 'undefined' && stopBtn) stopBtn.addEventListener('click', stopMedia);
  if (typeof progressBar !== 'undefined' && progressBar) progressBar.addEventListener('click', onProgressClick);
  if (typeof volumeSlider !== 'undefined' && volumeSlider) volumeSlider.addEventListener('input', (e) => setVolumeFromSlider(e.target.value));
  if (typeof fullscreenBtn !== 'undefined' && fullscreenBtn) fullscreenBtn.addEventListener('click', toggleFullscreen);
  if (typeof audioBtn !== 'undefined' && audioBtn) audioBtn.addEventListener('click', () => { if (video) loadMediaFromPlaylist(document.querySelector('.playlist-item[data-type="audio"]')) });
  if (typeof themeBtn !== 'undefined' && themeBtn) themeBtn.addEventListener('click', () => {
    body.classList.toggle('dark-theme');
    const isDark = body.classList.contains('dark-theme');
    localStorage.setItem(THEME_KEY, isDark ? 'dark' : 'light');
  });
  if (playlistItems.length) {
    playlistItems.forEach(it => {
      it.addEventListener('click', () => loadMediaFromPlaylist(it));
    });
  }

  // Keyboard shortcuts
  document.addEventListener('keydown', (e) => {
    if (['INPUT','TEXTAREA'].includes((e.target && e.target.tagName) || '')) return;
    switch (e.code) {
      case 'Space':
        e.preventDefault();
        togglePlayPause();
        break;
      case 'ArrowLeft':
        if (video) {
          video.currentTime = Math.max(0, video.currentTime - 5);
          updateProgress();
        }
        break;
      case 'ArrowRight':
        if (video) {
          video.currentTime = Math.min(video.duration || 0, video.currentTime + 5);
          updateProgress();
        }
        break;
      case 'KeyM':
        if (video) video.muted = !video.muted;
        break;
      case 'KeyF':
        toggleFullscreen();
        break;
    }
  });

  // Init defaults
  // Theme persistence
  const savedTheme = localStorage.getItem(THEME_KEY);
  if (savedTheme === 'dark') {
    body.classList.add('dark-theme');
  }
  // Volume default
  if (video) video.volume = 0.7;
  if (volumeSlider) volumeSlider.value = video ? video.volume : 0.7;
  // Metadata loaded
  if (video) video.addEventListener('loadedmetadata', () => {
    if (durationEl) durationEl.textContent = formatTime(video.duration);
  });
  // Initial overlay
  if (videoOverlay) videoOverlay.style.display = 'flex';
})();
