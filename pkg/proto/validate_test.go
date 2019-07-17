package proto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidPackageName(t *testing.T) {
	test := assert.New(t)

	testcases := []struct {
		Input string
		Valid bool
	}{
		{"aa", true},
		{"a.a", true},
		{"a_a", true},
		{"a@a", true},
		{"a+a", true},
		{"a_a", true},
		{"a-a", true},
		{"@a-a", false},
		{"a", false},
		{"a/a", false},
		{"aa\\", false},
		{"aa\"", false},
		{"aa'", false},
	}

	for _, testcase := range testcases {
		actual := IsValidPackageName(testcase.Input)

		test.Equal(testcase.Valid, actual, testcase.Input)
	}
}
