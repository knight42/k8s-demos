package podstatus

import (
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
)

func (o *Options) printPods() {
	podsList := make([]*corev1.Pod, 0, len(o.pods))
	for _, pod := range o.pods {
		podsList = append(podsList, pod)
	}
	sort.Slice(podsList, func(i, j int) bool {
		return podsList[i].Name < podsList[j].Name
	})
	for _, pod := range podsList {
		_ = o.PrintPod(pod, false)
	}
	_ = o.writer.Render()
}

// See also https://github.com/kubernetes/kubernetes/blob/master/pkg/printers/internalversion/printers.go#L579
func (o *Options) PrintPod(obj runtime.Object, flush bool) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("object is not a Pod: %#v", obj)
	}
	var (
		readyCount int
		totalCount       = len(pod.Spec.Containers)
		restarts   int32 = 0
	)

	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		ct := pod.Status.InitContainerStatuses[i]
		restarts += ct.RestartCount
		switch {
		case ct.State.Terminated != nil && ct.State.Terminated.ExitCode == 0:
			continue
		case ct.State.Terminated != nil:
			if len(ct.State.Terminated.Reason) != 0 {
				if ct.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", ct.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", ct.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + ct.State.Terminated.Reason
			}
			initializing = true
		case ct.State.Waiting != nil && len(ct.State.Waiting.Reason) > 0 && ct.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + ct.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
	}

	if !initializing {
		restarts = 0
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			restarts += container.RestartCount
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
				readyCount++
			}
		}

		if pod.DeletionTimestamp != nil && pod.Status.Reason == "NodeLost" {
			reason = "Unknown"
		} else if pod.DeletionTimestamp != nil {
			reason = "Terminating"
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			reason = "Running"
		}
	}

	lastReason := "<none>"
	for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
		container := pod.Status.ContainerStatuses[i]
		lastTermState := container.LastTerminationState
		if lastTermState.Terminated != nil {
			t := lastTermState.Terminated
			lastReason = fmt.Sprintf("%s:%d", t.Reason, t.ExitCode)
		} else if lastTermState.Waiting != nil {
			lastReason = lastTermState.Waiting.Reason
		}
	}

	nodeName := pod.Spec.NodeName
	hostIP := pod.Status.HostIP
	podIP := pod.Status.PodIP
	age := "<none>"

	if podIP == "" {
		podIP = "<none>"
	}
	if nodeName == "" {
		nodeName = "<none>"
	}
	if hostIP == "" {
		hostIP = "<none>"
	}
	if pod.Status.StartTime != nil {
		age = duration.ShortHumanDuration(time.Since(pod.Status.StartTime.Time))
	}

	args := []interface{}{
		pod.Name,
		fmt.Sprintf("%d/%d", readyCount, totalCount),
		reason,
		lastReason,
		restarts,
		podIP,
		hostIP,
		nodeName,
		age,
	}
	if flush {
		_ = o.writer.AppendAndFlush(args...)
	} else {
		o.writer.Append(args...)
	}
	return nil
}
