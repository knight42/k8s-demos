package podstatus

import (
	"fmt"

	"github.com/knight42/k8s-tools/pkg"
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
		selector = labels.FormatLabels(obj.Spec.Selector.MatchLabels)
	} else if obj, err := stsCli.Get(o.name, getOpt); err == nil {
		selector = labels.FormatLabels(obj.Spec.Selector.MatchLabels)
	} else if obj, err := dsCli.Get(o.name, getOpt); err == nil {
		selector = labels.FormatLabels(obj.Spec.Selector.MatchLabels)
	} else {
		return fmt.Errorf("not found: %s", o.name)
	}

	lstOpt := metav1.ListOptions{
		LabelSelector: selector,
	}
	result, err := podCli.List(lstOpt)
	if err != nil {
		return err
	}
	for _, item := range result.Items {
		var restart int32 = 0
		status := string(item.Status.Phase)
		for _, ctSta := range item.Status.ContainerStatuses {
			if restart != 0 && ctSta.RestartCount != 0 {
				restart = ctSta.RestartCount
			}
			if ctSta.State.Terminated != nil {
				status = ctSta.State.Terminated.Reason
			} else if ctSta.State.Waiting != nil {
				status = ctSta.State.Waiting.Reason
			}
		}
		fmt.Printf("name=%s status=%s restart=%d hostIP=%s nodeName=%s\n", item.Name, status, restart, item.Status.HostIP, item.Spec.NodeName)
	}
	return nil
}
