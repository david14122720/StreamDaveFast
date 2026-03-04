package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"web-player-backend/processor"
)

// RAM Cache para segmentos (Optimización de velocidad)
type CachedSegment struct {
	Data       []byte
	LastAccess time.Time
}

var (
	segmentCache = make(map[string]*CachedSegment)
	cacheMutex   sync.Mutex
)

var queue *processor.Queue

// VideoInfo contiene la información de un video para la API
type VideoInfo struct {
	Name        string `json:"name"`
	FileName    string `json:"file_name"`
	Size        int64  `json:"size"`
	IsProcessed bool   `json:"is_processed"`
	ManifestURL string `json:"manifest_url,omitempty"`
	DirectURL   string `json:"direct_url,omitempty"`
}

func main() {
	// Verificar FFmpeg
	if err := processor.CheckFFmpeg(); err != nil {
		fmt.Println("⚠️  ADVERTENCIA: FFmpeg no encontrado. La transcodificación no funcionará.")
		fmt.Println("   Por favor, asegúrate de que FFmpeg esté instalado y en tu PATH.")
	} else {
		fmt.Println("✅ FFmpeg detectado correctamente.")
	}

	// Crear directorios necesarios
	os.MkdirAll("./Videos", 0755)
	os.MkdirAll("./processed", 0755)

	// Iniciar limpiador de RAM (borra segmentos inactivos cada 30 segundos)
	go func() {
		for {
			time.Sleep(30 * time.Second)
			cacheMutex.Lock()
			count := 0
			for path, seg := range segmentCache {
				if time.Since(seg.LastAccess) > 30*time.Second {
					delete(segmentCache, path)
					count++
				}
			}
			if count > 0 {
				fmt.Printf("🧹 Limpiador RAM: Se liberaron %d segmentos (30s inactividad)\n", count)
			}
			cacheMutex.Unlock()
		}
	}()

	// Iniciar cola de procesamiento (Detección dinámica de núcleos)
	numWorkers := runtime.NumCPU()
	if numWorkers > 4 {
		numWorkers = 2 // No saturar si hay muchos núcleos, FFmpeg ya usa hilos
	} else if numWorkers < 1 {
		numWorkers = 1
	}
	fmt.Printf("👷 Iniciando pool de procesamiento con %d workers\n", numWorkers)
	queue = processor.NewQueue(numWorkers)
	defer queue.Close()

	// Crear mux para manejar rutas
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/videos", handleListVideos)
	mux.HandleFunc("/api/videos/", handleGetVideo)
	mux.HandleFunc("/api/upload", handleUploadVideo)
	mux.HandleFunc("/api/process", handleProcessVideo)
	mux.HandleFunc("/api/process/all", handleProcessAllVideos)
	mux.HandleFunc("/api/jobs", handleListJobs)
	mux.HandleFunc("/api/jobs/", handleGetJob)
	mux.HandleFunc("/api/delete", handleDeleteVideo)

	// Servir segmentos DASH con headers correctos
	mux.HandleFunc("/processed/", handleDASHFiles)

	// Servir videos directos con soporte de Range requests
	mux.HandleFunc("/Videos/", handleVideoFiles)

	// Servir archivos estáticos del frontend
	mux.Handle("/", http.FileServer(http.Dir("./")))

	port := "8080"
	localIP := getLocalIP()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("🚀 StreamDaveFast Ultra iniciado\n")
	fmt.Printf("🏠 Local:   http://localhost:%s\n", port)
	if localIP != "" {
		fmt.Printf("🌐 Red:     http://%s:%s\n", localIP, port)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📁 Videos:     ./Videos/")
	fmt.Println("📦 Procesados: ./processed/")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Servidor HTTP con CORS (escuchando en todas las interfaces)
	handler := corsMiddleware(mux)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// getLocalIP intenta obtener la IP local real de la red
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// corsMiddleware agrega headers CORS a todas las respuestas
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Range")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleDASHFiles sirve archivos DASH optimizados con RAM y sendfile
func handleDASHFiles(w http.ResponseWriter, r *http.Request) {
	filePath := "." + r.URL.Path
	ext := filepath.Ext(filePath)

	// Solo cacheamos segmentos (.m4s), no el manifiesto (.mpd) para evitar desincronización
	if ext == ".m4s" {
		cacheMutex.Lock()
		if seg, exists := segmentCache[filePath]; exists {
			seg.LastAccess = time.Now()
			cacheMutex.Unlock()
			w.Header().Set("X-Cache", "HIT-RAM")
			w.Header().Set("Content-Type", "video/iso.segment")
			w.Write(seg.Data)
			return
		}
		cacheMutex.Unlock()

		// Si no está en cache, leerlo y guardarlo
		data, err := os.ReadFile(filePath)
		if err == nil {
			cacheMutex.Lock()
			segmentCache[filePath] = &CachedSegment{
				Data:       data,
				LastAccess: time.Now(),
			}
			cacheMutex.Unlock()
		}
	}

	// Establecer Content-Type correcto
	switch ext {
	case ".mpd":
		w.Header().Set("Content-Type", "application/dash+xml")
		w.Header().Set("Cache-Control", "no-cache")
	case ".m4s":
		w.Header().Set("Content-Type", "video/iso.segment")
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	}

	// Usar http.ServeFile para activar 'sendfile' (Kernel a Tarjeta Red sin CPU)
	http.ServeFile(w, r, filePath)
}

// handleVideoFiles sirve videos con soporte de Range requests para seeking
func handleVideoFiles(w http.ResponseWriter, r *http.Request) {
	filePath := "." + r.URL.Path
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeFile(w, r, filePath)
}

// handleListVideos devuelve la lista de videos disponibles (originales y procesados)
func handleListVideos(w http.ResponseWriter, r *http.Request) {
	// Mapa para agrupar videos por nombre
	videoMap := make(map[string]*VideoInfo)

	// 1. Escanear videos originales subidos (si aún existen)
	files, _ := os.ReadDir("./Videos")
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext != ".mp4" && ext != ".mkv" && ext != ".webm" && ext != ".avi" && ext != ".mov" {
			continue
		}

		name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		info, _ := file.Info()

		v := &VideoInfo{
			Name:      name,
			FileName:  file.Name(),
			Size:      info.Size(),
			DirectURL: "/Videos/" + file.Name(),
		}
		videoMap[sanitizeName(name)] = v
	}

	// 2. Escanear videos ya procesados
	processedDirs, _ := os.ReadDir("./processed")
	for _, dir := range processedDirs {
		if !dir.IsDir() {
			continue
		}

		dirName := dir.Name()
		manifestPath := filepath.Join("processed", dirName, "manifest.mpd")

		if _, err := os.Stat(manifestPath); err == nil {
			v, exists := videoMap[dirName]
			if !exists {
				v = &VideoInfo{
					Name: dirName,
				}
				videoMap[dirName] = v
			}
			v.IsProcessed = true
			v.ManifestURL = "/" + manifestPath
		}
	}

	var videos []VideoInfo
	for _, v := range videoMap {
		videos = append(videos, *v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(videos)
}

// handleUploadVideo maneja la subida de videos
func handleUploadVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(2 << 30)

	file, header, err := r.FormFile("video")
	if err != nil {
		jsonError(w, "Error al leer el archivo: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := map[string]bool{".mp4": true, ".mkv": true, ".webm": true, ".avi": true, ".mov": true}
	if !validExts[ext] {
		jsonError(w, "Formato de video no soportado. Usa: mp4, mkv, webm, avi, mov", http.StatusBadRequest)
		return
	}

	dstPath := filepath.Join("Videos", header.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		jsonError(w, "Error al guardar el archivo", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		jsonError(w, "Error al copiar el archivo", http.StatusInternalServerError)
		return
	}

	fmt.Printf("📤 Video subido: %s (%d MB)\n", header.Filename, header.Size/(1024*1024))

	autoProcess := r.FormValue("auto_process")
	if autoProcess == "true" {
		name := strings.TrimSuffix(header.Filename, ext)
		outputDir := filepath.Join("processed", sanitizeName(name))
		jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())
		queue.Enqueue(jobID, dstPath, outputDir, header.Filename)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Video subido correctamente: " + header.Filename,
		"path":    dstPath,
	})
}

// handleProcessVideo inicia el procesamiento de un video específico
func handleProcessVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FileName string `json:"file_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if req.FileName == "" {
		jsonError(w, "file_name es requerido", http.StatusBadRequest)
		return
	}

	inputPath := filepath.Join("Videos", req.FileName)
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		jsonError(w, "Video no encontrado: "+req.FileName, http.StatusNotFound)
		return
	}

	name := strings.TrimSuffix(req.FileName, filepath.Ext(req.FileName))
	outputDir := filepath.Join("processed", sanitizeName(name))
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())


	job := queue.Enqueue(jobID, inputPath, outputDir, req.FileName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "queued",
		"job_id":  job.ID,
		"message": fmt.Sprintf("Video '%s' encolado para procesamiento", req.FileName),
	})
}

// handleProcessAllVideos procesa todos los videos no procesados
func handleProcessAllVideos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	files, err := os.ReadDir("./Videos")
	if err != nil {
		jsonError(w, "No se pudo leer la carpeta Videos", http.StatusInternalServerError)
		return
	}

	var queued []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext != ".mp4" && ext != ".mkv" && ext != ".webm" && ext != ".avi" && ext != ".mov" {
			continue
		}

		name := strings.TrimSuffix(file.Name(), ext)
		outputDir := filepath.Join("processed", sanitizeName(name))
		manifestPath := filepath.Join(outputDir, "manifest.mpd")

		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			inputPath := filepath.Join("Videos", file.Name())
			jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())
			queue.Enqueue(jobID, inputPath, outputDir, file.Name())
			queued = append(queued, file.Name())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "queued",
		"count":   len(queued),
		"videos":  queued,
		"message": fmt.Sprintf("%d videos encolados para procesamiento", len(queued)),
	})
}

// handleListJobs devuelve el estado de todos los trabajos
func handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs := queue.GetAllJobs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

// handleGetJob devuelve el estado de un trabajo específico
func handleGetJob(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		jsonError(w, "ID de trabajo no proporcionado", http.StatusBadRequest)
		return
	}
	jobID := parts[len(parts)-1]

	job, ok := queue.GetJob(jobID)
	if !ok {
		jsonError(w, "Trabajo no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// sanitizeName limpia el nombre del archivo para usarlo como directorio
func sanitizeName(name string) string {
	replacer := strings.NewReplacer(
		" ", "-",
		"(", "",
		")", "",
		"[", "",
		"]", "",
		"'", "",
		"\"", "",
		"&", "and",
	)
	return strings.ToLower(replacer.Replace(name))
}

// jsonError devuelve un error JSON
func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// handleGetVideo devuelve información detallada de un video específico
func handleGetVideo(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		jsonError(w, "Nombre de video no proporcionado", http.StatusBadRequest)
		return
	}
	videoName := parts[len(parts)-1]

	filePath := filepath.Join("Videos", videoName)
	info, err := os.Stat(filePath)
	if err != nil {
		jsonError(w, "Video no encontrado", http.StatusNotFound)
		return
	}

	name := strings.TrimSuffix(videoName, filepath.Ext(videoName))
	sanitizedName := sanitizeName(name)
	manifestPath := filepath.Join("processed", sanitizedName, "manifest.mpd")

	isProcessed := false
	if _, err := os.Stat(manifestPath); err == nil {
		isProcessed = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(VideoInfo{
		Name:         name,
		FileName:     videoName,
		Size:         info.Size(),
		IsProcessed:  isProcessed,
		ManifestURL:  func() string { if isProcessed { return "/" + manifestPath }; return "" }(),
		DirectURL:    "/Videos/" + videoName,
	})
}

// handleDeleteVideo elimina un video y sus archivos procesados
func handleDeleteVideo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		jsonError(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FileName string `json:"file_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if req.FileName == "" {
		jsonError(w, "file_name es requerido", http.StatusBadRequest)
		return
	}

	videoPath := filepath.Join("Videos", req.FileName)
	if err := os.Remove(videoPath); err != nil {
		jsonError(w, "Error al eliminar el video: "+err.Error(), http.StatusInternalServerError)
		return
	}

	name := strings.TrimSuffix(req.FileName, filepath.Ext(req.FileName))
	processedDir := filepath.Join("processed", sanitizeName(name))
	os.RemoveAll(processedDir)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Video eliminado: " + req.FileName,
	})
}
