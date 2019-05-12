package main

import (
	"github.com/knight42/k8s-tools/pkg/roles"
)

func main() {
	_ = roles.NewCmd().Execute()
}
