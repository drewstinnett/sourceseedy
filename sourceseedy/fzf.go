package sourceseedy

import (
	"io"
	"os"
	"os/exec"
	"strings"
)

func Fzf(data io.Reader) (string, error) {
	var result strings.Builder
	cmd := exec.Command("fzf")
	cmd.Stdout = &result
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	_, err = io.Copy(stdin, data)
	//_, err = data.WriteTo(stdin)
	if err != nil {
		return "", err
	}
	err = stdin.Close()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.String()), nil
}
