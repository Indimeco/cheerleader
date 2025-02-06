package main

import "testing"

func TestValidApiPaths(t *testing.T) {
	type test struct {
		input string
		want  apiDefinition
	}
	testCases := []test{
		{input: "/banana/123/scores", want: apiDefinition{path: "/{game}/{player_id}/scores", game: "banana", playerId: "123"}},
		{input: "/pp1/abc/scores", want: apiDefinition{path: "/{game}/{player_id}/scores", game: "pp1", playerId: "abc"}},
		{input: "/pp1/abc/scores/", want: apiDefinition{path: "/{game}/{player_id}/scores", game: "pp1", playerId: "abc"}},
		{input: "/1/sasa/scores", want: apiDefinition{path: "/{game}/{player_id}/scores", game: "1", playerId: "sasa"}},
		{input: "/duck/goose/ranks", want: apiDefinition{path: "/{game}/{player_id}/ranks", game: "duck", playerId: "goose"}},
		{input: "/duck/scores", want: apiDefinition{path: "/{game}/scores", game: "duck", playerId: ""}},
		{input: "/duck/ranks", want: apiDefinition{path: "/{game}/ranks", game: "duck", playerId: ""}},
		{input: "/duck/ranks/", want: apiDefinition{path: "/{game}/ranks", game: "duck", playerId: ""}},
	}

	for _, tc := range testCases {
		got, _ := eventPathToApiDefinition(tc.input)
		if tc.want.path != got.path {
			t.Errorf("want %v, got %v, input %v", tc.want.path, got.path, tc.input)
		}
		if tc.want.game != got.game {
			t.Errorf("want %v, got %v, input %v", tc.want.game, got.game, tc.input)
		}
		if tc.want.playerId != got.playerId {
			t.Errorf("want %v, got %v, input %v", tc.want.playerId, got.playerId, tc.input)
		}
	}
}

func TestInvalidApiPaths(t *testing.T) {
	type test struct {
		input string
	}
	testCases := []test{
		{input: ""},
		{input: "///"},
		{input: "/duck/123//"},
		{input: "/&*/<div></div>"},
		{input: "/duck"},
		{input: "/duck/"},
		{input: "/duck/123/scores/rabbits"},
		{input: "/duck/score"},
	}

	for _, tc := range testCases {
		got, err := eventPathToApiDefinition(tc.input)
		if err == nil || got.path != "" {
			t.Errorf("want %q, got %q, input %q, err %v", "", got, tc.input, err)
		}
	}
}
