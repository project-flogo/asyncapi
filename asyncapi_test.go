package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func TestAsyncApi(t *testing.T) {
	testAPI := func(file string) {
		t.Log(file)
		current, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			err = os.Chdir(current)
			if err != nil {
				t.Fatal(err)
			}
		}()

		tmp, err := ioutil.TempDir("", "generate_api")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(tmp)
		cmd := exec.Command("./asyncapi", "-input", file, "-type", "flogoapiapp", "-output", tmp)
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = os.Chdir(tmp)
		if err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("go", "build")
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = os.RemoveAll(tmp)
		if err != nil {
			t.Fatal(err)
		}
	}
	testJSON := func(file string) {
		t.Log(file)
		current, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			err = os.Chdir(current)
			if err != nil {
				t.Fatal(err)
			}
		}()

		tmp, err := ioutil.TempDir("", "generate_json")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(tmp)
		cmd := exec.Command("./asyncapi", "-input", file, "-type", "flogodescriptor", "-output", tmp)
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = os.Chdir(tmp)
		if err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("flogo", "create", "app")
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("mv", "support.go", "app/src")
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = os.Chdir("app")
		if err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("flogo", "build")
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = os.RemoveAll(tmp)
		if err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command("go", "build")
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	files := [...]string{
		"examples/eftl/asyncapi.yml",
		"examples/eftl/asyncapi_secure.yml",
		"examples/http/asyncapi.yml",
		"examples/http/asyncapi_secure.yml",
		"examples/kafka/asyncapi.yml",
		"examples/kafka/asyncapi_secure.yml",
		"examples/mqtt/asyncapi.yml",
		"examples/mqtt/asyncapi_secure.yml",
		"examples/websocket/asyncapi.yml",
		"examples/websocket/asyncapi_secure.yml",
	}
	for _, file := range files {
		testAPI(file)
		testJSON(file)
	}
}
