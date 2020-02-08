package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/lighttiger2505/sqls/ast"
	"github.com/lighttiger2505/sqls/ast/astutil"
	"github.com/lighttiger2505/sqls/database"
	"github.com/lighttiger2505/sqls/dialect"
	"github.com/lighttiger2505/sqls/parser"
	"github.com/lighttiger2505/sqls/token"
)

type CompletionType int

const (
	_ CompletionType = iota
	CompletionTypeKeyword
	CompletionTypeFunction
	CompletionTypeAlias
	CompletionTypeColumn
	CompletionTypeTable
	CompletionTypeView
	CompletionTypeChange
	CompletionTypeUser
	CompletionTypeDatabase
)

func (ct CompletionType) String() string {
	switch ct {
	case CompletionTypeKeyword:
		return "Keyword"
	case CompletionTypeFunction:
		return "Function"
	case CompletionTypeAlias:
		return "Alias"
	case CompletionTypeColumn:
		return "Column"
	case CompletionTypeTable:
		return "Table"
	case CompletionTypeView:
		return "View"
	case CompletionTypeChange:
		return "Change"
	case CompletionTypeUser:
		return "User"
	case CompletionTypeDatabase:
		return "Database"
	default:
		return ""
	}
}

var keywords = []string{
	"ACCESS", "ADD", "ALL", "ALTER TABLE", "AND", "ANY", "AS",
	"ASC", "AUTO_INCREMENT", "BEFORE", "BEGIN", "BETWEEN",
	"BIGINT", "BINARY", "BY", "CASE", "CHANGE MASTER TO", "CHAR",
	"CHARACTER SET", "CHECK", "COLLATE", "COLUMN", "COMMENT",
	"COMMIT", "CONSTRAINT", "CREATE", "CURRENT",
	"CURRENT_TIMESTAMP", "DATABASE", "DATE", "DECIMAL", "DEFAULT",
	"DELETE FROM", "DESC", "DESCRIBE", "DROP",
	"ELSE", "END", "ENGINE", "ESCAPE", "EXISTS", "FILE", "FLOAT",
	"FOR", "FOREIGN KEY", "FORMAT", "FROM", "FULL", "FUNCTION",
	"GRANT", "GROUP BY", "HAVING", "HOST", "IDENTIFIED", "IN",
	"INCREMENT", "INDEX", "INSERT INTO", "INT", "INTEGER",
	"INTERVAL", "INTO", "IS", "JOIN", "KEY", "LEFT", "LEVEL",
	"LIKE", "LIMIT", "LOCK", "LOGS", "LONG", "MASTER",
	"MEDIUMINT", "MODE", "MODIFY", "NOT", "NULL", "NUMBER",
	"OFFSET", "ON", "OPTION", "OR", "ORDER BY", "OUTER", "OWNER",
	"PASSWORD", "PORT", "PRIMARY", "PRIVILEGES", "PROCESSLIST",
	"PURGE", "REFERENCES", "REGEXP", "RENAME", "REPAIR", "RESET",
	"REVOKE", "RIGHT", "ROLLBACK", "ROW", "ROWS", "ROW_FORMAT",
	"SAVEPOINT", "SELECT", "SESSION", "SET", "SHARE", "SHOW",
	"SLAVE", "SMALLINT", "SMALLINT", "START", "STOP", "TABLE",
	"THEN", "TINYINT", "TO", "TRANSACTION", "TRIGGER", "TRUNCATE",
	"UNION", "UNIQUE", "UNSIGNED", "UPDATE", "USE", "USER",
	"USING", "VALUES", "VARCHAR", "VIEW", "WHEN", "WHERE", "WITH",
}

type Completer struct {
	Conn         *database.MySQLDB
	TableColumns map[string][]*database.TableInfo
}

func NewCompleter() *Completer {
	db := database.NewMysqlDB("root:root@tcp(127.0.0.1:13306)/world")
	return &Completer{
		Conn:         db,
		TableColumns: map[string][]*database.TableInfo{},
	}
}

func (c *Completer) Init() error {
	if err := c.Conn.Open(); err != nil {
		return err
	}
	defer c.Conn.Close()
	tableColumns, err := c.Conn.TableColumns()
	if err != nil {
		return err
	}
	c.TableColumns = tableColumns
	return nil
}

func completionTypeIs(completionTypes []CompletionType, expect CompletionType) bool {
	for _, t := range completionTypes {
		if t == expect {
			return true
		}
	}
	return false
}

func (c *Completer) complete(text string, params CompletionParams) ([]CompletionItem, error) {
	// parse query
	src := bytes.NewBuffer([]byte(text))
	p, err := parser.NewParser(src, &dialect.GenericSQLDialect{})
	if err != nil {
		return nil, err
	}
	parsed, err := p.Parse()
	if err != nil {
		return nil, err
	}

	targetTables := parser.ExtractTable(parsed)

	// fetch database infomation
	columnCandinates := []CompletionItem{}
	for _, info := range targetTables {
		if info.Name != "" {
			if columns, ok := c.TableColumns[strings.ToUpper(info.Name)]; ok {
				for _, column := range columns {
					candinate := CompletionItem{
						Label:      column.Name,
						InsertText: column.Name,
						// Kind:       FieldCompletion,
					}
					columnCandinates = append(columnCandinates, candinate)
				}
			}
		}
	}

	// create completion items
	completionItems := []CompletionItem{}
	pos := token.Pos{Line: params.Position.Line + 1, Col: params.Position.Character}
	cTypes := getCompletionTypes(parsed, pos)
	log.Printf("completion types, %s", cTypes)
	switch {
	case completionTypeIs(cTypes, CompletionTypeKeyword):
		for _, k := range keywords {
			item := CompletionItem{
				Label:      k,
				InsertText: k,
				// Kind:       KeywordCompletion,
			}
			completionItems = append(completionItems, item)
		}
	case completionTypeIs(cTypes, CompletionTypeColumn):
		completionItems = append(completionItems, columnCandinates...)
	}
	return completionItems, nil
}

func getCompletionTypes(root ast.TokenList, pos token.Pos) []CompletionType {
	var res []CompletionType
	log.Printf("getCompletionTypes pos %+v", pos)

	nodeWalker := parser.NewNodeWalker(root, pos)
	log.Printf("cur node %s", nodeWalker.CurPath.CurNode)

	switch {
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"SET", "ORDER BY", "DISTINCT"})):
	// 	res = []CompletionType{
	// 		CompletionTypeColumn,
	// 		CompletionTypeTable,
	// 	}
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"AS"})):
	// 	res = []CompletionType{}
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"TO"})):
	// 	res = []CompletionType{
	// 		CompletionTypeChange,
	// 	}
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"USER", "FOR"})):
	// 	res = []CompletionType{
	// 		CompletionTypeUser,
	// 	}
	case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"SELECT", "WHERE", "HAVING"})):
		res = []CompletionType{
			CompletionTypeColumn,
			CompletionTypeTable,
			CompletionTypeView,
			CompletionTypeFunction,
		}
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"JOIN", "COPY", "FROM", "UPDATE", "INTO", "DESCRIBE", "TRUNCATE", "DESC", "EXPLAIN"})):
	// 	res = []CompletionType{
	// 		CompletionTypeColumn,
	// 		CompletionTypeTable,
	// 		CompletionTypeView,
	// 		CompletionTypeFunction,
	// 	}
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"ON"})):
	// 	res = []CompletionType{
	// 		CompletionTypeColumn,
	// 		CompletionTypeTable,
	// 		CompletionTypeView,
	// 		CompletionTypeFunction,
	// 	}
	// case nodeWalker.PrevNodesIs(true, genKeywordMatcher([]string{"USE", "DATABASE", "TEMPLATE", "CONNECT"})):
	// 	res = []CompletionType{
	// 		CompletionTypeDatabase,
	// 	}
	default:
		res = []CompletionType{
			CompletionTypeKeyword,
		}
	}
	return res
}

func genKeywordMatcher(keywords []string) astutil.NodeMatcher {
	return astutil.NodeMatcher{
		ExpectKeyword: keywords,
	}
}

// func getLastToken(tokens []*sqltoken.Token, line, char int) (int, *sqltoken.Token) {
// 	pos := sqltoken.Pos{
// 		Line: line,
// 		Col:  char,
// 	}
// 	var curIndex int
// 	var curToken *sqltoken.Token
// 	for i, token := range tokens {
// 		if 0 <= sqltoken.ComparePos(pos, token.From) {
// 			curToken = token
// 			curIndex = i
// 			if 0 >= sqltoken.ComparePos(pos, token.To) {
// 				return curIndex, curToken
// 			}
// 		}
// 	}
// 	return curIndex, curToken
// }

func getLine(text string, line int) string {
	scanner := bufio.NewScanner(strings.NewReader(text))
	i := 1
	for scanner.Scan() {
		if i == line {
			return scanner.Text()
		}
		i++
	}
	return ""
}

func getLastWord(text string, line, char int) string {
	t := getBeforeCursorText(text, line, char)
	s := getLine(t, line)

	reg := regexp.MustCompile(`\w+`)
	ss := reg.FindAllString(s, -1)
	if len(ss) == 0 {
		return ""
	}
	return ss[len(ss)-1]
}

func getBeforeCursorText(text string, line, char int) string {
	writer := bytes.NewBufferString("")
	scanner := bufio.NewScanner(strings.NewReader(text))

	i := 1
	for scanner.Scan() {
		if i == line {
			t := scanner.Text()
			writer.Write([]byte(t[:char]))
			break
		}
		writer.Write([]byte(fmt.Sprintln(scanner.Text())))
		i++
	}
	return writer.String()
}
