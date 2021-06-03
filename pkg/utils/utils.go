package utils

import (
	"io/ioutil"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"sigs.k8s.io/yaml"
	"k8s.io/client-go/util/jsonpath"
)

func DetectSopsYaml(filename string) (bool, error) {
	return DetectJsonPath(filename, "$.sops.lastmodified")
}

func DetectJsonPath(filename string, path string) (bool, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return false, fmt.Errorf("ioutil.ReadFile: %w", err)
	}

	var data interface{}
	err = yaml.Unmarshal(bytes, &data)
	if err != nil {
	    return false, fmt.Errorf("yaml.Unmarshal: %w", err)
	}

	j := jsonpath.New("DetectSopsYaml").AllowMissingKeys(true)

	err = j.Parse(fmt.Sprintf("{%s}",path))
	if err != nil {
		return false, fmt.Errorf("jsonpath.Parse: %w", err)
	}

	res, err := j.FindResults(data)
	if err != nil {
		return false, fmt.Errorf("jsonpath.FindResults: %w", err)
	}

	return len(res[0]) > 0, nil
}

func Exec(binargs string, filename string) ([]byte, error) {
	args := strings.Split(binargs, " ")
	for i, arg := range args {
		if arg == "{}" {
			args[i] = filename
		}
	}
	path, err := exec.LookPath(args[0])
	if err != nil {
	    return nil, err
	}
	args[0] = path
	cmd := exec.Command(path, args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	return cmd.Output()
}
