package user

// Principal is the authenticated account identity resolved from Anubis.
type Principal struct {
	UserID string
	Email  string
}
