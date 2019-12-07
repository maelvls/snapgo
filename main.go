package main

import (
	"bytes"
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
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func InlineSnapshotUpdate(snapshotFilename string, snapshotLine int, replacement string) error {
	logrus.Debugf("updating snapshot at %s:%d\n", snapshotFilename, snapshotLine)

	f := token.NewFileSet()

	node, err := parser.ParseFile(f, snapshotFilename, nil, parser.AllErrors)
	if err != nil {
		logrus.Errorf("%v\n", err)
		os.Exit(1)
	}

	newExpr, err := parser.ParseExprFrom(f, snapshotFilename, replacement, parser.AllErrors)
	if err != nil {
		logrus.Errorf("%v\n", err)
		os.Exit(1)
	}

	if logrus.GetLevel() == logrus.TraceLevel {
		err = format.Node(os.Stdout, f, newExpr)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	}

	found := false
	visit := func(node ast.Node) bool {
		if node == nil {
			return true
		}
		switch node := node.(type) {
		case *ast.CallExpr:
			call := node
			fun := call.Fun
			switch node := fun.(type) {
			case *ast.SelectorExpr: // foo.ReadFile
				if f.Position(node.Pos()).Line == snapshotLine && node.Sel.Name == "InlineSnapshot" {
					logrus.Tracef("found InlineSnapshot at %s:\n  from %s\n to   %s\n",
						f.Position(node.Pos()).String(),
						debugFmtToStr(f, call.Args[0]),
						debugFmtToStr(f, newExpr),
					)
					call.Args[0] = newExpr
					found = true
					return false
				}
			case *ast.Ident: // ReadFile
				if f.Position(node.Pos()).Line == snapshotLine && node.Name == "InlineSnapshot" {
					logrus.Tracef("found InlineSnapshot at %s:\n  from %s\n to   %s\n",
						f.Position(node.Pos()).String(),
						debugFmtToStr(f, call.Args[0]),
						debugFmtToStr(f, newExpr),
					)
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
		return fmt.Errorf("could not find")
	}

	logrus.Tracef("writing to %s: %v", snapshotFilename, debugFmtToStr(f, node))
	file, err := os.Create(snapshotFilename)
	if err != nil {
		return errors.Wrap(err, "when opening file for updating snapshot")
	}

	err = format.Node(file, f, node)
	if err != nil {
		return errors.Wrap(err, "when writing to file for updating snapshot")
	}

	return nil
}

func debugFmtToStr(f *token.FileSet, n ast.Node) string {
	b := &bytes.Buffer{}
	err := format.Node(b, f, n)
	if err != nil {
		panic(err)
	}
	return b.String()
}

var (
	updateSnapshots = flag.Bool("u", false, "update inline snapshots")
	verbose         = flag.Bool("v", false, "verbose mode")
)

func main() {
	flag.Parse()
	if *verbose {
		logrus.SetLevel(logrus.TraceLevel)
	}

}

type M struct {
	Want             interface{}
	snapshotFilename string
	snapshotLine     int
}

func (m M) Matches(got interface{}) bool {
	if *updateSnapshots {
		str := render.AsCode(got)
		err := InlineSnapshotUpdate(m.snapshotFilename, m.snapshotLine, str)
		if err != nil {
			panic(err)
		}
		return true
	}
	return reflect.DeepEqual(m.Want, got)
}

func (m M) String() string {
	return fmt.Sprintf("%v", m.Want)
}

func InlineSnapshot(want interface{}) gomock.Matcher {
	m := M{Want: want}
	if *updateSnapshots {
		_, fileName, fileLine, ok := runtime.Caller(1)
		if !ok {
			panic(fmt.Errorf("runtime.Caller: could not find which filename:line"))
		}
		logrus.Debugf("InlineSnapshot at %s:%d\n", fileName, fileLine)
		m.snapshotFilename = fileName
		m.snapshotLine = fileLine
	}
	return m
}
