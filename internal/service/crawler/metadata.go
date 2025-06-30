package crawler

import (
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
)

// ServiceName is the canonical name of the crawler service used in metadata.
const ServiceName = "crawler"

// CurrentVersion represents the current version of the crawler service's metadata schema.
const CurrentVersion = "1.0.0"

// --- Canonical Metadata Keys ---
// These constants define the standardized keys for accessing crawler-specific
// information within the common.Metadata proto's service_specific field.
// Adhering to these keys is crucial for cross-service compatibility and for
// AI agents to reliably interpret and manipulate crawler-related data.
const (
	// CrawlerNamespace is the top-level key for all crawler-specific metadata.
	CrawlerNamespace = "crawler"

	// TaskDetails contains information about the crawl task itself.
	TaskDetails = "task_details"
	// TaskType specifies the kind of crawl (e.g., "html", "file", "video").
	TaskType = "type"
	// TaskTarget is the resource to be crawled (e.g., URL, file path).
	TaskTarget = "target"
	// TaskDepth indicates the recursion depth for the crawl.
	TaskDepth = "depth"

	// ResultSummary holds the outcome of a completed crawl.
	ResultSummary = "result_summary"
	// ResultStatus indicates if the crawl was successful, failed, etc.
	ResultStatus = "status"
	// ResultExtractedLinks contains URLs or paths found during the crawl.
	ResultExtractedLinks = "extracted_links"
	// ResultErrorMessage provides details on why a crawl failed.
	ResultErrorMessage = "error_message"

	// SecurityAnalysis contains findings from the SecurityWorker.
	SecurityAnalysis = "security_analysis"
	// SecurityMalwareDetected indicates if malware was found.
	SecurityMalwareDetected = "malware_detected"
	// SecurityPiiRedacted indicates if Personally Identifiable Information was redacted.
	SecurityPiiRedacted = "pii_redacted"
	// SecurityHighEntropy indicates if the content has high entropy (potentially obfuscated/encrypted).
	SecurityHighEntropy = "high_entropy"

	// VideoMetadata holds data extracted from video files.
	VideoMetadata = "video_metadata"
	// VideoAudioPath is the path to the extracted audio file.
	VideoAudioPath = "audio_path"
	// VideoDuration is the duration of the video.
	VideoDuration = "duration"
	// VideoFormat is the container format of the video (e.g., "mp4").
	VideoFormat = "format"
	// VideoCodec is the video codec used (e.g., "h264").
	VideoCodec = "video_codec"
)

// GetTaskType extracts the crawl task type from the metadata.
// It provides a standardized way for services and AI agents to understand the nature of a crawl task.
func GetTaskType(meta *commonpb.Metadata) string {
	if meta == nil {
		return ""
	}
	vars := metadata.ExtractServiceVariables(meta, CrawlerNamespace)
	details, ok := vars[TaskDetails].(map[string]interface{})
	if !ok {
		return ""
	}

	taskType, _ := details[TaskType].(string)
	return taskType
}

// GetVideoAudioPath extracts the path to the extracted audio file from video metadata.
// This is a key integration point for AI workflows, such as passing the audio
// to a speech-to-text transcription service.
func GetVideoAudioPath(meta *commonpb.Metadata) string {
	if meta == nil {
		return ""
	}
	vars := metadata.ExtractServiceVariables(meta, CrawlerNamespace)
	videoMeta, ok := vars[VideoMetadata].(map[string]interface{})
	if !ok {
		return ""
	}

	audioPath, _ := videoMeta[VideoAudioPath].(string)
	return audioPath
}
