package processor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// QualityProfile define una variante de calidad para la transcodificación
type QualityProfile struct {
	Name         string
	MaxWidth     int
	MaxHeight    int
	VideoBitrate string
	Label        string
}

// VideoCodec define el codec de salida usado para el DASH.
type VideoCodec string

const (
	CodecH264 VideoCodec = "h264"
	CodecH265 VideoCodec = "h265"
	CodecAV1  VideoCodec = "av1"
)

// EncoderConfig contiene la configuración concreta que se pasará a FFmpeg.
type EncoderConfig struct {
	Codec       VideoCodec
	Encoder     string
	DisplayName string
	BitrateRate float64
}

// CheckFFmpeg verifica que FFmpeg esté instalado y tenga los codecs necesarios
func CheckFFmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	return cmd.Run()
}

// Perfiles de calidad estándar (escalera de bitrate optimizada para evitar VBV underflow)
var QualityProfiles = []QualityProfile{
	{Name: "144p", MaxWidth: 256, MaxHeight: 144, VideoBitrate: "200k", Label: "Ultra Económico (GPRS/Edge)"},
	{Name: "240p", MaxWidth: 426, MaxHeight: 240, VideoBitrate: "400k", Label: "Económico (3G)"},
	{Name: "480p", MaxWidth: 854, MaxHeight: 480, VideoBitrate: "1500k", Label: "Estándar (WiFi)"},
	{Name: "720p", MaxWidth: 1280, MaxHeight: 720, VideoBitrate: "3000k", Label: "HD (4G/Fibra)"},
	{Name: "1080p", MaxWidth: 1920, MaxHeight: 1080, VideoBitrate: "5000k", Label: "Full HD (Pro)"},
	{Name: "1440p", MaxWidth: 2560, MaxHeight: 1440, VideoBitrate: "8000k", Label: "2K (Ultra HD)"},
	{Name: "2160p", MaxWidth: 3840, MaxHeight: 2160, VideoBitrate: "15000k", Label: "4K (Cine)"},
}

// TranscodeResult contiene información sobre la transcodificación completada
type TranscodeResult struct {
	VideoName    string    `json:"video_name"`
	ManifestPath string    `json:"manifest_path"`
	Qualities    []string  `json:"qualities"`
	Duration     float64   `json:"duration_seconds"`
	ProcessedAt  time.Time `json:"processed_at"`
	Codec        string    `json:"codec"`
	Encoder      string    `json:"encoder"`
	HLSManifest  string    `json:"hls_manifest,omitempty"`
}

// HardwareDetector detecta qué aceleración por hardware está disponible
type HardwareDetector struct {
	VAAPI bool
	QSV   bool
	NVENC bool
}

// DetectHardware detecta la aceleración por hardware disponible
func DetectHardware() HardwareDetector {
	detector := HardwareDetector{}

	// Verificar VAAPI
	cmd := exec.Command("ffmpeg", "-hide_banner", "-encoders", "2>/dev/null")
	output, _ := cmd.Output()
	outputStr := string(output)

	if strings.Contains(outputStr, "h264_vaapi") || strings.Contains(outputStr, "hevc_vaapi") {
		vaCmd := exec.Command("vainfo")
		if vaCmd.Run() == nil {
			detector.VAAPI = true
		}
	}

	// Verificar QSV
	if strings.Contains(outputStr, "h264_qsv") {
		detector.QSV = true
	}

	// Verificar NVENC
	if strings.Contains(outputStr, "h264_nvenc") {
		detector.NVENC = true
	}

	return detector
}

// GetEncoderConfig devuelve la configuración del encoder basada en el hardware disponible
func GetEncoderConfig() EncoderConfig {
	available := AvailableEncoders()
	preferred := strings.ToLower(strings.TrimSpace(os.Getenv("STREAM_VIDEO_CODEC")))
	if preferred == "" {
		preferred = "auto"
	}

	candidates := []VideoCodec{CodecAV1, CodecH265, CodecH264}
	switch preferred {
	case "av1":
		candidates = []VideoCodec{CodecAV1, CodecH265, CodecH264}
	case "h265", "hevc":
		candidates = []VideoCodec{CodecH265, CodecAV1, CodecH264}
	case "h264", "avc":
		candidates = []VideoCodec{CodecH264, CodecAV1, CodecH265}
	case "auto":
	default:
		fmt.Printf("⚠️ STREAM_VIDEO_CODEC=%q no reconocido; usando auto\n", preferred)
	}

	for _, codec := range candidates {
		if config, ok := encoderForCodec(codec, available); ok {
			return config
		}
	}

	return EncoderConfig{Codec: CodecH264, Encoder: "libx264", DisplayName: "H.264 / AVC", BitrateRate: 1.0}
}

// AvailableEncoders devuelve los encoders de FFmpeg disponibles.
func AvailableEncoders() string {
	cmd := exec.Command("ffmpeg", "-hide_banner", "-encoders")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return string(output)
}

func encoderForCodec(codec VideoCodec, available string) (EncoderConfig, bool) {
	switch codec {
	case CodecAV1:
		if strings.Contains(available, "libsvtav1") {
			return EncoderConfig{Codec: CodecAV1, Encoder: "libsvtav1", DisplayName: "AV1", BitrateRate: 0.55}, true
		}
		if strings.Contains(available, "libaom-av1") {
			return EncoderConfig{Codec: CodecAV1, Encoder: "libaom-av1", DisplayName: "AV1", BitrateRate: 0.55}, true
		}
	case CodecH265:
		if strings.Contains(available, "libx265") {
			return EncoderConfig{Codec: CodecH265, Encoder: "libx265", DisplayName: "H.265 / HEVC", BitrateRate: 0.65}, true
		}
	case CodecH264:
		if strings.Contains(available, "libx264") {
			return EncoderConfig{Codec: CodecH264, Encoder: "libx264", DisplayName: "H.264 / AVC", BitrateRate: 1.0}, true
		}
	}
	return EncoderConfig{}, false
}

// GetVideoDuration obtiene la duración de un video en segundos
func GetVideoDuration(inputPath string) (string, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return "0", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetVideoResolution obtiene la resolución del video original
func GetVideoResolution(inputPath string) (int, int, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		inputPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	var w, h int
	fmt.Sscanf(strings.TrimSpace(string(output)), "%dx%d", &w, &h)
	return w, h, nil
}

// HasVideoStream valida que FFprobe pueda leer al menos un stream de video.
func HasVideoStream(inputPath string) error {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_type",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(output)) != "video" {
		return fmt.Errorf("no se encontró un stream de video válido")
	}
	return nil
}

// SelectProfiles elige los perfiles de calidad adecuados según la resolución del video original
func SelectProfiles(width, height int) []QualityProfile {
	var selected []QualityProfile
	shortEdge := width
	if height < shortEdge {
		shortEdge = height
	}
	for _, profile := range QualityProfiles {
		targetShortEdge := profile.MaxHeight
		// La calidad se decide por el lado corto para soportar horizontal, vertical, cuadrado y ultrawide.
		if targetShortEdge <= (shortEdge + 10) {
			selected = append(selected, profile)
		}
	}
	// Si el video es muy pequeño, al menos incluir la calidad más baja
	if len(selected) == 0 {
		selected = append(selected, QualityProfiles[0])
	}
	return selected
}

func adjustedBitrate(profile QualityProfile, config EncoderConfig) string {
	var bitrateNum int
	fmt.Sscanf(profile.VideoBitrate, "%dk", &bitrateNum)
	if bitrateNum <= 0 {
		return profile.VideoBitrate
	}
	return fmt.Sprintf("%dk", int(float64(bitrateNum)*config.BitrateRate))
}

func dashScaleFilter(profile QualityProfile) string {
	landscapeWidth := profile.MaxWidth
	landscapeHeight := profile.MaxHeight
	portraitWidth := profile.MaxHeight
	portraitHeight := profile.MaxWidth
	return fmt.Sprintf(
		"scale=w='if(gte(iw,ih),%d,%d)':h='if(gte(iw,ih),%d,%d)':force_original_aspect_ratio=decrease:force_divisible_by=2:flags=lanczos",
		landscapeWidth,
		portraitWidth,
		landscapeHeight,
		portraitHeight,
	)
}

func appendCodecOptions(args []string, streamIndex int, profile QualityProfile, config EncoderConfig, videoBitrate string, maxRate string, bufSize string) []string {
	args = append(args,
		fmt.Sprintf("-c:v:%d", streamIndex), config.Encoder,
		fmt.Sprintf("-b:v:%d", streamIndex), videoBitrate,
		"-filter:v:"+fmt.Sprintf("%d", streamIndex), dashScaleFilter(profile),
		fmt.Sprintf("-g:v:%d", streamIndex), "120",
	)

	switch config.Codec {
	case CodecAV1:
		args = append(args,
			fmt.Sprintf("-preset:v:%d", streamIndex), "8",
			fmt.Sprintf("-svtav1-params:v:%d", streamIndex), "rc=1",
			fmt.Sprintf("-pix_fmt:v:%d", streamIndex), "yuv420p",
		)
	case CodecH265:
		args = append(args,
			fmt.Sprintf("-maxrate:v:%d", streamIndex), maxRate,
			fmt.Sprintf("-bufsize:v:%d", streamIndex), bufSize,
			fmt.Sprintf("-preset:v:%d", streamIndex), "fast",
			fmt.Sprintf("-crf:v:%d", streamIndex), "24",
			fmt.Sprintf("-tag:v:%d", streamIndex), "hvc1",
			fmt.Sprintf("-x265-params:v:%d", streamIndex), "keyint=120:min-keyint=120:scenecut=0:open-gop=0",
			fmt.Sprintf("-pix_fmt:v:%d", streamIndex), "yuv420p",
		)
	case CodecH264:
		h264Profile := "main"
		level := "4.0"
		// IMPLEMENTACIÓN DE CRF DINÁMICO:
		// Para 1080p usamos CRF 18 (Alta calidad), para el resto usamos 22
		// Esto optimiza el espacio sin sacrificar la experiencia en Full HD.
		crf := "22"
		if profile.Name == "1080p" {
			h264Profile = "high"
			level = "4.1"
			crf = "18"
		}
		args = append(args,
			fmt.Sprintf("-maxrate:v:%d", streamIndex), maxRate,
			fmt.Sprintf("-bufsize:v:%d", streamIndex), bufSize,
			fmt.Sprintf("-preset:v:%d", streamIndex), "fast",
			fmt.Sprintf("-profile:v:%d", streamIndex), h264Profile,
			fmt.Sprintf("-level:v:%d", streamIndex), level,
			fmt.Sprintf("-crf:v:%d", streamIndex), crf,
			fmt.Sprintf("-x264-params:v:%d", streamIndex), "nal-hrd=vbr:keyint=120:min-keyint=120",
			fmt.Sprintf("-pix_fmt:v:%d", streamIndex), "yuv420p",
		)
	}

	return args
}

// TranscodeVideo procesa un video a DASH con múltiples calidades
func TranscodeVideo(inputPath string, outputDir string) (*TranscodeResult, error) {
	startTime := time.Now()

	// Detectar hardware disponible
	hw := DetectHardware()
	encoderConfig := GetEncoderConfig()

	if hw.VAAPI {
		fmt.Printf("🟡 VAAPI detectado; usando encoder de software para compatibilidad DASH multi-stream\n")
	} else if hw.QSV {
		fmt.Printf("🟡 QSV detectado; usando encoder de software para compatibilidad DASH multi-stream\n")
	} else if hw.NVENC {
		fmt.Printf("🟡 NVENC detectado; usando encoder de software para compatibilidad DASH multi-stream\n")
	} else {
		fmt.Printf("🔴 Usando CPU (%s)\n", encoderConfig.Encoder)
	}
	fmt.Printf("🎞️ Codec de salida: %s (%s)\n", encoderConfig.DisplayName, encoderConfig.Encoder)

	// Limpiar directorio de salida si existe
	if _, err := os.Stat(outputDir); err == nil {
		fmt.Printf("🗑️ Limpiando directorio existente: %s\n", outputDir)
		if err := os.RemoveAll(outputDir); err != nil {
			return nil, fmt.Errorf("error limpiando directorio: %w", err)
		}
	}

	// Crear directorio de salida
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("error creando directorio de salida: %w", err)
	}

	// Obtener resolución del video original
	width, height, err := GetVideoResolution(inputPath)
	if err != nil {
		fmt.Printf("⚠️  No se pudo detectar resolución, usando todos los perfiles: %v\n", err)
		width, height = 1920, 1080
	}
	fmt.Printf("📐 Resolución original: %dx%d\n", width, height)

	// Seleccionar perfiles adecuados
	profiles := SelectProfiles(width, height)
	fmt.Printf("🎯 Perfiles seleccionados: %d variantes\n", len(profiles))

	// Construir argumentos de FFmpeg
	args := []string{}

	args = append(args,
		"-i", inputPath,
		"-y", // Sobrescribir sin preguntar
	)

	// 1. Añadir flujos de VIDEO (Múltiples calidades)
	for i, p := range profiles {
		// Parseamos el bitrate para cálculos de VBV (Buffer Verifier)
		var bitrateNum int
		videoBitrate := adjustedBitrate(p, encoderConfig)
		fmt.Sscanf(videoBitrate, "%dk", &bitrateNum)

		// VBV con más margen para evitar underflow en escenas de acción
		maxRate := fmt.Sprintf("%dk", int(float64(bitrateNum)*1.5)) // 50% de margen (antes era 20%)
		bufSize := fmt.Sprintf("%dk", bitrateNum*3)                 // Buffer de 3s (antes era 2s)

		// Optimización específica para 1080p (mayor calidad)
		if p.Name == "1080p" && encoderConfig.Codec == CodecH264 {
			maxRate = "6000k"
			bufSize = "12000k"
		}

		args = append(args,
			"-map", "0:v:0",
		)
		args = appendCodecOptions(args, i, p, encoderConfig, videoBitrate, maxRate, bufSize)
	}

	// 2. Añadir flujo de AUDIO ÚNICO (Master Audio)
	// Usamos un solo flujo de audio para todas las calidades para evitar cortes al cambiar de resolución
	args = append(args,
		"-map", "0:a:0?",
		"-c:a:0", "aac",
		"-b:a:0", "128k",
		"-maxrate:a:0", "128k", // CBR Audio (Clave para evitar micro-cortes)
		"-bufsize:a:0", "128k",
		"-ar:0", "48000", // 48kHz (División perfecta para segmentos de 5s)
		"-ac:0", "2",
		"-af", "aresample=async=1:first_pts=0", // Alineación de audio al tiempo cero
	)

	// Opciones globales de encoding (para calidad pro y concurrencia)
	args = append(args,
		"-threads", "0",
		"-force_key_frames", "expr:gte(t,n_forced*5)", // Keyframe basado en tiempo real
		"-sc_threshold", "0", // Desactivar detección de cambio de escena
		"-avoid_negative_ts", "make_zero",
		"-map_metadata", "-1",
		"-movflags", "+faststart",
	)

	// Configuración DASH (Nivel Pro)
	args = append(args,
		"-f", "dash",
		"-seg_duration", "5",
		"-index_correction", "1",
		"-use_timeline", "1",
		"-use_template", "1",
		"-dash_segment_type", "mp4", // Asegura formato compatible sin cabeceras extra
		"-init_seg_name", "init-$RepresentationID$.m4s",
		"-media_seg_name", "chunk-$RepresentationID$-$Number%05d$.m4s",
		"-adaptation_sets", "id=0,streams=v id=1,streams=a",
		filepath.Join(outputDir, "manifest.mpd"),
	)

	cmd := exec.Command("ffmpeg", args...)

	// Capturar salida para logs
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	fmt.Printf("🎬 Iniciando transcodificación de %s...\n", filepath.Base(inputPath))
	fmt.Printf("   🎵 Audio: Master Audio AAC 128kbps (Continuo)\n")
	for _, p := range profiles {
		fmt.Printf("   📺 %s (%s) - Video: %s\n", p.Name, p.Label, adjustedBitrate(p, encoderConfig))
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error en transcodificación: %w", err)
	}

	// Pass 2: HLS muxer (remux only, no re-encode).
	// Reads the DASH encoded ladder from disk and rewraps it as HLS
	// (.m3u8 + .ts segments) so iOS Safari can play the same encoded
	// content via Shaka's HLS engine. See decision A2.
	hlsManifestPath := filepath.Join(outputDir, "master.m3u8")
	hlsArgs := []string{
		"-y",
		"-i", filepath.Join(outputDir, "manifest.mpd"),
		"-c", "copy",
		"-f", "hls",
		"-hls_time", "6",
		"-hls_segment_filename", filepath.Join(outputDir, "seg-%03d.ts"),
		"-master_pl_name", "master.m3u8",
		hlsManifestPath,
	}
	hlsCmd := exec.Command("ffmpeg", hlsArgs...)
	hlsCmd.Stderr = os.Stderr
	hlsCmd.Stdout = os.Stdout
	fmt.Printf("📺 Generando HLS (remux desde DASH encoded ladder)...\n")
	if err := hlsCmd.Run(); err != nil {
		return nil, fmt.Errorf("error en pass HLS: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("✅ Transcodificación completada en %s\n", elapsed.Round(time.Second))

	// Obtener nombres de calidades
	qualityNames := make([]string, len(profiles))
	for i, p := range profiles {
		qualityNames[i] = p.Name
	}
	duration := 0.0
	if durationText, err := GetVideoDuration(inputPath); err == nil {
		duration, _ = strconv.ParseFloat(durationText, 64)
	}

	return &TranscodeResult{
		VideoName:    filepath.Base(inputPath),
		ManifestPath: filepath.Join(outputDir, "manifest.mpd"),
		Qualities:    qualityNames,
		Duration:     duration,
		ProcessedAt:  time.Now(),
		Codec:        string(encoderConfig.Codec),
		Encoder:      encoderConfig.Encoder,
		HLSManifest:  hlsManifestPath,
	}, nil
}
