package parser

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/token"
)

func TestParseStatement(t *testing.T) {
	var input string
	var stmts []*ast.Statement

	input = "select 1;select 2;select 3;"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 4, "select 1;")
	testPos(t, stmts[0], genPosOneline(1), genPosOneline(9))
	testStatement(t, stmts[1], 4, "select 2;")
	testPos(t, stmts[1], genPosOneline(10), genPosOneline(18))
	testStatement(t, stmts[2], 4, "select 3;")
	testPos(t, stmts[2], genPosOneline(19), genPosOneline(27))

	input = "select 1;select 2;select 3"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 4, "select 1;")
	testPos(t, stmts[0], genPosOneline(1), genPosOneline(9))
	testStatement(t, stmts[1], 4, "select 2;")
	testPos(t, stmts[1], genPosOneline(10), genPosOneline(18))
	testStatement(t, stmts[2], 3, "select 3")
	testPos(t, stmts[2], genPosOneline(19), genPosOneline(26))
}

func TestParseParenthesis(t *testing.T) {
	testcases := []struct {
		name    string
		input   string
		checkFn func(t *testing.T, stmts []*ast.Statement, input string)
	}{
		{
			name:  "single",
			input: "(3)",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				t.Helper()
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testParenthesis(t, list[0], input)
				testPos(t, stmts[0], genPosOneline(1), genPosOneline(3))
			},
		},
		{
			name:  "with operator",
			input: "(3 - 4)",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				t.Helper()
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testParenthesis(t, list[0], input)
				testPos(t, stmts[0], genPosOneline(1), genPosOneline(7))
			},
		},
		{
			name:  "inner parenthesis",
			input: "(1 * 2 + (3 - 4))",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				t.Helper()
				testStatement(t, stmts[0], 1, input)
				list := stmts[0].GetTokens()
				testParenthesis(t, list[0], input)
				testPos(t, stmts[0], genPosOneline(1), genPosOneline(17))
			},
		},
		{
			name:  "with select",
			input: "select (select (x3) x2) and (y2) bar",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				t.Helper()
				testStatement(t, stmts[0], 9, input)

				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testParenthesis(t, list[2], "(select (x3) x2)")
				testItem(t, list[3], " ")
				testItem(t, list[4], "and")
				testItem(t, list[5], " ")
				testParenthesis(t, list[6], "(y2)")
				testItem(t, list[7], " ")
				testIdentifier(t, list[8], `bar`)

				parenthesis := testTokenList(t, list[2], 7).GetTokens()
				testItem(t, parenthesis[0], "(")
				testItem(t, parenthesis[1], "select")
				testItem(t, parenthesis[2], " ")
				testParenthesis(t, parenthesis[3], "(x3)")
				testItem(t, parenthesis[4], " ")
				testIdentifier(t, parenthesis[5], "x2")
				testItem(t, parenthesis[6], ")")
			},
		},
		{
			name:  "not close parenthesis",
			input: "select (select (x3) x2 and (y2) bar",
			checkFn: func(t *testing.T, stmts []*ast.Statement, input string) {
				t.Helper()

				list := stmts[0].GetTokens()
				testItem(t, list[0], "select")
				testItem(t, list[1], " ")
				testItem(t, list[2], "(")
				testItem(t, list[3], "select")
				testItem(t, list[4], " ")
				testParenthesis(t, list[5], "(x3)")
				testItem(t, list[6], " ")
				testIdentifier(t, list[7], "x2")
				testItem(t, list[8], " ")
				testItem(t, list[9], "and")
				testItem(t, list[10], " ")
				testParenthesis(t, list[11], "(y2)")
				testItem(t, list[12], " ")
				testIdentifier(t, list[13], "bar")
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			stmts := parseInit(t, tt.input)
			tt.checkFn(t, stmts, tt.input)
		})
	}
}

func TestParseWhere(t *testing.T) {
	input := "select * from foo where bar = 1 order by id desc"
	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 13, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testItem(t, list[2], "*")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from foo ")

	testWhere(t, list[5], "where bar = 1 ")
	testItem(t, list[6], "order")
	testItem(t, list[7], " ")
	testItem(t, list[8], "by")
	testItem(t, list[9], " ")
	testIdentifier(t, list[10], "id")
	testItem(t, list[11], " ")
	testItem(t, list[12], "desc")

	where := testTokenList(t, list[5], 4).GetTokens()
	testItem(t, where[0], "where")
	testItem(t, where[1], " ")
	testComparison(t, where[2], "bar = 1")
	testItem(t, where[3], " ")
}

func TestParseFrom(t *testing.T) {
	var input string
	var stmts []*ast.Statement
	var list []ast.Node

	input = "select * from abc"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 5, input)
	list = stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testPos(t, list[0], genPosOneline(1), genPosOneline(7))
	testItem(t, list[1], " ")
	testPos(t, list[1], genPosOneline(7), genPosOneline(8))
	testItem(t, list[2], "*")
	testPos(t, list[2], genPosOneline(8), genPosOneline(9))
	testItem(t, list[3], " ")
	testPos(t, list[3], genPosOneline(9), genPosOneline(10))
	testFrom(t, list[4], "from abc")
	testPos(t, list[4], genPosOneline(10), genPosOneline(15))

	input = "select from abc"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 3, input)
	list = stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testFrom(t, list[2], "from abc")

	input = "select * from "
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 5, input)
	list = stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testItem(t, list[2], "*")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from ")
	testPos(t, list[4], genPosOneline(10), genPosOneline(14))

	list = testTokenList(t, list[4], 2).GetTokens()
	testItem(t, list[0], "from")
	testPos(t, list[0], genPosOneline(10), genPosOneline(14))
	testItem(t, list[1], " ")
	testPos(t, list[1], genPosOneline(14), genPosOneline(15))
}

func TestParseJoin(t *testing.T) {
	input := "select * from abc join efd"

	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 6, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testItem(t, list[2], "*")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from abc ")
	testJoin(t, list[5], "join efd")
}

func TestParseJoin_WithOn(t *testing.T) {
	input := "select * from abc join efd on abc.id = efd.id"

	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 9, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testItem(t, list[2], "*")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from abc ")
	testJoin(t, list[5], "join efd ")
	testItem(t, list[6], "on")
	testItem(t, list[7], " ")
	testComparison(t, list[8], "abc.id = efd.id")
}

func TestParseWhere_NotFoundClose(t *testing.T) {
	input := "select * from foo where bar = 1"
	src := bytes.NewBuffer([]byte(input))
	parser, err := NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}

	got, err := parser.Parse()
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}
	wantStmtLen := 1
	if wantStmtLen != len(got.GetTokens()) {
		t.Errorf("Statements does not contain %d statements, got %d", wantStmtLen, len(got.GetTokens()))
	}
	var stmts []*ast.Statement
	for _, node := range got.GetTokens() {
		stmt, ok := node.(*ast.Statement)
		if !ok {
			t.Fatalf("invalid type want Statement got %T", stmt)
		}
		stmts = append(stmts, stmt)
	}
	testStatement(t, stmts[0], 6, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testItem(t, list[2], "*")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from foo ")
	testWhere(t, list[5], "where bar = 1")

	where := testTokenList(t, list[5], 3).GetTokens()
	testItem(t, where[0], "where")
	testItem(t, where[1], " ")
	testComparison(t, where[2], "bar = 1")
}

func TestParseWhere_WithParenthesis(t *testing.T) {
	input := "select x from (select y from foo where bar = 1) z"
	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 5, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testIdentifier(t, list[2], "x")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from (select y from foo where bar = 1) z")

	from := testTokenList(t, list[4], 5).GetTokens()
	parenthesis := testTokenList(t, from[2], 11).GetTokens()
	testItem(t, parenthesis[0], "(")
	testItem(t, parenthesis[1], "select")
	testItem(t, parenthesis[2], " ")
	testIdentifier(t, parenthesis[3], "y")
	testItem(t, parenthesis[4], " ")
	testItem(t, parenthesis[5], "from")
	testItem(t, parenthesis[6], " ")
	testIdentifier(t, parenthesis[7], "foo")
	testItem(t, parenthesis[8], " ")
	testWhere(t, parenthesis[9], "where bar = 1")
	testItem(t, parenthesis[10], ")")
}

func TestParseFunction(t *testing.T) {
	input := `foo()`
	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 1, input)

	list := stmts[0].GetTokens()
	testFunction(t, list[0], "foo()")
}

func TestParsePeriod_Double(t *testing.T) {
	input := `a.*, b.id`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 1, input)

	list := stmts[0].GetTokens()
	testIdentifierList(t, list[0], input)

	il := testTokenList(t, list[0], 4).GetTokens()
	testMemberIdentifier(t, il[0], "a.*")
	testItem(t, il[1], ",")
	testItem(t, il[2], " ")
	testMemberIdentifier(t, il[3], "b.id")
}

func TestParsePeriod_WithWildcard(t *testing.T) {
	input := `a.*`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 1, input)

	list := stmts[0].GetTokens()
	testMemberIdentifier(t, list[0], "a.*")
}

func TestParsePeriod_Invalid(t *testing.T) {
	input := `a.`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 1, input)

	list := stmts[0].GetTokens()
	testMemberIdentifier(t, list[0], "a.")
}

func TestParsePeriod_InvalidWithSelect(t *testing.T) {
	input := `SELECT foo. FROM foo`
	stmts := parseInit(t, input)

	testStatement(t, stmts[0], 5, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "SELECT")
	testItem(t, list[1], " ")
	testMemberIdentifier(t, list[2], "foo.")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "FROM foo")
}

func TestParseIdentifier(t *testing.T) {
	input := `select foo.bar from "myschema"."table"`
	src := bytes.NewBuffer([]byte(input))
	parser, err := NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}

	got, err := parser.Parse()
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}
	wantStmtLen := 1
	if wantStmtLen != len(got.GetTokens()) {
		t.Errorf("Statements does not contain %d statements, got %d", wantStmtLen, len(got.GetTokens()))
	}
	var stmts []*ast.Statement
	for _, node := range got.GetTokens() {
		stmt, ok := node.(*ast.Statement)
		if !ok {
			t.Fatalf("invalid type want Statement got %T", stmt)
		}
		stmts = append(stmts, stmt)
	}
	testStatement(t, stmts[0], 5, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testMemberIdentifier(t, list[2], "foo.bar")
	testItem(t, list[3], " ")
	testFrom(t, list[4], `from "myschema"."table"`)
}

func TestParseOperator(t *testing.T) {
	var input string
	var stmts []*ast.Statement
	var list []ast.Node

	input = "foo+100"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testOperator(t, list[0], input)

	input = "foo + 100"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testOperator(t, list[0], input)

	input = "foo*100"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testOperator(t, list[0], input)
}

func TestParseComparison(t *testing.T) {
	var input string
	var stmts []*ast.Statement
	var list []ast.Node

	input = "foo = 25.5"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)

	input = "foo = 'bar'"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)

	input = "(3 + 4) = 7"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)

	input = "foo = DATE(bar.baz)"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)

	input = "foo = DATE(bar.baz)"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)

	input = "DATE(foo.bar) = bar.baz"
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testComparison(t, list[0], input)
}

func TestParseAliased(t *testing.T) {
	input := `select foo as bar from mytable`
	stmts := parseInit(t, input)
	testStatement(t, stmts[0], 5, input)

	list := stmts[0].GetTokens()
	testItem(t, list[0], "select")
	testItem(t, list[1], " ")
	testAliased(t, list[2], "foo as bar")
	testItem(t, list[3], " ")
	testFrom(t, list[4], "from mytable")
}

func TestParseIdentifierList(t *testing.T) {
	var input string
	var stmts []*ast.Statement
	var list []ast.Node

	input = `foo, bar`
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testIdentifierList(t, list[0], input)

	input = `sum(a), sum(b)`
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testIdentifierList(t, list[0], input)

	input = `sum(a) as x, b as y`
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testIdentifierList(t, list[0], input)

	input = `foo, bar, hoge`
	stmts = parseInit(t, input)
	testStatement(t, stmts[0], 1, input)
	list = stmts[0].GetTokens()
	testIdentifierList(t, list[0], input)
}

func parseInit(t *testing.T, input string) []*ast.Statement {
	t.Helper()
	src := bytes.NewBuffer([]byte(input))
	parser, err := NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}

	parsed, err := parser.Parse()
	if err != nil {
		t.Fatalf("error %+v\n", err)
	}

	var stmts []*ast.Statement
	for _, node := range parsed.GetTokens() {
		stmt, ok := node.(*ast.Statement)
		if !ok {
			t.Fatalf("invalid type want Statement parsed %T", stmt)
		}
		stmts = append(stmts, stmt)
	}
	return stmts
}

func testTokenList(t *testing.T, node ast.Node, length int) ast.TokenList {
	t.Helper()
	list, ok := node.(ast.TokenList)
	if !ok {
		t.Fatalf("invalid type want GetTokens got %T", node)
	}
	if length != len(list.GetTokens()) {
		t.Fatalf("Statements does not contain %d statements, got %d", length, len(list.GetTokens()))
	}
	return list
}

func testStatement(t *testing.T, stmt *ast.Statement, length int, expect string) {
	t.Helper()
	if length != len(stmt.GetTokens()) {
		t.Fatalf("Statements does not contain %d statements, got %d, (expect %q got: %q)", length, len(stmt.GetTokens()), expect, stmt.String())
	}
	if expect != stmt.String() {
		t.Errorf("expected %q, got %q", expect, stmt.String())
	}
}

func testItem(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	item, ok := node.(*ast.Item)
	if !ok {
		t.Errorf("invalid type want Item got %T", node)
	}
	if expect != item.String() {
		t.Errorf("expected %q, got %q", expect, item.String())
	}
}

func testMemberIdentifier(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.MemberIdentifer)
	if !ok {
		t.Errorf("invalid type want MemberIdentifer got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testIdentifier(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Identifer)
	if !ok {
		t.Errorf("invalid type want Identifier got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testOperator(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Operator)
	if !ok {
		t.Errorf("invalid type want Operator got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testComparison(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Comparison)
	if !ok {
		t.Errorf("invalid type want Comparison got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testParenthesis(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Parenthesis)
	if !ok {
		t.Errorf("invalid type want Parenthesis got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testFunction(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.FunctionLiteral)
	if !ok {
		t.Errorf("invalid type want Function got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testWhere(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.WhereClause)
	if !ok {
		t.Errorf("invalid type want Where got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testFrom(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.FromClause)
	if !ok {
		t.Errorf("invalid type want From got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testJoin(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.JoinClause)
	if !ok {
		t.Errorf("invalid type want Join got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testAliased(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.Aliased)
	if !ok {
		t.Errorf("invalid type want Identifier got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testIdentifierList(t *testing.T, node ast.Node, expect string) {
	t.Helper()
	_, ok := node.(*ast.IdentiferList)
	if !ok {
		t.Errorf("invalid type want IdentiferList got %T", node)
	}
	if expect != node.String() {
		t.Errorf("expected %q, got %q", expect, node.String())
	}
}

func testPos(t *testing.T, node ast.Node, pos, end token.Pos) {
	t.Helper()
	if !reflect.DeepEqual(pos, node.Pos()) {
		t.Errorf("PosExpected %+v, got %+v", pos, node.Pos())
	}
	if !reflect.DeepEqual(end, node.End()) {
		t.Errorf("EndExpected %+v, got %+v", end, node.End())
	}
}

func genPosOneline(col int) token.Pos {
	return token.Pos{Line: 1, Col: col}
}
