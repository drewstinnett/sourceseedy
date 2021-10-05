package project_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

var testBase string

func TestMain(m *testing.M) {
	// exec setUp function
	setUp()
	// exec test and this returns an exit code to pass to os
	retCode := m.Run()
	// exec tearDown function
	tearDown()
	// If exit code is distinct of zero,
	// the test will be failed (red)
	os.Exit(retCode)
}

func setUp() {
	var err error
	testBase, err = ioutil.TempDir("", "sourceseedy-tests")
	if err != nil {
		panic(err)
	}
	for _, item := range []string{
		"fake.com/somenamespace/someproject/.git",
		"badhost/somenamespace/someproject/.git",
		"badhost/.fakedir/thing/.git",
		"fake.com/emptydir/",
	} {
		err := os.MkdirAll(path.Join(testBase, item), 0o755)
		if err != nil {
			panic(err)
		}
	}
}

func tearDown() {
	os.RemoveAll(testBase)
}
