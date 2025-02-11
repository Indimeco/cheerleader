package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/indimeco/cheerleader/internal/ddb"
	"github.com/indimeco/cheerleader/internal/models"
)

type Handler struct {
	Database HandlerDatabase
	Logger   *slog.Logger
}

type HandlerDatabase interface {
	PutScore(context.Context, models.Score) error
	GetTopPlayerScores(context.Context, models.PlayerScoreRequest) ([]models.Score, error)
	GetTopRanks(context.Context, models.RanksRequest) (models.Ranks, error)
}

func New(ctx context.Context) (Handler, error) {
	ddbClient, err := ddb.New(ctx)
	if err != nil {
		return Handler{}, fmt.Errorf("Failed to get database: %w", err)
	}

	return Handler{
		Database: ddbClient,
		Logger:   slog.Default(),
	}, nil
}

func (h Handler) PutScore(ctx context.Context, score models.Score) error {
	err := h.Database.PutScore(ctx, score)
	if err != nil {
		return fmt.Errorf("Failed to put score: %w", err)
	}

	return nil
}

func (h Handler) GetTopPlayerScores(ctx context.Context, scoreRequest models.PlayerScoreRequest) ([]models.Score, error) {
	scores, err := h.Database.GetTopPlayerScores(ctx, scoreRequest)
	if err != nil {
		return nil, fmt.Errorf("Failed to get top player scores: %w", err)
	}
	return scores, nil
}

func (h Handler) GetTopRanks(ctx context.Context, ranksRequest models.RanksRequest) (models.Ranks, error) {
	ranks, err := h.Database.GetTopRanks(ctx, ranksRequest)
	if err != nil {
		return nil, fmt.Errorf("Failed to get top ranks: %w", err)
	}
	return ranks, nil
}

func (h Handler) GetRanksAroundPlayer(ctx context.Context, game string, playerId string, around int) (models.Ranks, error) {
	playerScores, err := h.Database.GetTopPlayerScores(ctx, models.PlayerScoreRequest{
		PlayerId:     playerId,
		ScoreRequest: models.ScoreRequest{Game: game, Limit: 1},
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to get top player score: %w", err)
	}
	// Player hasn't scored yet
	if len(playerScores) < 1 {
		return models.Ranks{}, nil
	}
	topScore := playerScores[0]
	ranks, err := h.Database.GetTopRanks(ctx, models.RanksRequest{})
	if err != nil {
		return nil, fmt.Errorf("Failed to get top ranks: %w", err)
	}

	index := ranks.BinarySearch(topScore.Score, 0, len(ranks))
	// Player is not ranked
	if index == -1 {
		return models.Ranks{}, nil
	}
	ranksAround := ranks.Around(index, around)

	return ranksAround, nil
}
