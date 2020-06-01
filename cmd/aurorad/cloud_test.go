package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNext_One(t *testing.T) {
	test := assert.New(t)

	cloud, _ := NewCloud("a", ConfigResources{CPU: 1}, 4)

	test.Equal("0", cloud.getNextCPU())
	test.Equal("1", cloud.getNextCPU())
	test.Equal("2", cloud.getNextCPU())
	test.Equal("3", cloud.getNextCPU())
	test.Equal("0", cloud.getNextCPU())
}

func TestNext_Two(t *testing.T) {
	test := assert.New(t)

	cloud, _ := NewCloud("a", ConfigResources{CPU: 2}, 4)

	test.Equal("0-1", cloud.getNextCPU())
	test.Equal("2-3", cloud.getNextCPU())
	test.Equal("0-1", cloud.getNextCPU())
}
