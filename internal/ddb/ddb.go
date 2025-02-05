package ddb

import (
	"context"
	"fmt"
	"maps"
	"strconv"

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
	sk := strconv.Itoa(s.Score)
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
	item, err := attributevalue.MarshalMap(score)
	keys := getDdbCompositeKey(score)
	maps.Copy(item, keys)

	if err != nil {
		return fmt.Errorf("Failed to marshall score: %w", err)
	}

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
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to query player scores: %w", err)
	}

	scores := make([]models.Score, 0, items.Count)
	for _, marshalledScore := range items.Items {
		var score models.Score
		err := attributevalue.UnmarshalMap(marshalledScore, score)
		if err != nil {
			return nil, fmt.Errorf("Failed to unmarshall a score: %w", err)
		}
		scores = append(scores, score)
	}
	return scores, nil
}
