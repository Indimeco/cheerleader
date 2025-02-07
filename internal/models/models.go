package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"strconv"
	"strings"
)

type Score struct {
	Game       string
	Score      int
	PlayerId   string
	PlayerName string
}

type ScoreRequest struct {
	Game  string
	Limit int
}

type PlayerScoreRequest struct {
	ScoreRequest
	PlayerId string
}

type Rank struct {
	Score      int    `dynamodbav:"sk"`
	Position   int    `dynamodbav:"-"`
	PlayerName string `dynamodbav:"pname"`
}

type Ranks []Rank

func NewScore(game string, playerId string, requestBody string) (Score, error) {
	type putNewScoreRequestBody struct {
		Score      int    `json:"score"`
		PlayerName string `json:"playerName"`
	}
	b := putNewScoreRequestBody{}
	err := json.Unmarshal([]byte(requestBody), &b)
	if err != nil {
		return Score{}, fmt.Errorf("Failed to parse response body: %w", err)
	}
	fmt.Println(b)
	if b.Score == 0 {
		return Score{}, errors.New("Expected a score")
	}
	if b.PlayerName == "" {
		return Score{}, errors.New("Expected a player_name")
	}
	if len(b.PlayerName) > 32 {
		return Score{}, errors.New("Player name was too long")
	}

	return Score{
		PlayerId:   playerId,
		PlayerName: b.PlayerName,
		Game:       game,
		Score:      b.Score,
	}, nil
}

func NewScoreRequest(params map[string]string, game string) (ScoreRequest, error) {
	limitStr, ok := params["limit"]
	if !ok {
		return ScoreRequest{}, errors.New("Expected a limit")
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return ScoreRequest{}, fmt.Errorf("Failed to parse limit: %w", err)
	}
	if limit > 100 || limit < 0 {
		return ScoreRequest{}, errors.New("Limit must be between 0 and 100")
	}

	return ScoreRequest{
		Game:  game,
		Limit: limit,
	}, nil
}

func NewPlayerScoreRequest(params map[string]string, game string, playerId string) (PlayerScoreRequest, error) {
	scoreRequest, err := NewScoreRequest(params, game)
	if err != nil {
		return PlayerScoreRequest{}, err
	}

	return PlayerScoreRequest{
		scoreRequest,
		playerId,
	}, nil
}

// Fulfills the Unmarshaler interface https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue#Unmarshaler
func (s *Score) UnmarshalDynamoDBAttributeValue(av types.AttributeValue) error {
	avM, ok := av.(*types.AttributeValueMemberM)
	if !ok {
		return nil
	}
	for k, kv := range avM.Value {
		switch k {
		case "pk":
			{
				str, ok := kv.(*types.AttributeValueMemberS)
				if !ok {
					return errors.New("Wrong type stored at pk")
				}
				split := strings.Split(str.Value, "|")
				s.PlayerId = split[0]
				s.Game = split[1]
			}
		case "sk":
			{

				i, ok := kv.(*types.AttributeValueMemberN)
				if !ok {
					return errors.New("Wrong type stored at sk")
				}
				score, err := strconv.Atoi(i.Value)
				if err != nil {
					return fmt.Errorf("Failed to parse sk for score: %w", err)
				}
				s.Score = score
			}
		case "pname":
			{

				str, ok := kv.(*types.AttributeValueMemberS)
				if !ok {
					return errors.New("Wrong type stored at pname")
				}
				s.PlayerName = str.Value
			}
		}
	}
	return nil
}

// Fulfills the Marshaler interface https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue#Marshaler
func (s *Score) MarshalDynamoDBAttributeValue() (types.AttributeValue, error) {
	m := make(map[string]types.AttributeValue)
	score := strconv.Itoa(s.Score)
	m["pk"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("%s|%s", s.PlayerId, s.Game)}
	m["sk"] = &types.AttributeValueMemberN{Value: score}
	m["game"] = &types.AttributeValueMemberS{Value: s.Game}
	m["pname"] = &types.AttributeValueMemberS{Value: s.PlayerName}
	return &types.AttributeValueMemberM{
		Value: m,
	}, nil
}

// func binarySearch returns -1 if the score is not in the ranks, otherwise the index of the score within the ranks
func (r Ranks) BinarySearch(score int, left int, right int) int {
	mid := (right-left)/2 + left
	if left > right {
		return -1
	}
	if r[mid].Score == score {
		return mid
	}
	if r[mid].Score > score {
		return r.BinarySearch(score, mid+1, right)
	}
	return r.BinarySearch(score, left, right-1)
}

func (r Ranks) Around(index int, around int) Ranks {
	if len(r)-1 < index {
		return Ranks{}
	}
	out := make(Ranks, 0, around*2+1)
	// ranks before
	for i := around; i > 0; i-- {
		if index-i >= 0 {
			out = append(out, r[index-i])
		}
	}
	// the index rank
	out = append(out, r[index])
	// ranks after
	for i := 1; i < around+1; i++ {
		if index+i <= len(r)-1 {
			out = append(out, r[index+i])
		}
	}
	return out
}
