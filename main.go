package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/indimeco/cheerleader/internal/ddb"
	"github.com/indimeco/cheerleader/internal/models"
)

type Handler struct {
	ddbClient *dynamodb.Client
	logger    *slog.Logger
	tableName string
}

var ddbClient *dynamodb.Client
var once sync.Once

func NewHandler(ctx context.Context) (Handler, error) {
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
		logger:    slog.Default(),
		tableName: tableName,
	}, nil
}

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var err error
	handler, err := NewHandler(ctx)
	if err != nil {
		// failure to get a handler is unrecoverable
		panic(fmt.Errorf("Failed to get handler: %w", err))
	}

	switch event.HTTPMethod {
	case "GET":
		params := event.QueryStringParameters
		scoreRequest, err := models.NewPlayerScoreRequestFromParams(params)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Body:       fmt.Sprint(err),
			}, err
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
		params := event.QueryStringParameters
		score, err := models.NewScoreFromParams(params)
		if err != nil {
			return events.APIGatewayProxyResponse{
				Body:       fmt.Sprint(err),
				StatusCode: http.StatusBadRequest,
			}, err
		}
		err = handler.putScore(ctx, score)
		if err != nil {
			return handler.internalServerError(err), err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusCreated,
		}, err
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusMethodNotAllowed,
			Body:       "Method not allowed",
		}, nil
	}
}

func (h Handler) internalServerError(err error) events.APIGatewayProxyResponse {
	h.logger.Error(fmt.Sprintf("Unexpected error: %v", err))
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       "Internal server error",
	}
}

func (h Handler) putScore(ctx context.Context, score models.Score) error {
	err := ddb.PutScore(ctx, h.tableName, h.ddbClient, score)
	if err != nil {
		return fmt.Errorf("Failed to put score: %w", err)
	}

	return nil
}

func (h Handler) getTopPlayerScores(ctx context.Context, scoreRequest models.PlayerScoreRequest) ([]models.Score, error) {
	scores, err := ddb.GetTopPlayerScores(ctx, h.tableName, h.ddbClient, scoreRequest)
	if err != nil {
		return nil, fmt.Errorf("Failed to get top player scores: %w", err)
	}
	return scores, nil
}

func main() {
	lambda.Start(handleRequest)
}
