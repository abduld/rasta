package main

import (
	"github.com/k0kubun/pp"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	//_ "llvm.org/llvm/bindings/go/llvm"
	"fmt"
	"strings"
)

type MExpr interface {
	Head() MExpr
	Length() int
	String() string
}

type MExprBase struct {
	Position token.Pos
}

type MExprComment struct {
	MExprBase
	Value string
}

type MExprString struct {
	MExprBase
	Value string
}

type MExprInteger struct {
	MExprBase
	Value int
}

type MExprSymbol struct {
	MExprBase
	Context string
	Name    string
}

type MExprNormal struct {
	MExprBase
	Hd        MExpr
	Arguments []MExpr
}
type Generator struct {
	Program chan MExpr
}

func (this *MExprNormal) Head() MExpr {
	return this.Hd
}
func (this *MExprNormal) Length() int {
	return len(this.Arguments)
}
func (this *MExprNormal) String() string {
	args := make([]string, len(this.Arguments))
	for ii, elem := range this.Arguments {
		args[ii] = elem.String()
	}
	return this.Hd.String() + "[" + strings.Join(args, ",") + "]"
}

func (*MExprSymbol) Head() MExpr {
	return &MExprSymbol{
		Context: "System",
		Name:    "Symbol",
	}
}
func (*MExprSymbol) Length() int {
	return 0
}
func (this *MExprSymbol) String() string {
	return this.Context + "`" + this.Name
}
func (*MExprString) Head() MExpr {
	return &MExprSymbol{
		Context: "System",
		Name:    "String",
	}
}
func (*MExprString) Length() int {
	return 0
}
func (this *MExprString) String() string {
	return "\"" + this.Value + "\""
}

func walkIdentList(v ast.Visitor, list []*ast.Ident) {
	for _, x := range list {
		ast.Walk(v, x)
	}
}

func walkExprList(v ast.Visitor, list []ast.Expr) {
	for _, x := range list {
		ast.Walk(v, x)
	}
}

func walkStmtList(v ast.Visitor, list []ast.Stmt) {
	for _, x := range list {
		ast.Walk(v, x)
	}
}

func walkDeclList(v ast.Visitor, list []ast.Decl) {
	for _, x := range list {
		ast.Walk(v, x)
	}
}

func (this *Generator) Visit(anode ast.Node) (w ast.Visitor) {
	switch node := anode.(type) {
	//case *ast.Comment:
	//	pp.Println(node.Text)
	//case *ast.FuncDecl:
	//	fmt.Println(node.Name.Name)
	//case *ast.CommentGroup:
	//	pp.Println(node.Text())
	//case *ast.Package:
	//	pp.Println(node)
	case *ast.SelectorExpr:
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)

		ast.Walk(gen, node.X)
		ast.Walk(gen, node.Sel)
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "GetField",
			},
			Arguments: []MExpr{
				<-gen.Program,
				<-gen.Program,
			},
		}
	case *ast.Ident:
		this.Program <- &MExprSymbol{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Context: "System",
			Name:    node.Name,
		}
	case *ast.TypeSpec:
		gen := &Generator{
			Program: make(chan MExpr),
		}
		defer close(gen.Program)
		ast.Walk(gen, node.Name)
		ast.Walk(gen, node.Type)

		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "DeclareType",
			},
			Arguments: []MExpr{
				<-gen.Program,
				<-gen.Program,
			},
		}
	case *ast.ImportSpec:
		var nm MExpr
		if node.Name == nil && node.Path == nil {
			nm = &MExprString{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Value: "Empty",
			}
		} else if node.Path != nil {
			nm = &MExprString{
				MExprBase: MExprBase{
					Position: node.Path.ValuePos,
				},
				Value: strings.Trim(node.Path.Value, "\""),
			}
		} else {
			nm = &MExprString{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Value: node.Name.Name,
			}
		}
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "Rasta",
				Name:    "Import",
			},
			Arguments: []MExpr{
				nm,
			},
		}
	case *ast.File:
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "System",
				Name:    "BeginPackage",
			},
			Arguments: []MExpr{
				&MExprString{
					MExprBase: MExprBase{
						Position: node.Pos(),
					},
					Value: node.Name.Name,
				},
			},
		}
		walkDeclList(
			&Generator{
				Program: this.Program,
			},
			node.Decls,
		)
		this.Program <- &MExprNormal{
			MExprBase: MExprBase{
				Position: node.Pos(),
			},
			Hd: &MExprSymbol{
				MExprBase: MExprBase{
					Position: node.Pos(),
				},
				Context: "System",
				Name:    "EndPackage",
			},
			Arguments: []MExpr{},
		}
		return nil

	}
	//pp.Println(node)
	return this
}

const code = `
package main


type VerifierFailureAction C.LLVMVerifierFailureAction

const (
	// verifier will print to stderr and abort()
	AbortProcessAction VerifierFailureAction = C.LLVMAbortProcessAction
	// verifier will print to stderr and return 1
	PrintMessageAction VerifierFailureAction = C.LLVMPrintMessageAction
	// verifier will just return 1
	ReturnStatusAction VerifierFailureAction = C.LLVMReturnStatusAction
)`

func main() {

	const pth = `/Users/abduld/Code/go/src/llvm.org/llvm/bindings/go/llvm/analysis.go`
	_, err := ioutil.ReadFile(pth)
	if err != nil {
		panic(err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, pth, string(code), 0)
	if err != nil {
		panic(err)
	}
	done := make(chan bool)
	gen := &Generator{
		Program: make(chan MExpr),
	}
	go func() {
		ast.Walk(gen, f)
		done <- true
	}()

	go func() {
		for {
			select {
			case mexpr := <-gen.Program:
				if mexpr != nil {
					fmt.Println("E   ", mexpr)
				}
			}
		}
	}()
	<-done
	pp.Println("Exit")
}
