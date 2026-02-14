package postgres

import "testing"

func TestIsBindParameterMismatch(t *testing.T) {
	t.Run("matches bind mismatch error", func(t *testing.T) {
		err := fakeErr("pq: bind message supplies 2 parameters, but prepared statement \"\" requires 1 (08P01)")
		if !isBindParameterMismatch(err) {
			t.Fatalf("expected true for bind mismatch error")
		}
	})

	t.Run("ignores unrelated error", func(t *testing.T) {
		err := fakeErr("pq: relation lineups does not exist")
		if isBindParameterMismatch(err) {
			t.Fatalf("expected false for unrelated error")
		}
	})
}

type fakeErr string

func (e fakeErr) Error() string { return string(e) }
