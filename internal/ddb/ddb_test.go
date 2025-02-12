package ddb

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	"github.com/google/go-cmp/cmp"
	"github.com/indimeco/cheerleader/internal/models"
	"github.com/testcontainers/testcontainers-go"
	tcdynamodb "github.com/testcontainers/testcontainers-go/modules/dynamodb"
)

var globalTestClient *dynamodb.Client

const testTableName = "test_table"

type DynamoDBLocalResolver struct {
	hostAndPort string
}

func (r *DynamoDBLocalResolver) ResolveEndpoint(ctx context.Context, params dynamodb.EndpointParameters) (endpoint smithyendpoints.Endpoint, err error) {
	return smithyendpoints.Endpoint{
		URI: url.URL{Host: r.hostAndPort, Scheme: "http"},
	}, nil
}

// func createTestDdbClient creates a new dynamodb client, a closer and an error
// the closer should always be called, regardless of if an error occurred during client creation
func createTestDdbClient(ctx context.Context) (*dynamodb.Client, func(), error) {
	dynamodbContainer, err := tcdynamodb.Run(ctx, "amazon/dynamodb-local:2.2.1")
	close := func() {
		if err := testcontainers.TerminateContainer(dynamodbContainer); err != nil {
			panic(fmt.Sprintf("Failed to terminate container: %v", err))
		}
	}
	if err != nil {
		return nil, close, fmt.Errorf("Failed to run dynamodb container: %w", err)
	}

	hostPort, err := dynamodbContainer.ConnectionString(context.Background())
	if err != nil {
		return nil, close, fmt.Errorf("Failed to get connection string: %w", err)
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
		Value: aws.Credentials{
			AccessKeyID:     "DUMMYIDEXAMPLE",
			SecretAccessKey: "DUMMYEXAMPLEKEY",
		},
	}))
	if err != nil {
		return nil, close, fmt.Errorf("Failed to load default config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg, dynamodb.WithEndpointResolverV2(&DynamoDBLocalResolver{hostAndPort: hostPort}))

	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("pk"),
				KeyType:       "HASH",
			},
			{
				AttributeName: aws.String("sk"),
				KeyType:       "SORT",
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("pk"),
				AttributeType: "S",
			},
			{
				AttributeName: aws.String("sk"),
				AttributeType: "N",
			},
			{
				AttributeName: aws.String("game"),
				AttributeType: "S",
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("GameScoresIndex"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("game"),
						KeyType:       "HASH",
					},
					{
						AttributeName: aws.String("sk"),
						KeyType:       "SORT",
					},
				},
				Projection: &types.Projection{
					ProjectionType:   "INCLUDE",
					NonKeyAttributes: []string{"pname", "ts"},
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(1),
					WriteCapacityUnits: aws.Int64(1),
				},
			},
		},
	})
	if err != nil {
		return nil, close, fmt.Errorf("Failed to create table: %w", err)
	}

	return client, close, nil
}

func createTestDynamoScoreDatabase() DynamoScoreDatabase {
	return DynamoScoreDatabase{
		client:    globalTestClient,
		rankLimit: 1000,
		tableName: testTableName,
	}
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	client, close, err := createTestDdbClient(ctx)
	defer close()
	if err != nil {
		panic(fmt.Sprintf("Failed to get test client: %v", err))
	}
	globalTestClient = client
	m.Run()
	return
}

func TestPutScore(t *testing.T) {
	ctx := context.Background()
	score := models.Score{
		PlayerId:   "1",
		PlayerName: "Bananalord",
		Game:       "Tetris",
		Score:      100,
	}
	d := createTestDynamoScoreDatabase()
	err := d.PutScore(ctx, score)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestGetTopPlayerScores(t *testing.T) {
	score1 := models.Score{
		PlayerId:   "2",
		PlayerName: "Bananalord",
		Game:       "Tetris",
		Score:      100,
	}
	score2 := models.Score{
		PlayerId:   "2",
		PlayerName: "Bananalord",
		Game:       "Tetris",
		Score:      150,
	}

	want := []models.Score{
		score2,
		score1,
	}

	d := createTestDynamoScoreDatabase()
	ctx := context.Background()
	err := d.PutScore(ctx, score1)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	err = d.PutScore(ctx, score2)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	scoreRequest := models.PlayerScoreRequest{
		ScoreRequest: models.ScoreRequest{
			Game:  "Tetris",
			Limit: 2,
		},
		PlayerId: "2",
	}
	scores, err := d.GetTopPlayerScores(ctx, scoreRequest)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if diff := cmp.Diff(want, scores); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGetTopPlayerScoresWithLimit(t *testing.T) {
	score1 := models.Score{
		PlayerId:   "2",
		PlayerName: "Bananalord",
		Game:       "Tetris",
		Score:      100,
	}
	score2 := models.Score{
		PlayerId:   "2",
		PlayerName: "Bananalord",
		Game:       "Tetris",
		Score:      150,
	}

	want := []models.Score{
		score2,
	}

	d := createTestDynamoScoreDatabase()
	ctx := context.Background()
	err := d.PutScore(ctx, score1)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	err = d.PutScore(ctx, score2)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	scoreRequest := models.PlayerScoreRequest{
		ScoreRequest: models.ScoreRequest{
			Game:  "Tetris",
			Limit: 1,
		},
		PlayerId: "2",
	}
	scores, err := d.GetTopPlayerScores(ctx, scoreRequest)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if diff := cmp.Diff(want, scores); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGetTopPlayerScoresWithUserGameIsolation(t *testing.T) {
	score1 := models.Score{
		PlayerId:   "3",
		PlayerName: "Joseph",
		Game:       "Tetris",
		Score:      1,
	}
	score2 := models.Score{
		PlayerId:   "4",
		PlayerName: "Apricot",
		Game:       "Tetris",
		Score:      2,
	}
	score3 := models.Score{
		PlayerId:   "4",
		PlayerName: "Apricot",
		Game:       "Fetch",
		Score:      3,
	}
	score4 := models.Score{
		PlayerId:   "3",
		PlayerName: "Joseph",
		Game:       "Fetch",
		Score:      4,
	}

	want := []models.Score{
		score2,
	}

	d := createTestDynamoScoreDatabase()
	ctx := context.Background()
	err := d.PutScore(ctx, score1)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	err = d.PutScore(ctx, score2)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	err = d.PutScore(ctx, score3)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	err = d.PutScore(ctx, score4)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	scoreRequest := models.PlayerScoreRequest{
		ScoreRequest: models.ScoreRequest{
			Game:  "Tetris",
			Limit: 10,
		},
		PlayerId: "4",
	}
	scores, err := d.GetTopPlayerScores(ctx, scoreRequest)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if diff := cmp.Diff(want, scores); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGetTopRanks(t *testing.T) {
	score1 := models.Score{
		PlayerId:   "2",
		PlayerName: "Bananalord",
		Game:       "Comedy",
		Score:      100,
		Timestamp:  333,
	}
	score2 := models.Score{
		PlayerId:   "2",
		PlayerName: "Bananalord",
		Game:       "Comedy",
		Score:      150,
		Timestamp:  111,
	}
	score3 := models.Score{
		PlayerId:   "5",
		PlayerName: "Mongoose",
		Game:       "Comedy",
		Score:      124,
		Timestamp:  222,
	}

	want := models.Ranks{
		{
			Position:   1,
			PlayerName: "Bananalord",
			Score:      150,
			Timestamp:  111,
		},
		{
			Position:   2,
			PlayerName: "Mongoose",
			Score:      124,
			Timestamp:  222,
		},
		{
			Position:   3,
			PlayerName: "Bananalord",
			Score:      100,
			Timestamp:  333,
		},
	}

	d := createTestDynamoScoreDatabase()
	ctx := context.Background()
	err := d.PutScore(ctx, score1)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	err = d.PutScore(ctx, score2)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	err = d.PutScore(ctx, score3)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	ranks, err := d.GetTopRanks(ctx, models.RanksRequest{Game: "Comedy"})
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
	if diff := cmp.Diff(want, ranks); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
