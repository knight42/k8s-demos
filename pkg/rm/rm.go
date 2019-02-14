package rm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/knight42/k8s-tools/pkg"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func printYaml(w io.Writer, data []byte) error {
	v := map[string]interface{}{}
	_ = json.Unmarshal(data, &v)
	encoder := yaml.NewEncoder(w)
	return encoder.Encode(v)
}

func saveObject(backupDir string, identifier string, v interface{}) error {
	fpath := path.Join(backupDir, fmt.Sprintf("%s_%s.yaml", identifier, time.Now().Format(time.RFC3339)))
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, _ := json.Marshal(v)
	return printYaml(f, data)
}

type RmOptions struct {
	configFlags *genericclioptions.ConfigFlags

	// Common user flags
	selector string

	// results of arg parsing
	namespace string
	backupDir string
	name      string
	kind      pkg.Kind
	clientSet *kubernetes.Clientset
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
			pkg.CheckError(o.Complete(cmd, args))
			pkg.CheckError(o.Validate())
			pkg.CheckError(o.Run())
		},
	}
	cmd.Flags().StringVarP(&o.selector, "selector", "l", o.selector, "Selector (label query) to filter on, not including uninitialized ones, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2).")

	flags := cmd.PersistentFlags()
	o.configFlags.AddFlags(flags)

	return cmd
}

func (o *RmOptions) Complete(cmd *cobra.Command, args []string) error {
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

	var (
		kind string
		name string
	)
	switch len(args) {
	case 1:
		kind = args[0]
	case 2:
		kind, name = args[0], args[1]
	default:
		return fmt.Errorf("too much args")
	}
	canonicalKind, valid := pkg.CanonicalizeKind(kind)
	if !valid {
		return fmt.Errorf("unknown object: %s", kind)
	}
	o.kind = canonicalKind
	o.name = name

	rawConfig, err := o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}
	clusterName := rawConfig.Contexts[rawConfig.CurrentContext].Cluster
	o.backupDir = path.Join(os.Getenv("HOME"), ".k8s-wastebin", clusterName, ns, o.kind.String())

	return nil
}

func (o *RmOptions) Validate() error {
	if len(o.selector) != 0 && len(o.name) != 0 {
		return fmt.Errorf("cannot specify selector and name at the same time")
	}
	return nil
}

func (o *RmOptions) Run() error {
	delProp := metav1.DeletePropagationForeground
	delOpt := &metav1.DeleteOptions{
		PropagationPolicy: &delProp,
	}

	lstOpt := metav1.ListOptions{
		LabelSelector: o.selector,
	}

	getOpt := metav1.GetOptions{}

	backupDir := o.backupDir
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	dplyCli := o.clientSet.AppsV1().Deployments(o.namespace)
	cjCli := o.clientSet.BatchV1beta1().CronJobs(o.namespace)
	cmCli := o.clientSet.CoreV1().ConfigMaps(o.namespace)
	svcCli := o.clientSet.CoreV1().Services(o.namespace)
	stsCli := o.clientSet.AppsV1().StatefulSets(o.namespace)
	dsCli := o.clientSet.AppsV1().DaemonSets(o.namespace)

	switch o.kind {
	case pkg.Deployments:
		if len(o.name) != 0 {
			item, err := dplyCli.Get(o.name, getOpt)
			if err != nil {
				return err
			}
			item.APIVersion = "apps/v1"
			item.Kind = "Deployment"
			if err := saveObject(backupDir, o.name, item); err != nil {
				return err
			}
			return dplyCli.Delete(o.name, delOpt)
		}

		result, err := dplyCli.List(lstOpt)
		if err != nil {
			return err
		}
		if len(result.Items) == 0 {
			return fmt.Errorf("No resources found.")
		}

		result.APIVersion = "v1"
		result.Kind = "List"
		if err := saveObject(backupDir, o.selector, result); err != nil {
			return err
		}
		return dplyCli.DeleteCollection(delOpt, lstOpt)

	case pkg.ConfigMaps:
		if len(o.name) != 0 {
			item, err := cmCli.Get(o.name, getOpt)
			if err != nil {
				return err
			}
			item.APIVersion = "v1"
			item.Kind = "ConfigMap"
			if err := saveObject(backupDir, o.name, item); err != nil {
				return err
			}
			return cmCli.Delete(o.name, delOpt)
		}

		result, err := cmCli.List(lstOpt)
		if err != nil {
			return err
		}
		if len(result.Items) == 0 {
			return fmt.Errorf("No resources found.")
		}

		result.APIVersion = "v1"
		result.Kind = "List"
		if err := saveObject(backupDir, o.selector, result); err != nil {
			return err
		}
		return cmCli.DeleteCollection(delOpt, lstOpt)

	case pkg.Services:
		if len(o.name) != 0 {
			item, err := svcCli.Get(o.name, getOpt)
			if err != nil {
				return err
			}
			item.APIVersion = "v1"
			item.Kind = "Service"
			if err := saveObject(backupDir, o.name, item); err != nil {
				return err
			}
			return svcCli.Delete(o.name, delOpt)
		}
		return fmt.Errorf("cannot delete services using selector\nhttps://github.com/kubernetes/kubernetes/issues/68468#issuecomment-419981870")

	case pkg.DaemonSets:
		if len(o.name) != 0 {
			item, err := dsCli.Get(o.name, getOpt)
			if err != nil {
				return err
			}
			item.APIVersion = "apps/v1"
			item.Kind = "DaemonSets"
			if err := saveObject(backupDir, o.name, item); err != nil {
				return err
			}
			return dsCli.Delete(o.name, delOpt)
		}

		result, err := dsCli.List(lstOpt)
		if err != nil {
			return err
		}
		if len(result.Items) == 0 {
			return fmt.Errorf("No resources found.")
		}

		result.APIVersion = "v1"
		result.Kind = "List"
		if err := saveObject(backupDir, o.selector, result); err != nil {
			return err
		}
		return dsCli.DeleteCollection(delOpt, lstOpt)

	case pkg.StatefulSets:
		if len(o.name) != 0 {
			item, err := stsCli.Get(o.name, getOpt)
			if err != nil {
				return err
			}
			item.APIVersion = "apps/v1"
			item.Kind = "Deployment"
			if err := saveObject(backupDir, o.name, item); err != nil {
				return err
			}
			return stsCli.Delete(o.name, delOpt)
		}

		result, err := stsCli.List(lstOpt)
		if err != nil {
			return err
		}
		if len(result.Items) == 0 {
			return fmt.Errorf("No resources found.")
		}

		result.APIVersion = "v1"
		result.Kind = "List"
		if err := saveObject(backupDir, o.selector, result); err != nil {
			return err
		}
		return stsCli.DeleteCollection(delOpt, lstOpt)

	case pkg.Cronjobs:
		if len(o.name) != 0 {
			item, err := cjCli.Get(o.name, getOpt)
			if err != nil {
				return err
			}
			item.APIVersion = "apps/v1"
			item.Kind = "Deployment"
			if err := saveObject(backupDir, o.name, item); err != nil {
				return err
			}
			return cjCli.Delete(o.name, delOpt)
		}

		result, err := cjCli.List(lstOpt)
		if err != nil {
			return err
		}
		if len(result.Items) == 0 {
			return fmt.Errorf("No resources found.")
		}

		result.APIVersion = "v1"
		result.Kind = "List"
		if err := saveObject(backupDir, o.selector, result); err != nil {
			return err
		}
		return cjCli.DeleteCollection(delOpt, lstOpt)

	default:
		return fmt.Errorf("unknown object: %s", o.kind)
	}
}
