package service

import (
	"context"
	"fmt"

	"GP_back_end_go/internal/mysql"
	"GP_back_end_go/models/dto"
	"GP_back_end_go/pkg/constants"
	"GP_back_end_go/pkg/db"
)

func EnqueuePreprocess(ctx context.Context, id uint64) (dto.QueueAckResponse, error) {
	return enqueueProcessingTask(ctx, id, constants.PreprocessQueueKey, constants.PreprocessActiveSetKey)
}

func EnqueueGraphBuild(ctx context.Context, id uint64) (dto.QueueAckResponse, error) {
	return enqueueProcessingTask(ctx, id, constants.GraphQueueKey, constants.GraphActiveSetKey)
}

func GetPreprocessStatus(ctx context.Context) (dto.StatusResponse, error) {
	pending, err := getQueuePending(ctx, constants.PreprocessQueueKey, constants.PreprocessActiveSetKey)
	if err != nil {
		return dto.StatusResponse{}, err
	}
	return dto.StatusResponse{
		Pending:  pending,
		QueueKey: constants.PreprocessQueueKey,
	}, nil
}

func GetGraphStatus(ctx context.Context) (dto.StatusResponse, error) {
	pending, err := getQueuePending(ctx, constants.GraphQueueKey, constants.GraphActiveSetKey)
	if err != nil {
		return dto.StatusResponse{}, err
	}
	return dto.StatusResponse{
		Pending:  pending,
		QueueKey: constants.GraphQueueKey,
	}, nil
}

func enqueueProcessingTask(ctx context.Context, id uint64, queueKey, activeSetKey string) (dto.QueueAckResponse, error) {
	if id == 0 {
		return dto.QueueAckResponse{}, fmt.Errorf("%w: id invalid", ErrBadRequest)
	}
	if db.DB.RDB == nil {
		return dto.QueueAckResponse{}, ErrRedisUnavailable
	}

	dao := mysql.NewCrawlTargetDAO()
	if _, err := dao.GetDetailByID(id); err != nil {
		return dto.QueueAckResponse{}, err
	}

	if err := db.DB.RDB.RPush(ctx, queueKey, fmt.Sprintf("%d", id)).Err(); err != nil {
		return dto.QueueAckResponse{}, err
	}

	pending, err := getQueuePending(ctx, queueKey, activeSetKey)
	if err != nil {
		return dto.QueueAckResponse{}, err
	}
	return dto.QueueAckResponse{
		Queued:   1,
		QueueKey: queueKey,
		Pending:  pending,
	}, nil
}

func getQueuePending(ctx context.Context, queueKey, activeSetKey string) (int64, error) {
	if db.DB.RDB == nil {
		return 0, ErrRedisUnavailable
	}
	pendingQueue, err := db.DB.RDB.LLen(ctx, queueKey).Result()
	if err != nil {
		return 0, err
	}
	activeCount, err := db.DB.RDB.SCard(ctx, activeSetKey).Result()
	if err != nil {
		return 0, err
	}
	return pendingQueue + activeCount, nil
}
