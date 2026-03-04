# 🚀 StreamDaveFast

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
