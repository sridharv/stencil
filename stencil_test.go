package stencil

import (
	"os"
	"path/filepath"
	"testing"

	"io/ioutil"

	"flag"

	"bytes"

	"github.com/sridharv/fakegopath"
)

var updateGoldens = flag.Bool("update-goldens", false, "If true, goldens are updated")

type testCase struct {
	name     string
	files    []fakegopath.SourceFile
	srcs     []string
	outfile  string
	expected string
}

func (c testCase) run(t *testing.T) {
	t.Run(c.name, func(t *testing.T) {
		tmp, err := fakegopath.NewTemporaryWithFiles("stencil_"+c.name, c.files)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		defer tmp.Reset()

		var dir string
		var f file
		mkdir := func(d string, p os.FileMode) error {
			if p != 0755 {
				t.Errorf("invalid filemode for dir: %v", p)
			}
			dir = d
			return nil
		}
		write := func(path string, data []byte, p os.FileMode) error {
			if p != 0644 {
				t.Errorf("invalid filemode for file: %v", p)
			}
			f.path, f.data = path, data
			return nil
		}
		srcs := make([]string, len(c.srcs))
		for i, s := range c.srcs {
			srcs[i] = filepath.Join(tmp.Src, s)
		}
		if err := process(srcs, mkdir, write); err != nil {
			t.Fatalf("%+v", err)
		}
		out := filepath.Join(tmp.Src, c.outfile)
		if dir != filepath.Dir(out) {
			t.Errorf("expected dir %s, got %s", filepath.Dir(out), dir)
		}
		if f.path != out {
			t.Errorf("expected file %s, got %s", out, f.path)
		}
		if *updateGoldens {
			if err := ioutil.WriteFile(c.expected, f.data, 0644); err != nil {
				t.Fatal(c.expected, ": failed to update golden", err)
			}
			return
		}
		golden, err := ioutil.ReadFile(c.expected)
		if err != nil {
			t.Fatal(c.expected, ": could not read golden", err)
		}
		if !bytes.Equal(golden, f.data) {
			t.Errorf("expected output:\n%s\ngot:\n%s", string(golden), string(f.data))
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
		srcs:     []string{"examples/setexamples/intersect.go"},
		outfile:  "examples/setexamples/vendor/collections/set/Element/string/set.go",
		expected: "testdata/set.string.golden",
	},
}

func TestStencil(t *testing.T) {
	for _, c := range cases {
		c.run(t)
	}
}
