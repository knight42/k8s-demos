package main

import (
	"fmt"
	"log"

	"github.com/knight42/k8s-demos/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
	cfg, err := pkg.BuildConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := metricsclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	node_metrics := clientset.MetricsV1beta1().NodeMetricses()
	metrics, err := node_metrics.List(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range metrics.Items {
		fmt.Println("======================")
		fmt.Printf("name: %s\n", item.ObjectMeta.Name)
		fmt.Printf("timestamp: %s\n", item.Timestamp)
		fmt.Printf("window: %s\n", item.Window)
		fmt.Printf("cpu: %s\n", item.Usage.Cpu())
		fmt.Printf("memory: %s\n", item.Usage.Memory())
	}
}
