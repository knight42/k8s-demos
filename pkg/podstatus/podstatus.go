// Podstatus print status of pods of Deployment/StatefulSet/DaemonSet
// Great thanks to `https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/get/get.go`
package podstatus

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/knight42/k8s-tools/pkg/tabwriter"
	"github.com/knight42/k8s-tools/pkg/utils"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

type Options struct {
	configFlags *genericclioptions.ConfigFlags

	namespace     string
	watch         bool
	watchOnly     bool
	labelSelector string

	args            []string
	writer          *tabwriter.Writer
	enforceResource bool
	namePattern     *regexp.Regexp
	pods            map[string]*corev1.Pod
}

func NewOptions() *Options {
	return &Options{
		configFlags: genericclioptions.NewConfigFlags(true),
		pods:        make(map[string]*corev1.Pod),
	}
}

func NewCmd() *cobra.Command {
	o := NewOptions()
	cmd := &cobra.Command{
		Use: "kubectl pods [NAME | -l label] [flags]",
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(cmd, args))
			utils.CheckError(o.Validate())
			utils.CheckError(o.Run())
		},
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&o.watch, "watch", "w", false, "After listing/getting the requested object, watch for changes.")
	flags.StringVarP(&o.labelSelector, "selector", "l", o.labelSelector, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	flags.BoolVar(&o.watchOnly, "watch-only", o.watchOnly, "Watch for changes to the requested object(s), without listing/getting first.")

	o.configFlags.AddFlags(cmd.PersistentFlags())

	return cmd
}

func (o *Options) Complete(cmd *cobra.Command, args []string) error {
	var err error

	o.namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	o.args = args
	switch len(args) {
	case 2:
		o.enforceResource = true
	case 1:
		o.namePattern, err = regexp.Compile(args[0])
		if err != nil {
			return err
		}
	}

	o.writer = tabwriter.New(os.Stdout)
	o.writer.SetHeader([]string{"name", "ready", "status", "last status", "restarts", "podip", "hostip", "node", "age"})

	return nil
}

func (o *Options) Validate() error {
	if len(o.labelSelector) == 0 && len(o.args) == 0 {
		return fmt.Errorf("must specify label selector or name")
	}
	if len(o.labelSelector) != 0 && len(o.args) != 0 {
		return fmt.Errorf("cannot use label selector and name at the same time")
	}
	return nil
}

func (o *Options) Run() error {
	var (
		r        *resource.Result
		err      error
		selector string
	)

	if len(o.labelSelector) != 0 {
		selector = o.labelSelector
	} else if o.enforceResource {
		r = newBuilder(o.configFlags).
			NamespaceParam(o.namespace).DefaultNamespace().
			SingleResourceType().
			ResourceTypeOrNameArgs(false, o.args...).
			Do()

		if err = r.Err(); err != nil {
			return err
		}

		obj, err := r.Object()
		if err != nil {
			return err
		}

		if isHPA(obj) {
			obj, err = getRefObject(obj, o.configFlags)
			if err != nil {
				return err
			}
		} else if isPod(obj) {
			pod := obj.(*corev1.Pod)
			return o.handleSinglePod(pod)
		}
		selector, err = getSelectorFromObject(obj)
		if err != nil {
			return err
		}
	} else {
		panic(fmt.Errorf("TODO"))
	}

	o.labelSelector = selector
	fmt.Printf("Selector: -l%s\n\n", selector)

	if o.watch || o.watchOnly {
		return o.watchPods()
	}

	r = newBuilder(o.configFlags).
		NamespaceParam(o.namespace).DefaultNamespace().
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

func (o *Options) handleSinglePod(pod *corev1.Pod) error {
	podEvtSelector := fields.AndSelectors(
		fields.OneTermEqualSelector("involvedObject.name", pod.Name),
		fields.OneTermEqualSelector("involvedObject.namespace", pod.Namespace),
		fields.OneTermEqualSelector("involvedObject.uid", string(pod.UID)),
	)
	r := newBuilder(o.configFlags).
		NamespaceParam(o.namespace).DefaultNamespace().
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
	fmt.Printf("Pod: %s/%s\n", pod.Namespace, pod.Name)

	if o.watch || o.watchOnly {
		return fmt.Errorf("watching single pod is not supported now")
	}
	_ = o.PrintPod(pod, false)
	_ = o.writer.Render()
	return nil
}

func (o *Options) watchPods() error {
	r := newBuilder(o.configFlags).
		NamespaceParam(o.namespace).DefaultNamespace().
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
			pod, ok := objToPrint.(*corev1.Pod)
			if !ok {
				continue
			}
			o.pods[string(pod.UID)] = pod
			_ = o.PrintPod(objToPrint, false)
		}
		_ = o.writer.Render()
	}

	watcher, err := r.Watch(rv)
	if err != nil {
		return err
	}

	defer watcher.Stop()

	for ev := range watcher.ResultChan() {
		pod, ok := ev.Object.(*corev1.Pod)
		if !ok {
			continue
		}
		n := len(o.pods) + 1
		for n > 0 {
			cursorUp(os.Stdout, 1)
			clearLine(os.Stdout)
			n--
		}
		switch ev.Type {
		case watch.Added:
			o.pods[string(pod.UID)] = pod
		case watch.Modified:
			o.pods[string(pod.UID)] = pod
		case watch.Deleted:
			delete(o.pods, string(pod.UID))
		}
		o.printPods()
	}
	return nil
}
