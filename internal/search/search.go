package search

import (
	"strings"
)

type Search struct {
	Search []string
	Data   string
}

func (search *Search) Find() bool {
	for _, row := range search.Search {
		var i int
		split := strings.Split(row, ",")

		for _, cell := range split {
			if strings.Contains(strings.ToLower(search.Data), strings.TrimSpace(strings.ToLower(cell))) {
				i++
			}
		}

		if i == len(split) {
			return true
		}
	}

	return false
}
