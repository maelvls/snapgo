package snap

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"runtime"

	"github.com/gdexlab/go-render/render"
	"github.com/pkg/errors"
)

var (
	updateSnapshots = flag.Bool("snap.update", false, "update inline snapshots")
)

type snapMatcher struct {
	want         interface{}
	snapFilename string
	snapLine     int
}

func (m snapMatcher) Matches(got interface{}) bool {
	if *updateSnapshots {
		str := render.AsCode(got)
		err := InlineSnapshotUpdate(m.snapFilename, m.snapLine, str)
		if err != nil {
			panic(err)
		}
		return true
	}
	return reflect.DeepEqual(m.want, got)
}

func (m snapMatcher) String() string {
	return fmt.Sprintf("%v", m.want)
}

func InlineSnapshot(want interface{}) snapMatcher {
	m := snapMatcher{want: want}
	if *updateSnapshots {
		_, fileName, fileLine, ok := runtime.Caller(1)
		if !ok {
			panic(fmt.Errorf("runtime.Caller: could not find which filename:line"))
		}
		m.snapFilename = fileName
		m.snapLine = fileLine
	}
	return m
}

func InlineSnapshotUpdate(snapFilename string, snapLine int, replacement string) error {
	f := token.NewFileSet()

	node, err := parser.ParseFile(f, snapFilename, nil, parser.AllErrors)
	if err != nil {
		return errors.Wrap(err, "while updating snapshot")
	}

	newExpr, err := parser.ParseExprFrom(f, snapFilename, replacement, parser.AllErrors)
	if err != nil {
		return errors.Wrap(err, "shouldn't happen: err while parsing snapshot update")
	}

	found := false
	visit := func(node ast.Node) bool {
		if node == nil || found {
			return false
		}
		switch node := node.(type) {
		case *ast.CallExpr:
			call := node
			fun := call.Fun
			switch node := fun.(type) {
			case *ast.SelectorExpr: // foo.ReadFile
				if f.Position(node.Pos()).Line == snapLine && node.Sel.Name == "InlineSnapshot" {
					call.Args[0] = newExpr
					found = true
					return false
				}
			case *ast.Ident: // ReadFile
				if f.Position(node.Pos()).Line == snapLine && node.Name == "InlineSnapshot" {
					call.Args[0] = newExpr
					found = true
					return false
				}
			}
		}

		return true
	}
	ast.Inspect(node, visit)
	if !found {
		return fmt.Errorf("could not find InlineSnapshot at %s:%d", snapFilename, snapLine)
	}

	file, err := os.Create(snapFilename)
	if err != nil {
		return errors.Wrap(err, "when opening file for updating snapshot")
	}

	err = format.Node(file, f, node)
	if err != nil {
		return errors.Wrap(err, "when writing to file for updating snapshot")
	}

	return nil
}
