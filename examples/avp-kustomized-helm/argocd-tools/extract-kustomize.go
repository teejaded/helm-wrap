package main

import (
	"os"
	"io/ioutil"
	"path"
	"fmt"

	"gopkg.in/yaml.v2"
)

type KustomizeFiles struct {
	Files map[string]string `yaml:"Kustomize"`
}

func main() {
	bytes, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	data := KustomizeFiles{}
	err = yaml.Unmarshal(bytes, &data)
	if err != nil {
		panic(err)
	}

	for k, v := range data.Files {
		f, err := os.Create(path.Base(k))
		if err != nil {
			panic(err)
		}
		f.WriteString(v)
		f.Close()
	}

	fmt.Printf("%s", string(bytes))
}
