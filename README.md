# 🚀 StreamDaveFast Ultra Rápido

**Plataforma de streaming de video de alto rendimiento**, optimizada para eliminar micro-stuttering, tearing y lag, incluso en calidades Full HD (1080p).

---

## 🎯 Propósito

StreamDaveFast es el primer paso hacia una plataforma de streaming profesional. Convierte cualquier archivo de video en un flujo adaptativo DASH que se ajusta automáticamente a la velocidad de internet del usuario, garantizando reproducción sin cortes y con latencia mínima gracias a su entrega desde RAM.

### ¿Cómo funciona el flujo optimizado?

```
                ┌─────────────────────────────┐
   MP4/MKV   →  │   FFmpeg Transcoder (Go)    │  →  Perfil High 4.1
   subido       │   GOP 5s | Keyframes Temporales│     VBV Estricto
                └──────────┬──────────────────┘
                           ↓
                ┌─────────────────────────────┐  →  Cache en RAM (45s)
                │    Segmentos DASH (.m4s)     │     Serving instantáneo
                │    + Manifiesto (.mpd)       │     Headers X-Cache
                └──────────┬──────────────────┘
                           ↓
                ┌─────────────────────────────┐  →  Buffer de 20s
                │       Shaka Player          │     Ajuste Auto de Audio
                │  Bitrate Adaptativo Auto    │     UI Throttling (250ms)
                └─────────────────────────────┘
```

> **¿Internet lento?** → Se reproduce en 144p/240p sin cortes  
> **¿Internet rápido?** → Sube a 1080p con VBV controlado (Sin picos de datos)  
> **¿Reproducción fluida?** → Entrega desde RAM para eliminar la latencia del disco

---

## ⚡ Características "Ultra"

### Motor de Procesamiento (Backend en Go)
- **RAM Cache System**: Los segmentos de video se cargan en la memoria RAM el primer acceso y se sirven instantáneamente (latencia <1ms).
- **Limpieza Automática**: Recolector de basura inteligente que libera segmentos inactivos cada 30 segundos.
- **Workers Pool**: Procesamiento concurrente que no bloquea la entrega de videos existentes.
- **Logging de Cache**: Headers `X-Cache: HIT-RAM` o `MISS-RAM` para monitoreo en tiempo real.

### Escalera de Bitrate (Transcodificación FFmpeg)
| Calidad | Resolución | Video Bitrate | Profile/Level | VBV (Max/Buf) |
|---------|-----------|---------------|---------------|---------------|
| 144p | 256x144 | 150 kbps | Main 4.0 | 180k / 300k |
| 240p | 426×240 | 350 kbps | Main 4.0 | 420k / 700k |
| 480p | 854×480 | 1,200 kbps | Main 4.0 | 1.4M / 2.4M |
| 720p | 1280×720 | 2,500 kbps | Main 4.0 | 3.0M / 5.0M |
| 1080p 🔥 | 1920×1080| 4,500 kbps| **High 4.1** | **5.0M / 10M** |

*   **Sincronización A/V**: Filtros `aresample=async=1` y `force_key_frames` cada 5s para evitar saltos.
*   **FPS Nativos**: Detección dinámica de frames (23.98, 24, 30, 60fps) sin conversiones forzadas.
*   **Audio CBR**: AAC a 128kbps constante para transiciones DASH silenciosas.

### Reproductor "YouTube Experience" (Frontend)
- **UI Optimizado**: Actualización de interfaz cada 250ms (ahorra CPU en 1080p).
- **Control Gestual**: Click simple para Play/Pausa, Doble-Click para Fullscreen.
- **Auto-Hide**: Controles inteligentes que desaparecen tras 3s de inactividad.
- **Anti-Gaps**: Shaka Player configurado para saltar huecos minúsculos (<0.3s) y autocorregir audio.
- **Buffer Adaptativo**: Meta de 20s para estabilidad sin saturar la memoria del navegador.

---

## 🛠 Stack Tecnológico

| Componente | Tecnología | Detalle Optimizado |
|-----------|-----------|-----------|
| **Backend** | Go (Golang) | RAM Caching & Goroutines |
| **Transcodificación** | FFmpeg (libx264) | **Preset Fast** (Estabilidad 1080p) |
| **Formato de salida** | DASH (MP4 Fragments) | Gapless Audio & VBV Enforced |
| **Reproductor** | Shaka Player | ABR con `lowerBitrateSwitching` |
| **UI** | Vanilla JS & CSS | Animaciones Glassmorphism |

---

## 🚀 Inicio Rápido

```bash
# 1. Preparar entorno
mkdir Videos processed

# 2. Iniciar el servidor
go build -o streamdavefast .
./streamdavefast

# 3. Acceder
# http://localhost:8080
```

---

## ⌨️ Atajos de Teclado

| Tecla | Acción |
|-------|--------|
| `Espacio` | Play / Pausa |
| `←` / `→` | Salto 10s |
| `M` | Silencio |
| `F` | Pantalla completa |
| `1`-`4` | Salto por porcentajes (Proximamente) |

---

## 🗺 Roadmap Completado
- [x] Servidor Go con RAM Cache
- [x] Transcodificación con VBV Estricto para 1080p
- [x] Sincronización de Audio AAC CBR
- [x] UI de Reproductor tipo YouTube
- [x] Autodetección de perfiles por resolución original
- [x] Limpieza de RAM automática
- [ ] Soporte HLS (iOS)
- [ ] Implementación de DRM básica
- [ ] Multi-idioma en audio

---

## 📄 Licencia
MIT License 2026 - StreamDaveFast Project

**Plataforma de streaming de video ultrarrápida con bitrate adaptativo**, diseñada para reproducir video con fluidez incluso en conexiones lentas, al estilo de YouTube o Netflix.

---

## 🎯 Propósito

StreamDaveFast es el primer paso hacia una plataforma de streaming profesional. Convierte cualquier archivo de video en un flujo adaptativo DASH que se ajusta automáticamente a la velocidad de internet del usuario, garantizando reproducción sin cortes.

### ¿Cómo funciona?

```
                ┌─────────────────────────────┐
   MP4/MKV   →  │   FFmpeg Transcoder (Go)    │
   subido       │   240p │ 480p │ 720p │ 1080p │
                └──────────┬──────────────────┘
                           ↓
                ┌─────────────────────────────┐
                │    Segmentos DASH (.m4s)     │
                │    + Manifiesto (.mpd)       │
                └──────────┬──────────────────┘
                           ↓
                ┌─────────────────────────────┐
                │       Shaka Player          │
                │  Bitrate Adaptativo Auto    │
                │  Buffer inteligente 30s     │
                └─────────────────────────────┘
```

> **Internet lento?** → Se reproduce en 240p sin cortes  
> **Internet rápido?** → Automáticamente sube a 1080p  
> **Internet inestable?** → Cambia de calidad al vuelo sin que el usuario lo note

---

## ⚡ Características

### Motor de Procesamiento (Backend)
- **Go (Golang)** como servidor HTTP de alto rendimiento
- **FFmpeg** para transcodificación profesional a múltiples calidades
- **Cola de procesamiento** con workers concurrentes (no bloquea el servidor)
- **CORS** habilitado para peticiones cross-origin
- **Range requests** para seeking instantáneo

### Escalera de Bitrate (Transcodificación)
| Calidad | Resolución | Video Bitrate | Audio | Caso de uso |
|---------|-----------|---------------|-------|-------------|
| 240p | 426×240 | 400 kbps | Master 128k | 2G/3G débil |
| 480p | 854×480 | 1,500 kbps | Master 128k | WiFi estable |
| 720p | 1280×720 | 3,000 kbps | Master 128k | 4G/Cable |
| 1080p | 1920×1080| 5,000 kbps | Master 128k | Fibra óptica |
| 1440p (2K) | 2560×1440| 9,000 kbps | Master 128k | High-end WiFi |
| 2160p (4K) | 3840×2160| 15,000 kbps| Master 128k | Fibra Giga |

### Reproductor Inteligente (Frontend)
- **Shaka Player** con bitrate adaptativo automático
- **MPEG-DASH** con segmentos de 4 segundos
- **Buffer inteligente**: 30s adelante, arranque rápido con 10s rebuffering goal
- **Estadísticas en vivo**: calidad, bitrate, buffer, frames perdidos
- **Controles YouTube Experience**: Autocultado tras inactividad, Click para Play/Pause, Doble-Click para Fullscreen
- **Subida de videos** con drag & drop
- **Procesamiento automático** al subir

---

## 🛠 Stack Tecnológico

| Componente | Tecnología | Propósito |
|-----------|-----------|-----------|
| **Backend** | Go (Golang) | Servidor HTTP, API REST, procesamiento |
| **Transcodificación** | FFmpeg (libx264 + AAC) | **Preset Slow** (Mejor compresión/calidad) |
| **Formato de salida** | MPEG-DASH (.mpd + .m4s) | Streaming adaptativo con **Single Audio Stream** |
| **Reproductor** | Shaka Player 4.x | ABR automático y controles personalizados |
| **Frontend** | HTML5, CSS3, JavaScript | Interfaz de usuario |
| **Tipografía** | Inter (Google Fonts) | Diseño moderno |
| **Iconos** | Font Awesome 6 | Iconografía |

---

## 📋 Requisitos

- **Go** 1.21+ instalado ([go.dev](https://go.dev))
- **FFmpeg** con soporte para `libx264` y `aac`
  ```bash
  # Verificar codecs
  ffmpeg -codecs | grep -E "libx264|aac"
  ```

---

## 🚀 Inicio Rápido

```bash
# 1. Clonar el proyecto
git clone <tu-repo>
cd Reproductor_web

# 2. Colocar videos en la carpeta Videos/
cp tu_video.mp4 Videos/

# 3. Iniciar el servidor
go run main.go

# 4. Abrir en el navegador
# http://localhost:8080
```

---

## 📡 API REST

### Listar videos
```
GET /api/videos
```
Devuelve la lista de videos con su estado (procesado o no).

### Subir video
```
POST /api/upload
Content-Type: multipart/form-data
Body: video=<archivo>&auto_process=true
```

### Procesar un video a DASH
```
POST /api/process
Content-Type: application/json
Body: {"file_name": "video.mp4"}
```

### Procesar todos los videos pendientes
```
POST /api/process/all
```

### Ver estado de trabajos
```
GET /api/jobs
GET /api/jobs/{job_id}
```

---

## ⌨️ Atajos de Teclado

| Tecla | Acción |
|-------|--------|
| `Espacio` | Play / Pausa |
| `←` / `→` | Retroceder / Avanzar 10s |
| `M` | Silenciar/Activar sonido |
| `F` | Pantalla completa |
| `N` | Video siguiente |
| `P` | Video anterior |

---

## 🗺 Roadmap (Plataforma de Streaming)

- [x] Servidor Go con API REST
- [x] Transcodificación FFmpeg multi-calidad
- [x] Segmentos DASH de 4 segundos
- [x] Shaka Player con ABR automático
- [x] Cola de procesamiento en background
- [x] Subida de videos con drag & drop
- [ ] Soporte HLS para dispositivos Apple
- [ ] Almacenamiento en Object Storage (S3/R2)
- [ ] CDN para distribución global
- [ ] SSL/HTTPS
- [ ] Autenticación de usuarios
- [ ] Thumbnails y previews
- [ ] Subtítulos (WebVTT)
- [ ] Soporte 4K y HDR

---

## 📄 Licencia

MIT License
