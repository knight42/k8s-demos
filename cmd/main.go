package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/knight42/k8s-tools/pkg/diagnose"
	"github.com/ruguoapp/k8suite"
	"k8s.io/client-go/kubernetes"
)

var rootCmd *cobra.Command

func init() {
	rootCmd = &cobra.Command{
		Use:   "k8s",
		Short: "help",
		Long:  `Long Help`,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	var (
		kubeCfg   string
		kubeCtx   string
		namespace string
	)
	rootCmd.PersistentFlags().StringVar(&namespace, "ns", "", "namespace")
	rootCmd.PersistentFlags().StringVar(&kubeCfg, "conf", "", "kube config path")
	rootCmd.PersistentFlags().StringVar(&kubeCtx, "context", "", "current context")

	diagCmd := &cobra.Command{
		Use: "diagnose",
		Run: func(cmd *cobra.Command, args []string) {
			k8sCfg, err := k8suite.BuildConfig(kubeCfg, kubeCtx)
			if err != nil {
				log.Fatal(err)
			}
			cliset := kubernetes.NewForConfigOrDie(k8sCfg)
			diagnose.Run(cliset, namespace)
		},
	}

	rootCmd.AddCommand(diagCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
