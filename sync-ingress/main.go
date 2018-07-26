package main

import (
	"log"

	"github.com/knight42/k8s-demos/pkg"

	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/watch"
)

func main() {
	config, err := pkg.BuildConfig()
	if err != nil {
		log.Fatal(err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	api := clientset.CoreV1()
	watcher, err := api.Services("infra").Watch(metav1.ListOptions{Watch: true})
	if err != nil {
		log.Fatal(err)
	}
	ch := watcher.ResultChan()
	for evt := range ch {
		svc, ok := evt.Object.(*corev1.Service)
		if !ok {
			log.Fatal("unexpected type")
		}
		switch evt.Type {
		case watch.Added:
			log.Println("Added", svc.Name)
		case watch.Modified:
			log.Println("Modified")
		case watch.Deleted:
			log.Println("Deleted")
		case watch.Error:
			log.Println("watcher error")
		}
	}
}
