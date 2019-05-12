package roles

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/knight42/k8s-tools/pkg/utils"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

type RolesOptions struct {
	configFlags *genericclioptions.ConfigFlags

	clientset *kubernetes.Clientset

	// results of arg parsing
	subjects []rbacv1.Subject
}

func NewRolesOptions() *RolesOptions {
	return &RolesOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmd() *cobra.Command {
	o := NewRolesOptions()

	cmd := &cobra.Command{
		Use: "kubectl roles [user:<user> | sa:<serviceaccount> | group:<group> ] [flags]",
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(cmd, args))
			utils.CheckError(o.Validate())
			utils.CheckError(o.Run())
		},
	}
	flags := cmd.PersistentFlags()
	o.configFlags.AddFlags(flags)

	return cmd
}

func (o *RolesOptions) Complete(cmd *cobra.Command, args []string) error {
	ns, _, err := o.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	for _, arg := range args {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) < 2 {
			return fmt.Errorf("invalid subject: %s", arg)
		}

		var sub rbacv1.Subject
		switch {
		case strings.HasPrefix(arg, "user:"):
			sub.Kind = rbacv1.UserKind
			sub.Name = parts[1]

		case strings.HasPrefix(arg, "group:"):
			sub.Kind = rbacv1.GroupKind
			sub.Name = parts[1]

		case strings.HasPrefix(arg, "serviceaccount:"):
			fallthrough
		case strings.HasPrefix(arg, "sa:"):
			sub.Kind = rbacv1.ServiceAccountKind
			sub.Namespace = ns
			sub.Name = parts[1]
		default:
			return fmt.Errorf("unknown subject: %s", arg)
		}
		o.subjects = append(o.subjects, sub)
	}

	restCfg, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	o.clientset, err = kubernetes.NewForConfig(restCfg)
	if err != nil {
		return err
	}

	return nil
}

func (o *RolesOptions) Validate() error {
	return nil
}

func (o *RolesOptions) Run() error {
	bindings, err := o.clientset.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	clusterRoleBindings := bindings.Items

	nsps, err := o.clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	namespaces := make([]string, len(nsps.Items))
	for i, nsp := range nsps.Items {
		namespaces[i] = nsp.Name
	}

	for _, sub := range o.subjects {
		fmt.Printf("%s:%s\n", sub.Kind, sub.Name)
		fmt.Println("===========")

		fmt.Println("ClusterRole:")
		for _, crBinding := range clusterRoleBindings {
			for _, subject := range crBinding.Subjects {
				if subject.Name == sub.Name &&
					subject.Kind == sub.Kind &&
					subject.Namespace == sub.Namespace {

					fmt.Printf("%s (binding: %s)\n", crBinding.RoleRef.Name, crBinding.Name)
				}
			}
		}
		fmt.Println()

		fmt.Println("Role:")
		if sub.Kind == rbacv1.ServiceAccountKind {
			bindings, err := o.clientset.RbacV1().RoleBindings(sub.Namespace).List(metav1.ListOptions{})
			if err != nil {
				return err
			}

			for _, binding := range bindings.Items {
				for _, subject := range binding.Subjects {
					if subject.Name == sub.Name &&
						subject.Kind == sub.Kind &&
						subject.Namespace == sub.Namespace {
						fmt.Printf("%s/%s (binding: %s)\n", sub.Namespace, binding.RoleRef.Name, binding.Name)
					}
				}
			}
		} else {
			for _, namespace := range namespaces {
				bindings, err := o.clientset.RbacV1().RoleBindings(namespace).List(metav1.ListOptions{})
				if err != nil {
					return err
				}

				for _, binding := range bindings.Items {
					for _, subject := range binding.Subjects {
						if subject.Name == sub.Name &&
							subject.Kind == sub.Kind &&
							subject.Namespace == sub.Namespace {
							fmt.Printf("%s/%s (binding: %s)\n", namespace, binding.RoleRef.Name, binding.Name)
						}
					}
				}
			}
		}

		fmt.Println()
	}
	return nil
}
