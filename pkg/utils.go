package pkg

import (
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Kind int

const (
	Unknown Kind = iota
	Deployments
	ConfigMaps
	Services
	DaemonSets
	StatefulSets
	Cronjobs
)

func (k Kind) String() string {
	switch k {
	case Deployments:
		return "deployments"
	case ConfigMaps:
		return "configmaps"
	case Services:
		return "services"
	case DaemonSets:
		return "daemonsets"
	case StatefulSets:
		return "statefulsets"
	case Cronjobs:
		return "cronjobs"
	default:
		return "unknown"
	}
}

func CanonicalizeKind(kind string) (Kind, bool) {
	switch kind {
	case "deployments", "deployment", "deploy":
		return Deployments, true
	case "configmaps", "configmap", "cm":
		return ConfigMaps, true
	case "services", "service", "svc":
		return Services, true
	case "daemonsets", "daemonset", "ds":
		return DaemonSets, true
	case "statefulsets", "statefulset", "sts":
		return StatefulSets, true
	case "cronjobs", "cronjob", "cj":
		return Cronjobs, true
	default:
		return Unknown, false
	}
}

func CheckError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func MangleObject(obj metav1.Object) {
	obj.SetSelfLink("")
	obj.SetResourceVersion("")
	obj.SetOwnerReferences(nil)
	obj.SetUID("")

	switch item := obj.(type) {
	case *corev1.ConfigMap:
		item.APIVersion = "v1"
		item.Kind = "ConfigMap"
	case *corev1.Service:
		item.Status.Reset()
		item.APIVersion = "v1"
		item.Kind = "Service"
	case *appsv1.Deployment:
		item.Status.Reset()
		item.APIVersion = "apps/v1"
		item.Kind = "Deployment"
	case *appsv1.StatefulSet:
		item.Status.Reset()
		item.APIVersion = "apps/v1"
		item.Kind = "StatefulSet"
	case *appsv1.DaemonSet:
		item.Status.Reset()
		item.APIVersion = "apps/v1"
		item.Kind = "DaemonSet"
	}
}
