package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/indimeco/cheerleader/internal/ddb"
	"github.com/indimeco/cheerleader/internal/models"
)

type Handler struct {
	ddbClient *dynamodb.Client
	Logger    *slog.Logger
	tableName string
}

var ddbClient *dynamodb.Client
var once sync.Once

func New(ctx context.Context) (Handler, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		return Handler{}, errors.New("No region specified in env")
	}

	var onceErr error
	once.Do(func() {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))

		if err != nil {
			onceErr = fmt.Errorf("Failed to get aws config: %w", err)
		}
		ddbClient = dynamodb.NewFromConfig(cfg)
	})
	if onceErr != nil {
		return Handler{}, onceErr
	}

	tableName := os.Getenv("DDB_TABLE")
	if tableName == "" {
		return Handler{}, errors.New("No ddb tablename specified in env")
	}

	return Handler{
		ddbClient: ddbClient,
		Logger:    slog.Default(),
		tableName: tableName,
	}, nil
}

func (h Handler) PutScore(ctx context.Context, score models.Score) error {
	err := ddb.PutScore(ctx, h.tableName, h.ddbClient, score)
	if err != nil {
		return fmt.Errorf("Failed to put score: %w", err)
	}

	return nil
}

func (h Handler) GetTopPlayerScores(ctx context.Context, scoreRequest models.PlayerScoreRequest) ([]models.Score, error) {
	scores, err := ddb.GetTopPlayerScores(ctx, h.tableName, h.ddbClient, scoreRequest)
	if err != nil {
		return nil, fmt.Errorf("Failed to get top player scores: %w", err)
	}
	return scores, nil
}

func (h Handler) GetTopRanks(ctx context.Context, game string) (models.Ranks, error) {
	ranks, err := ddb.GetTopRanks(ctx, h.tableName, h.ddbClient, game)
	if err != nil {
		return nil, fmt.Errorf("Failed to get top ranks: %w", err)
	}
	return ranks, nil
}

func (h Handler) GetRanksAroundPlayer(ctx context.Context, game string, playerId string, around int) (models.Ranks, error) {
	playerScores, err := ddb.GetTopPlayerScores(ctx, h.tableName, h.ddbClient, models.PlayerScoreRequest{
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
	ranks, err := ddb.GetTopRanks(ctx, h.tableName, h.ddbClient, game)
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
