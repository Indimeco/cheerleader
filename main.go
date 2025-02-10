package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/indimeco/cheerleader/internal/ddb"
	"github.com/indimeco/cheerleader/internal/models"
)

type handler struct {
	ddbClient *dynamodb.Client
	logger    *slog.Logger
	tableName string
}

var ddbClient *dynamodb.Client
var once sync.Once

func newHandler(ctx context.Context) (handler, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		return handler{}, errors.New("No region specified in env")
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
		return handler{}, onceErr
	}

	tableName := os.Getenv("DDB_TABLE")
	if tableName == "" {
		return handler{}, errors.New("No ddb tablename specified in env")
	}

	return handler{
		ddbClient: ddbClient,
		logger:    slog.Default(),
		tableName: tableName,
	}, nil
}

type apiDefinition struct {
	path     string
	playerId string
	game     string
}

func eventPathToApiDefinition(path string) (apiDefinition, error) {
	type apiDescription struct {
		regex           regexp.Regexp
		hasGamePath     bool
		hasPlayerIdPath bool
	}
	apiPaths := map[string]apiDescription{
		"/{game}/{player_id}/scores": {regex: *regexp.MustCompile(`^/[\w\d]+/[\w\d]+/scores/?$`), hasGamePath: true, hasPlayerIdPath: true},
		"/{game}/{player_id}/ranks":  {regex: *regexp.MustCompile(`^/[\w\d]+/[\w\d]+/ranks/?$`), hasGamePath: true, hasPlayerIdPath: true},
		"/{game}/scores":             {regex: *regexp.MustCompile(`^/[\w\d]+/scores/?$`), hasGamePath: true},
		"/{game}/ranks":              {regex: *regexp.MustCompile(`^/[\w\d]+/ranks/?$`), hasGamePath: true},
	}

	for k, v := range apiPaths {
		definition := apiDefinition{}
		if v.regex.Match([]byte(path)) {
			definition.path = k
			parts := strings.Split(path, "/")
			if v.hasGamePath {
				definition.game = parts[1]
			}
			if v.hasPlayerIdPath {
				definition.playerId = parts[2]
			}

			return definition, nil
		}
	}

	return apiDefinition{}, errors.New("No matching api found")
}

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var err error
	handler, err := newHandler(ctx)
	if err != nil {
		// failure to get a handler is unrecoverable
		panic(fmt.Errorf("Failed to get handler: %w", err))
	}

	api, err := eventPathToApiDefinition(event.Path)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Body:       "Resource not found",
		}, nil
	}

	switch api.path {
	case "/{game}/{player_id}/scores":
		{

			switch event.HTTPMethod {
			case "GET":
				params := event.QueryStringParameters
				scoreRequest, err := models.NewPlayerScoreRequest(params, api.game, api.playerId)
				if err != nil {
					return events.APIGatewayProxyResponse{
						StatusCode: http.StatusBadRequest,
						Body:       fmt.Sprint(err),
					}, nil
				}
				scores, err := handler.getTopPlayerScores(ctx, scoreRequest)
				if err != nil {
					return handler.internalServerError(err), err
				}
				out, err := json.Marshal(&scores)
				if err != nil {
					return handler.internalServerError(err), err
				}
				return events.APIGatewayProxyResponse{
					Body:       string(out),
					StatusCode: http.StatusOK,
				}, nil

			case "PUT":
				score, err := models.NewScore(api.game, api.playerId, event.Body)
				if err != nil {
					return events.APIGatewayProxyResponse{
						Body:       fmt.Sprint(err),
						StatusCode: http.StatusBadRequest,
					}, nil
				}
				err = handler.putScore(ctx, score)
				if err != nil {
					return handler.internalServerError(err), err
				}
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusCreated,
				}, nil
			default:
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusMethodNotAllowed,
					Body:       "Method not allowed",
				}, nil
			}
		}
	case "/{game}/{player_id}/ranks":
		{
			if event.HTTPMethod != "GET" {
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusMethodNotAllowed,
					Body:       "Method not allowed",
				}, nil
			}
			ranksRequest, err := models.NewPlayerRanksRequest(event.QueryStringParameters, api.game, api.playerId)
			if err != nil {
				return events.APIGatewayProxyResponse{
					Body:       fmt.Sprint(err),
					StatusCode: http.StatusBadRequest,
				}, nil
			}
			ranksAround, err := handler.getRanksAroundPlayer(ctx, ranksRequest.Game, ranksRequest.PlayerId, ranksRequest.Around)
			if err != nil {
				return handler.internalServerError(err), err
			}
			out, err := json.Marshal(&ranksAround)
			if err != nil {
				return handler.internalServerError(err), err
			}
			return events.APIGatewayProxyResponse{
				Body:       string(out),
				StatusCode: http.StatusOK,
			}, nil
		}
	case "/{game}/ranks":
		{
			if event.HTTPMethod != "GET" {
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusMethodNotAllowed,
					Body:       "Method not allowed",
				}, nil
			}
			ranks, err := handler.getTopRanks(ctx, api.game)
			if err != nil {
				return handler.internalServerError(err), err
			}
			out, err := json.Marshal(&ranks)
			if err != nil {
				return handler.internalServerError(err), err
			}
			return events.APIGatewayProxyResponse{
				Body:       string(out),
				StatusCode: http.StatusOK,
			}, nil
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       "Failed to handle request",
	}, nil
}

func (h handler) internalServerError(err error) events.APIGatewayProxyResponse {
	h.logger.Error(fmt.Sprintf("Unexpected error: %v", err))
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       "Internal server error",
	}
}

func (h handler) putScore(ctx context.Context, score models.Score) error {
	err := ddb.PutScore(ctx, h.tableName, h.ddbClient, score)
	if err != nil {
		return fmt.Errorf("Failed to put score: %w", err)
	}

	return nil
}

func (h handler) getTopPlayerScores(ctx context.Context, scoreRequest models.PlayerScoreRequest) ([]models.Score, error) {
	scores, err := ddb.GetTopPlayerScores(ctx, h.tableName, h.ddbClient, scoreRequest)
	if err != nil {
		return nil, fmt.Errorf("Failed to get top player scores: %w", err)
	}
	return scores, nil
}

func (h handler) getTopRanks(ctx context.Context, game string) (models.Ranks, error) {
	ranks, err := ddb.GetTopRanks(ctx, h.tableName, h.ddbClient, game)
	if err != nil {
		return nil, fmt.Errorf("Failed to get top ranks: %w", err)
	}
	return ranks, nil
}

func (h handler) getRanksAroundPlayer(ctx context.Context, game string, playerId string, around int) (models.Ranks, error) {
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

func main() {
	lambda.Start(handleRequest)
}
