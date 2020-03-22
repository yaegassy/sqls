package handler

import (
	"context"
	"log"
	"net"
	"reflect"
	"testing"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/lighttiger2505/sqls/internal/lsp"
)

type TestContext struct {
	h          jsonrpc2.Handler
	conn       *jsonrpc2.Conn
	connServer *jsonrpc2.Conn
	server     *Server
	ctx        context.Context
}

func newTestContext() *TestContext {
	server := NewServer()
	handler := jsonrpc2.HandlerWithError(server.Handle)
	ctx := context.Background()
	return &TestContext{
		h:      handler,
		ctx:    ctx,
		server: server,
	}
}

func (tx *TestContext) setup(t *testing.T) {
	t.Helper()
	tx.initServer(t)
}

func (tx *TestContext) tearDown() {
	if tx.conn != nil {
		if err := tx.conn.Close(); err != nil {
			log.Fatal("conn.Close:", err)
		}
	}

	if tx.connServer != nil {
		if err := tx.connServer.Close(); err != nil {
			log.Fatal("connServer.Close:", err)
		}
	}
}

func (tx *TestContext) initServer(t *testing.T) {
	t.Helper()

	// Prepare the server and client connection.
	client, server := net.Pipe()
	tx.connServer = jsonrpc2.NewConn(tx.ctx, jsonrpc2.NewBufferedStream(server, jsonrpc2.VSCodeObjectCodec{}), tx.h)
	tx.conn = jsonrpc2.NewConn(tx.ctx, jsonrpc2.NewBufferedStream(client, jsonrpc2.VSCodeObjectCodec{}), tx.h)

	// Initialize Langage Server
	params := lsp.InitializeParams{
		InitializationOptions: lsp.InitializeOptions{},
	}
	if err := tx.conn.Call(tx.ctx, "initialize", params, nil); err != nil {
		t.Fatal("conn.Call initialize:", err)
	}
}

func TestInitialized(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	want := lsp.InitializeResult{
		lsp.ServerCapabilities{
			TextDocumentSync: lsp.TDSKFull,
			HoverProvider:    false,
			CompletionProvider: &lsp.CompletionOptions{
				TriggerCharacters: []string{"."},
			},
			CodeActionProvider:              true,
			DefinitionProvider:              false,
			DocumentFormattingProvider:      false,
			DocumentRangeFormattingProvider: false,
		},
	}
	var got lsp.InitializeResult
	params := lsp.InitializeParams{
		InitializationOptions: lsp.InitializeOptions{},
	}
	if err := tx.conn.Call(tx.ctx, "initialize", params, &got); err != nil {
		t.Fatal("conn.Call initialize:", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("not match \n%+v\n%+v", want, got)
	}
}

func TestFileWatch(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	uri := "file:///Users/octref/Code/css-test/test.sql"
	openText := "SELECT * FROM todo ORDER BY id ASC"
	changeText := "SELECT * FROM todo ORDER BY name ASC"

	didOpenParams := lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:        uri,
			LanguageID: "sql",
			Version:    0,
			Text:       openText,
		},
	}
	if err := tx.conn.Call(tx.ctx, "textDocument/didOpen", didOpenParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didOpen:", err)
	}
	tx.testFile(t, didOpenParams.TextDocument.URI, didOpenParams.TextDocument.Text)

	didChangeParams := lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			URI:     uri,
			Version: 1,
		},
		ContentChanges: []lsp.TextDocumentContentChangeEvent{
			lsp.TextDocumentContentChangeEvent{
				Range: lsp.Range{
					Start: lsp.Position{
						Line:      1,
						Character: 1,
					},
					End: lsp.Position{
						Line:      1,
						Character: 1,
					},
				},
				RangeLength: 1,
				Text:        changeText,
			},
		},
	}
	if err := tx.conn.Call(tx.ctx, "textDocument/didChange", didChangeParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didChange:", err)
	}
	tx.testFile(t, didChangeParams.TextDocument.URI, didChangeParams.ContentChanges[0].Text)

	didSaveParams := lsp.DidSaveTextDocumentParams{
		Text:         openText,
		TextDocument: lsp.TextDocumentIdentifier{uri},
	}
	if err := tx.conn.Call(tx.ctx, "textDocument/didSave", didSaveParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didSave:", err)
	}
	tx.testFile(t, didSaveParams.TextDocument.URI, didSaveParams.Text)

	didCloseParams := lsp.DidCloseTextDocumentParams{lsp.TextDocumentIdentifier{uri}}
	if err := tx.conn.Call(tx.ctx, "textDocument/didClose", didCloseParams, nil); err != nil {
		t.Fatal("conn.Call textDocument/didClose:", err)
	}
	_, ok := tx.server.files[didCloseParams.TextDocument.URI]
	if ok {
		t.Errorf("found opened file. URI:%s", didCloseParams.TextDocument.URI)
	}
}

func TestComplete(t *testing.T) {
	tx := newTestContext()
	tx.setup(t)
	defer tx.tearDown()

	didChangeConfigurationParams := lsp.DidChangeConfigurationParams{
		Settings: struct {
			SQLS struct {
				Driver         string "json:\"driver\""
				DataSourceName string "json:\"dataSourceName\""
			} "json:\"sqls\""
		}{
			SQLS: struct {
				Driver         string "json:\"driver\""
				DataSourceName string "json:\"dataSourceName\""
			}{
				Driver:         "mock",
				DataSourceName: "",
			},
		},
	}
	if err := tx.conn.Call(tx.ctx, "workspace/didChangeConfiguration", didChangeConfigurationParams, nil); err != nil {
		t.Fatal("conn.Call workspace/didChangeConfiguration:", err)
	}

	uri := "file:///Users/octref/Code/css-test/test.sql"
	testcases := []struct {
		name  string
		input string
		line  int
		col   int
		want  []string
	}{
		{
			name:  "select identifier",
			input: "select  from city",
			line:  0,
			col:   7,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select identifier with table alias",
			input: "select  from city as c",
			line:  0,
			col:   7,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"c",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select identifier with table alias without as",
			input: "select  from city c",
			line:  0,
			col:   7,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"c",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select identifier filterd",
			input: "select Cou from city",
			line:  0,
			col:   10,
			want: []string{
				"CountryCode",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select identifier list",
			input: "select id, cou from city",
			line:  0,
			col:   14,
			want: []string{
				"CountryCode",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "select has aliased table identifier",
			input: "select c. from city as c",
			line:  0,
			col:   9,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
			},
		},
		{
			name:  "select has aliased without as table identifier",
			input: "select c. from city as c",
			line:  0,
			col:   9,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
			},
		},
		{
			name:  "from identifier",
			input: "select CountryCode from ",
			line:  0,
			col:   24,
			want: []string{
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "from identifier filterd",
			input: "select CountryCode from co",
			line:  0,
			col:   26,
			want: []string{
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "join identifier",
			input: "select CountryCode from city left join ",
			line:  0,
			col:   39,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "join identifier filterd",
			input: "select CountryCode from city left join co",
			line:  0,
			col:   41,
			want: []string{
				"CountryCode",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "join on identifier filterd",
			input: "select CountryCode from city left join country on co",
			line:  0,
			col:   52,
			want: []string{
				"Code",
				"Continent",
				"Code2",
			},
		},
		{
			name:  "ORDER BY identifier",
			input: "SELECT ID, Name FROM city ORDER BY ",
			line:  0,
			col:   35,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "GROUP BY identifier",
			input: "SELECT CountryCode, COUNT(*) FROM city GROUP BY ",
			line:  0,
			col:   48,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "FROM identifiers in the sub query",
			input: "SELECT * FROM (SELECT * FROM ",
			line:  0,
			col:   29,
			want: []string{
				"city",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "filterd FROM identifiers in the sub query",
			input: "SELECT * FROM (SELECT * FROM co",
			line:  0,
			col:   29,
			want: []string{
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "filterd SELECT identifiers in the sub query",
			input: "SELECT * FROM (SELECT Cou FROM city)",
			line:  0,
			col:   25,
			want: []string{
				"CountryCode",
				"country",
				"countrylanguage",
			},
		},
		{
			name:  "SELECT identifiers by sub query",
			input: "SELECT  FROM (SELECT ID as city_id, Name as city_name FROM city) as t",
			line:  0,
			col:   7,
			want: []string{
				"city_id",
				"city_name",
			},
		},
		{
			name:  "SELECT identifiers with multiple statement forcused first",
			input: "SELECT c. FROM city as c;SELECT c. FROM country as c;",
			line:  0,
			col:   9,
			want: []string{
				"ID",
				"Name",
				"CountryCode",
				"District",
				"Population",
			},
		},
		{
			name:  "SELECT identifiers with multiple statement forcused second",
			input: "SELECT c. FROM city as c;SELECT c. FROM country as c;",
			line:  0,
			col:   34,
			want: []string{
				"Code",
				"Name",
				"CountryCode",
				"Region",
				"SurfaceArea",
				"IndepYear",
				"LifeExpectancy",
				"GNP",
				"GNPOld",
				"LocalName",
				"GovernmentForm",
				"HeadOfState",
				"Capital",
				"Code2",
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			// Open dummy file
			didOpenParams := lsp.DidOpenTextDocumentParams{
				TextDocument: lsp.TextDocumentItem{
					URI:        uri,
					LanguageID: "sql",
					Version:    0,
					Text:       tt.input,
				},
			}
			if err := tx.conn.Call(tx.ctx, "textDocument/didOpen", didOpenParams, nil); err != nil {
				t.Fatal("conn.Call textDocument/didOpen:", err)
			}
			tx.testFile(t, didOpenParams.TextDocument.URI, didOpenParams.TextDocument.Text)
			// Create completion params
			commpletionParams := lsp.CompletionParams{
				TextDocumentPositionParams: lsp.TextDocumentPositionParams{
					TextDocument: lsp.TextDocumentIdentifier{
						URI: uri,
					},
					Position: lsp.Position{
						Line:      tt.line,
						Character: tt.col,
					},
				},
				CompletionContext: lsp.CompletionContext{
					TriggerKind:      0,
					TriggerCharacter: nil,
				},
			}

			var got []lsp.CompletionItem
			if err := tx.conn.Call(tx.ctx, "textDocument/completion", commpletionParams, &got); err != nil {
				t.Fatal("conn.Call textDocument/completion:", err)
			}
			testCompletionItem(t, tt.want, got)
		})
	}
}

func testCompletionItem(t *testing.T, expectLabels []string, gotItems []lsp.CompletionItem) {
	t.Helper()

	itemMap := map[string]struct{}{}
	for _, item := range gotItems {
		itemMap[item.Label] = struct{}{}
	}

	for _, el := range expectLabels {
		_, ok := itemMap[el]
		if !ok {
			t.Errorf("not included label, expect %q", el)
		}
	}
}

func (tx *TestContext) testFile(t *testing.T, uri, text string) {
	f, ok := tx.server.files[uri]
	if !ok {
		t.Errorf("not found opened file. URI:%s", uri)
	}
	if f.Text != text {
		t.Errorf("not match %s. got: %s", text, f.Text)
	}
}