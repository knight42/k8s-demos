package podstatus

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

func getSelectorFromObject(obj runtime.Object) string {
	switch actual := obj.(type) {
	// Deployment
	case *appsv1.Deployment:
		return labels.FormatLabels(actual.Spec.Selector.MatchLabels)
	case *appsv1beta1.Deployment:
		return labels.FormatLabels(actual.Spec.Selector.MatchLabels)
	case *appsv1beta2.Deployment:
		return labels.FormatLabels(actual.Spec.Selector.MatchLabels)
	case *extensionsv1beta1.Deployment:
		return labels.FormatLabels(actual.Spec.Selector.MatchLabels)

	// DaemonSet
	case *appsv1.DaemonSet:
		return labels.FormatLabels(actual.Spec.Selector.MatchLabels)
	case *appsv1beta2.DaemonSet:
		return labels.FormatLabels(actual.Spec.Selector.MatchLabels)
	case *extensionsv1beta1.DaemonSet:
		return labels.FormatLabels(actual.Spec.Selector.MatchLabels)

	// StatefulSet
	case *appsv1.StatefulSet:
		return labels.FormatLabels(actual.Spec.Selector.MatchLabels)
	case *appsv1beta1.StatefulSet:
		return labels.FormatLabels(actual.Spec.Selector.MatchLabels)
	case *appsv1beta2.StatefulSet:
		return labels.FormatLabels(actual.Spec.Selector.MatchLabels)

	default:
		return fmt.Sprintf("unknown object: %s", obj.GetObjectKind().GroupVersionKind())
	}
}
