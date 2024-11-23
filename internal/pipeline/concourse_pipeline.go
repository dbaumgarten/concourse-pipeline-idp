package pipeline

import "fmt"

type ConcoursePipeline struct {
	Team string
	Name string
}

func (p ConcoursePipeline) String() string {
	return fmt.Sprintf("%s/%s", p.Team, p.Name)
}
