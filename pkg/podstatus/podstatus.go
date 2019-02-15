package podstatus

import (
	"fmt"
	"os"

	"github.com/knight42/k8s-tools/pkg"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

type PodStatusOptions struct {
	configFlags *genericclioptions.ConfigFlags

	// results of arg parsing
	namespace string
	name      string
	clientSet *kubernetes.Clientset
}

func NewPodStatusOptions() *PodStatusOptions {
	return &PodStatusOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmd() *cobra.Command {
	o := NewPodStatusOptions()

	cmd := &cobra.Command{
		Use: "kubectl podstatus [deployment|statefulset|daemonset]",
		Run: func(cmd *cobra.Command, args []string) {
			pkg.CheckError(o.Complete(cmd, args))
			pkg.CheckError(o.Validate())
			pkg.CheckError(o.Run())
		},
	}

	flags := cmd.PersistentFlags()
	o.configFlags.AddFlags(flags)

	return cmd
}

func (o *PodStatusOptions) Complete(cmd *cobra.Command, args []string) error {
	loader := o.configFlags.ToRawKubeConfigLoader()
	restConfig, err := loader.ClientConfig()
	if err != nil {
		return err
	}
	cliset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	o.clientSet = cliset

	ns, _, _ := loader.Namespace()
	o.namespace = ns

	o.name = args[0]

	return nil
}

func (o *PodStatusOptions) Validate() error {
	return nil
}

func (o *PodStatusOptions) Run() error {
	var selector string

	getOpt := metav1.GetOptions{}

	podCli := o.clientSet.CoreV1().Pods(o.namespace)
	dplyCli := o.clientSet.AppsV1().Deployments(o.namespace)
	stsCli := o.clientSet.AppsV1().StatefulSets(o.namespace)
	dsCli := o.clientSet.AppsV1().DaemonSets(o.namespace)

	if obj, err := dplyCli.Get(o.name, getOpt); err == nil {
		fmt.Printf("Deployment: %s/%s\n", o.namespace, o.name)
		selector = labels.FormatLabels(obj.Spec.Selector.MatchLabels)
	} else if obj, err := stsCli.Get(o.name, getOpt); err == nil {
		fmt.Printf("StatefulSet: %s/%s\n", o.namespace, o.name)
		selector = labels.FormatLabels(obj.Spec.Selector.MatchLabels)
	} else if obj, err := dsCli.Get(o.name, getOpt); err == nil {
		fmt.Printf("DaemonSet: %s/%s\n", o.namespace, o.name)
		selector = labels.FormatLabels(obj.Spec.Selector.MatchLabels)
	} else {
		return fmt.Errorf("not found: %s/%s", o.namespace, o.name)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Ready", "Status", "Restarts", "IP", "Node"})

	lstOpt := metav1.ListOptions{
		LabelSelector: selector,
	}
	result, err := podCli.List(lstOpt)
	if err != nil {
		return err
	}
	for _, item := range result.Items {
		var (
			restarts        int32 = 0
			containersCount int   = len(item.Spec.Containers)
			readyCount      int
		)
		status := string(item.Status.Phase)
		for _, ctSta := range item.Status.ContainerStatuses {
			if ctSta.Ready {
				readyCount += 1
			} else if ctSta.State.Terminated != nil {
				status = ctSta.State.Terminated.Reason
			} else if ctSta.State.Waiting != nil {
				status = ctSta.State.Waiting.Reason
			}

			if restarts == 0 && ctSta.RestartCount != 0 {
				restarts = ctSta.RestartCount
			}
		}
		table.Append([]string{
			item.Name,
			fmt.Sprintf("%d/%d", readyCount, containersCount),
			status,
			fmt.Sprint(restarts),
			item.Status.HostIP,
			item.Spec.NodeName,
		})
	}
	table.Render()
	return nil
}
