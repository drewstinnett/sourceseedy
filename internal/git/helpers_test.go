package git_test

import (
	"io/ioutil"
	"os"
	"path"
)

func BuildTestStruct() string {
	base, err := ioutil.TempDir("", "sourceseedy-tests")
	if err != nil {
		panic(err)
	}
	for _, item := range []string{
		"fake.com/somenamespace/someproject/.git",
		"badhost/somenamespace/someproject/.git",
		"badhost/.fakedir/thing/.git",
		"fake.com/emptydir/",
	} {
		err := os.MkdirAll(path.Join(base, item), 0o755)
		if err != nil {
			panic(err)
		}
	}
	return base
}
