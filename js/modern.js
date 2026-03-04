(function() {
  const body = document.body;
  const video = document.getElementById('mediaVideo');
  const playBtn = document.getElementById('playBtn');
  const stopBtn = document.getElementById('stopBtn');
  const volumeBtn = document.getElementById('volumeBtn');
  const volumeSlider = document.getElementById('volumeSlider');
  const fullscreenBtn = document.getElementById('fullscreenBtn');
  const videoContainer = document.getElementById('videoContainer');
  const videoOverlay = document.getElementById('videoOverlay');
  const progressBar = document.getElementById('progressBar');
  const progressFill = document.getElementById('progressFill');
  const currentTimeEl = document.getElementById('currentTime');
  const durationEl = document.getElementById('duration');
  const playlistItems = document.querySelectorAll('.playlist-item');
  const themeBtn = document.getElementById('themeBtn');
  const fileInput = document.getElementById('fileInput');
  const playlistContent = document.getElementById('playlistContent');

  let playing = false;
  let progressInterval = null;
  const THEME_KEY = 'player-theme';

  function formatTime(t) {
    if (isNaN(t) || t === Infinity || !t) return '00:00';
    const m = Math.floor(t / 60);
    const s = Math.floor(t % 60);
    return (m < 10 ? '0' : '') + m + ':' + (s < 10 ? '0' : '') + s;
  }

  function setPlayIcon(isPlaying) {
    if (!playBtn) return;
    playBtn.innerHTML = isPlaying ? '<i class="fas fa-pause"></i>' : '<i class="fas fa-play"></i>';
  }

  function hideOverlay() {
    if (videoOverlay) videoOverlay.style.display = 'none';
  }

  function showOverlay() {
    if (videoOverlay) videoOverlay.style.display = 'flex';
  }

  function updateProgress() {
    if (!video) return;
    const d = video.duration || 0;
    const c = video.currentTime || 0;
    const pct = d > 0 ? (c / d) * 100 : 0;
    if (progressFill) progressFill.style.width = pct + '%';
    if (currentTimeEl) currentTimeEl.textContent = formatTime(c);
    
    if (video.ended) {
      playing = false;
      setPlayIcon(false);
      showOverlay();
      if (progressInterval) {
        clearInterval(progressInterval);
        progressInterval = null;
      }
    }
  }

  function playMedia() {
    if (!video) return;
    video.play();
    playing = true;
    setPlayIcon(true);
    hideOverlay();
    if (!progressInterval) progressInterval = setInterval(updateProgress, 250);
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
    showOverlay();
    if (progressInterval) {
      clearInterval(progressInterval);
      progressInterval = null;
    }
  }

  function onProgressClick(e) {
    if (!video || !progressBar) return;
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

  function loadVideo(src, title) {
    if (!video) return;
    video.src = src;
    video.load();
    video.play();
    hideOverlay();
    if (progressInterval) {
      clearInterval(progressInterval);
      progressInterval = null;
    }
    playing = true;
    setPlayIcon(true);
    progressInterval = setInterval(updateProgress, 250);
    
    // Update playlist selection
    document.querySelectorAll('.playlist-item').forEach(item => {
      item.classList.remove('active');
    });
  }

  function addToPlaylist(src, title) {
    const item = document.createElement('div');
    item.className = 'playlist-item';
    item.setAttribute('data-src', src);
    item.setAttribute('data-type', 'video');
    item.innerHTML = '<i class="fas fa-video"></i><span>' + title + '</span>';
    item.addEventListener('click', function() {
      loadVideo(src, title);
      document.querySelectorAll('.playlist-item').forEach(i => i.classList.remove('active'));
      this.classList.add('active');
    });
    playlistContent.appendChild(item);
  }

  // Event Listeners
  if (playBtn) playBtn.addEventListener('click', togglePlayPause);
  if (stopBtn) stopBtn.addEventListener('click', stopMedia);
  if (videoOverlay) videoOverlay.addEventListener('click', togglePlayPause);
  if (progressBar) progressBar.addEventListener('click', onProgressClick);
  if (volumeSlider) volumeSlider.addEventListener('input', (e) => setVolumeFromSlider(e.target.value));
  if (fullscreenBtn) fullscreenBtn.addEventListener('click', toggleFullscreen);
  
  // File input handler
  if (fileInput) {
    fileInput.addEventListener('change', function(e) {
      const file = e.target.files[0];
      if (file) {
        const url = URL.createObjectURL(file);
        const title = file.name.replace(/\.[^/.]+$/, '');
        loadVideo(url, title);
        addToPlaylist(url, title);
      }
    });
  }
  
  if (themeBtn) {
    themeBtn.addEventListener('click', () => {
      body.classList.toggle('dark-theme');
      const isDark = body.classList.contains('dark-theme');
      localStorage.setItem(THEME_KEY, isDark ? 'dark' : 'light');
      themeBtn.innerHTML = isDark ? '<i class="fas fa-sun"></i>' : '<i class="fas fa-moon"></i>';
    });
  }

  if (playlistItems.length) {
    playlistItems.forEach(it => {
      it.addEventListener('click', function() {
        const src = this.getAttribute('data-src');
        const title = this.querySelector('span').textContent;
        loadVideo(src, title);
        document.querySelectorAll('.playlist-item').forEach(i => i.classList.remove('active'));
        this.classList.add('active');
      });
    });
  }

  // Keyboard shortcuts
  document.addEventListener('keydown', (e) => {
    if (['INPUT', 'TEXTAREA'].includes(e.target.tagName)) return;
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
        if (video) {
          video.muted = !video.muted;
          if (volumeBtn) {
            volumeBtn.innerHTML = video.muted ? '<i class="fas fa-volume-mute"></i>' : '<i class="fas fa-volume-up"></i>';
          }
        }
        break;
      case 'KeyF':
        toggleFullscreen();
        break;
    }
  });

  // Initialize
  const savedTheme = localStorage.getItem(THEME_KEY);
  if (savedTheme === 'dark') {
    body.classList.add('dark-theme');
    if (themeBtn) themeBtn.innerHTML = '<i class="fas fa-sun"></i>';
  }

  if (video) {
    video.volume = 0.7;
    video.addEventListener('loadedmetadata', () => {
      if (durationEl) durationEl.textContent = formatTime(video.duration);
    });
    video.addEventListener('play', () => {
      playing = true;
      setPlayIcon(true);
      hideOverlay();
      if (!progressInterval) progressInterval = setInterval(updateProgress, 250);
    });
    video.addEventListener('pause', () => {
      playing = false;
      setPlayIcon(false);
    });
    video.addEventListener('ended', () => {
      playing = false;
      setPlayIcon(false);
      showOverlay();
      if (progressInterval) {
        clearInterval(progressInterval);
        progressInterval = null;
      }
    });
  }

  if (volumeSlider) volumeSlider.value = 0.7;
  
  if (videoOverlay) {
    videoOverlay.style.display = 'flex';
  }
})();
