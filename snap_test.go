package snap

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type A struct {
	some string
	b    *B
}

type B struct {
	data []string
}

func Test_main(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	assert.True(t, InlineSnapshot(&A{some: "aa", b: &B{data: []string{"hhe"}}}).Matches(&A{some: "aa", b: &B{data: []string{"hhe"}}}))
}

func debugFmtToStr(f *token.FileSet, n ast.Node) string {
	b := &bytes.Buffer{}
	err := format.Node(b, f, n)
	if err != nil {
		panic(err)
	}
	return b.String()
}
