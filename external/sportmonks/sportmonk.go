package sportmonks

type ResponseTopScorers struct {
	Data       []TopscorerItem `json:"data"`
	Pagination `json:"pagination"`
}
type Pagination struct {
	Count       int     `json:"count"`
	PerPage     int     `json:"per_page"`
	CurrentPage int     `json:"current_page"`
	NextPage    *string `json:"next_page"`
	HasMore     bool    `json:"has_more"`
}

type TopscorerItem struct {
	ID            int64       `json:"id"`
	SeasonID      int64       `json:"season_id"`
	PlayerID      int64       `json:"player_id"`
	TypeID        int64       `json:"type_id"`
	Position      int         `json:"position"`
	Total         int         `json:"total"`
	ParticipantID int64       `json:"participant_id"`
	Type          StatType    `json:"type"`
	Player        Player      `json:"player"`
	Participant   Participant `json:"participant"`
	Season        Season      `json:"season"`
}

type StatType struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Code          string `json:"code"`
	DeveloperName string `json:"developer_name"`
	ModelType     string `json:"model_type"`
	StatGroup     string `json:"stat_group"`
}
type Nationality struct {
	ID           int64    `json:"id"`
	ContinentID  int64    `json:"continent_id"`
	Name         string   `json:"name"`
	OfficialName string   `json:"official_name"`
	FifaName     string   `json:"fifa_name"`
	ISO2         string   `json:"iso2"`
	ISO3         string   `json:"iso3"`
	Latitude     string   `json:"latitude"`  // API kirim string
	Longitude    string   `json:"longitude"` // API kirim string
	Borders      []string `json:"borders"`
	ImagePath    string   `json:"image_path"`
}
type Position struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Code          string  `json:"code"`
	DeveloperName string  `json:"developer_name"`
	ModelType     string  `json:"model_type"`
	StatGroup     *string `json:"stat_group"` // bisa null
}
type Player struct {
	ID                 int64       `json:"id"`
	SportID            int64       `json:"sport_id"`
	CountryID          int64       `json:"country_id"`
	NationalityID      int64       `json:"nationality_id"`
	CityID             *int64      `json:"city_id"` // bisa null di contoh lain
	PositionID         int64       `json:"position_id"`
	DetailedPositionID *int64      `json:"detailed_position_id"`
	TypeID             int64       `json:"type_id"`
	Nationality        Nationality `json:"nationality"`
	CommonName         string      `json:"common_name"`
	Firstname          string      `json:"firstname"`
	Lastname           string      `json:"lastname"`
	Name               string      `json:"name"`
	DisplayName        string      `json:"display_name"`
	ImagePath          string      `json:"image_path"`
	Position           Position    `json:"position"`
	Height             *int        `json:"height"`
	Weight             *int        `json:"weight"`
	DateOfBirth        string      `json:"date_of_birth"` // kalau mau lebih strict: time.Time + custom unmarshal
	Gender             string      `json:"gender"`
}

type Participant struct {
	ID          int64   `json:"id"`
	SportID     int64   `json:"sport_id"`
	CountryID   int64   `json:"country_id"`
	VenueID     *int64  `json:"venue_id"`
	Gender      string  `json:"gender"`
	Name        string  `json:"name"`
	ShortCode   *string `json:"short_code"`
	ImagePath   string  `json:"image_path"`
	Founded     *int    `json:"founded"`
	Type        string  `json:"type"`
	Placeholder bool    `json:"placeholder"`

	LastPlayedAt *string `json:"last_played_at"` // format: "YYYY-MM-DD HH:MM:SS"
}

type Season struct {
	ID                      int64   `json:"id"`
	SportID                 int64   `json:"sport_id"`
	LeagueID                int64   `json:"league_id"`
	TieBreakerRuleID        *int64  `json:"tie_breaker_rule_id"`
	Name                    string  `json:"name"`
	Finished                bool    `json:"finished"`
	Pending                 bool    `json:"pending"`
	IsCurrent               bool    `json:"is_current"`
	StartingAt              string  `json:"starting_at"`               // "YYYY-MM-DD"
	EndingAt                string  `json:"ending_at"`                 // "YYYY-MM-DD"
	StandingsRecalculatedAt *string `json:"standings_recalculated_at"` // "YYYY-MM-DD HH:MM:SS"
	GamesInCurrentWeek      bool    `json:"games_in_current_week"`
}
