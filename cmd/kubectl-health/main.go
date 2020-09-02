package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := &cobra.Command{
		Use:                   "kubectl health",
		Long:                  "Access the /healthz endpoint of the cluster",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := configFlags.ToRawKubeConfigLoader().RawConfig()
			if err != nil {
				return err
			}
			if config.Contexts == nil {
				return fmt.Errorf("empty context")
			}
			ctx, ok := config.Contexts[config.CurrentContext]
			if !ok {
				return fmt.Errorf("context not found: %s", config.CurrentContext)
			}
			cluster, ok := config.Clusters[ctx.Cluster]
			if !ok {
				return fmt.Errorf("cluster not found: %s", ctx.Cluster)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Context: %s\n", config.CurrentContext)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Server: %s\n", ctx.Cluster)

			caCert := x509.NewCertPool()
			caCert.AppendCertsFromPEM(cluster.CertificateAuthorityData)
			tlsConfig := &tls.Config{
				RootCAs: caCert,
			}
			client := http.Client{
				Timeout:   time.Second * 5,
				Transport: &http.Transport{TLSClientConfig: tlsConfig},
			}

			u, err := url.Parse(cluster.Server)
			if err != nil {
				return fmt.Errorf("parse server url: %w", err)
			}
			u.Path = "/healthz"
			u.RawQuery = "verbose"
			resp, err := client.Get(u.String())
			if err != nil {
				return err
			}
			var out io.Writer
			if resp.StatusCode != http.StatusOK {
				out = cmd.ErrOrStderr()
				_, _ = fmt.Fprintf(out, "ERROR: Unexpected status code: %d\n", resp.StatusCode)
			} else {
				out = cmd.OutOrStdout()
			}
			_, err = io.Copy(out, resp.Body)
			return err
		},
	}
	configFlags.AddFlags(cmd.Flags())

	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
