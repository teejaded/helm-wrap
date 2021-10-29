package helmwrapper

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/kylelemons/godebug/diff"
)

var hw *HelmWrapper

func init() {
	os.Setenv("HELMWRAP_CONFIG", `[
		{"action":"transform-values","filter":"$.sops.lastmodified","command":"sops -d {}"},
		{"action":"shell-exec","command":"$HELM"}
	]`)
	hw, _ = NewHelmWrapper()
}

func TestNewHelmWrapper(t *testing.T) {
	// TODO
}

func TestErrorf(t *testing.T) {
	hw.Errors = []error{}
	err := hw.errorf("test %s %d %t", "a", 1, true)
	if hw.Errors[0] != err {
		t.Errorf("errorf(test %%s %%d %%t, a, 1, true) = %s; want %s", hw.Errors[0], err)
	}
}

func TestPipeWriter(t *testing.T) {
	// TODO
}

func TestValuesArg(t *testing.T) {
	res, _, err := hw.valuesArg([]string{"-f", "cat.yaml"})
	if res != "cat.yaml" || err != nil {
		t.Errorf("valuesArg([]string{\"-f\", \"cat.yaml\"}) = %s, %s; want cat.yaml, <nil>", res, "cat.yaml")
	}

	res, _, err = hw.valuesArg([]string{"--values", "cat.yaml"})
	if res != "cat.yaml" || err != nil {
		t.Errorf("valuesArg([]string{\"--valuse\", \"cat.yaml\"}) = %s, %s; want cat.yaml, <nil>", res, "cat.yaml")
	}

	res, _, err = hw.valuesArg([]string{"--values=cat.yaml"})
	if res != "cat.yaml" || err != nil {
		t.Errorf("valuesArg([]string{\"--values=cat.yaml\"}) = %s, %s; want cat.yaml, <nil>", res, "cat.yaml")
	}
}

func TestReplaceValueFileArg(t *testing.T) {
	args := []string{"-f", "cat.yaml"}
	hw.replaceValueFileArg(args, "dog.yaml")
	if args[1] != "dog.yaml" {
		t.Errorf("args[1] = %s; want dog.yaml", args[1])
	}

	args = []string{"--values", "cat.yaml"}
	hw.replaceValueFileArg(args, "dog.yaml")
	if args[1] != "dog.yaml" {
		t.Errorf("args[1] = %s; want dog.yaml", args[1])
	}

	args = []string{"--values=cat.yaml"}
	hw.replaceValueFileArg(args, "dog.yaml")
	if args[0] != "--values=dog.yaml" {
		t.Errorf("args[1] = %s; want --values=dog.yaml", args[1])
	}
}

func TestMkTmpDir(t *testing.T) {
	// ensure no errors
	cleanFn, err := hw.mkTmpDir()
	if err != nil {
		t.Errorf("mkTmpDir error: %s", err)
	}

	// dir exists
	if _, err = os.Stat(hw.temporaryDirectory); err != nil {
		t.Errorf("mkTmpDir stat error: %s", err)
	}

	// ensure dir is deleted
	cleanFn()
	if _, err = os.Stat(hw.temporaryDirectory); err == nil {
		t.Errorf("mkTmpDir cleanup func did not work")
	} else if !os.IsNotExist(err) {
		t.Errorf("mkTmpDir cleanup something went wrong: %s", err)
	}
}

func TestMkPipe(t *testing.T) {
	// ensure no errors
	err := hw.mkPipe("cat.yaml")
	if err != nil {
		t.Errorf("mkPipe error: %s", err)
	}

	// file exists
	if _, err = os.Stat("cat.yaml"); err != nil {
		t.Errorf("mkPipe stat error: %s", err)
	}

	os.Remove("cat.yaml")
}

func TestRunHelm(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = writer
	wg := new(sync.WaitGroup)
	var out1 bytes.Buffer
	go func() {
		wg.Add(1)
		defer wg.Done()
		_, err = io.Copy(&out1, reader)
		if err != nil {
			t.Errorf("io.Copy error: %s", err)
		}
	}()

	os.Args = []string{
		"./helm-wrap",
		"template",
		"../../test/charts/test",
		"--values=../../test/charts/test/values-enc.yaml",
		"--values=../../test/charts/test/extra.yaml",
	}
	hw.RunHelm()
	writer.Close()
	wg.Wait()

	reader, writer, err = os.Pipe()
	if err != nil {
		panic(err)
	}
	var out2 bytes.Buffer
	go func() {
		wg.Add(1)
		defer wg.Done()
		_, err = io.Copy(&out2, reader)
		if err != nil {
			t.Errorf("io.Copy error: %s", err)
		}
	}()

	args := []string{
		hw.helmBinPath,
		"template",
		"../../test/charts/test",
		"--values=../../test/charts/test/values-dec.yaml",
		"--values=../../test/charts/test/extra.yaml",
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr

	cmd.Run()
	writer.Close()
	wg.Wait()

	if !bytes.Equal(out1.Bytes(), out2.Bytes()) {
		diff := diff.Diff(out1.String(), out2.String())
		t.Errorf("unexpected RunHelm output: \n%s", diff)
	}
}
