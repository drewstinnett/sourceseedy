package finder

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/apex/log"
	"github.com/drewstinnett/sourceseedy/internal/project"
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

func FzfProjects(base string) (string, error) {
	projects, err := project.ListAllProjectFullIDs(base)
	if err != nil {
		return "", err
	}
	r := new(bytes.Buffer)
	var thing string
	for _, p := range projects {
		line := fmt.Sprintf(p + "\n")
		r.Write([]byte(line))

	}
	thing, err = Fzf(r)
	if err != nil {
		return "", err
	}
	return thing, nil
}

func fzfWithFilter(command string, input func(in io.WriteCloser)) string {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	cmd := exec.Command(shell, "-c", command)
	cmd.Stderr = os.Stderr
	in, _ := cmd.StdinPipe()
	go func() {
		input(in)
		in.Close()
	}()
	result, _ := cmd.Output()
	return string(result)
}

func StreamFzfProjects(base, filter string) (string, error) {
	var namespaces []project.Namespace
	hs, err := project.ListHosts(base)
	if err != nil {
		return "", err
	}
	for _, h := range hs {
		ns, err := h.ListNamespaces()
		if err != nil {
			return "", err
		}
		namespaces = append(namespaces, ns...)
	}
	var fzfCmd string
	if filter != "" {
		fzfCmd = fmt.Sprintf("fzf +m -q \"%v\"", filter)
	} else {
		fzfCmd = "fzf +m"
	}
	filtered := fzfWithFilter(fzfCmd, func(in io.WriteCloser) {
		var wg sync.WaitGroup
		for _, namespace := range namespaces {
			wg.Add(1)
			go func(namespace project.Namespace) {
				defer wg.Done()
				projects, err := namespace.ListProjects()
				if err != nil {
					log.WithError(err)
				}
				for _, project := range projects {
					fmt.Fprintln(in, project.FullID())
					// batch = append(batch, project.FullID())
				}
			}(namespace)
		}
		wg.Wait()
	})
	// fmt.Println(filtered)
	return strings.TrimSuffix(filtered, "\n"), nil
}
