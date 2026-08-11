package generation

import (
	"time"

	"yingyan.local/backend/internal/model"
)

const estimatedDurationPerOutput = 90 * time.Second

// EstimatedProgress combines settled outputs with a time-based estimate because
// OpenAI-compatible image providers do not expose intermediate render progress.
func EstimatedProgress(task model.GenerationTask, now time.Time) int {
	if isTerminalStatus(task.Status) {
		return 100
	}
	if task.Status != StatusProcessing || task.OutputCount <= 0 {
		return 0
	}

	settled := task.CompletedOutputs + task.FailedOutputs
	realProgress := settled * 100 / task.OutputCount
	startedAt := task.StartedAt
	if startedAt == nil {
		return max(realProgress, 5)
	}

	elapsed := now.Sub(*startedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	estimatedTotal := estimatedDurationPerOutput * time.Duration(task.OutputCount)
	timeProgress := 5 + int(elapsed*85/estimatedTotal)
	return min(90, max(realProgress, timeProgress))
}

func isTerminalStatus(status string) bool {
	switch status {
	case StatusCompleted, StatusPartial, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}
