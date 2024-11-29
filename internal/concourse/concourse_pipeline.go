package concourse

import "fmt"

type Pipeline struct {
	Team string
	Name string
}

func (p Pipeline) String() string {
	return fmt.Sprintf("%s/%s", p.Team, p.Name)
}
