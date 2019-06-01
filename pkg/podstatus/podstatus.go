// Podstatus print status of pods of Deployment/StatefulSet/DaemonSet
// Great thanks to `https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/get/get.go`
package podstatus

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/knight42/k8s-tools/pkg/tabwriter"
	"github.com/knight42/k8s-tools/pkg/utils"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	OwnersAnnotationKey                 = "jike.app/owners"
	LastUserAnnotationKey               = "jike.app/last-user"
	LastOperationAnnotationKey          = "jike.app/last-operation"
	LastOperationTimestampAnnotationKey = "jike.app/last-operation-ts"
)

const (
	KindPod         = "Pod"
	KindDeployment  = "Deployment"
	KindStatefulSet = "StatefulSet"
	KindDaemonSet   = "DaemonSet"
)

var (
	errNotFound    = fmt.Errorf("not found")
	errUnknwonKind = fmt.Errorf("unknown kind")
)

type PodStatusOptions struct {
	configFlags *genericclioptions.ConfigFlags

	namespace     string
	watch         bool
	watchOnly     bool
	labelSelector string

	args             []string
	writer           *tabwriter.Writer
	enforceNamespace bool
	enforceResource  bool
}

func NewPodStatusOptions() *PodStatusOptions {
	return &PodStatusOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmd() *cobra.Command {
	o := NewPodStatusOptions()
	cmd := &cobra.Command{
		Use: "kubectl podstatus [NAME | -l label] [flags]",
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(cmd, args))
			utils.CheckError(o.Validate())
			utils.CheckError(o.Run())
		},
	}
	cmd.Flags().BoolVarP(&o.watch, "watch", "w", false, "After listing/getting the requested object, watch for changes.")
	cmd.Flags().StringVarP(&o.labelSelector, "selector", "l", o.labelSelector, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	cmd.Flags().BoolVar(&o.watchOnly, "watch-only", o.watchOnly, "Watch for changes to the requested object(s), without listing/getting first.")

	flags := cmd.PersistentFlags()
	o.configFlags.AddFlags(flags)

	return cmd
}

func (o *PodStatusOptions) newBuilder() *resource.Builder {
	return resource.NewBuilder(o.configFlags).
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(o.namespace).DefaultNamespace().
		Latest()
}

func (o *PodStatusOptions) Complete(cmd *cobra.Command, args []string) error {
	var err error

	o.namespace, o.enforceNamespace, err = o.configFlags.
		ToRawKubeConfigLoader().
		Namespace()
	if err != nil {
		return err
	}

	o.args = args
	if len(args) > 0 {
		name := args[0]
		if strings.ContainsRune(name, '/') || len(args) == 2 {
			o.enforceResource = true
		}
	}

	o.writer = tabwriter.New(os.Stdout)
	o.writer.SetHeader([]string{"name", "ready", "status", "last status", "restarts", "podip", "hostip", "node", "age"})

	return nil
}

func (o *PodStatusOptions) Validate() error {
	if len(o.labelSelector) == 0 && len(o.args) == 0 {
		return fmt.Errorf("must specify label selector or name")
	}
	if len(o.labelSelector) != 0 && len(o.args) != 0 {
		return fmt.Errorf("cannot use label selector and name at the same time")
	}
	return nil
}

// See also https://github.com/kubernetes/kubernetes/blob/master/pkg/printers/internalversion/printers.go#L579
func (o *PodStatusOptions) PrintPod(obj runtime.Object, needFlush bool) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("object is not a Pod: %#v", obj)
	}
	var (
		readyCount int
		totalCount int   = len(pod.Spec.Containers)
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
	if needFlush {
		_ = o.writer.AppendAndFlush(args...)
	} else {
		o.writer.Append(args...)
	}
	return nil
}

func (o *PodStatusOptions) Run() error {
	var (
		r        *resource.Result
		err      error
		selector string
	)

	if len(o.labelSelector) != 0 {
		selector = o.labelSelector
	} else {
		if o.enforceResource {
			r = o.newBuilder().
				SingleResourceType().
				ResourceTypeOrNameArgs(false, o.args...).
				Flatten().
				Do()
		} else {
			name := o.args[0]
			r = o.newBuilder().
				ContinueOnError().
				ResourceTypeOrNameArgs(false, "deploy/"+name, "sts/"+name, "ds/"+name).
				Flatten().
				Do().
				IgnoreErrors(errors.IsNotFound)
		}

		if err = r.Err(); err != nil {
			return err
		}

		infos, err := r.Infos()
		if err != nil {
			return err
		}
		if len(infos) == 0 {
			return errNotFound
		}

		info := infos[0]
		obj := info.Object
		mapping := info.ResourceMapping()
		switch mapping.GroupVersionKind.Kind {
		case KindDeployment:
			var antns map[string]string
			metaobj := obj.(metav1.Object)
			selector = getSelectorFromObject(obj)
			antns = metaobj.GetAnnotations()
			fmt.Printf("Deployment: %s/%s\n", info.Namespace, info.Name)
			if val, ok := antns[OwnersAnnotationKey]; ok {
				fmt.Printf("Owners: %s\n", val)
			}
			if val, ok := antns[LastUserAnnotationKey]; ok {
				fmt.Printf("Last User: %s\n", val)
			}
			if val, ok := antns[LastOperationAnnotationKey]; ok {
				fmt.Printf("Last Operation: %s\n", val)
			}
			if val, ok := antns[LastOperationTimestampAnnotationKey]; ok {
				fmt.Printf("Last Timestamp: %s\n", val)
			}
		case KindStatefulSet:
			selector = getSelectorFromObject(obj)
			fmt.Printf("StatefulSet: %s/%s\n", info.Namespace, info.Name)
		case KindDaemonSet:
			selector = getSelectorFromObject(obj)
			fmt.Printf("DaemonSet: %s/%s\n", info.Namespace, info.Name)
		case KindPod:
			return o.handleSinglePod(info)
		default:
			return errUnknwonKind
		}
	}

	o.labelSelector = selector
	fmt.Printf("Selector: -l%s\n\n", selector)

	if o.watch || o.watchOnly {
		return o.watchPods()
	}

	r = o.newBuilder().
		LabelSelector(selector).
		ResourceTypes("pods").
		Flatten().
		Do()

	if err := r.Err(); err != nil {
		return err
	}

	err = r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}

		return o.PrintPod(info.Object, false)
	})
	_ = o.writer.Render()

	return err
}

func (o *PodStatusOptions) handleSinglePod(info *resource.Info) error {
	obj := info.Object
	metaobj := obj.(metav1.Object)
	podEvtSelector := fields.AndSelectors(
		fields.OneTermEqualSelector("involvedObject.name", info.Name),
		fields.OneTermEqualSelector("involvedObject.namespace", info.Namespace),
		fields.OneTermEqualSelector("involvedObject.uid", string(metaobj.GetUID())),
	)
	r := o.newBuilder().
		FieldSelectorParam(podEvtSelector.String()).
		SingleResourceType().
		ResourceTypes("events").
		Flatten().
		Do()
	if err := r.Err(); err != nil {
		return err
	}
	evtPrinter := tabwriter.New(os.Stdout)
	evtPrinter.SetHeader([]string{"Type", "Reason", "Age", "From", "Message"})
	err := r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		evt, ok := info.Object.(*corev1.Event)
		if !ok {
			return fmt.Errorf("not event: %s", info.Object)
		}
		age := fmt.Sprintf(
			"%s (x%d over %s)",
			duration.ShortHumanDuration(time.Since(evt.LastTimestamp.Time)),
			evt.Count,
			duration.ShortHumanDuration(evt.LastTimestamp.Sub(evt.FirstTimestamp.Time)),
		)
		from := fmt.Sprintf("%s, %s", evt.Source.Component, evt.Source.Host)
		evtPrinter.Append(evt.Type, evt.Reason, age, from, evt.Message)
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("Events:\n")
	_ = evtPrinter.Render()
	fmt.Println()
	fmt.Printf("Pod: %s/%s\n", info.Namespace, info.Name)

	if o.watch || o.watchOnly {
		return fmt.Errorf("watching single pod is not supported now")
	}
	_ = o.PrintPod(info.Object, false)
	_ = o.writer.Render()
	return nil
}

func (o *PodStatusOptions) watchPods() error {
	r := o.newBuilder().
		SingleResourceType().
		LabelSelector(o.labelSelector).
		ResourceTypes("pods").
		Do()

	if err := r.Err(); err != nil {
		return err
	}

	obj, err := r.Object()
	if err != nil {
		return err
	}

	rv, err := meta.NewAccessor().ResourceVersion(obj)
	if err != nil {
		return err
	}

	if !o.watchOnly {
		objsToPrint, _ := meta.ExtractList(obj)
		for _, objToPrint := range objsToPrint {
			_ = o.PrintPod(objToPrint, false)
		}
		_ = o.writer.Render()
	}

	intf, err := r.Watch(rv)
	if err != nil {
		return err
	}

	defer intf.Stop()
	evChan := intf.ResultChan()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer signal.Stop(sigChan)

	for {
		select {
		case ev := <-evChan:
			err := o.PrintPod(ev.Object, true)
			if err != nil {
				return nil
			}
		case <-sigChan:
			return nil
		}
	}
}
