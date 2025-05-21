package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	// "github.com/alecthomas/repr"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
)

type Generator interface {
	toNode() (error, *ast.Node)
}

var NotImplemented = errors.New("Not implemented error")

type SExpr struct {
	Terminal string   `parser:"@Terminal"`
	Ident    string   `parser:"| @Ident"`
	Exprs    []*SExpr `parser:"| LP @@* RP"`
}

type Document struct {
	Definitions Definitions `parser:"LP 'sl_document' LP 'sl_definition_block' @@ RP RP"`
}

type Definitions struct {
	FunctionDefintions []*FunctionDefintion `parser:"LP 'sl_function_definitions' 'functions' @@ (';' @@)* RP"`
}

type FunctionDefintion struct {
	ImplicitDefinition ImplicitDefintion `parser:"LP 'function_definition' @@ RP"`
}

type ImplicitDefintion struct {
	Name        string             `parser:"LP 'implicit_function_definition' @Ident"`
	Arguments   []*PatternTypePair `parser:"LP 'parameter_types' LP @@* RP RP"`
	RetrunTypes []*IdentTypePair   `parser:"LP 'identifier_type_pair_list' @@* RP"`
	PostExpr    PostExpr           `parser:"LP 'post_expression' 'post' @@* RP RP"`
}

type Type struct {
	BasicType string `parser:"LP 'type' LP 'basic_type' @Ident RP RP"`
}

type PatternTypePair struct {
	PatternList PatternList `parser:"LP 'pattern_type_pair_list' LP 'pattern_list' @@ RP"`
	Type        Type        `parser:"':' @@ RP"`
}

type PatternList struct {
	Patterns []string `parser:"LP 'pattern' @Ident RP (',' LP 'pattern' @Ident RP)*"`
}

type IdentTypePair struct {
	Name string `parser:"LP 'identifier_type_pair' @Ident"`
	Type Type   `parser:"':' @@ RP"`
}

type PostExpr struct {
	Exp Expression `parser:"LP @@ RP"`
}

type Expression struct {
	Variable string      `parser:"'expression' LP 'variable' LP 'name' @Ident RP RP"`
	Negation *Expression `parser:"|'expression' 'not' LP @@ RP"`
	Lhs      *Expression `parser:"|'expression' LP @@ RP"`
	Op       string      `parser:"@('=' | 'and' | 'or')"`
	Rhs      *Expression `parser:"LP @@ RP"`
}

type Equality struct {
	Lhs Expression `parser:"LP @@ RP"`
	Rhs Expression `parser:"'=' LP @@ RP"`
}

type And struct {
	Lhs Expression `parser:"LP @@ RP"`
	Rhs Expression `parser:"'and' LP @@ RP"`
}

type Mjau struct {
	Mjau string `parser:"Terminal"`
}

func (s *Document) toNode() (error, *ast.File) {
	var func_decls []ast.Decl
	if len(s.Definitions.FunctionDefintions) > 0 {
		func_decls = make([]ast.Decl, len(s.Definitions.FunctionDefintions)*2)
		for i := 0; i < len(func_decls); i += 2 {
			err, main_decl, post_decl := s.Definitions.FunctionDefintions[i/2].ImplicitDefinition.toNode()
			if err != nil {
				return fmt.Errorf("%v; failed to decl", err), nil
			}
			func_decls[i] = main_decl
			func_decls[i+1] = post_decl
		}

	}
	return nil, &ast.File{
		Name:      ast.NewIdent("main"),
		Decls:     func_decls,
		GoVersion: "1.24,3",
	}
}

func (s *Expression) toNode() (error, ast.Expr) {
	if s.Variable != "" {
		return nil, ast.NewIdent(s.Variable)
	}
	if s.Negation != nil {
		err, rhs := s.Negation.toNode()
		if err != nil {
			return fmt.Errorf("%v; failed to get rhs", err), nil
		}
		return nil, &ast.UnaryExpr{
			Op: token.NOT,
			X:  rhs,
		}
	}
	if s.Op != "" {
		err, rhs := s.Rhs.toNode()
		if err != nil {
			return fmt.Errorf("%v; failed to get rhs", err), nil
		}
		err, lhs := s.Lhs.toNode()
		if err != nil {
			return fmt.Errorf("%v; failed to get lsh", err), nil
		}
		var op token.Token
		switch s.Op {
		case "and":
			op = token.LAND
			break
		case "or":
			op = token.LOR
			break
		case "=":
			op = token.EQL
			break
		default:
			return fmt.Errorf("%s is a silly operator", s.Op), nil
		}
		return nil, &ast.BinaryExpr{
			Op: op,
			X:  lhs,
			Y:  rhs,
		}
	}
	return fmt.Errorf("%v: silly expression", NotImplemented), nil
}

func (s *IdentTypePair) toNode() (error, *ast.Field) {
	err, type_node := s.Type.toNode()
	if err != nil {
		return fmt.Errorf("%v: Failed to get type", err), nil
	}

	return nil, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(s.Name)},
		Type:  type_node,
	}
}

func (s *ImplicitDefintion) toNode() (error, ast.Decl, ast.Decl) {
	args := make([]*ast.Field, len(s.Arguments))
	for i := range args {
		err, field := s.Arguments[i].toNode()
		if err != nil {
			return fmt.Errorf("%v: Failed to get field from node", err), nil, nil
		}
		args[i] = field
	}
	returns := make([]*ast.Field, len(s.RetrunTypes))
	for i := range returns {
		err, field := s.RetrunTypes[i].toNode()
		if err != nil {
			return fmt.Errorf("%v: Failed to get field from node", err), nil, nil
		}
		returns[i] = field
	}

	params := &ast.FieldList{
		List: args,
	}
	return_types := &ast.FieldList{
		List: returns,
	}

	err, post_decl := makePostCheck(&s.PostExpr, s.Name, params, return_types)
	if err != nil {
		return fmt.Errorf("%v: Failed at making post", err), nil, nil
	}

	main_decl := &ast.FuncDecl{
		Name: ast.NewIdent(s.Name),
		Type: &ast.FuncType{
			Params:  params,
			Results: return_types,
		},
	}

	return nil, main_decl, post_decl
}

func makePostCheck(s *PostExpr, name string, params *ast.FieldList, return_types *ast.FieldList) (error, *ast.FuncDecl) {
	err, return_exp := s.Exp.toNode()
	if err != nil {
		return fmt.Errorf("%v: Could not fuck around with thing", err), nil
	}
	return nil, &ast.FuncDecl{
		Name: ast.NewIdent(fmt.Sprintf("POST_%s", name)),
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: append(params.List, return_types.List...),
			},
			Results: &ast.FieldList{
				List: []*ast.Field{{
					Names:   []*ast.Ident{},
					Type:    ast.NewIdent("bool"),
					Tag:     &ast.BasicLit{},
					Comment: &ast.CommentGroup{},
				}},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{&ast.ReturnStmt{
				Results: []ast.Expr{return_exp},
			}},
		},
	}

}

func (s *Type) toNode() (error, ast.Expr) {
	if s.BasicType == "" {
		return fmt.Errorf("%v: Fancy types are not implemented", NotImplemented), nil
	}
	return nil, ast.NewIdent(s.BasicType)
}

func (s *PatternTypePair) toNode() (error, *ast.Field) {
	idents := make([]*ast.Ident, len(s.PatternList.Patterns))
	for i := range idents {
		idents[i] = ast.NewIdent(s.PatternList.Patterns[i])
	}
	err, type_node := s.Type.toNode()
	if err != nil {
		return fmt.Errorf("%v: Failed to get type", err), nil
	}

	return nil, &ast.Field{
		Names: idents,
		Type:  type_node,
	}
}

var basicLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Whitespace", Pattern: `[ \r\t\n]+`},
	{Name: "LP", Pattern: `[(]`},
	{Name: "RP", Pattern: `[)]`},
	{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
	{Name: "Terminal", Pattern: `[^\s()]+`},
	{Name: "Any", Pattern: `.+`},
})

var parser = participle.MustBuild[Document](
	participle.Lexer(basicLexer),
	participle.Elide("Whitespace"),
	participle.UseLookahead(2),
)

var cli struct {
	EBNF  bool     `help:"Dump EBNF."`
	Files []string `arg:"" optional:"" type:"existingfile" help:"GraphQL schema files to parse."`
}

func main() {
	ctx := kong.Parse(&cli)
	if cli.EBNF {
		fmt.Println(parser.String())
		ctx.Exit(0)
	}
	for _, file := range cli.Files {
		r, err := os.Open(file)
		ctx.FatalIfErrorf(err)
		ast, err := parser.Parse(file, r)
		r.Close()
		//repr.Println(ast)
		var buf bytes.Buffer
		err, thing := ast.toNode()
		ctx.FatalIfErrorf(err)
		printer.Fprint(&buf, token.NewFileSet(), thing)
		fmt.Println(buf.String())
		ctx.FatalIfErrorf(err)
	}
}
