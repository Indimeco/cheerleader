package models

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

func NewScoreFromParams(params map[string]string) (Score, error) {
	game, ok := params["game"]
	if !ok {
		return Score{}, errors.New("Expected a game")
	}
	playerId, ok := params["player_id"]
	if !ok {
		return Score{}, errors.New("Expected a player_id")
	}
	sScore, ok := params["score"]
	if !ok {
		return Score{}, errors.New("Expected a score")
	}
	score, err := strconv.Atoi(sScore)
	if err != nil {
		return Score{}, fmt.Errorf("Unable to parse score: %w", err)
	}
	playerName, ok := params["player_name"]
	if !ok {
		return Score{}, errors.New("Expected a player_name")
	}

	return Score{
		PlayerId:   playerId,
		PlayerName: playerName,
		Game:       game,
		Score:      score,
	}, nil
}

func NewScoreRequestFromParams(params map[string]string) (ScoreRequest, error) {
	game, ok := params["game"]
	if !ok {
		return ScoreRequest{}, errors.New("Expected a game")
	}
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

func NewPlayerScoreRequestFromParams(params map[string]string) (PlayerScoreRequest, error) {
	scoreRequest, err := NewScoreRequestFromParams(params)
	if err != nil {
		return PlayerScoreRequest{}, err
	}
	playerId, ok := params["player_id"]
	if !ok {
		return PlayerScoreRequest{}, errors.New("Expected a player_id")
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
		case "PlayerName":
			{

				str, ok := kv.(*types.AttributeValueMemberS)
				if !ok {
					return errors.New("Wrong type stored at PlayerName")
				}
				s.PlayerName = str.Value
			}
		}
	}
	return nil
}
