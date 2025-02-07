package ddb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/indimeco/cheerleader/internal/models"
)

func getDdbPk(playerId string, game string) string {
	return fmt.Sprintf("%v|%v", playerId, game)
}

func getDdbCompositeKey(s models.Score) map[string]types.AttributeValue {
	pk := getDdbPk(s.PlayerId, s.Game)
	sk := s.Score
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

func PutScore(ctx context.Context, tableName string, client *dynamodb.Client, score models.Score) error {
	item, err := attributevalue.MarshalMap(&score)

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})

	return err
}

func GetTopPlayerScores(ctx context.Context, tableName string, client *dynamodb.Client, scoreRequest models.PlayerScoreRequest) ([]models.Score, error) {
	keyEx := expression.Key("pk").Equal(expression.Value(getDdbPk(scoreRequest.PlayerId, scoreRequest.Game)))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to build key expression: %w", err)
	}
	items, err := client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &tableName,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		Limit:                     aws.Int32(int32(scoreRequest.Limit)),
		ScanIndexForward:          aws.Bool(false), // reverse the sort order to get the highest scores
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to query player scores: %w", err)
	}

	scores := make([]models.Score, 0, items.Count)
	for _, marshalledScore := range items.Items {
		var score models.Score
		err := attributevalue.UnmarshalMap(marshalledScore, &score)
		if err != nil {
			return nil, fmt.Errorf("Failed to unmarshall a score: %w", err)
		}
		scores = append(scores, score)
	}
	return scores, nil
}

/**
* ddbMaxRanksLimit is set in order to simplify the possible load placed on any given instance of the service
* the constraint prevents expensive query operations and ensures fast response time by limiting the rank requests to a single 'page' of data
* 1000 is very appoximately the maximum theoretical size that DDB can return in one response given the 1MB limitation and the maximum data size of a single rank
 */
const ddbMaxRanksLimit = 1000

func GetTopRanks(ctx context.Context, tableName string, client *dynamodb.Client, game string) ([]models.Rank, error) {
	keyEx := expression.Key("game").Equal(expression.Value(game))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to build key expression: %w", err)
	}
	items, err := client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &tableName,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		Limit:                     aws.Int32(ddbMaxRanksLimit),
		ScanIndexForward:          aws.Bool(false), // reverse the sort order to get the highest scores
		IndexName:                 aws.String("GameScoresIndex"),
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to query player ranks: %w", err)
	}

	ranks := make([]models.Rank, 0, items.Count)
	for i, marshalledRank := range items.Items {
		var rank models.Rank
		err := attributevalue.UnmarshalMap(marshalledRank, &rank)
		if err != nil {
			return nil, fmt.Errorf("Failed to unmarshall a rank: %w", err)
		}
		rank.Position = i + 1
		ranks = append(ranks, rank)
	}
	return ranks, nil
}
