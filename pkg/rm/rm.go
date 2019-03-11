package rm

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/knight42/k8s-tools/pkg/scheme"
	"github.com/knight42/k8s-tools/pkg/utils"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
)

type RmOptions struct {
	configFlags *genericclioptions.ConfigFlags

	// Common user flags
	selector  string
	namespace string

	// results of arg parsing
	backupDir string
	args      []string
}

func NewRmOptions() *RmOptions {
	return &RmOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmd() *cobra.Command {
	o := NewRmOptions()

	cmd := &cobra.Command{
		Use: "kubectl rm TYPE [NAME | -l label] [flags]",
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(cmd, args))
			utils.CheckError(o.Validate())
			utils.CheckError(o.Run())
		},
	}
	cmd.Flags().StringVarP(&o.selector, "selector", "l", o.selector, "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2).")

	flags := cmd.PersistentFlags()
	o.configFlags.AddFlags(flags)

	return cmd
}

func (o *RmOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	ns, _, err := o.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	o.namespace = ns

	rawConfig, err := o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}
	clusterName := rawConfig.Contexts[rawConfig.CurrentContext].Cluster

	o.backupDir = path.Join(os.Getenv("HOME"), ".k8s-wastebin", clusterName, ns)
	if err := os.MkdirAll(o.backupDir, 0755); err != nil {
		return err
	}

	return nil
}

func (o *RmOptions) Validate() error {
	return nil
}

func (o *RmOptions) Run() error {
	allArgs := os.Args[1:]

	identifier := strings.Join(allArgs, "_")
	identifier = strings.ReplaceAll(identifier, " ", "_")
	fpath := path.Join(o.backupDir, fmt.Sprintf("%s_rm_%s.yaml", time.Now().Format(time.RFC3339), identifier))
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	yamlPrinter := printers.YAMLPrinter{}

	policy := metav1.DeletePropagationForeground
	delOpt := &metav1.DeleteOptions{
		PropagationPolicy: &policy,
	}

	r := resource.NewBuilder(o.configFlags).
		LabelSelector(o.selector).
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(o.namespace).DefaultNamespace().
		SingleResourceType().
		ResourceTypeOrNameArgs(false, o.args...).
		Latest().
		Flatten().
		Do()

	if err := r.Err(); err != nil {
		return err
	}

	var deletedInfos []*resource.Info

	err = r.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}

		_, err = resource.NewHelper(info.Client, info.Mapping).DeleteWithOptions(info.Namespace, info.Name, delOpt)
		if err != nil {
			return err
		}

		deletedInfos = append(deletedInfos, info)

		obj := info.Object
		obj.GetObjectKind().SetGroupVersionKind(info.Mapping.GroupVersionKind)
		err = yamlPrinter.PrintObj(obj, f)
		if err != nil {
			return err
		}

		_, _ = fmt.Fprintln(f, "---")
		return nil
	})

	if err != nil {
		return err
	}

	if len(deletedInfos) == 0 {
		return fmt.Errorf("not found")
	}

	for _, info := range deletedInfos {
		gvk := info.Mapping.GroupVersionKind
		kindStr := fmt.Sprintf("%s.%s", strings.ToLower(gvk.Kind), gvk.Group)
		fmt.Printf("%s `%s/%s` deleted\n", kindStr, info.Namespace, info.Name)
	}

	return nil
}
