package main

import (
	"github.com/knight42/k8s-tools/pkg/rm"
)

func main() {
	_ = rm.NewCmd().Execute()
}
