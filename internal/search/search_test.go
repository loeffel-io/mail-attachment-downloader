package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFind(t *testing.T) {
	tests := []struct {
		search   *Search
		expected bool
	}{
		{
			search: &Search{
				Search: []string{
					"apple",
					"youtube",
				},
				Data: "youtube",
			},
			expected: true,
		},
		{
			search: &Search{
				Search: []string{
					"apple",
					"youtube",
				},
				Data: "test",
			},
			expected: false,
		},
		{
			search: &Search{
				Search: []string{
					"invoice, apple",
					"movie",
				},
				Data: "your invoice from apple",
			},
			expected: true,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.search.Find())
	}
}
