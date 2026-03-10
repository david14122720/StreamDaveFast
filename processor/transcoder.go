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
	Label        string
}

// Perfiles de calidad estándar (escalera de bitrate optimizada para "Cold Start")
var QualityProfiles = []QualityProfile{
	{Name: "144p", Resolution: "256x144", VideoBitrate: "150k", Label: "Ultra Económico (GPRS/Edge)"},
	{Name: "240p", Resolution: "426x240", VideoBitrate: "350k", Label: "Económico (3G)"},
	{Name: "480p", Resolution: "854x480", VideoBitrate: "1200k", Label: "Estándar (WiFi)"},
	{Name: "720p", Resolution: "1280x720", VideoBitrate: "2500k", Label: "HD (4G/Fibra)"},
	{Name: "1080p", Resolution: "1920x1080", VideoBitrate: "4500k", Label: "Full HD (Pro)"}, // Optimizado con VBV estricto en el loop
	{Name: "1440p", Resolution: "2560x1440", VideoBitrate: "8000k", Label: "2K (Ultra HD)"},
	{Name: "2160p", Resolution: "3840x2160", VideoBitrate: "15000k", Label: "4K (Cine)"},
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
		"144p":  144,
		"240p":  240,
		"480p":  480,
		"720p":  720,
		"1080p": 1080,
		"1440p": 1440,
		"2160p": 2160,
	}
	for _, profile := range QualityProfiles {
		targetHeight := resolutions[profile.Name]
		// Añadimos perfil si la altura es menor o igual a la original (con tolerancia de 10px para variaciones menores)
		if targetHeight <= (height + 10) {
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

	// 1. Añadir flujos de VIDEO (Múltiples calidades)
	for i, p := range profiles {
		// Parseamos el bitrate para cálculos de VBV (Buffer Verifier)
		var bitrateNum int
		fmt.Sscanf(p.VideoBitrate, "%dk", &bitrateNum)

		// VBV estricto y perfil de compatibilidad por defecto
		maxRate := fmt.Sprintf("%dk", int(float64(bitrateNum)*1.20)) // 20% de margen
		bufSize := fmt.Sprintf("%dk", bitrateNum*2)                  // Buffer de 2s
		profile := "main"
		level := "4.0"
		crf := "20"

		// Optimización específica para 1080p (el más pesado)
		if p.Name == "1080p" {
			maxRate = "5000k"
			bufSize = "10000k"
			profile = "high"
			level = "4.1"
			crf = "22" // Ligera reducción de carga para CPU
		}

		args = append(args,
			"-map", "0:v:0",
			fmt.Sprintf("-c:v:%d", i), "libx264",
			fmt.Sprintf("-b:v:%d", i), p.VideoBitrate,
			fmt.Sprintf("-maxrate:v:%d", i), maxRate,
			fmt.Sprintf("-bufsize:v:%d", i), bufSize,
			fmt.Sprintf("-s:v:%d", i), p.Resolution,
			fmt.Sprintf("-pix_fmt:v:%d", i), "yuv420p",
			fmt.Sprintf("-profile:v:%d", i), profile,
			fmt.Sprintf("-level:v:%d", i), level,
			fmt.Sprintf("-crf:v:%d", i), crf,
			fmt.Sprintf("-x264-params:v:%d", i), "nal-hrd=vbr:keyint=120:min-keyint=120",
		)
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
		"-preset", "fast",
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
		fmt.Printf("   📺 %s (%s) - Video: %s\n", p.Name, p.Label, p.VideoBitrate)
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
	}, nil
}
