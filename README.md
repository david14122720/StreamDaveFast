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
                └──────────┬──────────────────┘
                           ↓
                ┌─────────────────────────────┐  →  Cache en RAM (LRU)
                │    Segmentos DASH (.m4s)     │     Serving instantáneo
                │    + Manifiesto (.mpd)       │     Headers X-Cache: HIT
                └──────────┬──────────────────┘
                           ↓
                ┌─────────────────────────────┐  →  Buffer Meta: 20s
                │       Shaka Player          │     Ajuste Auto de Audio
                │  Bitrate Adaptativo (ABR)   │     UI Throttling (250ms)
                └─────────────────────────────┘
```

> **¿Internet lento?** → Se reproduce en 144p/240p sin cortes  
> **¿Internet rápido?** → Sube a 1080p con VBV controlado (Sin picos de datos)  
> **¿Reproducción fluida?** → Entrega desde RAM para eliminar la latencia del disco

---

## ⚡ Características "Ultra"

### Motor de Procesamiento (Backend en Go)
- **RAM Cache System**: Los segmentos de video se cargan en la memoria RAM en el primer acceso y se sirven instantáneamente (latencia <1ms).
- **Limpieza Automática**: Recolector de basura inteligente que libera segmentos inactivos cada 30 segundos para optimizar el uso de memoria.
- **Workers Pool**: Procesamiento concurrente balanceado (CPU vs I/O) que no bloquea la entrega de videos existentes.
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
- [ ] Soporte HLS para dispositivos iOS
- [ ] Thumbnails de previsualización al pasar el ratón por la barra
- [ ] Soporte para subtítulos externos (SRT/VTT)

---

## 📄 Licencia
MIT License 2026 - StreamDaveFast Project

---
*Desarrollado con ❤️ para una experiencia de streaming ultra fluida.*
