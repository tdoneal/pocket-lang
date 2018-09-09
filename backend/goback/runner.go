package goback

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

func RunFile(filePath string) string {

	gopath := "../outexec"
	cleanr(gopath)
	fmt.Println("cleaned target directory", gopath)

	dst := gopath + "/out.go"
	copyFile(filePath, dst)
	fmt.Println("copied source file from", filePath, "to", dst)

	// copy runtime libs
	outLibPath := "../outexec/lib.go"
	copyRuntimeLib("./backend/goback/runtime.go", outLibPath)
	fmt.Println("created runtime lib at", outLibPath)

	fmt.Println("running file", filePath)
	output, _ := exec.Command("go", "run", dst, outLibPath).CombinedOutput()

	fmt.Println("Output:")
	fmt.Println(string(output))
	fmt.Println("Run done.")

	return string(output)
}

func cleanr(path string) {
	os.RemoveAll(path)
	os.MkdirAll(path, 0700)
}

func copyRuntimeLib(src string, dst string) {
	dat, err := ioutil.ReadFile(src)
	if err != nil {
		panic(err)
	}
	fmt.Println("input lib file:")
	iconts := string(dat)

	// clean it up a bit
	iconts = strings.Replace(iconts, "goback", "main", 1)

	ioutil.WriteFile(dst, ([]byte)(iconts), 0644)
}

// copy the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
