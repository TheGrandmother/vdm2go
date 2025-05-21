package main

import "fmt"
import "os"
import "github.com/alecthomas/participle/v2"
import "github.com/alecthomas/kong"
import "github.com/alecthomas/repr"

type SExpr struct {
	Terminal string   `parser:"@Ident | @Char | @':'"`
	Exprs    []*SExpr `parser:"| '(' @@* ')'"`
}

var parser = participle.MustBuild[SExpr](
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
		repr.Println(ast)
		ctx.FatalIfErrorf(err)
	}
}
