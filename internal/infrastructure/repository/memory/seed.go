package memory

import (
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
)

const (
	LeagueIDLiga1Indonesia = "idn-liga-1-2025"
	LeagueIDPremierLeague  = "eng-premier-league-2025"
)

func SeedLeagues() []league.League {
	return []league.League{
		{
			ID:          LeagueIDLiga1Indonesia,
			Name:        "Liga 1 Indonesia",
			CountryCode: "ID",
			Season:      "2025/2026",
			IsDefault:   true,
		},
		{
			ID:          LeagueIDPremierLeague,
			Name:        "Premier League",
			CountryCode: "GB",
			Season:      "2025/2026",
			IsDefault:   false,
		},
	}
}

func SeedTeams() []team.Team {
	return []team.Team{
		{ID: "idn-persija", LeagueID: LeagueIDLiga1Indonesia, Name: "Persija Jakarta", Short: "PSJ", ImageURL: "https://cdn.example.com/teams/idn-persija.png", PrimaryColor: "#B91C1C", SecondaryColor: "#F87171"},
		{ID: "idn-persib", LeagueID: LeagueIDLiga1Indonesia, Name: "Persib Bandung", Short: "PSB", ImageURL: "https://cdn.example.com/teams/idn-persib.png", PrimaryColor: "#1D4ED8", SecondaryColor: "#60A5FA"},
		{ID: "idn-persebaya", LeagueID: LeagueIDLiga1Indonesia, Name: "Persebaya Surabaya", Short: "PRB", ImageURL: "https://cdn.example.com/teams/idn-persebaya.png", PrimaryColor: "#047857", SecondaryColor: "#34D399"},
		{ID: "idn-baliutd", LeagueID: LeagueIDLiga1Indonesia, Name: "Bali United", Short: "BU", ImageURL: "https://cdn.example.com/teams/idn-baliutd.png", PrimaryColor: "#991B1B", SecondaryColor: "#F59E0B"},
		{ID: "idn-psm", LeagueID: LeagueIDLiga1Indonesia, Name: "PSM Makassar", Short: "PSM", ImageURL: "https://cdn.example.com/teams/idn-psm.png", PrimaryColor: "#7C2D12", SecondaryColor: "#FB923C"},
		{ID: "idn-borneo", LeagueID: LeagueIDLiga1Indonesia, Name: "Borneo FC", Short: "BOR", ImageURL: "https://cdn.example.com/teams/idn-borneo.png", PrimaryColor: "#9A3412", SecondaryColor: "#FDBA74"},
		{ID: "eng-ars", LeagueID: LeagueIDPremierLeague, Name: "Arsenal", Short: "ARS", ImageURL: "https://cdn.example.com/teams/eng-ars.png", PrimaryColor: "#B91C1C", SecondaryColor: "#FCD34D"},
		{ID: "eng-liv", LeagueID: LeagueIDPremierLeague, Name: "Liverpool", Short: "LIV", ImageURL: "https://cdn.example.com/teams/eng-liv.png", PrimaryColor: "#B91C1C", SecondaryColor: "#E5E7EB"},
	}
}

func SeedPlayers() []player.Player {
	return []player.Player{
		{ID: "idn-gk-01", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persija", Name: "Andritany Ardhiyasa", Position: player.PositionGoalkeeper, Price: 90, ImageURL: "https://cdn.example.com/players/idn-gk-01.png"},
		{ID: "idn-gk-02", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persib", Name: "Teja Paku Alam", Position: player.PositionGoalkeeper, Price: 85, ImageURL: "https://cdn.example.com/players/idn-gk-02.png"},
		{ID: "idn-gk-03", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-psm", Name: "Reza Arya", Position: player.PositionGoalkeeper, Price: 80, ImageURL: "https://cdn.example.com/players/idn-gk-03.png"},
		{ID: "idn-def-01", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persija", Name: "Hansamu Yama", Position: player.PositionDefender, Price: 88, ImageURL: "https://cdn.example.com/players/idn-def-01.png"},
		{ID: "idn-def-02", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persib", Name: "Nick Kuipers", Position: player.PositionDefender, Price: 92, ImageURL: "https://cdn.example.com/players/idn-def-02.png"},
		{ID: "idn-def-03", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persebaya", Name: "Dusan Stevanovic", Position: player.PositionDefender, Price: 84, ImageURL: "https://cdn.example.com/players/idn-def-03.png"},
		{ID: "idn-def-04", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-baliutd", Name: "Ricky Fajrin", Position: player.PositionDefender, Price: 80, ImageURL: "https://cdn.example.com/players/idn-def-04.png"},
		{ID: "idn-def-05", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persebaya", Name: "Arief Catur", Position: player.PositionDefender, Price: 72, ImageURL: "https://cdn.example.com/players/idn-def-05.png"},
		{ID: "idn-def-06", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-psm", Name: "Yuran Fernandes", Position: player.PositionDefender, Price: 86, ImageURL: "https://cdn.example.com/players/idn-def-06.png"},
		{ID: "idn-mid-01", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persija", Name: "Maciej Gajos", Position: player.PositionMidfielder, Price: 98, ImageURL: "https://cdn.example.com/players/idn-mid-01.png"},
		{ID: "idn-mid-02", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persib", Name: "Marc Klok", Position: player.PositionMidfielder, Price: 99, ImageURL: "https://cdn.example.com/players/idn-mid-02.png"},
		{ID: "idn-mid-03", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persebaya", Name: "Bruno Moreira", Position: player.PositionMidfielder, Price: 95, ImageURL: "https://cdn.example.com/players/idn-mid-03.png"},
		{ID: "idn-mid-04", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-baliutd", Name: "Eber Bessa", Position: player.PositionMidfielder, Price: 97, ImageURL: "https://cdn.example.com/players/idn-mid-04.png"},
		{ID: "idn-mid-05", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-baliutd", Name: "Mitsuru Maruoka", Position: player.PositionMidfielder, Price: 90, ImageURL: "https://cdn.example.com/players/idn-mid-05.png"},
		{ID: "idn-mid-06", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persib", Name: "Dedi Kusnandar", Position: player.PositionMidfielder, Price: 78, ImageURL: "https://cdn.example.com/players/idn-mid-06.png"},
		{ID: "idn-mid-07", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-borneo", Name: "Stefano Lilipaly", Position: player.PositionMidfielder, Price: 94, ImageURL: "https://cdn.example.com/players/idn-mid-07.png"},
		{ID: "idn-fwd-01", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persija", Name: "Gustavo Almeida", Position: player.PositionForward, Price: 105, ImageURL: "https://cdn.example.com/players/idn-fwd-01.png"},
		{ID: "idn-fwd-02", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persib", Name: "David da Silva", Position: player.PositionForward, Price: 108, ImageURL: "https://cdn.example.com/players/idn-fwd-02.png"},
		{ID: "idn-fwd-03", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-persebaya", Name: "Paulo Henrique", Position: player.PositionForward, Price: 100, ImageURL: "https://cdn.example.com/players/idn-fwd-03.png"},
		{ID: "idn-fwd-04", LeagueID: LeagueIDLiga1Indonesia, TeamID: "idn-psm", Name: "Adilson Silva", Position: player.PositionForward, Price: 97, ImageURL: "https://cdn.example.com/players/idn-fwd-04.png"},
		{ID: "eng-gk-01", LeagueID: LeagueIDPremierLeague, TeamID: "eng-ars", Name: "David Raya", Position: player.PositionGoalkeeper, Price: 92, ImageURL: "https://cdn.example.com/players/eng-gk-01.png"},
		{ID: "eng-def-01", LeagueID: LeagueIDPremierLeague, TeamID: "eng-ars", Name: "William Saliba", Position: player.PositionDefender, Price: 96, ImageURL: "https://cdn.example.com/players/eng-def-01.png"},
		{ID: "eng-mid-01", LeagueID: LeagueIDPremierLeague, TeamID: "eng-liv", Name: "Dominik Szoboszlai", Position: player.PositionMidfielder, Price: 98, ImageURL: "https://cdn.example.com/players/eng-mid-01.png"},
		{ID: "eng-fwd-01", LeagueID: LeagueIDPremierLeague, TeamID: "eng-liv", Name: "Darwin Nunez", Position: player.PositionForward, Price: 104, ImageURL: "https://cdn.example.com/players/eng-fwd-01.png"},
	}
}

func SeedFixtures() []fixture.Fixture {
	return []fixture.Fixture{
		{
			ID:         "fx-idn-001",
			LeagueID:   LeagueIDLiga1Indonesia,
			Gameweek:   1,
			HomeTeam:   "Persija Jakarta",
			AwayTeam:   "Persib Bandung",
			HomeTeamID: "idn-persija",
			AwayTeamID: "idn-persib",
			KickoffAt:  time.Date(2026, 2, 14, 19, 0, 0, 0, time.UTC),
			Venue:      "Jakarta International Stadium",
		},
		{
			ID:         "fx-idn-002",
			LeagueID:   LeagueIDLiga1Indonesia,
			Gameweek:   1,
			HomeTeam:   "Persebaya Surabaya",
			AwayTeam:   "Bali United",
			HomeTeamID: "idn-persebaya",
			AwayTeamID: "idn-baliutd",
			KickoffAt:  time.Date(2026, 2, 15, 12, 30, 0, 0, time.UTC),
			Venue:      "Gelora Bung Tomo",
		},
		{
			ID:         "fx-idn-003",
			LeagueID:   LeagueIDLiga1Indonesia,
			Gameweek:   2,
			HomeTeam:   "Persib Bandung",
			AwayTeam:   "Persebaya Surabaya",
			HomeTeamID: "idn-persib",
			AwayTeamID: "idn-persebaya",
			KickoffAt:  time.Date(2026, 2, 21, 12, 30, 0, 0, time.UTC),
			Venue:      "Gelora Bandung Lautan Api",
		},
		{
			ID:         "fx-idn-004",
			LeagueID:   LeagueIDLiga1Indonesia,
			Gameweek:   2,
			HomeTeam:   "Bali United",
			AwayTeam:   "Persija Jakarta",
			HomeTeamID: "idn-baliutd",
			AwayTeamID: "idn-persija",
			KickoffAt:  time.Date(2026, 2, 22, 12, 30, 0, 0, time.UTC),
			Venue:      "Kapten I Wayan Dipta",
		},
		{
			ID:         "fx-idn-005",
			LeagueID:   LeagueIDLiga1Indonesia,
			Gameweek:   3,
			HomeTeam:   "Persija Jakarta",
			AwayTeam:   "Persebaya Surabaya",
			HomeTeamID: "idn-persija",
			AwayTeamID: "idn-persebaya",
			KickoffAt:  time.Date(2026, 2, 28, 12, 30, 0, 0, time.UTC),
			Venue:      "Jakarta International Stadium",
		},
		{
			ID:         "fx-idn-006",
			LeagueID:   LeagueIDLiga1Indonesia,
			Gameweek:   3,
			HomeTeam:   "Persib Bandung",
			AwayTeam:   "Bali United",
			HomeTeamID: "idn-persib",
			AwayTeamID: "idn-baliutd",
			KickoffAt:  time.Date(2026, 3, 1, 12, 30, 0, 0, time.UTC),
			Venue:      "Gelora Bandung Lautan Api",
		},
		{
			ID:         "fx-eng-001",
			LeagueID:   LeagueIDPremierLeague,
			Gameweek:   1,
			HomeTeam:   "Arsenal",
			AwayTeam:   "Liverpool",
			HomeTeamID: "eng-ars",
			AwayTeamID: "eng-liv",
			KickoffAt:  time.Date(2026, 2, 14, 15, 0, 0, 0, time.UTC),
			Venue:      "Emirates Stadium",
		},
	}
}
