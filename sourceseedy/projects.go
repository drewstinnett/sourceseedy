package sourceseedy

import "fmt"

type Project struct {
	Name      string
	Host      string
	Namespace string
	Directory string
}

func (p Project) FullID() string {
	return fmt.Sprintf("%v/%v/%v", p.Host, p.Namespace, p.Name)
}
