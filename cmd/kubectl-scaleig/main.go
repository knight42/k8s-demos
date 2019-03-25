package main

import (
	"github.com/knight42/k8s-tools/pkg/scaleig"
)

func main() {
	_ = scaleig.NewCmd().Execute()
}
