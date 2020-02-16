package podstatus

import (
	"fmt"
	"io"

	"github.com/morikuni/aec"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/scheme"
)

func clearLine(out io.Writer) {
	eraseMode := aec.EraseModes.All
	cl := aec.EraseLine(eraseMode)
	_, _ = fmt.Fprint(out, cl)
}

func cursorUp(out io.Writer, l uint) {
	_, _ = fmt.Fprint(out, aec.Up(l))
}

func isHPA(obj runtime.Object) bool {
	switch obj.(type) {
	case *autoscalingv1.HorizontalPodAutoscaler:
		return true
	case *autoscalingv2beta1.HorizontalPodAutoscaler:
		return true
	case *autoscalingv2beta2.HorizontalPodAutoscaler:
		return true
	}
	return false
}

func isPod(obj runtime.Object) bool {
	switch obj.(type) {
	case *corev1.Pod:
		return true
	}
	return false
}

func newBuilder(f genericclioptions.RESTClientGetter) *resource.Builder {
	return resource.NewBuilder(f).
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		Latest()
}

func getMatchLabels(selector *metav1.LabelSelector) (string, error) {
	if selector == nil {
		return "", fmt.Errorf("nil labelSelector")
	}
	return labels.FormatLabels(selector.MatchLabels), nil
}

func getSelectorFromObject(obj runtime.Object) (string, error) {
	switch actual := obj.(type) {
	// Service
	case *corev1.Service:
		return labels.FormatLabels(actual.Spec.Selector), nil

	// Deployment
	case *appsv1.Deployment:
		return getMatchLabels(actual.Spec.Selector)
	case *appsv1beta1.Deployment:
		return getMatchLabels(actual.Spec.Selector)
	case *appsv1beta2.Deployment:
		return getMatchLabels(actual.Spec.Selector)
	case *extensionsv1beta1.Deployment:
		return getMatchLabels(actual.Spec.Selector)

	// DaemonSet
	case *appsv1.DaemonSet:
		return getMatchLabels(actual.Spec.Selector)
	case *appsv1beta2.DaemonSet:
		return getMatchLabels(actual.Spec.Selector)
	case *extensionsv1beta1.DaemonSet:
		return getMatchLabels(actual.Spec.Selector)

	// StatefulSet
	case *appsv1.StatefulSet:
		return getMatchLabels(actual.Spec.Selector)
	case *appsv1beta1.StatefulSet:
		return getMatchLabels(actual.Spec.Selector)
	case *appsv1beta2.StatefulSet:
		return getMatchLabels(actual.Spec.Selector)

	// CronJob
	case *batchv1beta1.CronJob:
		return labels.FormatLabels(actual.Spec.JobTemplate.Spec.Template.Labels), nil

	// Job
	case *batchv1.Job:
		return getMatchLabels(actual.Spec.Selector)

	// ReplicaSet
	case *extensionsv1beta1.ReplicaSet:
		return getMatchLabels(actual.Spec.Selector)
	case *appsv1.ReplicaSet:
		return getMatchLabels(actual.Spec.Selector)
	case *appsv1beta2.ReplicaSet:
		return getMatchLabels(actual.Spec.Selector)

	default:
		return "", fmt.Errorf("unknown object: %#v", obj)
	}
}

func getRefObject(obj runtime.Object, f genericclioptions.RESTClientGetter) (runtime.Object, error) {
	var (
		ns, name, kind, apiVersion string
	)
	switch actual := obj.(type) {
	case *autoscalingv1.HorizontalPodAutoscaler:
		ref := actual.Spec.ScaleTargetRef
		ns = actual.Namespace
		name, kind, apiVersion = ref.Name, ref.Kind, ref.APIVersion
	case *autoscalingv2beta1.HorizontalPodAutoscaler:
		ref := actual.Spec.ScaleTargetRef
		ns = actual.Namespace
		name, kind, apiVersion = ref.Name, ref.Kind, ref.APIVersion
	case *autoscalingv2beta2.HorizontalPodAutoscaler:
		ref := actual.Spec.ScaleTargetRef
		ns = actual.Namespace
		name, kind, apiVersion = ref.Name, ref.Kind, ref.APIVersion
	default:
		return nil, fmt.Errorf("not hpa: %v", obj)
	}

	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, err
	}
	gvk := gv.WithKind(kind)
	mapper, err := f.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	result := newBuilder(f).
		NamespaceParam(ns).DefaultNamespace().
		ResourceNames(mapping.Resource.Resource, name).SingleResourceType().
		Do()
	if err := result.Err(); err != nil {
		return nil, err
	}
	return result.Object()
}
