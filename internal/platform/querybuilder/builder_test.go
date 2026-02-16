package querybuilder

import "testing"

func TestSelectBuilder(t *testing.T) {
	query, args, err := Select("id", "name").
		From("users").
		Where(Eq("tenant_id", "t1"), IsNull("deleted_at")).
		OrderBy("id").
		Limit(10).
		ToSQL()
	if err != nil {
		t.Fatalf("build select query: %v", err)
	}

	wantQuery := "SELECT id, name FROM users WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY id LIMIT 10"
	if query != wantQuery {
		t.Fatalf("unexpected query:\nwant: %s\ngot:  %s", wantQuery, query)
	}
	if len(args) != 1 || args[0] != "t1" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestInsertBuilder(t *testing.T) {
	query, args, err := InsertInto("users").
		Columns("id", "name").
		Values("u1", "name-1").
		Suffix("RETURNING id").
		ToSQL()
	if err != nil {
		t.Fatalf("build insert query: %v", err)
	}

	wantQuery := "INSERT INTO users (id, name) VALUES ($1, $2) RETURNING id"
	if query != wantQuery {
		t.Fatalf("unexpected query:\nwant: %s\ngot:  %s", wantQuery, query)
	}
	if len(args) != 2 || args[0] != "u1" || args[1] != "name-1" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestUpdateBuilder(t *testing.T) {
	query, args, err := Update("users").
		Set("name", "new").
		SetExpr("updated_at", "NOW()").
		Where(Eq("id", "u1")).
		ToSQL()
	if err != nil {
		t.Fatalf("build update query: %v", err)
	}

	wantQuery := "UPDATE users SET name = $1, updated_at = NOW() WHERE id = $2"
	if query != wantQuery {
		t.Fatalf("unexpected query:\nwant: %s\ngot:  %s", wantQuery, query)
	}
	if len(args) != 2 || args[0] != "new" || args[1] != "u1" {
		t.Fatalf("unexpected args: %+v", args)
	}
}
