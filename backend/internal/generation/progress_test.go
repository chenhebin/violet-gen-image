package generation

import (
	"testing"
	"time"

	"yingyan.local/backend/internal/model"
)

func TestEstimatedProgress(t *testing.T) {
	startedAt := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		task model.GenerationTask
		now  time.Time
		want int
	}{
		{name: "queued", task: model.GenerationTask{Status: StatusQueued, OutputCount: 1}, now: startedAt, want: 0},
		{name: "processing starts at five", task: model.GenerationTask{Status: StatusProcessing, OutputCount: 1, StartedAt: &startedAt}, now: startedAt, want: 5},
		{name: "processing advances with time", task: model.GenerationTask{Status: StatusProcessing, OutputCount: 1, StartedAt: &startedAt}, now: startedAt.Add(45 * time.Second), want: 47},
		{name: "processing is capped", task: model.GenerationTask{Status: StatusProcessing, OutputCount: 1, StartedAt: &startedAt}, now: startedAt.Add(3 * time.Minute), want: 90},
		{name: "settled output is authoritative", task: model.GenerationTask{Status: StatusProcessing, OutputCount: 4, CompletedOutputs: 2, StartedAt: &startedAt}, now: startedAt.Add(time.Second), want: 50},
		{name: "completed", task: model.GenerationTask{Status: StatusCompleted, OutputCount: 1, CompletedOutputs: 1}, now: startedAt, want: 100},
		{name: "failed", task: model.GenerationTask{Status: StatusFailed, OutputCount: 1, FailedOutputs: 1}, now: startedAt, want: 100},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := EstimatedProgress(test.task, test.now); got != test.want {
				t.Fatalf("EstimatedProgress() = %d, want %d", got, test.want)
			}
		})
	}
}
