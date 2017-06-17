package stencil

import (
	"path/filepath"
	"testing"

	"io/ioutil"

	"flag"

	"bytes"

	"os"

	"strings"

	"go/ast"
	"go/parser"
	"go/token"

	"josharian/apply"

	"go/printer"

	"github.com/pkg/errors"
	"github.com/sridharv/fakegopath"
)

var updateGoldens = flag.Bool("update-goldens", false, "If true, goldens are updated")

type outFile struct {
	path   string
	golden string
}

type testCase struct {
	name    string
	files   []fakegopath.SourceFile
	srcs    []string
	outs    []outFile
	process func([]string) ([]file, error)
}

func (c testCase) run(t *testing.T) {
	t.Run(c.name, func(t *testing.T) {
		tmp, err := fakegopath.NewTemporaryWithFiles("stencil_"+c.name, c.files)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		defer tmp.Reset()

		srcs := make([]string, len(c.srcs))
		for i, s := range c.srcs {
			srcs[i] = filepath.Join(tmp.Src, s)
		}
		proc := c.process
		if proc == nil {
			proc = process
		}
		files, err := proc(srcs)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		for i, o := range c.outs {
			out := filepath.Join(tmp.Src, o.path)
			f := files[i]
			if !strings.HasSuffix(f.path, out) {
				t.Errorf("expected file %s, got %s", out, f.path)
			}
			if *updateGoldens {
				if err := ioutil.WriteFile(o.golden, f.data, 0644); err != nil {
					t.Error(o.golden, ": failed to update golden", err)
				}
				continue
			}
			golden, err := ioutil.ReadFile(o.golden)
			if err != nil {
				t.Fatal(o.golden, ": could not read golden", err)
			}
			if !bytes.Equal(golden, f.data) {
				t.Errorf("expected output:\n%s\ngot:\n%s", string(golden), string(f.data))
			}
		}
	})
}

var cases = []testCase{
	{
		name: "Set_String_SingleFile",
		files: []fakegopath.SourceFile{
			{Src: "testdata/set.go", Dest: "collections/set/set.go"},
			{Src: "testdata/set.intersect.go", Dest: "examples/setexamples/intersect.go"},
		},
		srcs: []string{"examples/setexamples/intersect.go"},
		outs: []outFile{
			{
				path:   "examples/setexamples/vendor/collections/set/Element/string/set.go",
				golden: "testdata/set.string.golden",
			},
		},
	},
	{
		name: "Basic_Float32_SingleFile",
		files: []fakegopath.SourceFile{
			{Src: "testdata/basic.go", Dest: "basic/basic.go"},
			{Src: "testdata/basic.use.go", Dest: "use/use.go"},
		},
		srcs: []string{"use/use.go"},
		outs: []outFile{
			{
				path:   "use/vendor/basic/int/float32/basic.go",
				golden: "testdata/basic.float32.golden",
			},
		},
	},
	{
		name: "Set_Interfaces_SingleFile",
		files: []fakegopath.SourceFile{
			{Src: "testdata/interfaces.go", Dest: "ifaces/interfaces.go"},
			{Src: "testdata/interfaces.use.go", Dest: "use/use.go"},
		},
		srcs: []string{"use/use.go"},
		outs: []outFile{
			{
				path:   "use/vendor/ifaces/interface/int/ifaces.go",
				golden: "testdata/interfaces.int.golden",
			},
		},
	},
	{
		name: "Set_String_Dir",
		files: []fakegopath.SourceFile{
			{Src: "testdata/set.go", Dest: "collections/set/set.go"},
			{Src: "testdata/set.intersect.go", Dest: "examples/setexamples/intersect.go"},
		},
		srcs: []string{"examples/setexamples/intersect.go"},
		outs: []outFile{
			{
				path:   "examples/setexamples/vendor/collections/set/Element/string/set.go",
				golden: "testdata/set.string.golden",
			},
		},
		process: func(p []string) ([]file, error) {
			d := filepath.Dir(p[0])
			cwd, err := os.Getwd()
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if err = os.Chdir(d); err != nil {
				return nil, errors.WithStack(err)
			}
			defer os.Chdir(cwd)
			return process([]string{})
		},
	},
}

func TestStencil(t *testing.T) {
	for _, c := range cases {
		c.run(t)
	}
}

func TestASTRewrite(t *testing.T) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "./testdata/set.go", nil, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	r := apply.Apply(f, func(c apply.ApplyCursor) bool {
		switch t := c.Node().(type) {
		case *ast.GenDecl:
			if len(t.Specs) == 0 {
				return true
			}
			spec, ok := t.Specs[0].(*ast.TypeSpec)
			if !ok {
				return true
			}

			if spec.Name.Name != "Element" {
				return true
			}
			c.Delete()
		}
		return true
	}, nil)
	c := &printer.Config{}
	c.Fprint(os.Stdout, fs, r)
}
