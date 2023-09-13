package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"golang.org/x/tools/go/packages"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	prefix    = flag.String("prefix", "", "comma-separated list of type name prefix, must be set")
	object    = flag.String("obj", "", "object type, must be set")
	output    = flag.String("output", "", "output filename; default event.go")
	buildTags = flag.String("tags", "", "comma-separated list of build tags to apply")
)

func Usage() {
	_, _ = fmt.Fprintf(os.Stderr, "Usage of trigger:\n")
	_, _ = fmt.Fprintf(os.Stderr, "\t trigger -prefix T -obj T\n")
	_, _ = fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("trigger: ")
	flag.Usage = Usage
	flag.Parse()
	if len(*prefix) == 0 || len(*object) == 0 {
		flag.Usage()
		os.Exit(2)
	}
	if len(*output) == 0 {
		ss := "event_gen.go"
		output = &ss
	}
	args := flag.Args()
	if len(args) == 0 {
		// Default: process whole package in current directory.
		args = []string{"."}
	}
	var tags []string
	if len(*buildTags) > 0 {
		tags = strings.Split(*buildTags, ",")
	}
	var dir string
	if len(args) == 1 && isDirectory(args[0]) {
		dir = args[0]
	} else {
		if len(tags) != 0 {
			log.Fatal("-tags option applies only to directories, not when files are specified")
		}
		dir = filepath.Dir(args[0])
	}
	g := &Generator{
		prefix: *prefix,
		object: *object,
		output: *output,
	}
	g.parsePackage(args, tags)
	g.printf("// Code generated by \"trigger %s\"; DO NOT EDIT.\n\n",
		strings.Join(os.Args[1:], " "))
	g.printf("package %s\n\n", g.packageName)
	g.generate()
	err := os.WriteFile(filepath.Join(dir, strings.ToLower(*output)),
		g.format(), 0644)
	if err != nil {
		log.Fatalf("write output:%s", err)
	}
}

type Generator struct {
	prefix, object   string
	output, filename string

	packageName string
	files       []*File
	buf         bytes.Buffer
}

func (g *Generator) parsePackage(patterns []string, tags []string) {
	cfg := &packages.Config{
		Mode:       packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Tests:      false,
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(tags, " "))},
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		log.Fatal(err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("error: %d packages found", len(pkgs))
	}
	g.packageName = pkgs[0].Name
	g.files = make([]*File, len(pkgs[0].Syntax))
	for i, file := range pkgs[0].Syntax {
		g.files[i] = &File{
			file: file,
		}
	}
}

func (g *Generator) printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(&g.buf, format, args...)
}

func (g *Generator) generate() {
	typeNames := make([]typeName, 0, 1024)
	for _, file := range g.files {
		file.prefix = g.prefix
		ast.Inspect(file.file, file.genDecl)
		typeNames = append(typeNames, file.typeNames...)
	}
	g.printf("type EventHandler struct {\n")
	for _, tn := range typeNames {
		if tn.needCheck {
			v := strings.ToLower(strings.TrimPrefix(tn.name, g.prefix))
			g.printf("\t%sCheck func(obj *%s, %s %s) bool\n",
				v, g.object, v, tn.name)
		}
	}
	g.printf("\n")
	for _, tn := range typeNames {
		v := strings.ToLower(strings.TrimPrefix(tn.name, g.prefix))
		g.printf("\t%sWatcher []func(obj *%s, %s %s)\n",
			v, g.object, v, tn.name)
	}
	g.printf("}\n\ntype EventTable struct{\n")
	g.printf("\to *%s\n", g.object)
	g.printf("\th *EventHandler\n}\n\n")

	g.printf("func (t *EventHandler) CreateObject(o *%s) *EventTable {\n", g.object)
	g.printf("\treturn &EventTable{\n")
	g.printf("\t\to: o,\n")
	g.printf("\t\th: t,\n")
	g.printf("\t}\n}\n\n")

	for _, tn := range typeNames {
		v := strings.ToLower(strings.TrimPrefix(tn.name, g.prefix))
		if tn.needCheck {
			g.printf("func (t *EventHandler) Check%s(f func(*%s, %s) bool) {\n",
				tn.name, g.object, tn.name)
			g.printf("\tt.%sCheck = f\n}\n\n", v)
		}
		g.printf("func (t *EventHandler) On%s(f func(*%s, %s)) {\n",
			tn.name, g.object, tn.name)
		g.printf("\tt.%sWatcher = append(t.%sWatcher, f)\n}\n\n",
			v, v)

		g.printf("func (t*EventTable) Trigger%s(e %s) {\n",
			tn.name, tn.name)
		if tn.needCheck {
			g.printf("\tif t.h.%sCheck != nil && !t.h.%sCheck(t.o, e) {\n",
				v, v)
			g.printf("\t\treturn\n\t}\n")
		}
		g.printf("\t for _, f := range t.h.%sWatcher {\n", v)
		g.printf("\t\tf(t.o, e)\n\t}\n}\n\n")
	}
}

func (g *Generator) format() []byte {
	src, err := format.Source(g.buf.Bytes())
	if err != nil {
		// Should never happen, but can arise when developing this code.
		// The user can compile the output to see the error.
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		return g.buf.Bytes()
	}
	return src
}

type typeName struct {
	name      string
	needCheck bool
}

type File struct {
	file      *ast.File
	prefix    string
	typeNames []typeName
}

func (f *File) genDecl(node ast.Node) bool {
	decl, ok := node.(*ast.GenDecl)
	if !ok || decl.Tok != token.TYPE {
		return true
	}
	for _, s := range decl.Specs {
		spec := s.(*ast.TypeSpec)
		if strings.HasPrefix(spec.Name.Name, f.prefix) {
			tn := typeName{
				name:      spec.Name.Name,
				needCheck: strings.Contains(spec.Doc.Text(), "need_check"),
			}
			log.Printf("type %s comment %v", tn.name, spec.Doc.Text())
			f.typeNames = append(f.typeNames, tn)
		}
	}
	return true
}

// isDirectory reports whether the named file is a directory.
func isDirectory(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		log.Fatal(err)
	}
	return info.IsDir()
}
