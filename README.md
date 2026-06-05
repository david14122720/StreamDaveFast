# 🚀 StreamDaveFast Ultra Rápido

**Plataforma de streaming de video de alto rendimiento**, optimizada para eliminar micro-stuttering, tearing y lag, incluso en calidades Full HD (1080p).

---

## 🎯 Propósito

StreamDaveFast es el primer paso hacia una plataforma de streaming profesional. Convierte cualquier archivo de video en un flujo adaptativo DASH que se ajusta automáticamente a la velocidad de internet del usuario, garantizando reproducción sin cortes y con latencia mínima gracias a su entrega desde RAM.

### ¿Cómo funciona el flujo optimizado?

```
                ┌─────────────────────────────┐
   MP4/MKV   →  │   FFmpeg Transcoder (Go)    │  →  Perfil High 4.1 (1080p)
   subido       │   GOP 5s | Keyframes Fixes  │     VBV Estricto (Anti-picos)
                │   Pass 1: DASH (encode)     │
                │   Pass 2: HLS  (remux -c copy)  → master.m3u8 + seg-*.ts
                └──────────┬──────────────────┘
                           ↓
                ┌─────────────────────────────┐  →  Cache LRU en RAM (256MB/10MB)
                │  Segmentos DASH (.m4s)      │     DASH + HLS comparten pool
                │  Segmentos HLS  (.ts)       │     Headers X-Cache: HIT
                │  Manifiestos .mpd / .m3u8   │
                └──────────┬──────────────────┘
                           ↓
                ┌─────────────────────────────┐  →  Buffer Meta: 20s
                │       Shaka Player          │     Ajuste Auto de Audio
                │  iOS Safari → HLS (.m3u8)   │     UI Throttling (250ms)
                │  Resto del mundo → DASH     │
                └─────────────────────────────┘
```

> **¿Internet lento?** → Se reproduce en 144p/240p sin cortes  
> **¿Internet rápido?** → Sube a 1080p con VBV controlado (Sin picos de datos)  
> **¿Reproducción fluida?** → Entrega desde RAM para eliminar la latencia del disco

---

## ⚡ Características "Ultra"
SISTEMA DE OPTIMIZACIÓN PRO:
- **Zero-Copy & Buffer Pool**: Implementación de `sync.Pool` en Go para reducir la presión del Garbage Collector y optimización de entrega vía `sendfile` para evitar copias innecesarias de memoria.
- **CRF Dinámico**: Ajuste inteligente de la calidad basada en la resolución (CRF 18 para 1080p, CRF 22 para resoluciones bajas) optimizando el espacio en disco sin sacrificar la experiencia visual.
- **RAM Cache System**: Los segmentos de video se cargan en la memoria RAM en el primer acceso y se sirven instantáneamente (latencia <1ms).
- **Limpieza Automática**: Recolector de basura inteligente que libera segmentos inactivos cada 30 segundos para optimizar el uso de memoria.
- **Workers Pool**: Procesamiento concurrente balanceado (CPU la I/O) que no bloquea la entrega de videos existentes.
- **Logging de Cache**: Headers HTTP técnicos para monitoreo y depuración de rendimiento.

### Escalera de Bitrate (Transcodificación FFmpeg)
| Calidad | Resolución | Video Bitrate | Profile/Level | VBV (Max/Buf) | CRF |
|---------|-----------|---------------|---------------|---------------|-----|
| 144p | 256x144 | 200 kbps | Main 4.0 | 300k / 600k | 20 |
| 240p | 426×240 | 400 kbps | Main 4.0 | 600k / 1.2M | 20 |
| 480p | 854×480 | 1,500 kbps | Main 4.0 | 2.2M / 4.5M | 20 |
| 720p | 1280×720 | 3,000 kbps | Main 4.0 | 4.5M / 9.0M | 20 |
| 1080p 🔥 | 1920×1080| 5,000 kbps| **High 4.1** | **6.0M / 12M** | **18** |

*   **Escalado Inteligente**: Usa `force_divisible_by=2` para evitar errores de codec con resoluciones impares.
*   **Filtro Lanczos**: Mejor calidad de escalado que el algoritmo por defecto.
*   **VBV Optimizado**: Mayor margen (50%) y buffer (3s) para evitar underflow en escenas de acción.
*   **FPS Nativos**: Detección dinámica de frames (23.98, 24, 30, 60fps) respetando la cadencia original.
*   **GOP Controlado**: `-x264-params "keyint=120:min-keyint=120"` para alineación perfecta de segmentos DASH.
*   **Audio CBR**: AAC a 128kbps constante y 48kHz para evitar "gaps" de silencio entre segmentos.

### Reproductor "YouTube Experience" (Frontend)
- **UI Optimizado**: Actualización de interfaz cada 250ms y obtención de estadísticas cada 2s para liberar el hilo principal del navegador.
- **Control Gestual**: Click simple para Play/Pausa, Doble-Click para Fullscreen, Rueda de ratón para Volumen.
- **Auto-Hide**: Controles inteligentes con transición suave que desaparecen tras 3s de inactividad.
- **Anti-Stuttering**: Configuración de Shaka Player para re-sincronizar automáticamente en caso de micro-gaps (<0.3s).
- **Buffer Adaptativo**: Meta conservadora de 20s de buffer adelante y 15s atrás para un equilibrio entre estabilidad y consumo de RAM.

---

## 🛠 Stack Tecnológico

| Componente | Tecnología | Detalle de Implementación |
|-----------|-----------|-----------|
| **Backend** | Go (Golang) | Servidor multi-hilo, LRU Cache en RAM |
| **Transcodificación** | FFmpeg (libx264) | **Preset Fast** con escalado Lanczos y VBV optimizado |
| **Formato de salida** | DASH (MP4 Fragments) | Segmentación dinámica con metadatos optimizados |
| **Reproductor** | Shaka Player (Google) | ABR habilitado con `lowerBitrateSwitching: true` |
| **Estilos** | CSS Moderno | Glassmorphism, animaciones suaves y modo oscuro nativo |

---

## 🛡️ Seguridad de Archivos

- **Borrado Seguro**: El archivo original solo se borra después de verificar que `manifest.mpd` existe y tiene contenido válido (>0 bytes).
- **Previene Pérdida de Datos**: Si la transcodificación falla, el video original se conserva para reintentar.

---

## ⚡️ Aceleración por Hardware

StreamDaveFast detecta automáticamente la aceleración por hardware disponible:

| Hardware | Encoder | Velocidad | Consumo |
|----------|---------|-----------|---------|
| **Intel GPU** | VAAPI | 5-10x más rápido | Bajo |
| **Intel GPU** | QSV | 5-10x más rápido | Bajo |
| **NVIDIA GPU** | NVENC | 10-20x más rápido | Medio |
| **CPU** | libx264 | Baseline | Alto |

### Verificar tu hardware:
```bash
# Intel VAAPI
vainfo

# FFmpeg con soporte de hardware
ffmpeg -hide_banner -encoders 2>&1 | grep -E "(vaapi|nvenc|qsv)"
```

---

## 🚀 Inicio Rápido

```bash
# 1. Clonar e ingresar
cd Reproductor_web

# 2. Compilar binario optimizado
go build -o streamdavefast .

# 3. Iniciar servidor
./streamdavefast

# 4. Acceder
# http://localhost:8080
```

### Codec de salida

Por defecto el servidor usa `STREAM_VIDEO_CODEC=auto`: intenta generar DASH en AV1, si FFmpeg no tiene AV1 usa H.265/HEVC y finalmente H.264 como fallback.

```bash
# Opciones: auto, av1, h265, h264
STREAM_VIDEO_CODEC=av1 ./streamdavefast
STREAM_VIDEO_CODEC=h265 ./streamdavefast
STREAM_VIDEO_CODEC=h264 ./streamdavefast
```

### HLS para iOS

Cada video produce dos manifests a partir del mismo encode (el pass HLS es un remux `-c copy` del ladder DASH, así que el costo de tiempo extra es ~0 y la calidad se preserva bit-a-bit):

| Manifest | MIME | Ruta |
|----------|------|------|
| DASH | `application/dash+xml` | `/processed/{video}/manifest.mpd` |
| HLS | `application/vnd.apple.mpegurl` | `/processed/{video}/master.m3u8` |
| HLS segments | `video/mp2t` | `/processed/{video}/seg-NNN.ts` |

Los segmentos DASH (`.m4s`) y HLS (`.ts`) comparten el mismo cache LRU en RAM (256 MB totales, 10 MB por entrada, eviction por `LastAccess`). Inspeccioná el uso en `/api/stats`:

```json
{ "cache_count": 42, "cache_bytes": 134217728, "cache_cap": 268435456, "status": "online" }
```

#### Regla de detección iOS

El frontend elige HLS sobre DASH sólo para Safari en iOS:

```js
const isIOS = /iP(hone|ad|od)|Safari/.test(navigator.userAgent)
  && !/CriOS|FxiOS|EdgiOS/.test(navigator.userAgent);
```

Los navegadores iOS de terceros (Chrome iOS, Firefox iOS, Edge iOS) traen su propio engine y SÍ soportan DASH, por eso los excluimos. Se usa sólo `navigator.userAgent` — `navigator.platform` está deprecado. La transformación `manifest.mpd` → `master.m3u8` se hace client-side vía `String.replace`, así que el backend no necesita un campo nuevo en `VideoInfo`.

### Códigos de respuesta de error

Todos los endpoints de la API devuelven errores en formato JSON:

```json
{ "error": "Mensaje legible por humanos" }
```

| Código | Cuándo | Endpoints |
|--------|--------|-----------|
| `400 Bad Request` | Body/filename inválido, sin video stream, JSON malformado | `/api/upload`, `/api/process`, `/api/delete`, `/api/jobs/{id}` |
| `404 Not Found` | Video o job inexistente | `/api/process`, `/api/jobs/{id}`, `/api/videos/{name}` |
| `405 Method Not Allowed` | Método HTTP no soportado | `/api/upload`, `/api/process`, `/api/process/all`, `/api/delete` |
| `500 Internal Server Error` | Fallo de I/O o FFmpeg inesperado | `/api/upload`, `/api/process/all`, `/api/delete` |

El segmento de video (GET `/processed/...`) usa los códigos HTTP estándar del `http.ServeFile` (200 / 206 / 404 / 416); el cuerpo en 4xx/5xx no es JSON.

---

## ⌨️ Atajos de Teclado

| Tecla | Acción |
|-------|--------|
| `Espacio` | Play / Pausa |
| `←` / `→` | Retroceder / Avanzar 10s |
| `M` | Silenciar / Activar sonido |
| `F` | Pantalla completa (Toggle) |
| `N` / `P` | Siguiente / Anterior video |

---

## 🗺 Roadmap Completado
- [x] Transcodificación con VBV Estricto para 1080p (Sin tirones)
- [x] Sistema de Cache en RAM con limpieza automática
- [x] UI de Reproductor tipo YouTube (Gestos y Auto-hide)
- [x] Detección de FPS originales y alineación de Keyframes
- [x] Reducción de carga en el hilo principal del navegador (UI Throttling)
- [x] Soporte para nombres de archivos con caracteres especiales
- [x] **Aceleración por hardware (VAAPI/QSV/NVENC)** - 5-10x más rápido
- [x] **Soporte HLS para dispositivos iOS** (remux desde DASH, cache LRU compartido)
- [ ] Thumbnails de previsualización al pasar el ratón por la barra
- [ ] Soporte para subtítulos externos (SRT/VTT)

---

## 📄 Licencia
MIT License 2026 - StreamDaveFast Project

---
*Desarrollado con ❤️ para una experiencia de streaming ultra fluida.*
