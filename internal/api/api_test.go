package api

import "testing"

func TestValidApiPaths(t *testing.T) {
	type test struct {
		input string
		want  ApiDefinition
	}
	testCases := []test{
		{input: "/banana/123/scores", want: ApiDefinition{Route: "/{game}/{player_id}/scores", Game: "banana", PlayerId: "123"}},
		{input: "/pp1/abc/scores", want: ApiDefinition{Route: "/{game}/{player_id}/scores", Game: "pp1", PlayerId: "abc"}},
		{input: "/pp1/abc/scores/", want: ApiDefinition{Route: "/{game}/{player_id}/scores", Game: "pp1", PlayerId: "abc"}},
		{input: "/1/sasa/scores", want: ApiDefinition{Route: "/{game}/{player_id}/scores", Game: "1", PlayerId: "sasa"}},
		{input: "/duck/goose/ranks", want: ApiDefinition{Route: "/{game}/{player_id}/ranks", Game: "duck", PlayerId: "goose"}},
		{input: "/duck/ranks", want: ApiDefinition{Route: "/{game}/ranks", Game: "duck", PlayerId: ""}},
		{input: "/duck/ranks/", want: ApiDefinition{Route: "/{game}/ranks", Game: "duck", PlayerId: ""}},
	}

	for _, tc := range testCases {
		got, _ := EventPathToApiDefinition(tc.input)
		if tc.want.Route != got.Route {
			t.Errorf("want %v, got %v, input %v", tc.want.Route, got.Route, tc.input)
		}
		if tc.want.Game != got.Game {
			t.Errorf("want %v, got %v, input %v", tc.want.Game, got.Game, tc.input)
		}
		if tc.want.PlayerId != got.PlayerId {
			t.Errorf("want %v, got %v, input %v", tc.want.PlayerId, got.PlayerId, tc.input)
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
		got, err := EventPathToApiDefinition(tc.input)
		if err == nil || got.Route != "" {
			t.Errorf("want %q, got %q, input %q, err %v", "", got, tc.input, err)
		}
	}
}
