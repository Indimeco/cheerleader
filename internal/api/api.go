package api

import (
	"errors"
	"regexp"
	"strings"
)

type ApiRoutes struct {
	ScoresByPlayer string
	RanksByPlayer  string
	Scores         string
	Ranks          string
}

func NewApiRoutes() ApiRoutes {
	return ApiRoutes{
		ScoresByPlayer: "/{game}/{player_id}/scores",
		RanksByPlayer:  "/{game}/{player_id}/ranks",
		Ranks:          "/{game}/ranks",
	}
}

type ApiDefinition struct {
	Route    string
	PlayerId string
	Game     string
}

func EventPathToApiDefinition(path string) (ApiDefinition, error) {
	type apiDescription struct {
		regex           regexp.Regexp
		hasGamePath     bool
		hasPlayerIdPath bool
	}
	routes := NewApiRoutes()
	apiPaths := map[string]apiDescription{
		routes.ScoresByPlayer: {regex: *regexp.MustCompile(`^/[\w\d]+/[\w\d]+/scores/?$`), hasGamePath: true, hasPlayerIdPath: true},
		routes.RanksByPlayer:  {regex: *regexp.MustCompile(`^/[\w\d]+/[\w\d]+/ranks/?$`), hasGamePath: true, hasPlayerIdPath: true},
		routes.Ranks:          {regex: *regexp.MustCompile(`^/[\w\d]+/ranks/?$`), hasGamePath: true},
	}

	for k, v := range apiPaths {
		definition := ApiDefinition{}
		if v.regex.Match([]byte(path)) {
			definition.Route = k
			parts := strings.Split(path, "/")
			if v.hasGamePath {
				definition.Game = parts[1]
			}
			if v.hasPlayerIdPath {
				definition.PlayerId = parts[2]
			}

			return definition, nil
		}
	}

	return ApiDefinition{}, errors.New("No matching api found")
}
