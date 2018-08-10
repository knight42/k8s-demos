package pkg

import (
	"flag"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func BuildConfig() (*rest.Config, error) {
	if os.Getenv("IN_CLUSTER") == "true" {
		return rest.InClusterConfig()
	}

	var (
		kubeconfig     *string
		currentContext *string
	)
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	currentContext = flag.String("context", "", "kube context")
	flag.Parse()

	if *currentContext == "" {
		return clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: *kubeconfig},
			&clientcmd.ConfigOverrides{
				CurrentContext: *currentContext,
			}).ClientConfig()
	}
}
