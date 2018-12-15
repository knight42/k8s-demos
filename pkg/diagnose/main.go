package diagnose

import (
	"fmt"
	"log"
	"os"

	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type containerState struct {
	StateStr   string       `yaml:"state_str,omitempty"`
	Reason     string       `yaml:"reason,omitempty"`
	Message    string       `yaml:"message,omitempty"`
	Signal     int32        `yaml:"signal,omitempty"`
	ExitCode   int32        `yaml:"exit_code,omitempty"`
	StartedAt  *metav1.Time `yaml:"started_at,omitempty"`
	FinishedAt *metav1.Time `yaml:"finished_at,omitempty"`
}

type ContainerStatus struct {
	Name      string          `yaml:"name"`
	Image     string          `yaml:"image"`
	State     *containerState `yaml:"state,omitempty"`
	LastState *containerState `yaml:"last_state,omitempty"`
}

type PodStatus struct {
	Phase            string            `yaml:"phase"`
	Name             string            `yaml:"name"`
	NodeName         string            `yaml:"node_name"`
	ContainersStatus []ContainerStatus `yaml:"containers_status"`
}

type UnhealthyDeployment struct {
	Name                string      `yaml:"name"`
	Namespace           string      `yaml:"namespace"`
	DesiredReplicas     int32       `yaml:"desired_replicas"`
	ReadyReplicas       int32       `yaml:"ready_replicas"`
	UpdatedReplicas     int32       `yaml:"updated_replicas"`
	AvailableReplicas   int32       `yaml:"available_replicas"`
	UnavailableReplicas int32       `yaml:"unavailable_replicas"`
	PodsStatus          []PodStatus `yaml:"pods_status"`
}

func extractState(s corev1.ContainerState) *containerState {
	switch {
	case s.Terminated != nil:
		return &containerState{
			Reason:     s.Terminated.Reason,
			Message:    s.Terminated.Message,
			Signal:     s.Terminated.Signal,
			ExitCode:   s.Terminated.ExitCode,
			StateStr:   "terminated",
			StartedAt:  &s.Terminated.StartedAt,
			FinishedAt: &s.Terminated.FinishedAt,
		}
	case s.Waiting != nil:
		return &containerState{
			Reason:   s.Waiting.Reason,
			Message:  s.Waiting.Message,
			StateStr: "waiting",
		}
	case s.Running != nil:
		return &containerState{
			StartedAt: &s.Running.StartedAt,
			StateStr:  "running",
		}
	default:
		return nil
	}
}

func getCtStatuses(ctstas []corev1.ContainerStatus) ([]ContainerStatus, bool) {
	var ctsStatus []ContainerStatus
	podHealthy := true
	for _, ctsta := range ctstas {
		if ctsta.Ready {
			continue
		}
		podHealthy = false
		s := ContainerStatus{
			Name:      ctsta.Name,
			Image:     ctsta.Image,
			State:     extractState(ctsta.State),
			LastState: extractState(ctsta.LastTerminationState),
		}
		ctsStatus = append(ctsStatus, s)
	}
	return ctsStatus, podHealthy
}

func Run(clientset *kubernetes.Clientset, ns string) {
	core_v1 := clientset.CoreV1()

	var unhealthyDepls []UnhealthyDeployment

	ds, err := clientset.AppsV1().Deployments(ns).List(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range ds.Items {
		if item.Status.UnavailableReplicas == 0 {
			continue
		}
		ns := item.ObjectMeta.Namespace // override empty Namespace
		u := UnhealthyDeployment{
			Name:                item.ObjectMeta.Name,
			Namespace:           ns,
			DesiredReplicas:     *item.Spec.Replicas,
			ReadyReplicas:       item.Status.ReadyReplicas,
			UpdatedReplicas:     item.Status.UpdatedReplicas,
			UnavailableReplicas: item.Status.UnavailableReplicas,
			AvailableReplicas:   item.Status.AvailableReplicas,
		}

		labelSelector := labels.FormatLabels(item.Spec.Selector.MatchLabels)
		lstopts := metav1.ListOptions{
			LabelSelector: labelSelector,
		}
		pods, err := core_v1.Pods(ns).List(lstopts)
		if err != nil {
			log.Printf("error=%s cause='list pods' labels=%s", err, labelSelector)
			continue
		}

		var podsStatus []PodStatus

		for _, pod := range pods.Items {
			var podCtStatuses []corev1.ContainerStatus
			copy(podCtStatuses, pod.Status.ContainerStatuses)
			podCtStatuses = append(podCtStatuses, pod.Status.InitContainerStatuses...)
			ctsStatus, podHealthy := getCtStatuses(podCtStatuses)

			if podHealthy {
				continue
			}
			podsStatus = append(podsStatus, PodStatus{
				Name:             pod.ObjectMeta.Name,
				NodeName:         pod.Spec.NodeName,
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

	data, _ := yaml.Marshal(unhealthyDepls)

	fmt.Println("Unhealthy Deployments:")
	fmt.Println("========================================================")
	_, _ = os.Stdout.Write(data)
}
