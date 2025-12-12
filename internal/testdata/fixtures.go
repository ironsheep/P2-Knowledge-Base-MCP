// Package testdata provides test fixtures for P2KB MCP tests.
package testdata

import (
	"embed"
)

//go:embed fixtures/*
var Fixtures embed.FS

// GetFixture returns the content of a fixture file.
func GetFixture(name string) ([]byte, error) {
	return Fixtures.ReadFile("fixtures/" + name)
}

// MustGetFixture returns the content of a fixture file or panics.
func MustGetFixture(name string) []byte {
	data, err := GetFixture(name)
	if err != nil {
		panic(err)
	}
	return data
}
