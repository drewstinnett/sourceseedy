package archive

import (
	"os/exec"
)

// Call external tar program. All the built in golang ones seem a bit janky 😭
func CreateArchive(base, project, dest string) error {
	cmd := exec.Command("tar", "cvzf", dest, "-C", base, project)
	err := cmd.Run()
	return err
}
