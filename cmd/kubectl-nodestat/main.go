package main

import (
	"github.com/knight42/k8s-tools/pkg/nodestat"
)

func main() {
	_ = nodestat.NewCmd().Execute()
}
