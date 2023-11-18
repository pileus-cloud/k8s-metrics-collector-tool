/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

func installPrometheus(namespace string) error {
	cmd_version := exec.Command("helm",  "version")
	// Capture the command's standard output and standard error
	output, err := cmd_version.CombinedOutput()

	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	fmt.Println(string(output))

	cmd_add := exec.Command("helm", "repo", "add", "--force-update", "prometheus-community", "https://prometheus-community.github.io/helm-charts")
	output_add, _ := cmd_add.CombinedOutput()

	fmt.Println(string(output_add))

	cmd_install := exec.Command("helm", "upgrade", "--install", "--create-namespace", "-n", namespace, "kube-prometheus-stack", "prometheus-community/kube-prometheus-stack")
	output_install, err_install := cmd_install.CombinedOutput()

	fmt.Println(string(output_install))
	if err_install != nil {
		return err_install
	}

	return err
}


// installpromCmd represents the installprom command
var installpromCmd = &cobra.Command{
	Use:   "installprom",
	Short: "Install kube-prometheus-stack",
	Long: `Install kube-prometheus-stack`,
	RunE: func(cmd *cobra.Command, args []string) error {
		namespace, _ := cmd.Flags().GetString("namespace")
		return installPrometheus(namespace)
	},
}

func init() {
	rootCmd.AddCommand(installpromCmd)
	installpromCmd.Flags().String("namespace", "anodot-k8s-metrics-collector", "k8s namespace for installing")
}
