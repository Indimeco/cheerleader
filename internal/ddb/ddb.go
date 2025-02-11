package ddb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/indimeco/cheerleader/internal/models"
)

type DynamoScoreDatabase struct {
	tableName string
	client    *dynamodb.Client
	rankLimit int
}

var ddbClient *dynamodb.Client
var once sync.Once

func New(ctx context.Context) (DynamoScoreDatabase, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		return DynamoScoreDatabase{}, errors.New("No region specified in env")
	}
	tableName := os.Getenv("DDB_TABLE")
	if tableName == "" {
		return DynamoScoreDatabase{}, errors.New("No ddb tablename specified in env")
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
		return DynamoScoreDatabase{}, onceErr
	}

	/**
	 * DdbMaxRanksLimit is set in order to simplify the possible load placed on any given instance of the service
	 * the constraint prevents expensive query operations and ensures fast response time by limiting the rank requests to a single 'page' of data
	 * 1000 is very appoximately the maximum theoretical size that DDB can return in one response given the 1MB limitation and the maximum data size of a single rank
	 */
	const ddbMaxRanksLimit = 1000

	return DynamoScoreDatabase{
		tableName: tableName,
		client:    ddbClient,
		rankLimit: ddbMaxRanksLimit,
	}, nil
}

func (d DynamoScoreDatabase) getDdbPk(playerId string, game string) string {
	return fmt.Sprintf("%v|%v", playerId, game)
}

func (d DynamoScoreDatabase) getDdbCompositeKey(s models.Score) map[string]types.AttributeValue {
	pk := d.getDdbPk(s.PlayerId, s.Game)
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

func (d DynamoScoreDatabase) PutScore(ctx context.Context, score models.Score) error {
	item, err := attributevalue.MarshalMap(&score)

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})

	return err
}

func (d DynamoScoreDatabase) GetTopPlayerScores(ctx context.Context, scoreRequest models.PlayerScoreRequest) ([]models.Score, error) {
	keyEx := expression.Key("pk").Equal(expression.Value(d.getDdbPk(scoreRequest.PlayerId, scoreRequest.Game)))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to build key expression: %w", err)
	}
	items, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &d.tableName,
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

func (d DynamoScoreDatabase) GetTopRanks(ctx context.Context, ranksRequest models.RanksRequest) (models.Ranks, error) {
	keyEx := expression.Key("game").Equal(expression.Value(ranksRequest.Game))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to build key expression: %w", err)
	}

	limit := d.rankLimit
	if ranksRequest.Limit > 0 {
		limit = min(d.rankLimit, ranksRequest.Limit)
	}
	items, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &d.tableName,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		Limit:                     aws.Int32(int32(limit)),
		ScanIndexForward:          aws.Bool(false), // reverse the sort order to get the highest scores
		IndexName:                 aws.String("GameScoresIndex"),
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to query player ranks: %w", err)
	}

	ranks := make(models.Ranks, 0, items.Count)
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
