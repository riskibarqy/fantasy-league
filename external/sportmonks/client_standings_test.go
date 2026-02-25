package sportmonks

import "testing"

func TestParseStandings_PrefersOverallDetailValues(t *testing.T) {
	t.Parallel()

	items := []map[string]any{
		{
			"participant_id": float64(6733),
			"position":       float64(1),
			"points":         float64(0),
			"details": map[string]any{
				"data": []any{
					standingDetail(119, "home-matches-played", 11),
					standingDetail(120, "away-matches-played", 11),
					standingDetail(121, "home-won", 8),
					standingDetail(122, "away-won", 8),
					standingDetail(129, "overall-matches-played", 22),
					standingDetail(130, "overall-won", 16),
					standingDetail(131, "overall-draw", 2),
					standingDetail(132, "overall-lost", 4),
					standingDetail(133, "overall-goals-for", 50),
					standingDetail(134, "overall-conceded", 27),
					standingDetail(179, "overall-goal-difference", 23),
					standingDetail(187, "overall-points", 50),
				},
			},
		},
	}

	parsed := parseStandings(items)
	if len(parsed) != 1 {
		t.Fatalf("expected one standing row, got=%d", len(parsed))
	}

	row := parsed[0]
	if row.Played != 22 {
		t.Fatalf("expected played=22, got=%d", row.Played)
	}
	if row.Won != 16 {
		t.Fatalf("expected won=16, got=%d", row.Won)
	}
	if row.Draw != 2 {
		t.Fatalf("expected draw=2, got=%d", row.Draw)
	}
	if row.Lost != 4 {
		t.Fatalf("expected lost=4, got=%d", row.Lost)
	}
	if row.GoalsFor != 50 {
		t.Fatalf("expected goals_for=50, got=%d", row.GoalsFor)
	}
	if row.GoalsAgainst != 27 {
		t.Fatalf("expected goals_against=27, got=%d", row.GoalsAgainst)
	}
	if row.GoalDifference != 23 {
		t.Fatalf("expected goal_difference=23, got=%d", row.GoalDifference)
	}
	if row.Points != 50 {
		t.Fatalf("expected points=50, got=%d", row.Points)
	}
}

func TestParseStandings_MapsOverallConcededWithoutTypeID(t *testing.T) {
	t.Parallel()

	items := []map[string]any{
		{
			"participant_id": float64(10211),
			"position":       float64(2),
			"details": map[string]any{
				"data": []any{
					map[string]any{
						"value": float64(22),
						"type": map[string]any{
							"data": map[string]any{
								"developer_name": "overall-matches-played",
							},
						},
					},
					map[string]any{
						"value": float64(29),
						"type": map[string]any{
							"data": map[string]any{
								"code": "overall-conceded",
							},
						},
					},
				},
			},
		},
	}

	parsed := parseStandings(items)
	if len(parsed) != 1 {
		t.Fatalf("expected one standing row, got=%d", len(parsed))
	}
	row := parsed[0]
	if row.Played != 22 {
		t.Fatalf("expected played=22, got=%d", row.Played)
	}
	if row.GoalsAgainst != 29 {
		t.Fatalf("expected goals_against=29, got=%d", row.GoalsAgainst)
	}
}

func standingDetail(typeID int64, developerName string, value int) map[string]any {
	return map[string]any{
		"type_id": float64(typeID),
		"value":   float64(value),
		"type": map[string]any{
			"data": map[string]any{
				"id":             float64(typeID),
				"developer_name": developerName,
			},
		},
	}
}
