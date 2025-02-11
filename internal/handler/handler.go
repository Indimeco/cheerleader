package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/indimeco/cheerleader/internal/api"
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

func (h Handler) PutScore(ctx context.Context, apiDefinition api.ApiDefinition, body string) events.APIGatewayProxyResponse {
	score, err := models.NewScore(apiDefinition.Game, apiDefinition.PlayerId, body)
	if err != nil {
		return h.ResponseBadRequest(err)
	}
	err = h.Database.PutScore(ctx, score)
	if err != nil {
		return h.ResponseInternalServerError(fmt.Errorf("Failed to put score: %w", err))
	}

	return h.ResponseCreated()
}

func (h Handler) GetTopPlayerScores(ctx context.Context, apiDefinition api.ApiDefinition, params map[string]string) events.APIGatewayProxyResponse {
	scoreRequest, err := models.NewPlayerScoreRequest(params, apiDefinition.Game, apiDefinition.PlayerId)
	if err != nil {
		return h.ResponseBadRequest(err)
	}
	scores, err := h.Database.GetTopPlayerScores(ctx, scoreRequest)
	if err != nil {
		return h.ResponseInternalServerError(fmt.Errorf("Failed to get top player scores: %w", err))
	}
	out, err := json.Marshal(&scores)
	if err != nil {
		return h.ResponseInternalServerError(fmt.Errorf("Failed to marshal top player scores: %w", err))
	}
	return h.ResponseOk(string(out))
}

func (h Handler) GetTopRanks(ctx context.Context, apiDefinition api.ApiDefinition, params map[string]string) events.APIGatewayProxyResponse {
	ranksRequest, err := models.NewRanksRequest(params, apiDefinition.Game)
	if err != nil {
		return h.ResponseBadRequest(err)
	}
	ranks, err := h.Database.GetTopRanks(ctx, ranksRequest)
	if err != nil {
		return h.ResponseInternalServerError(fmt.Errorf("Failed to get top ranks: %w", err))
	}
	out, err := json.Marshal(&ranks)
	if err != nil {
		return h.ResponseInternalServerError(fmt.Errorf("Failed to marshal top ranks: %w", err))
	}
	return h.ResponseOk(string(out))
}

func (h Handler) GetRanksAroundPlayer(ctx context.Context, apiDefinition api.ApiDefinition, params map[string]string) events.APIGatewayProxyResponse {
	ranksRequest, err := models.NewPlayerRanksRequest(params, apiDefinition.Game, apiDefinition.PlayerId)
	if err != nil {
		return h.ResponseBadRequest(err)
	}

	playerScores, err := h.Database.GetTopPlayerScores(ctx, models.PlayerScoreRequest{
		PlayerId:     apiDefinition.PlayerId,
		ScoreRequest: models.ScoreRequest{Game: apiDefinition.Game, Limit: 1},
	})
	if err != nil {
		return h.ResponseInternalServerError(fmt.Errorf("Failed to get top player score: %w", err))
	}

	// Player hasn't scored yet
	if len(playerScores) < 1 {
		return h.ResponseOk("[]")
	}
	topScore := playerScores[0]
	ranks, err := h.Database.GetTopRanks(ctx, models.RanksRequest{})
	if err != nil {
		return h.ResponseInternalServerError(fmt.Errorf("Failed to get top ranks: %w", err))
	}

	index := ranks.BinarySearch(topScore.Score, 0, len(ranks))
	// Player is not ranked
	if index == -1 {
		return h.ResponseOk("[]")
	}
	ranksAround := ranks.Around(index, ranksRequest.Around)

	out, err := json.Marshal(&ranksAround)
	if err != nil {
		return h.ResponseInternalServerError(fmt.Errorf("Failed to marshal player ranks: %w", err))
	}
	return h.ResponseOk(string(out))
}

func (h Handler) ResponseInternalServerError(err error) events.APIGatewayProxyResponse {
	h.Logger.Error(fmt.Sprintf("Unexpected error: %v", err))
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       "Internal server error",
	}
}

func (h Handler) ResponseOk(data string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       data,
	}
}

func (h Handler) ResponseCreated() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
	}
}

func (h Handler) ResponseBadRequest(err error) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprint(err),
		StatusCode: http.StatusBadRequest,
	}
}

func (h Handler) ResponseMethodNotAllowed() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusMethodNotAllowed,
		Body:       "Method not allowed",
	}
}

func (h Handler) ResponseNotFound() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusNotFound,
		Body:       "Resource not found",
	}
}
