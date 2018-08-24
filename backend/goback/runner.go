package goback

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

func RunFile(filePath string) {
	fmt.Println("Running file", filePath)

	gopath := "../outexec"
	cleanr(gopath)
	fmt.Println("cleaned target directory", gopath)

	dst := gopath + "/out.go"
	copyFile(filePath, dst)
	fmt.Println("copied source file from", filePath, "to", dst)

	output, _ := exec.Command("go", "run", dst).CombinedOutput()

	fmt.Println("Output:")
	fmt.Println(string(output))
	fmt.Println("Run done.")
}

func cleanr(path string) {
	os.RemoveAll(path)
	os.MkdirAll(path, 0700)
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
