package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestAsyncApi(t *testing.T) {
	test := func(file string) {
		t.Log(file)
		err := os.Mkdir("test", 0777)
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("./asyncapi", "-input", file, "-type", "flogoapiapp", "-output", "./test/")
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = os.Chdir("test")
		if err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("go", "build")
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = os.Chdir("..")
		if err != nil {
			t.Fatal(err)
		}
		err = os.RemoveAll("./test")
		if err != nil {
			t.Fatal(err)
		}

		err = os.Mkdir("test", 0777)
		if err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("./asyncapi", "-input", file, "-type", "flogodescriptor", "-output", "./test/")
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = os.Chdir("test")
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
		err = os.Chdir("../..")
		if err != nil {
			t.Fatal(err)
		}
		err = os.RemoveAll("./test")
		if err != nil {
			t.Fatal(err)
		}
	}
	test("examples/eftl/asyncapi.yml")
	test("examples/eftl/asyncapi_secure.yml")
	test("examples/http/asyncapi.yml")
	test("examples/http/asyncapi_secure.yml")
	test("examples/kafka/asyncapi.yml")
	test("examples/kafka/asyncapi_secure.yml")
	test("examples/mqtt/asyncapi.yml")
	test("examples/mqtt/asyncapi_secure.yml")
	test("examples/websocket/asyncapi.yml")
	test("examples/websocket/asyncapi_secure.yml")
}
