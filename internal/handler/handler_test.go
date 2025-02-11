package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/indimeco/cheerleader/internal/api"
	"github.com/indimeco/cheerleader/internal/models"
)

type testDatabase struct{}

func (t testDatabase) PutScore(ctx context.Context, score models.Score) error {
	if score.Game == "error" {
		return errors.New("an error occurred")
	}
	return nil
}

func (testDatabase) GetTopPlayerScores(ctx context.Context, scoreRequest models.PlayerScoreRequest) ([]models.Score, error) {
	if scoreRequest.Game == "error" {
		return nil, errors.New("an error occurred")
	}
	return []models.Score{
		{
			PlayerId:   "2",
			PlayerName: "Bananalord",
			Game:       "Tetris",
			Score:      100,
		},
		{
			PlayerId:   "2",
			PlayerName: "Bananalord",
			Game:       "Tetris",
			Score:      50,
		},
		{
			PlayerId:   "2",
			PlayerName: "Bananalord",
			Game:       "Tetris",
			Score:      25,
		},
	}, nil
}

func (testDatabase) GetTopRanks(ctx context.Context, ranksRequest models.RanksRequest) (models.Ranks, error) {
	if ranksRequest.Game == "error" {
		return nil, errors.New("an error occurred")
	}
	return models.Ranks{
		{
			Position:   1,
			PlayerName: "Bananalord",
			Score:      150,
		},
		{
			Position:   2,
			PlayerName: "Mongoose",
			Score:      124,
		},
		{
			Position:   3,
			PlayerName: "Bananalord",
			Score:      100,
		},
	}, nil
}

func createTestHandler() Handler {
	return Handler{
		Logger:   slog.New(slog.NewJSONHandler(io.Discard, nil)),
		Database: testDatabase{},
	}
}

func TestPutScore(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()
	apiDefinition := api.ApiDefinition{
		Game:     "Tetris",
		PlayerId: "2",
	}
	body := `{"score": 10, "playerName": "goose"}`
	response := handler.PutScore(ctx, apiDefinition, body)

	if response.StatusCode != 201 {
		t.Errorf("want %v, got %v", 201, response.StatusCode)
	}
}

func TestPutScoreInvalidDb(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()
	apiDefinition := api.ApiDefinition{
		Game:     "error",
		PlayerId: "2",
	}
	body := `{"score": 10, "playerName": "goose"}`
	response := handler.PutScore(ctx, apiDefinition, body)

	if response.StatusCode != 500 {
		t.Errorf("want %v, got %v", 500, response.StatusCode)
	}
}

func TestPutScoreInvalidBody(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()
	apiDefinition := api.ApiDefinition{
		Game:     "error",
		PlayerId: "2",
	}
	body := `abcdefg`
	response := handler.PutScore(ctx, apiDefinition, body)

	if response.StatusCode != 400 {
		t.Errorf("want %v, got %v", 400, response.StatusCode)
	}
}

func TestGetTopPlayerScores(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()
	apiDefinition := api.ApiDefinition{
		Game:     "Tetris",
		PlayerId: "2",
	}
	params := map[string]string{"limit": "10"}
	response := handler.GetTopPlayerScores(ctx, apiDefinition, params)

	if response.StatusCode != 200 {
		t.Errorf("want %v, got %v", 200, response.StatusCode)
	}
}

func TestGetTopPlayerScoresDbError(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()
	apiDefinition := api.ApiDefinition{
		Game:     "error",
		PlayerId: "2",
	}
	params := map[string]string{"limit": "10"}
	response := handler.GetTopPlayerScores(ctx, apiDefinition, params)

	if response.StatusCode != 500 {
		t.Errorf("want %v, got %v", 500, response.StatusCode)
	}
}

func TestGetRanksAroundPlayer(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()
	apiDefinition := api.ApiDefinition{
		Game:     "Tetris",
		PlayerId: "2",
	}
	params := map[string]string{"ranks_around": "1"}
	response := handler.GetRanksAroundPlayer(ctx, apiDefinition, params)

	if response.StatusCode != 200 {
		t.Errorf("want %v, got %v", 200, response.StatusCode)
	}
}

func TestGetRanksAroundPlayerDbError(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()
	apiDefinition := api.ApiDefinition{
		Game:     "error",
		PlayerId: "2",
	}
	params := map[string]string{"ranks_around": "1"}
	response := handler.GetRanksAroundPlayer(ctx, apiDefinition, params)

	if response.StatusCode != 500 {
		t.Errorf("want %v, got %v", 500, response.StatusCode)
	}
}

func TestGetTopRanks(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()
	apiDefinition := api.ApiDefinition{
		Game:     "Tetris",
		PlayerId: "2",
	}
	params := map[string]string{"limit": "10"}
	response := handler.GetTopRanks(ctx, apiDefinition, params)

	if response.StatusCode != 200 {
		t.Errorf("want %v, got %v", 200, response.StatusCode)
	}
}

func TestGetTopRanksDbError(t *testing.T) {
	handler := createTestHandler()
	ctx := context.Background()
	apiDefinition := api.ApiDefinition{
		Game:     "error",
		PlayerId: "2",
	}
	params := map[string]string{"limit": "1"}
	response := handler.GetTopRanks(ctx, apiDefinition, params)

	if response.StatusCode != 500 {
		t.Errorf("want %v, got %v", 500, response.StatusCode)
	}
}
