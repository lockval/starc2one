package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"go.starlark.net/lib/json"
	"go.starlark.net/lib/math"
	"go.starlark.net/lib/time"
	"go.starlark.net/repl"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type entry struct {
	globals starlark.StringDict
	err     error
}

var (
	cache = make(map[string]*entry)

	moduleCount = 0

	module2exportsID  = &syntax.Ident{Name: "module2exports"}
	module2functionID = &syntax.Ident{Name: "module2function"}
	module2function   = &syntax.DefStmt{Name: module2functionID}

	argFile   = flag.String("file", "", "execute compile file in repl OR execute source file")
	argOutput = flag.String("output", "", "compile file to output")
	argSuffix = flag.String("suffix", "", "eg:\".star\". add suffix,will make more like module name,like this:\"path/module1\"")
)

func init() {
	// non-standard dialect flags
	flag.BoolVar(&resolve.AllowSet, "set", resolve.AllowSet, "allow set data type")
	flag.BoolVar(&resolve.AllowRecursion, "recursion", resolve.AllowRecursion, "allow while statements and recursive functions")
	flag.BoolVar(&resolve.AllowGlobalReassign, "globalreassign", resolve.AllowGlobalReassign, "allow reassignment of globals, and if/for/while statements at top level")
}

func addstmts(module string, sf *syntax.File) {

	moduleCount++

	defID := &syntax.Ident{Name: "f" + strconv.Itoa(moduleCount)}
	def := &syntax.DefStmt{
		Name: defID,
		// Params: params,
		// Body:   body,
	}

	myExportID := &syntax.Ident{Name: "myExport"}
	myExportIsDict := &syntax.AssignStmt{
		Op:  syntax.EQ,
		LHS: myExportID,
		RHS: &syntax.DictExpr{},
	}

	def.Body = append(def.Body, myExportIsDict)

	regMod := &syntax.AssignStmt{
		Op:  syntax.EQ,
		LHS: &syntax.IndexExpr{X: module2exportsID, Y: &syntax.Literal{Token: syntax.STRING, Value: module}},
		RHS: myExportID,
	}
	def.Body = append(def.Body, regMod)

	for _, v := range sf.Stmts {

		loadstmt, ok := v.(*syntax.LoadStmt)
		if ok {
			for idi := range loadstmt.From {

				// def.Params = append(def.Params, &syntax.Ident{Name: loadstmt.To[idi].Name})

				getTo := &syntax.AssignStmt{
					Op:  syntax.EQ,
					LHS: &syntax.Ident{Name: loadstmt.To[idi].Name},
					RHS: &syntax.IndexExpr{X: &syntax.IndexExpr{X: module2exportsID, Y: loadstmt.Module}, Y: &syntax.Literal{Token: syntax.STRING, Value: loadstmt.From[idi].Name}},
				}
				def.Body = append(def.Body, getTo)

			}

		} else {

			def.Body = append(def.Body, v)

			switch xxstmt := v.(type) {
			case *syntax.DefStmt:
				regv := &syntax.AssignStmt{
					Op:  syntax.EQ,
					LHS: &syntax.IndexExpr{X: myExportID, Y: &syntax.Literal{Token: syntax.STRING, Value: xxstmt.Name.Name}},
					RHS: xxstmt.Name,
				}
				def.Body = append(def.Body, regv)
			case *syntax.AssignStmt:
				xxID, ok := xxstmt.LHS.(*syntax.Ident)
				if ok {
					regv := &syntax.AssignStmt{
						Op:  syntax.EQ,
						LHS: &syntax.IndexExpr{X: myExportID, Y: &syntax.Literal{Token: syntax.STRING, Value: xxID.Name}},
						RHS: xxID,
					}
					def.Body = append(def.Body, regv)
				}
			}

		}

	}

	module2function.Body = append(module2function.Body, def)

	es := &syntax.ExprStmt{
		X: &syntax.CallExpr{Fn: defID},
	}

	module2function.Body = append(module2function.Body, es)
}

func Load(_ *starlark.Thread, module string) (starlark.StringDict, error) {

	e, ok := cache[module]
	if e == nil {
		if ok {
			// request for package whose loading is in progress
			return nil, fmt.Errorf("cycle in load graph")
		}

		// Add a placeholder to indicate "load in progress".
		cache[module] = nil

		// Load it.
		thread := &starlark.Thread{Name: "exec " + module, Load: Load}

		var predeclared starlark.StringDict = nil
		sf, program, err := starlark.SourceProgram(module+*argSuffix, nil, predeclared.Has)
		if err != nil {
			return nil, err
		}

		globals, err := program.Init(thread, predeclared)
		globals.Freeze()
		// for k := range globals {
		// 	println("global:", k)
		// }

		addstmts(module, sf)

		e = &entry{globals, err}

		// Update the cache.
		cache[module] = e
	}
	return e.globals, e.err

}

func main() {

	flag.Parse()

	if *argFile == "" {
		log.Fatal("missing parameter -f")
	}

	// Ideally this statement would update the predeclared environment.
	// TODO(adonovan): plumb predeclared env through to the REPL.
	starlark.Universe["json"] = json.Module
	starlark.Universe["time"] = time.Module
	starlark.Universe["math"] = math.Module

	if *argOutput != "" {

		if *argOutput == *argFile {
			log.Fatal("input and output is same")
		}

		println("execute", *argFile, "and compile to", *argOutput)

		var stmts []syntax.Stmt

		module2exports := &syntax.AssignStmt{
			Op:  syntax.EQ,
			LHS: module2exportsID,
			RHS: &syntax.DictExpr{},
		}
		stmts = append(stmts, module2exports)

		stmts = append(stmts, module2function)

		es := &syntax.ExprStmt{
			X: &syntax.CallExpr{Fn: module2functionID},
		}
		stmts = append(stmts, es)

		_, err := Load(nil, *argFile)
		check(err)

		var predeclared starlark.StringDict = nil
		f := &syntax.File{Stmts: stmts}
		program, err := starlark.FileProgram(f, predeclared.Has)
		check(err)

		file, err := os.OpenFile(*argOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		check(err)
		defer file.Close()

		program.Write(file)

	} else {
		println("execute", *argFile, "in REPL")

		file, err := os.OpenFile(*argFile, os.O_RDONLY, 0600)
		check(err)
		defer file.Close()
		program, err := starlark.CompiledProgram(file)
		check(err)

		thread := &starlark.Thread{Name: "exec " + *argFile}

		globals, err := program.Init(thread, nil)
		check(err)
		globals.Freeze()
		for k := range globals {
			println("global:", k)
		}

		repl.REPL(thread, globals)

	}

}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
