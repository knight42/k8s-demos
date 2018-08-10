package main

import (
	"flag"
	"log"

	"github.com/knight42/k8s-demos/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func main() {
	ns := flag.String("namespace", "", "the namespace scope")

	cfg, err := pkg.BuildConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ds, err := clientset.AppsV1().Deployments(*ns).List(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range ds.Items {
		if item.Status.UnavailableReplicas != 0 {
			log.Printf("%s(%s)", item.Name, item.Namespace)
		}
	}
}
