package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	extv1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Service struct {
	Name      string
	Port      int32
	Namespace string
}

var (
	KubeConfig        string = ""
	IngressNamespaces string = ""
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func init() {
	if home := homeDir(); home != "" {
		KubeConfig = filepath.Join(home, ".kube", "config")
	}
	IngressNamespaces = os.Getenv("NAMESPACES_TO_USE_INTERNAL_INGRESS")
}

func FromServices(ns string, svcs []Service) *extv1.Ingress {
	ing := &extv1.Ingress{}

	objMeta := metav1.ObjectMeta{}
	objMeta.SetName("auto-ingress")
	objMeta.SetNamespace(ns)
	objMeta.SetAnnotations(map[string]string{
		"nginx.ingress.kubernetes.io/rewrite-target": "/",
		"nginx.ingress.kubernetes.io/ssl-redirect":   "false",
		"kubernetes.io/ingress.class":                "nginx",
	})

	httpIngVal := new(extv1.HTTPIngressRuleValue)
	for _, svc := range svcs {
		httpIngVal.Paths = append(httpIngVal.Paths, extv1.HTTPIngressPath{
			Backend: extv1.IngressBackend{
				ServiceName: svc.Name,
				ServicePort: intstr.FromInt(int(svc.Port)),
			},
			Path: fmt.Sprintf("/__%s.%s.%d__", svc.Name, svc.Namespace, svc.Port),
		})
	}

	rule := extv1.IngressRule{}
	rule.HTTP = httpIngVal

	ing.ObjectMeta = objMeta
	ing.Spec.Rules = []extv1.IngressRule{rule}
	return ing
}

func BuildConfig() (*rest.Config, error) {
	var (
		config *rest.Config
		err    error
	)
	// use the current context in kubeconfig
	if os.Getenv("IN_CLUSTER") == "true" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", KubeConfig)
	}
	return config, err
}

func CreateIngress(clientset *kubernetes.Clientset, ns string) error {
	ext_api := clientset.ExtensionsV1beta1()
	svcLst, err := clientset.CoreV1().Services(ns).List(metav1.ListOptions{})
	svcItems := svcLst.Items
	if len(svcItems) == 0 {
		return nil
	}

	xs := []Service{}
	for _, svc := range svcItems {
		if svc.Spec.Type == corev1.ServiceTypeExternalName {
			continue
		}
		if len(svc.Spec.Ports) == 0 {
			continue
		}
		for _, p := range svc.Spec.Ports {
			if p.Port == 53 || p.Protocol == "UDP" {
				continue
			}
			xs = append(xs, Service{
				Name:      svc.Name,
				Port:      p.Port,
				Namespace: ns,
			})
		}
	}
	ing := FromServices(ns, xs)
	log.Printf("Create %d rules in %s", len(xs), ns)
	_, err = ext_api.Ingresses(ns).Update(ing)
	if errors.IsNotFound(err) {
		_, err = ext_api.Ingresses(ns).Create(ing)
	}
	return err
}

func main() {
	cfg, err := BuildConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if IngressNamespaces == "" {
		log.Fatal("Empty namespace")
	}
	namespaces := strings.Split(IngressNamespaces, ",")
	for _, ns := range namespaces {
		err = CreateIngress(clientset, ns)
		if err != nil {
			log.Println(err)
		}
	}
}
