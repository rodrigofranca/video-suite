package jobcontracts

// The job types, in one table (docs/adr/0036). Adding a job type is one
// entry here and its handler in the worker that runs it. What depends on the
// type reads the table: the stream it is queued on, the shared contracts
// file, the API's run-again check, the name the dashboard shows, and the
// tests that every worker handles its types.

// Workers that run jobs.
const (
	WorkerMedia = "media" // apps/worker (Go, FFmpeg): stream StreamJobs
	WorkerAI    = "ai"    // apps/ai-worker (Python): stream StreamAIJobs
)

// JobType describes one type of job.
type JobType struct {
	Name   string
	Label  string // what people call it, as the dashboard shows it
	Worker string // WorkerMedia or WorkerAI
	// Rerun: someone may run it again by hand (POST /assets/{id}/jobs), with
	// an input the API builds from the video alone. Exports and translations
	// need choices, so they run only when an export asks for them.
	Rerun bool
}

// JobTypes lists every job type: the Pipeline in order, then the jobs that
// run on request.
var JobTypes = []JobType{
	{JobTypeExtractMetadata, "Metadata", WorkerMedia, true},
	{JobTypeGenerateThumbnail, "Thumbnail", WorkerMedia, true},
	{JobTypeTranscodeHLS, "HLS encoding", WorkerMedia, true},
	{JobTypeExtractAudio, "Audio", WorkerMedia, true},
	{JobTypeTranscribeAudio, "Transcription", WorkerAI, true},
	{JobTypeAnalyzeTranscript, "Summary", WorkerAI, true},
	{JobTypeSuggestClips, "Clips", WorkerAI, true},
	{JobTypeTrackFaces, "Face tracking", WorkerAI, true},
	{JobTypeExportClip, "Clip export", WorkerMedia, false},
	{JobTypeTranslateTranscript, "Translation", WorkerAI, false},
}

// Lookup returns a job type by name.
func Lookup(name string) (JobType, bool) {
	for _, t := range JobTypes {
		if t.Name == name {
			return t, true
		}
	}
	return JobType{}, false
}

// Label is what people call a job type; an unknown one is called by its name.
func Label(name string) string {
	if t, ok := Lookup(name); ok {
		return t.Label
	}
	return name
}

// TypesOf lists the job types a worker runs, in table order.
func TypesOf(worker string) []string {
	var names []string
	for _, t := range JobTypes {
		if t.Worker == worker {
			names = append(names, t.Name)
		}
	}
	return names
}
