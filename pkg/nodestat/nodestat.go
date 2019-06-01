package nodestat

import (
	"fmt"
	"os"
	"sort"

	"github.com/knight42/k8s-tools/pkg/tabwriter"
	"github.com/knight42/k8s-tools/pkg/utils"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsv1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

func memoryInMB(bytes int64) int64 {
	return bytes / (1024 * 1024)
}

type NodeStatOptions struct {
	configFlags *genericclioptions.ConfigFlags

	namespace     string
	name          string
	labelSelector string

	args             []string
	enforceNamespace bool
	metricsClient    metricsclientset.Interface
	writer           *tabwriter.Writer
}

func NewNodeStatOptions() *NodeStatOptions {
	return &NodeStatOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmd() *cobra.Command {
	o := NewNodeStatOptions()

	cmd := &cobra.Command{
		Use: "kubectl nodestat [NAME | -l label] [flags]",
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(cmd, args))
			utils.CheckError(o.Validate())
			utils.CheckError(o.Run())
		},
	}
	cmd.Flags().StringVarP(&o.labelSelector, "selector", "l", o.labelSelector, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")

	flags := cmd.PersistentFlags()
	o.configFlags.AddFlags(flags)

	return cmd
}

func (o *NodeStatOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var (
		err error
	)

	o.namespace, o.enforceNamespace, err = o.configFlags.
		ToRawKubeConfigLoader().
		Namespace()
	if err != nil {
		return err
	}

	restCfg, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	o.metricsClient, err = metricsclientset.NewForConfig(restCfg)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		o.name = args[0]
	}

	o.writer = tabwriter.New(os.Stdout)

	return nil
}

func (o *NodeStatOptions) newBuilder() *resource.Builder {
	return resource.NewBuilder(o.configFlags).
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		Flatten().
		Latest()
}

func (o *NodeStatOptions) Validate() error {
	if len(o.args) > 1 {
		return fmt.Errorf("only one node can be specified")
	}
	return nil
}

func (o *NodeStatOptions) getNodeMetrics() (*metricsapi.NodeMetricsList, error) {
	nmCli := o.metricsClient.MetricsV1beta1().NodeMetricses()
	nml := &metricsapi.NodeMetricsList{}

	if len(o.name) == 0 {
		ml, err := nmCli.List(metav1.ListOptions{
			LabelSelector: o.labelSelector,
		})
		if err != nil {
			return nil, err
		}

		err = metricsv1beta1api.Convert_v1beta1_NodeMetricsList_To_metrics_NodeMetricsList(ml, nml, nil)
		if err != nil {
			return nil, err
		}
	} else {
		var nm metricsapi.NodeMetrics
		m, err := nmCli.Get(o.name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		err = metricsv1beta1api.Convert_v1beta1_NodeMetrics_To_metrics_NodeMetrics(m, &nm, nil)
		if err != nil {
			return nil, err
		}
		nml.Items = []metricsapi.NodeMetrics{nm}
	}

	return nml, nil
}

func (o *NodeStatOptions) Run() error {
	nml, err := o.getNodeMetrics()
	if err != nil {
		return err
	}

	if len(nml.Items) == 0 {
		return fmt.Errorf("metrics not available yet")
	}

	args := append([]string{"nodes"}, o.args...)
	r := o.newBuilder().
		SingleResourceType().
		LabelSelector(o.labelSelector).
		ResourceTypeOrNameArgs(true, args...).
		Do()

	if err := r.Err(); err != nil {
		return err
	}

	allocatable := make(map[string]corev1.ResourceList)
	err = r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}

		n := info.Object.(*corev1.Node)
		allocatable[n.Name] = n.Status.Allocatable
		return nil
	})
	if err != nil {
		return err
	}

	return o.printResourceUsage(nml.Items, allocatable)
}

func (o *NodeStatOptions) printResourceUsage(nodeMetrics []metricsapi.NodeMetrics, allocatable map[string]corev1.ResourceList) error {
	sort.Slice(nodeMetrics, func(i, j int) bool {
		return nodeMetrics[i].Name < nodeMetrics[j].Name
	})

	o.writer.SetHeader([]string{"name", "CPU(usage/total)", "requests/limits", "memory(usage/total)", "requests/limits"})

	maxNodes := len(nodeMetrics)
	var usage corev1.ResourceList
	for i, m := range nodeMetrics {
		err := scheme.Scheme.Convert(&m.Usage, &usage, nil)
		if err != nil {
			return err
		}

		percent := 100 * float32(i+1) / float32(maxNodes)
		fmt.Printf("\r\033[2KProgress: %3.2f%% (%d/%d), Node: %s", percent, i+1, maxNodes, m.Name)
		totalReqsAndLims, err := o.getTotalRequestsAndLimits(m.Name)
		if err != nil {
			return err
		}

		cpuReqs := totalReqsAndLims[corev1.ResourceRequestsCPU]
		memReqs := totalReqsAndLims[corev1.ResourceRequestsMemory]
		cpuLims := totalReqsAndLims[corev1.ResourceLimitsCPU]
		memLims := totalReqsAndLims[corev1.ResourceLimitsMemory]

		cpuUsage := usage.Cpu()
		memUsage := usage.Memory()

		total := allocatable[m.Name]
		cpuTotal := total.Cpu()
		memTotal := total.Memory()

		fractionCpuReqs := float64(0)
		fractionCpuLimits := float64(0)
		fractionMemReqs := float64(0)
		fractionMemLimits := float64(0)
		fractionCpuUsage := float64(0)
		fractionMemUsage := float64(0)

		if cpuTotal.Value() != 0 {
			fractionCpuUsage = float64(cpuUsage.MilliValue()) / float64(cpuTotal.MilliValue()) * 100
			fractionCpuReqs = float64(cpuReqs.MilliValue()) / float64(cpuTotal.MilliValue()) * 100
			fractionCpuLimits = float64(cpuLims.MilliValue()) / float64(cpuTotal.MilliValue()) * 100
		}

		if memTotal.Value() != 0 {
			fractionMemUsage = float64(memUsage.Value()) / float64(memTotal.Value()) * 100
			fractionMemReqs = float64(memReqs.Value()) / float64(memTotal.Value()) * 100
			fractionMemLimits = float64(memLims.Value()) / float64(memTotal.Value()) * 100
		}

		o.writer.Append(
			m.Name,
			fmt.Sprintf("%vm(%.1f%%)/%vm", cpuUsage.MilliValue(), fractionCpuUsage, cpuTotal.MilliValue()),
			fmt.Sprintf("%vm(%.1f%%)/%vm(%.1f%%)", cpuReqs.MilliValue(), fractionCpuReqs, cpuLims.MilliValue(), fractionCpuLimits),
			fmt.Sprintf("%vMi(%.1f%%)/%vMi", memoryInMB(memUsage.Value()), fractionMemUsage, memoryInMB(memTotal.Value())),
			fmt.Sprintf("%vMi(%.1f%%)/%vMi(%.1f%%)", memoryInMB(memReqs.Value()), fractionMemReqs, memoryInMB(memLims.Value()), fractionMemLimits),
		)
	}
	fmt.Println()
	return o.writer.Render()
}

func (o *NodeStatOptions) getTotalRequestsAndLimits(nodeName string) (corev1.ResourceList, error) {
	fieldSelector := fields.AndSelectors(
		fields.OneTermEqualSelector("spec.nodeName", nodeName),
		fields.OneTermNotEqualSelector("status.phase", string(corev1.PodSucceeded)),
		fields.OneTermNotEqualSelector("status.phase", string(corev1.PodFailed)),
	)
	r := o.newBuilder().
		ResourceTypes("pods").
		FieldSelectorParam(fieldSelector.String()).
		Do()

	if err := r.Err(); err != nil {
		return nil, err
	}

	totalReqs, totalLimits := corev1.ResourceList{}, corev1.ResourceList{}

	r.Visit(func(info *resource.Info, err error) error {
		pod := info.Object.(*corev1.Pod)
		podReqs, podLimits := utils.PodRequestsAndLimits(pod)

		for name, curVal := range podReqs {
			if totalVal, found := totalReqs[name]; !found {
				totalReqs[name] = curVal.DeepCopy()
			} else {
				totalVal.Add(curVal)
				totalReqs[name] = totalVal
			}
		}

		for name, curVal := range podLimits {
			if totalVal, found := totalLimits[name]; !found {
				totalLimits[name] = curVal.DeepCopy()
			} else {
				totalVal.Add(curVal)
				totalLimits[name] = totalVal
			}
		}
		return nil
	})

	total := corev1.ResourceList{
		corev1.ResourceRequestsCPU:    totalReqs[corev1.ResourceCPU],
		corev1.ResourceRequestsMemory: totalReqs[corev1.ResourceMemory],

		corev1.ResourceLimitsCPU:    totalLimits[corev1.ResourceCPU],
		corev1.ResourceLimitsMemory: totalLimits[corev1.ResourceMemory],
	}

	return total, nil
}
