package models

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/go-cmp/cmp"
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

	result, err := NewScoreFromParams("tag", "goosey", body)
	if err != nil {
		t.Errorf("want nil, got %v", err)
	}
	diff := cmp.Diff(want, result)
	if diff != "" {
		t.Errorf("mismatch (want +, got -)\n%v", diff)
	}
}
