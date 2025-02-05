package models

import (
	"errors"
	"fmt"
	"strconv"
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
		return ScoreRequest{}, errors.New("Expected a player_name")
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return ScoreRequest{}, fmt.Errorf("Failed to parse limit: %w", err)
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
