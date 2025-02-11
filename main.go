package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/indimeco/cheerleader/internal/api"
	"github.com/indimeco/cheerleader/internal/handler"
)

func handleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var err error
	h, err := handler.New(ctx)
	if err != nil {
		// failure to get a handler is unrecoverable
		panic(fmt.Errorf("Failed to get handler: %w", err))
	}

	apiRoutes := api.NewApiRoutes()
	apiDefinition, err := api.EventPathToApiDefinition(event.Path)
	if err != nil {
		return h.ResponseNotFound(), nil
	}

	params := event.QueryStringParameters
	body := event.Body

	switch apiDefinition.Route {
	case apiRoutes.ScoresByPlayer:
		{

			switch event.HTTPMethod {
			case "GET":
				return h.GetTopPlayerScores(ctx, apiDefinition, params), nil
			case "PUT":
				return h.PutScore(ctx, apiDefinition, body), nil
			default:
				return h.ResponseMethodNotAllowed(), nil
			}
		}
	case apiRoutes.RanksByPlayer:
		{
			if event.HTTPMethod != "GET" {
				return h.ResponseMethodNotAllowed(), nil
			}
			return h.GetRanksAroundPlayer(ctx, apiDefinition, params), nil
		}
	case apiRoutes.Ranks:
		{
			if event.HTTPMethod != "GET" {
				return h.ResponseMethodNotAllowed(), nil
			}
			return h.GetTopRanks(ctx, apiDefinition, params), nil
		}
	}

	return h.ResponseInternalServerError(fmt.Errorf("Unhandled API escaped with path %q method %q ", event.Path, event.HTTPMethod)), nil
}

func main() {
	lambda.Start(handleRequest)
}
