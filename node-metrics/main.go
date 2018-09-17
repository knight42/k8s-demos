package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/knight42/k8s-utils/pkg"

	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// type doc: https://godoc.org/github.com/kubernetes/metrics/pkg/apis/metrics
// client api doc: https://godoc.org/github.com/kubernetes/metrics/pkg/client/clientset/versioned
// resouce doc: https://godoc.org/k8s.io/apimachinery/pkg/api/resource#Quantity

type NodeMetric struct {
	Name        string `yaml:"name"`
	UsedCpu     string `yaml:"used_cpu"`
	UsedMemory  string `yaml:"used_memory"`
	AvailCpu    string `yaml:"avail_cpu"`
	AvailMemory string `yaml:"avail_memory"`
}

type ContainerResource struct {
	CpuUsage   string `yaml:"cpu_usage"`
	MemUsage   string `yaml:"mem_usage"`
	CpuRequest string `yaml:"cpu_request"`
	MemRequest string `yaml:"mem_request"`
	CpuLimit   string `yaml:"cpu_limit"`
	MemLimit   string `yaml:"mem_limit"`
}

type PodMetric struct {
	Namespace  string                        `yaml:"namespace"`
	Containers map[string]*ContainerResource `yaml:"containers"`
}

func printOverView(clientset *kubernetes.Clientset, metricsCliset *metricsclientset.Clientset) {
	nodeMetricCli := metricsCliset.MetricsV1beta1().NodeMetricses()

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	var nodeMetrics []NodeMetric
	for _, node := range nodes.Items {
		m := NodeMetric{
			Name: node.ObjectMeta.Name,
		}

		nm, err := nodeMetricCli.Get(node.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			log.Printf("error=%s node=%s", err, node.ObjectMeta.Name)
		}
		allocCpu := node.Status.Allocatable.Cpu()
		allocMem := node.Status.Allocatable.Memory()
		allocCpu.Sub(*nm.Usage.Cpu())
		allocMem.Sub(*nm.Usage.Memory())
		m.AvailCpu = allocCpu.String()
		m.AvailMemory = allocMem.String()
		m.UsedCpu = nm.Usage.Cpu().String()
		m.UsedMemory = nm.Usage.Memory().String()
		nodeMetrics = append(nodeMetrics, m)
	}

	if len(nodeMetrics) == 0 {
		return
	}

	data, _ := yaml.Marshal(nodeMetrics)
	fmt.Println(string(data))
}

func printNodeDetail(clientset *kubernetes.Clientset, metricsCliset *metricsclientset.Clientset, nodeName string) {
	podsCli := clientset.CoreV1().Pods(metav1.NamespaceAll)
	fieldSet := fields.Set{
		"spec.nodeName": nodeName,
		"status.phase":  "Running",
	}
	pods, err := podsCli.List(metav1.ListOptions{
		FieldSelector: fieldSet.String(),
	})
	if err != nil {
		log.Fatal(err)
	}

	result := make(map[string]PodMetric, len(pods.Items))
	for _, p := range pods.Items {
		pm := PodMetric{}
		pm.Namespace = p.ObjectMeta.Namespace
		pm.Containers = make(map[string]*ContainerResource, len(p.Spec.Containers))
		for _, ct := range p.Spec.Containers {
			var ctRes ContainerResource
			ctRes.CpuLimit = ct.Resources.Limits.Cpu().String()
			ctRes.MemLimit = ct.Resources.Limits.Memory().String()
			ctRes.CpuRequest = ct.Resources.Requests.Cpu().String()
			ctRes.MemRequest = ct.Resources.Requests.Memory().String()
			pm.Containers[ct.Name] = &ctRes
		}

		podMetricCli := metricsCliset.MetricsV1beta1().PodMetricses(pm.Namespace)
		m, err := podMetricCli.Get(p.Name, metav1.GetOptions{})
		if err != nil {
			log.Println(err)
			continue
		}
		for _, ct := range m.Containers {
			pm.Containers[ct.Name].CpuUsage = ct.Usage.Cpu().String()
			pm.Containers[ct.Name].MemUsage = ct.Usage.Memory().String()
		}
		result[p.Name] = pm
	}

	if len(result) == 0 {
		return
	}

	data, _ := yaml.Marshal(result)
	fmt.Println(string(data))
}

func main() {
	nodeName := flag.String("node", "", "node name")

	cfg, err := pkg.BuildConfigFromFlag()
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	metricsCliset, err := metricsclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if *nodeName == "" {
		printOverView(clientset, metricsCliset)
	} else {
		printNodeDetail(clientset, metricsCliset, *nodeName)
	}
}
