package export

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	yaml "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type Config struct {
	Selector  string
	Namespace string
	Name      string
	Resouce   string
}

func cleanMeta(meta *metav1.ObjectMeta) {
	meta.OwnerReferences = nil
	meta.ResourceVersion = ""
	meta.SelfLink = ""
	meta.UID = ""
}

func printYaml(obj runtime.Object) {
	jbytes, _ := json.Marshal(obj)
	v := map[string]interface{}{}
	_ = json.Unmarshal(jbytes, &v)
	if _, ok := v["status"]; ok {
		delete(v, "status")
	}
	encoder := yaml.NewEncoder(os.Stdout)
	_ = encoder.Encode(v)
}

func mutateConfigMap(item *corev1.ConfigMap) {
	cleanMeta(&item.ObjectMeta)
	item.APIVersion = "v1"
	item.Kind = "ConfigMap"
}

func mutateDeployment(item *appsv1.Deployment) {
	cleanMeta(&item.ObjectMeta)
	item.APIVersion = "apps/v1"
	item.Kind = "Deployment"
	item.Status.Reset()
}

func Run(clientset *kubernetes.Clientset, cfg Config) {
	lstOpt := metav1.ListOptions{
		LabelSelector: cfg.Selector,
	}
	getOpt := metav1.GetOptions{}

	cmCli := clientset.CoreV1().ConfigMaps(cfg.Namespace)
	dplyCli := clientset.AppsV1().Deployments(cfg.Namespace)
	switch cfg.Resouce {
	case "cm", "configmap":
		if cfg.Name != "" {
			item, err := cmCli.Get(cfg.Name, getOpt)
			if err != nil {
				log.Fatal(err)
			}
			mutateConfigMap(item)
			printYaml(item)
			return
		}

		result, err := cmCli.List(lstOpt)
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range result.Items {
			mutateConfigMap(&item)
			printYaml(&item)
			fmt.Println("---")
		}

	case "deploy", "deployment", "deployments":
		if cfg.Name != "" {
			item, err := dplyCli.Get(cfg.Name, getOpt)
			if err != nil {
				log.Fatal(err)
			}
			mutateDeployment(item)
			printYaml(item)
			return
		}

		result, err := dplyCli.List(lstOpt)
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range result.Items {
			mutateDeployment(&item)
			printYaml(&item)
			fmt.Println("---")
		}

	default:
		log.Fatalf("unknown resource: %s", cfg.Resouce)
	}

}
