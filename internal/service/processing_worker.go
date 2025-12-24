package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"GP_back_end_go/pkg/constants"
	"GP_back_end_go/pkg/db"
	"GP_back_end_go/pkg/log"

	go_redis "github.com/go-redis/redis/v8"
)

const defaultWorkerCount = 3

func StartPreprocessWorkers(ctx context.Context) {
	startAsyncWorkers(ctx, "PreprocessWorker", constants.PreprocessQueueKey, constants.PreprocessActiveSetKey, defaultWorkerCount, func(jobCtx context.Context, id uint64) error {
		_, err := RunPreprocessLLM(jobCtx, id)
		return err
	})
}

func StartGraphWorkers(ctx context.Context) {
	startAsyncWorkers(ctx, "GraphWorker", constants.GraphQueueKey, constants.GraphActiveSetKey, defaultWorkerCount, func(jobCtx context.Context, id uint64) error {
		_, err := BuildGraphFromProcessed(jobCtx, id)
		return err
	})
}

func startAsyncWorkers(
	ctx context.Context,
	name string,
	queueKey string,
	activeSetKey string,
	workerCount int,
	handler func(context.Context, uint64) error,
) {
	if workerCount < 1 {
		workerCount = 1
	}

	log.Info(name, "start workers", "count", workerCount, "queue", queueKey, "active_set", activeSetKey)
	for i := 0; i < workerCount; i++ {
		go func(idx int) {
			workerName := fmt.Sprintf("%s-%d", name, idx+1)
			for {
				select {
				case <-ctx.Done():
					log.Info(workerName, "context done, exit")
					return
				default:
				}

				if db.DB.RDB == nil {
					time.Sleep(500 * time.Millisecond)
					continue
				}

				res, err := db.DB.RDB.BRPop(ctx, 5*time.Second, queueKey).Result()
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					log.Info(workerName, "context canceled", "err", err)
					return
				}
				if err == go_redis.Nil {
					continue
				}
				if err != nil {
					log.Error(workerName, "BRPop failed", err.Error(), "queue", queueKey)
					time.Sleep(time.Second)
					continue
				}
				if len(res) < 2 {
					continue
				}

				rawID := strings.TrimSpace(res[1])
				if rawID == "" {
					continue
				}

				if err := db.DB.RDB.SAdd(ctx, activeSetKey, rawID).Err(); err != nil {
					log.Error(workerName, "SAdd active failed", err.Error(), "id_raw", rawID)
				}

				id, err := strconv.ParseUint(rawID, 10, 64)
				if err != nil {
					log.Error(workerName, "parse id failed", err.Error(), "raw", rawID)
					_ = db.DB.RDB.SRem(ctx, activeSetKey, rawID).Err()
					continue
				}

				start := time.Now()
				if err := handler(ctx, id); err != nil {
					log.Error(workerName, "job failed", err.Error(), "id", id)
				} else {
					log.Info(workerName, "job done", "id", id, "duration_ms", time.Since(start).Milliseconds())
				}

				if err := db.DB.RDB.SRem(ctx, activeSetKey, rawID).Err(); err != nil {
					log.Error(workerName, "active cleanup failed", err.Error(), "id", id)
				}
			}
		}(i)
	}
}
