package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSafeScale(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SafeScale Suite")
}
