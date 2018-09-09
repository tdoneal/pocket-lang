package pktest

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"pocket-lang/backend/goback"
	"pocket-lang/frontend/pocket"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

func RunCase(inFile string) {
	dat, err := ioutil.ReadFile(inFile)
	if err != nil {
		panic(err)
	}
	fmt.Println("input file:")
	fmt.Println(string(dat))
	sdat := string(dat)
	sections := strings.Split(sdat, ">>>")
	for i := 0; i < len(sections)-1; i += 2 {
		src := sections[i]
		desOutput := SanitizeOutput(sections[i+1])
		actOutput := SanitizeOutput(CompileAndRunSrc(src))
		fmt.Println("wanted", desOutput)
		fmt.Println("got", actOutput)
		if desOutput != actOutput {
			panic("failed")
		}
	}

}

func SanitizeOutput(op string) string {
	rv := strings.Replace(op, "\r\n", "\n", -1)
	rv = strings.TrimSpace(rv)
	return rv
}

func ListDirFiles(inDir string) []string {
	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		panic(err)
	}
	rv := []string{}
	for _, file := range files {
		fullPath := filepath.Join(inDir, file.Name())
		fmt.Println("fullPath", fullPath)
		rv = append(rv, fullPath)
	}
	return rv
}

func CompileAndRunSrc(inSrc string) string {
	fmt.Println("input file:")
	fmt.Println(string(inSrc))

	tokens := pocket.Tokenize(string(inSrc))
	fmt.Println("final tokens:\n", spew.Sdump(tokens))

	parsed := pocket.Parse(tokens)
	fmt.Println("final parsed:\n", pocket.PrettyPrint(parsed))

	xformed := pocket.Xform(parsed)
	fmt.Println("final xformed:\n", pocket.PrettyPrint(xformed))

	genned := goback.Generate(parsed)
	fmt.Println("final generated:\n", genned)

	err := ioutil.WriteFile("./outcode/out.go", []byte(genned), 0644)
	if err != nil {
		panic(err)
	}

	return goback.RunFile("./outcode/out.go")
}

func CompileAndRunFile(inPath string) (output string) {
	dat, err := ioutil.ReadFile(inPath)
	if err != nil {
		panic(err)
	}
	return CompileAndRunSrc(string(dat))
}
