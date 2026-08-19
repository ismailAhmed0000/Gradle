package queue

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"gradle-go-backend/internal/models"
)

const (
	extractInkQueue   = "jobs:extract_ink"
	compositePDFQueue = "jobs:composite_pdf"
)

var queueNameByJobType = map[models.JobType]string{
	models.JobTypeExtractInk:   extractInkQueue,
	models.JobTypeCompositePDF: compositePDFQueue,
}

type JobQueue struct {
	client *redis.Client
}

func NewJobQueue(redisURL string) (*JobQueue, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parsing REDIS_URL: %w", err)
	}
	return &JobQueue{client: redis.NewClient(opts)}, nil
}

// Push enqueues a job for the worker, which BLPOPs from the head of the list.
// Pushing to the tail keeps jobs FIFO.
func (q *JobQueue) Push(ctx context.Context, jobType models.JobType, referenceID string) error {
	queueName, ok := queueNameByJobType[jobType]
	if !ok {
		return fmt.Errorf("unknown job type: %s", jobType)
	}
	if err := q.client.RPush(ctx, queueName, referenceID).Err(); err != nil {
		return fmt.Errorf("pushing %s job %s: %w", jobType, referenceID, err)
	}
	return nil
}
