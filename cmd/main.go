package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/knight42/k8s-tools/pkg/diagnose"
	"github.com/knight42/k8s-tools/pkg/export"
	"github.com/ruguoapp/k8suite"
	"k8s.io/client-go/kubernetes"
)

var rootCmd *cobra.Command

func getClientset(kubeCfg, kubeCtx string) *kubernetes.Clientset {
	k8sCfg, err := k8suite.BuildConfig(kubeCfg, kubeCtx)
	if err != nil {
		log.Fatal(err)
	}
	return kubernetes.NewForConfigOrDie(k8sCfg)
}

func init() {
	rootCmd = &cobra.Command{
		Use:   "k8s",
		Short: "",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	var (
		kubeCfg   string
		kubeCtx   string
		namespace string
	)
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "namespace")
	rootCmd.PersistentFlags().StringVar(&kubeCfg, "conf", "", "kube config path")
	rootCmd.PersistentFlags().StringVar(&kubeCtx, "context", "", "current context")

	diagCmd := &cobra.Command{
		Use: "diagnose",
		Run: func(cmd *cobra.Command, args []string) {
			cliset := getClientset(kubeCfg, kubeCtx)
			diagnose.Run(cliset, namespace)
		},
	}

	exportCmd := &cobra.Command{
		Use:  "export",
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			selector, err := cmd.Flags().GetString("selector")
			if err != nil {
				log.Fatal(err)
			}
			cliset := getClientset(kubeCfg, kubeCtx)
			cfg := export.Config{
				Selector:  selector,
				Resouce:   args[0],
				Namespace: namespace,
			}
			switch len(args) {
			case 2:
				cfg.Name = args[1]
			}
			export.Run(cliset, cfg)
		},
	}
	exportCmd.Flags().StringP("selector", "l", "", "Selector (label query) to filter on")

	rootCmd.AddCommand(diagCmd)
	rootCmd.AddCommand(exportCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
