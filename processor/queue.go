package processor

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// JobStatus representa el estado de un trabajo de procesamiento
type JobStatus string

const (
	StatusQueued     JobStatus = "queued"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

// Job representa un trabajo de transcodificación en la cola
type Job struct {
	ID         string           `json:"id"`
	InputPath  string           `json:"input_path"`
	OutputDir  string           `json:"output_dir"`
	VideoName  string           `json:"video_name"`
	Status     JobStatus        `json:"status"`
	Progress   int              `json:"progress"` // 0-100
	Error      string           `json:"error,omitempty"`
	Result     *TranscodeResult `json:"result,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	StartedAt  *time.Time       `json:"started_at,omitempty"`
	FinishedAt *time.Time       `json:"finished_at,omitempty"`
}

// Queue gestiona los trabajos de transcodificación en segundo plano
type Queue struct {
	jobs    map[string]*Job
	pending chan *Job
	mu      sync.RWMutex
	wg      sync.WaitGroup
}

// NewQueue crea una nueva cola de procesamiento
func NewQueue(workers int) *Queue {
	q := &Queue{
		jobs:    make(map[string]*Job),
		pending: make(chan *Job, 100), // Buffer de 100 trabajos
	}

	// Iniciar workers
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker(i + 1)
	}

	return q
}

// Enqueue añade un nuevo trabajo a la cola
func (q *Queue) Enqueue(id, inputPath, outputDir, videoName string) *Job {
	job := &Job{
		ID:        id,
		InputPath: inputPath,
		OutputDir: outputDir,
		VideoName: videoName,
		Status:    StatusQueued,
		Progress:  0,
		CreatedAt: time.Now(),
	}

	q.mu.Lock()
	q.jobs[id] = job
	q.mu.Unlock()

	q.pending <- job
	fmt.Printf("📋 Trabajo encolado: %s (%s)\n", videoName, id)
	return job
}

// GetJob obtiene el estado de un trabajo
func (q *Queue) GetJob(id string) (*Job, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	job, ok := q.jobs[id]
	return job, ok
}

// GetAllJobs devuelve todos los trabajos
func (q *Queue) GetAllJobs() []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()
	jobs := make([]*Job, 0, len(q.jobs))
	for _, job := range q.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// worker procesa trabajos de la cola
func (q *Queue) worker(id int) {
	defer q.wg.Done()
	for job := range q.pending {
		fmt.Printf("🔧 Worker %d procesando: %s\n", id, job.VideoName)

		q.mu.Lock()
		job.Status = StatusProcessing
		now := time.Now()
		job.StartedAt = &now
		q.mu.Unlock()

		// Ejecutar transcodificación
		result, err := TranscodeVideo(job.InputPath, job.OutputDir)

		q.mu.Lock()
		finishedAt := time.Now()
		job.FinishedAt = &finishedAt
		if err != nil {
			job.Status = StatusFailed
			job.Error = err.Error()
			fmt.Printf("❌ Worker %d falló en: %s - %v\n", id, job.VideoName, err)
		} else {
			job.Status = StatusCompleted
			job.Progress = 100
			job.Result = result
			fmt.Printf("✅ Worker %d completó: %s\n", id, job.VideoName)

			// Una vez que el archivo está procesado, borramos el original
			fmt.Printf("♻️ Borrando archivo original para ahorrar espacio: %s\n", job.InputPath)
			if removeErr := os.Remove(job.InputPath); removeErr != nil {
				fmt.Printf("⚠️ No se pudo borrar el archivo original: %v\n", removeErr)
			}
		}
		q.mu.Unlock()
	}
}

// Close cierra la cola y espera a que los workers terminen
func (q *Queue) Close() {
	close(q.pending)
	q.wg.Wait()
}
