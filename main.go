package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

type Score struct {
	game       string
	score      int
	playerId   int
	playerName string
}

// GetKey returns the composite key for DDB
func (s Score) GetKey() map[string]types.AttributeValue {
	pk := fmt.Sprintf("%v|%v", strconv.Itoa(s.playerId), s.game)
	sk := strconv.Itoa(s.score)
	pkAttr, err := attributevalue.Marshal(pk)
	if err != nil {
		panic(err)
	}
	skAttr, err := attributevalue.Marshal(sk)
	if err != nil {
		panic(err)
	}
	return map[string]types.AttributeValue{"pk": pkAttr, "sk": skAttr}
}

func NewScoreFromParams(params map[string]string) (Score, error) {
	game, ok := params["game"]
	if !ok {
		return Score{}, errors.New("Expected a game")
	}
	sPlayerId, ok := params["player_id"]
	if !ok {
		return Score{}, errors.New("Expected a player_id")
	}
	playerId, err := strconv.Atoi(sPlayerId)
	if err != nil {
		return Score{}, errors.New("Unable to parse player_id")
	}
	sScore, ok := params["score"]
	if !ok {
		return Score{}, errors.New("Expected a score")
	}
	score, err := strconv.Atoi(sScore)
	if err != nil {
		return Score{}, errors.New("Unable to parse score")
	}
	playerName, ok := params["player_name"]
	if !ok {
		return Score{}, errors.New("Expected a player_name")
	}

	return Score{
		playerId:   playerId,
		playerName: playerName,
		game:       game,
		score:      score,
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
	case "PUT":
		params := event.QueryStringParameters
		score, err := NewScoreFromParams(params)
		if err != nil {
			return events.APIGatewayProxyResponse{
				Body:       fmt.Sprint(err),
				StatusCode: http.StatusBadRequest,
			}, err
		}
		err = handler.putScore(ctx, score)
		if err == nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusCreated,
			}, err
		}
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusMethodNotAllowed,
			Body:       "Method not allowed",
		}, nil
	}

	// catch-all generic error handling
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       "Internal server error",
	}, err
}

func (h Handler) putScore(ctx context.Context, score Score) error {
	item, err := attributevalue.MarshalMap(score)
	keys := score.GetKey()
	maps.Copy(item, keys)

	if err != nil {
		return fmt.Errorf("Failed to marshall score: %w", err)
	}

	_, err = h.ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(h.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("Failed to put score: %w", err)
	}

	return nil
}

func main() {
	lambda.Start(handleRequest)
}
