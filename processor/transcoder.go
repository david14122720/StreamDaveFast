package processor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// QualityProfile define una variante de calidad para la transcodificación
type QualityProfile struct {
	Name         string
	Resolution   string
	VideoBitrate string
	AudioBitrate string
	Label        string
}

// Perfiles de calidad estándar (escalera de bitrate)
var QualityProfiles = []QualityProfile{
	{Name: "240p", Resolution: "426x240", VideoBitrate: "400k", AudioBitrate: "64k", Label: "Baja (2G/3G)"},
	{Name: "480p", Resolution: "854x480", VideoBitrate: "1500k", AudioBitrate: "128k", Label: "Media (WiFi)"},
	{Name: "720p", Resolution: "1280x720", VideoBitrate: "3000k", AudioBitrate: "192k", Label: "Alta (4G/Cable)"},
	{Name: "1080p", Resolution: "1920x1080", VideoBitrate: "5000k", AudioBitrate: "256k", Label: "Full HD (Fibra)"},
}

// TranscodeResult contiene información sobre la transcodificación completada
type TranscodeResult struct {
	VideoName    string    `json:"video_name"`
	ManifestPath string    `json:"manifest_path"`
	Qualities    []string  `json:"qualities"`
	Duration     float64   `json:"duration_seconds"`
	ProcessedAt  time.Time `json:"processed_at"`
}

// CheckFFmpeg verifica que FFmpeg esté instalado y tenga los codecs necesarios
func CheckFFmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	return cmd.Run()
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

// SelectProfiles elige los perfiles de calidad adecuados según la resolución del video original
func SelectProfiles(width, height int) []QualityProfile {
	var selected []QualityProfile
	resolutions := map[string]int{
		"240p":  240,
		"480p":  480,
		"720p":  720,
		"1080p": 1080,
	}
	for _, profile := range QualityProfiles {
		targetHeight := resolutions[profile.Name]
		if targetHeight <= height {
			selected = append(selected, profile)
		}
	}
	// Si el video es muy pequeño, al menos incluir la calidad más baja
	if len(selected) == 0 {
		selected = append(selected, QualityProfiles[0])
	}
	return selected
}

// TranscodeVideo procesa un video a DASH con múltiples calidades
func TranscodeVideo(inputPath string, outputDir string) (*TranscodeResult, error) {
	startTime := time.Now()

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
	args := []string{
		"-i", inputPath,
		"-y", // Sobrescribir sin preguntar
	}

	// Añadir cada perfil de calidad como stream separado
	for i, p := range profiles {
		args = append(args,
			// Map video y audio para este perfil
			"-map", "0:v:0",
			"-map", "0:a:0?", // el ? evita fallo si no hay audio

			// Codec de video H.264
			fmt.Sprintf("-c:v:%d", i), "libx264",
			fmt.Sprintf("-b:v:%d", i), p.VideoBitrate,
			fmt.Sprintf("-s:v:%d", i), p.Resolution,
			fmt.Sprintf("-profile:v:%d", i), "main",

			// Codec de audio AAC
			fmt.Sprintf("-c:a:%d", i), "aac",
			fmt.Sprintf("-b:a:%d", i), p.AudioBitrate,
			fmt.Sprintf("-ac:%d", i), "2",
		)
	}

	// Opciones globales de encoding (aplican a todos los streams de video)
	args = append(args,
		"-preset", "fast", // Balance velocidad/calidad
		"-g", "48", // Keyframe cada 48 frames (~2s a 24fps)
		"-keyint_min", "48", // Forzar keyframes regulares
		"-sc_threshold", "0", // Desactivar scene detection para keyframes predecibles
	)

	// Configuración DASH
	args = append(args,
		"-f", "dash",
		"-seg_duration", "4", // Segmentos de 4 segundos
		"-use_timeline", "1", // Timeline para Shaka
		"-use_template", "1", // Templates para nombres de segmento
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
	for _, p := range profiles {
		fmt.Printf("   📺 %s (%s) - Video: %s, Audio: %s\n", p.Name, p.Label, p.VideoBitrate, p.AudioBitrate)
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error en transcodificación: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("✅ Transcodificación completada en %s\n", elapsed.Round(time.Second))

	// Obtener nombres de calidades
	qualityNames := make([]string, len(profiles))
	for i, p := range profiles {
		qualityNames[i] = p.Name
	}

	return &TranscodeResult{
		VideoName:    filepath.Base(inputPath),
		ManifestPath: filepath.Join(outputDir, "manifest.mpd"),
		Qualities:    qualityNames,
		Duration:     elapsed.Seconds(),
		ProcessedAt:  time.Now(),
	}, nil
}
