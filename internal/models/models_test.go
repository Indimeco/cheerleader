package models

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMarshalScore(t *testing.T) {
	s := Score{PlayerId: "123", PlayerName: "David Bowie", Game: "Singing", Score: 505}
	wantPk := "123|Singing"
	wantSk := 505
	wantPname := "David Bowie"
	wantGame := "Singing"
	result, err := attributevalue.Marshal(&s)
	if err != nil {
		t.Errorf("want nil, got %v", err)
	}
	resultPk := result.(*types.AttributeValueMemberM).Value["pk"].(*types.AttributeValueMemberS).Value
	resultSk := result.(*types.AttributeValueMemberM).Value["sk"].(*types.AttributeValueMemberN).Value
	resultGame := result.(*types.AttributeValueMemberM).Value["game"].(*types.AttributeValueMemberS).Value
	resultPname := result.(*types.AttributeValueMemberM).Value["pname"].(*types.AttributeValueMemberS).Value

	if resultPk != wantPk {
		t.Errorf("mismatch want %v, got %v", wantPk, resultPk)
	}
	if resultSk != resultSk {
		t.Errorf("mismatch want %v, got %v", wantSk, resultSk)
	}
	if resultGame != wantGame {
		t.Errorf("mismatch want %v, got %v", wantGame, resultGame)
	}
	if resultPname != wantPname {
		t.Errorf("mismatch want %v, got %v", wantPname, resultPname)
	}
}

func TestUnmarshalScore(t *testing.T) {
	av := &types.AttributeValueMemberM{
		Value: map[string]types.AttributeValue{
			"pk":    &types.AttributeValueMemberS{Value: "123|Singing"},
			"sk":    &types.AttributeValueMemberN{Value: "505"},
			"game":  &types.AttributeValueMemberS{Value: "Singing"},
			"pname": &types.AttributeValueMemberS{Value: "David Bowie"},
		},
	}
	want := Score{PlayerId: "123", PlayerName: "David Bowie", Game: "Singing", Score: 505}
	result := Score{}
	err := attributevalue.Unmarshal(av, &result)
	if err != nil {
		t.Errorf("want nil, got %v", err)
	}
	diff := cmp.Diff(want, result)
	if diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestNewScoreFromParams(t *testing.T) {
	want := Score{
		Game:       "tag",
		Score:      99,
		PlayerId:   "goosey",
		PlayerName: "BIG GOOSE",
	}

	body := `
	{
		"score": 99,
		"playerName": "BIG GOOSE"
	}
	`

	result, err := NewScore("tag", "goosey", body)
	if err != nil {
		t.Errorf("want nil, got %v", err)
	}
	diff := cmp.Diff(want, result, cmpopts.IgnoreFields(Score{}, "Timestamp"))
	if diff != "" {
		t.Errorf("mismatch (want +, got -)\n%v", diff)
	}
}

func TestRanksBinarySearch(t *testing.T) {
	ranks := Ranks{
		{Position: 1, PlayerName: "Albus", Score: 100},
		{Position: 2, PlayerName: "Harry", Score: 50},
		{Position: 3, PlayerName: "Potter", Score: 20},
		{Position: 4, PlayerName: "Dobby", Score: 10},
	}

	search := ranks.BinarySearch(100, 0, len(ranks))
	if search != 0 {
		t.Errorf("wanted to find index at %v, got %v", 0, search)
	}
	search = ranks.BinarySearch(50, 0, len(ranks))
	if search != 1 {
		t.Errorf("wanted to find index at %v, got %v", 1, search)
	}
	search = ranks.BinarySearch(20, 0, len(ranks))
	if search != 2 {
		t.Errorf("wanted to find index at %v, got %v", 2, search)
	}
	search = ranks.BinarySearch(10, 0, len(ranks))
	if search != 3 {
		t.Errorf("wanted to find index at %v, got %v", 3, search)
	}
}

func TestRanksAround(t *testing.T) {

	r0 := Rank{Position: 1, PlayerName: "Albus", Score: 100}
	r1 := Rank{Position: 2, PlayerName: "Harry", Score: 50}
	r2 := Rank{Position: 3, PlayerName: "Potter", Score: 20}
	r3 := Rank{Position: 4, PlayerName: "Dobby", Score: 10}
	ranks := Ranks{
		r0, r1, r2, r3,
	}

	got := ranks.Around(0, 1)
	want := Ranks{r0, r1}
	diff := cmp.Diff(want, got)
	if diff != "" {
		t.Errorf("mismatch (-want, got +)\n%v", diff)
	}

	got = ranks.Around(1, 2)
	want = Ranks{r0, r1, r2, r3}
	diff = cmp.Diff(want, got)
	if diff != "" {
		t.Errorf("mismatch (-want, got +)\n%v", diff)
	}

	got = ranks.Around(2, 1)
	want = Ranks{r1, r2, r3}
	diff = cmp.Diff(got, want)
	diff = cmp.Diff(want, got)
	if diff != "" {
		t.Errorf("mismatch (-want, got +)\n%v", diff)
	}
}
