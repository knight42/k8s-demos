package main

import (
	"github.com/knight42/k8s-tools/pkg/podstatus"
)

func main() {
	_ = podstatus.NewCmd().Execute()
}
