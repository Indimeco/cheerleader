package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/indimeco/cheerleader/internal/api"
	"github.com/indimeco/cheerleader/internal/handler"
	"github.com/indimeco/cheerleader/internal/models"
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
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Body:       "Resource not found",
		}, nil
	}

	switch apiDefinition.Route {
	case apiRoutes.ScoresByPlayer:
		{

			switch event.HTTPMethod {
			case "GET":
				params := event.QueryStringParameters
				scoreRequest, err := models.NewPlayerScoreRequest(params, apiDefinition.Game, apiDefinition.PlayerId)
				if err != nil {
					return events.APIGatewayProxyResponse{
						StatusCode: http.StatusBadRequest,
						Body:       fmt.Sprint(err),
					}, nil
				}
				scores, err := h.GetTopPlayerScores(ctx, scoreRequest)
				if err != nil {
					return internalServerError(&h, err), err
				}
				out, err := json.Marshal(&scores)
				if err != nil {
					return internalServerError(&h, err), err
				}
				return events.APIGatewayProxyResponse{
					Body:       string(out),
					StatusCode: http.StatusOK,
				}, nil

			case "PUT":
				score, err := models.NewScore(apiDefinition.Game, apiDefinition.PlayerId, event.Body)
				if err != nil {
					return events.APIGatewayProxyResponse{
						Body:       fmt.Sprint(err),
						StatusCode: http.StatusBadRequest,
					}, nil
				}
				err = h.PutScore(ctx, score)
				if err != nil {
					return internalServerError(&h, err), err
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
	case apiRoutes.RanksByPlayer:
		{
			if event.HTTPMethod != "GET" {
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusMethodNotAllowed,
					Body:       "Method not allowed",
				}, nil
			}
			ranksRequest, err := models.NewPlayerRanksRequest(event.QueryStringParameters, apiDefinition.Game, apiDefinition.PlayerId)
			if err != nil {
				return events.APIGatewayProxyResponse{
					Body:       fmt.Sprint(err),
					StatusCode: http.StatusBadRequest,
				}, nil
			}
			ranksAround, err := h.GetRanksAroundPlayer(ctx, ranksRequest.Game, ranksRequest.PlayerId, ranksRequest.Around)
			if err != nil {
				return internalServerError(&h, err), err
			}
			out, err := json.Marshal(&ranksAround)
			if err != nil {
				return internalServerError(&h, err), err
			}
			return events.APIGatewayProxyResponse{
				Body:       string(out),
				StatusCode: http.StatusOK,
			}, nil
		}
	case apiRoutes.Ranks:
		{
			if event.HTTPMethod != "GET" {
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusMethodNotAllowed,
					Body:       "Method not allowed",
				}, nil
			}
			ranks, err := h.GetTopRanks(ctx, apiDefinition.Game)
			if err != nil {
				return internalServerError(&h, err), err
			}
			out, err := json.Marshal(&ranks)
			if err != nil {
				return internalServerError(&h, err), err
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

func internalServerError(handler *handler.Handler, err error) events.APIGatewayProxyResponse {
	handler.Logger.Error(fmt.Sprintf("Unexpected error: %v", err))
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       "Internal server error",
	}
}

func main() {
	lambda.Start(handleRequest)
}
