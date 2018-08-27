package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/knight42/k8s-utils/pkg"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type ContainerStatus struct {
	State   string `yaml:"state"`
	Image   string `yaml:"image"`
	Message string `yaml:"message"`
	Reason  string `yaml:"reason`
}

type PodStatus struct {
	Phase            string            `yaml:"phase"`
	ContainersStatus []ContainerStatus `yaml:"containers_status"`
}

type UnhealthyDeployment struct {
	Name                string      `yaml:"name"`
	Namespace           string      `yaml:"namespace"`
	DesiredReplicas     int32       `yaml:"desired_replicas"`
	ReadyReplicas       int32       `yaml:"ready_replicas"`
	UpdatedReplicas     int32       `yaml:"updated_replicas"`
	UnavailableReplicas int32       `yaml:"unavailable_replicas"`
	PodsStatus          []PodStatus `yaml:"pods_status"`
}

func main() {
	ns := flag.String("namespace", "", "the namespace scope")

	cfg, err := pkg.BuildConfigFromFlag()
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	corev1 := clientset.CoreV1()

	var unhealthyDepls []UnhealthyDeployment

	ds, err := clientset.AppsV1().Deployments(*ns).List(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Unhealthy Deployments:")
	fmt.Println("========================================================")
	for _, item := range ds.Items {
		if item.Status.UnavailableReplicas == 0 {
			continue
		}
		ns := item.ObjectMeta.Namespace
		u := UnhealthyDeployment{
			Name:                item.ObjectMeta.Name,
			Namespace:           ns,
			DesiredReplicas:     *item.Spec.Replicas,
			ReadyReplicas:       item.Status.ReadyReplicas,
			UpdatedReplicas:     item.Status.UpdatedReplicas,
			UnavailableReplicas: item.Status.UnavailableReplicas,
		}

		labelSelector := labels.FormatLabels(item.Spec.Selector.MatchLabels)
		lstopts := metav1.ListOptions{
			LabelSelector: labelSelector,
		}
		pods, err := corev1.Pods(ns).List(lstopts)
		if err != nil {
			log.Printf("error=%s cause='list pods' labels=%s", err, labelSelector)
			continue
		}

		var podsStatus []PodStatus

		for _, pod := range pods.Items {
			healthy := true

			var ctsStatus []ContainerStatus
			for _, ctsta := range pod.Status.ContainerStatuses {
				if ctsta.Ready {
					continue
				}
				healthy = false
				switch {
				case ctsta.State.Terminated != nil:
					ctsStatus = append(ctsStatus, ContainerStatus{
						State:   "terminated",
						Image:   ctsta.Image,
						Message: ctsta.State.Terminated.Message,
						Reason:  ctsta.State.Terminated.Reason,
					})
				case ctsta.State.Waiting != nil:
					ctsStatus = append(ctsStatus, ContainerStatus{
						State:   "waiting",
						Image:   ctsta.Image,
						Message: ctsta.State.Waiting.Message,
						Reason:  ctsta.State.Waiting.Reason,
					})
				}
			}

			if healthy {
				continue
			}
			podsStatus = append(podsStatus, PodStatus{
				Phase:            string(pod.Status.Phase),
				ContainersStatus: ctsStatus,
			})
		}

		u.PodsStatus = podsStatus
		unhealthyDepls = append(unhealthyDepls, u)
	}

	if len(unhealthyDepls) == 0 {
		return
	}

	data, err := yaml.Marshal(unhealthyDepls)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
}
