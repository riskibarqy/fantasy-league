package querybuilder

import (
	"fmt"
	"strconv"
	"strings"
)

type Condition interface {
	appendSQL(buf *strings.Builder, args *[]any, argIndex *int)
}

type eqCondition struct {
	column string
	value  any
}

func Eq(column string, value any) Condition {
	return eqCondition{column: column, value: value}
}

func (c eqCondition) appendSQL(buf *strings.Builder, args *[]any, argIndex *int) {
	buf.WriteString(c.column)
	buf.WriteString(" = ")
	buf.WriteString(placeholder(*argIndex))
	*args = append(*args, c.value)
	*argIndex = *argIndex + 1
}

type inCondition struct {
	column string
	values []any
}

func In(column string, values []any) Condition {
	return inCondition{column: column, values: values}
}

func (c inCondition) appendSQL(buf *strings.Builder, args *[]any, argIndex *int) {
	if len(c.values) == 0 {
		buf.WriteString("1=0")
		return
	}

	buf.WriteString(c.column)
	buf.WriteString(" IN (")
	for i, v := range c.values {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(placeholder(*argIndex))
		*args = append(*args, v)
		*argIndex = *argIndex + 1
	}
	buf.WriteString(")")
}

type isNullCondition struct {
	column string
}

func IsNull(column string) Condition {
	return isNullCondition{column: column}
}

func (c isNullCondition) appendSQL(buf *strings.Builder, _ *[]any, _ *int) {
	buf.WriteString(c.column)
	buf.WriteString(" IS NULL")
}

type exprCondition struct {
	expr string
	args []any
}

func Expr(expr string, args ...any) Condition {
	return exprCondition{expr: expr, args: args}
}

func (c exprCondition) appendSQL(buf *strings.Builder, args *[]any, argIndex *int) {
	buf.WriteString(rewritePlaceholders(c.expr, c.args, args, argIndex))
}

type eqLiteralCondition struct {
	column string
	value  string
}

func EqLiteral(column, value string) Condition {
	return eqLiteralCondition{column: column, value: value}
}

func (c eqLiteralCondition) appendSQL(buf *strings.Builder, _ *[]any, _ *int) {
	buf.WriteString(c.column)
	buf.WriteString(" = ")
	buf.WriteString(quoteLiteral(c.value))
}

type SelectBuilder struct {
	columns []string
	table   string
	where   []Condition
	groupBy []string
	orderBy []string
	limit   int
}

func Select(columns ...string) *SelectBuilder {
	return &SelectBuilder{columns: append([]string(nil), columns...)}
}

func (b *SelectBuilder) From(table string) *SelectBuilder {
	b.table = table
	return b
}

func (b *SelectBuilder) Where(conditions ...Condition) *SelectBuilder {
	b.where = append(b.where, conditions...)
	return b
}

func (b *SelectBuilder) OrderBy(parts ...string) *SelectBuilder {
	b.orderBy = append(b.orderBy, parts...)
	return b
}

func (b *SelectBuilder) GroupBy(parts ...string) *SelectBuilder {
	b.groupBy = append(b.groupBy, parts...)
	return b
}

func (b *SelectBuilder) Limit(limit int) *SelectBuilder {
	b.limit = limit
	return b
}

func (b *SelectBuilder) ToSQL() (string, []any, error) {
	if len(b.columns) == 0 {
		return "", nil, fmt.Errorf("select columns are required")
	}
	if strings.TrimSpace(b.table) == "" {
		return "", nil, fmt.Errorf("select table is required")
	}

	var buf strings.Builder
	buf.WriteString("SELECT ")
	buf.WriteString(strings.Join(b.columns, ", "))
	buf.WriteString(" FROM ")
	buf.WriteString(b.table)

	args := make([]any, 0, len(b.where))
	argIndex := 1
	appendWhereClause(&buf, b.where, &args, &argIndex)
	appendGroupByClause(&buf, b.groupBy)
	appendOrderByClause(&buf, b.orderBy)
	appendLimitClause(&buf, b.limit)

	return buf.String(), args, nil
}

type InsertBuilder struct {
	table   string
	columns []string
	rows    [][]any
	suffix  string
}

func InsertInto(table string) *InsertBuilder {
	return &InsertBuilder{table: table}
}

func (b *InsertBuilder) Columns(columns ...string) *InsertBuilder {
	b.columns = append([]string(nil), columns...)
	return b
}

func (b *InsertBuilder) Values(values ...any) *InsertBuilder {
	b.rows = append(b.rows, append([]any(nil), values...))
	return b
}

func (b *InsertBuilder) Suffix(sql string) *InsertBuilder {
	b.suffix = strings.TrimSpace(sql)
	return b
}

func (b *InsertBuilder) ToSQL() (string, []any, error) {
	if strings.TrimSpace(b.table) == "" {
		return "", nil, fmt.Errorf("insert table is required")
	}
	if len(b.columns) == 0 {
		return "", nil, fmt.Errorf("insert columns are required")
	}
	if len(b.rows) == 0 {
		return "", nil, fmt.Errorf("insert values are required")
	}

	var buf strings.Builder
	buf.WriteString("INSERT INTO ")
	buf.WriteString(b.table)
	buf.WriteString(" (")
	buf.WriteString(strings.Join(b.columns, ", "))
	buf.WriteString(") VALUES ")

	args := make([]any, 0, len(b.rows)*len(b.columns))
	argIndex := 1
	for rowIdx, row := range b.rows {
		if len(row) != len(b.columns) {
			return "", nil, fmt.Errorf("insert row %d has %d values, expected %d", rowIdx, len(row), len(b.columns))
		}
		if rowIdx > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("(")
		for colIdx, value := range row {
			if colIdx > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(placeholder(argIndex))
			args = append(args, value)
			argIndex++
		}
		buf.WriteString(")")
	}

	if b.suffix != "" {
		buf.WriteString(" ")
		buf.WriteString(rewritePlaceholders(b.suffix, nil, &args, &argIndex))
	}

	return buf.String(), args, nil
}

type setClause struct {
	column string
	value  any
	expr   bool
}

type UpdateBuilder struct {
	table  string
	sets   []setClause
	where  []Condition
	suffix string
}

func Update(table string) *UpdateBuilder {
	return &UpdateBuilder{table: table}
}

func (b *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	b.sets = append(b.sets, setClause{column: column, value: value})
	return b
}

func (b *UpdateBuilder) SetExpr(column, expr string, args ...any) *UpdateBuilder {
	b.sets = append(b.sets, setClause{
		column: column,
		value:  exprCondition{expr: expr, args: args},
		expr:   true,
	})
	return b
}

func (b *UpdateBuilder) Where(conditions ...Condition) *UpdateBuilder {
	b.where = append(b.where, conditions...)
	return b
}

func (b *UpdateBuilder) Suffix(sql string) *UpdateBuilder {
	b.suffix = strings.TrimSpace(sql)
	return b
}

func (b *UpdateBuilder) ToSQL() (string, []any, error) {
	if strings.TrimSpace(b.table) == "" {
		return "", nil, fmt.Errorf("update table is required")
	}
	if len(b.sets) == 0 {
		return "", nil, fmt.Errorf("update sets are required")
	}

	var buf strings.Builder
	buf.WriteString("UPDATE ")
	buf.WriteString(b.table)
	buf.WriteString(" SET ")

	args := make([]any, 0, len(b.sets)+len(b.where))
	argIndex := 1
	for i, s := range b.sets {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(s.column)
		buf.WriteString(" = ")

		if s.expr {
			expr, ok := s.value.(exprCondition)
			if !ok {
				return "", nil, fmt.Errorf("invalid expression set value for %s", s.column)
			}
			buf.WriteString(rewritePlaceholders(expr.expr, expr.args, &args, &argIndex))
			continue
		}

		buf.WriteString(placeholder(argIndex))
		args = append(args, s.value)
		argIndex++
	}

	appendWhereClause(&buf, b.where, &args, &argIndex)
	if b.suffix != "" {
		buf.WriteString(" ")
		buf.WriteString(rewritePlaceholders(b.suffix, nil, &args, &argIndex))
	}

	return buf.String(), args, nil
}

func appendWhereClause(buf *strings.Builder, conditions []Condition, args *[]any, argIndex *int) {
	if len(conditions) == 0 {
		return
	}
	buf.WriteString(" WHERE ")
	for i, c := range conditions {
		if i > 0 {
			buf.WriteString(" AND ")
		}
		c.appendSQL(buf, args, argIndex)
	}
}

func appendOrderByClause(buf *strings.Builder, orderBy []string) {
	if len(orderBy) == 0 {
		return
	}
	buf.WriteString(" ORDER BY ")
	buf.WriteString(strings.Join(orderBy, ", "))
}

func appendGroupByClause(buf *strings.Builder, groupBy []string) {
	if len(groupBy) == 0 {
		return
	}
	buf.WriteString(" GROUP BY ")
	buf.WriteString(strings.Join(groupBy, ", "))
}

func appendLimitClause(buf *strings.Builder, limit int) {
	if limit <= 0 {
		return
	}
	buf.WriteString(" LIMIT ")
	buf.WriteString(strconv.Itoa(limit))
}

func placeholder(i int) string {
	return "$" + strconv.Itoa(i)
}

func rewritePlaceholders(expr string, exprArgs []any, args *[]any, argIndex *int) string {
	if len(exprArgs) == 0 {
		return expr
	}

	var out strings.Builder
	next := 0
	for i := 0; i < len(expr); i++ {
		if expr[i] == '?' {
			if next >= len(exprArgs) {
				out.WriteByte('?')
				continue
			}
			out.WriteString(placeholder(*argIndex))
			*args = append(*args, exprArgs[next])
			*argIndex = *argIndex + 1
			next++
			continue
		}
		out.WriteByte(expr[i])
	}
	return out.String()
}

func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
