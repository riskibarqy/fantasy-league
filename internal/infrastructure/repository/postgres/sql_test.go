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

func TestIsUnnamedPreparedStatementMissing(t *testing.T) {
	t.Run("matches statement missing message", func(t *testing.T) {
		err := fakeErr("pq: unnamed prepared statement does not exist (26000)")
		if !isUnnamedPreparedStatementMissing(err) {
			t.Fatalf("expected true for statement missing error")
		}
	})

	t.Run("matches by 26000 code", func(t *testing.T) {
		err := fakeErr("pq: prepared statement missing (26000)")
		if !isUnnamedPreparedStatementMissing(err) {
			t.Fatalf("expected true for 26000 prepared statement error")
		}
	})

	t.Run("ignores unrelated error", func(t *testing.T) {
		err := fakeErr("pq: relation lineups does not exist")
		if isUnnamedPreparedStatementMissing(err) {
			t.Fatalf("expected false for unrelated error")
		}
	})
}

func TestQuoteLiteral(t *testing.T) {
	got := quoteLiteral("o'hara")
	if got != "'o''hara'" {
		t.Fatalf("unexpected quoted literal: %s", got)
	}
}

type fakeErr string

func (e fakeErr) Error() string { return string(e) }
