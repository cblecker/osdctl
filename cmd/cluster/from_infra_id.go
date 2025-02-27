package cluster

import (
	"encoding/json"
	"fmt"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/osdctl/internal/utils/globalflags"
	"github.com/openshift/osdctl/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"os"
	"strings"
)

type fromInfraIdOptions struct {
	globalOpts *globalflags.GlobalOptions
}

type clusterIdentification struct {
	ID         string `json:"id" yaml:"ID"`
	ExternalID string `json:"external_id" yaml:"ExternalID"`
	Name       string `json:"name" yaml:"Name"`
}

func (ci clusterIdentification) String() string {
	return fmt.Sprintf("ID:\t\t%s\nExternal ID:\t%s\nName:\t\t%s\n", ci.ID, ci.ExternalID, ci.Name)
}

func newCmdFromInfraId(globalOpts *globalflags.GlobalOptions) *cobra.Command {
	opts := &fromInfraIdOptions{
		globalOpts,
	}
	return &cobra.Command{
		Use:               "from-infra-id",
		Short:             "Get cluster ID and external ID from a given infrastructure ID commonly used by Splunk",
		Args:              cobra.ExactArgs(1),
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(opts.run(cmd, args))
		},
	}
}

func getClusterNameFromInfraId(infraId string) (string, error) {
	const separator = "-"
	clusterNameAndHash := strings.Split(infraId, separator)
	if len(clusterNameAndHash) < 2 {
		return "", fmt.Errorf("expected infrastructure ID format to be <name>-<hash> but got: %s", infraId)
	}
	return strings.Join(clusterNameAndHash[:len(clusterNameAndHash)-1], separator), nil
}

func (ops *fromInfraIdOptions) run(cmd *cobra.Command, args []string) error {
	ocmClient := utils.CreateConnection()
	defer func() {
		if err := ocmClient.Close(); err != nil {
			fmt.Printf("cannot close the ocmClient (possible memory leak): %q", err)
		}
	}()

	infraId := args[0]
	clusterName, err := getClusterNameFromInfraId(infraId)
	if err != nil {
		return err
	}
	clusters, err := utils.ApplyFilters(
		ocmClient,
		[]string{fmt.Sprintf("name like '%s%%'", clusterName)},
	)
	if err != nil {
		return fmt.Errorf("could not retrieve clusters for %s", infraId)
	}
	for _, cluster := range clusters {
		if cluster.InfraID() == infraId {
			return renderOutput(cluster, ops.globalOpts.Output)
		}
	}
	_, err = fmt.Fprintf(os.Stderr, "No clusters found matching %s\n", infraId)
	return err
}

func renderOutput(cluster *v1.Cluster, outputFormat string) error {
	ci := clusterIdentification{
		ID:         cluster.ID(),
		ExternalID: cluster.ExternalID(),
		Name:       cluster.Name(),
	}
	if outputFormat == "json" {
		jsonOutput, err := json.Marshal(ci)
		if err != nil {
			return fmt.Errorf("error marshaling cluster data to JSON %q", err)
		}
		fmt.Println(string(jsonOutput))
	} else if outputFormat == "yaml" {
		yamlOutput, err := yaml.Marshal(ci)
		if err != nil {
			return fmt.Errorf("error marshaling cluster data to YAML %q", err)
		}
		fmt.Println(string(yamlOutput))
	} else if outputFormat == "env" {
		fmt.Printf("CLUSTER_ID='%s'\nCLUSTER_EXTERNAL_ID='%s'\nCLUSTER_NAME='%s'\n", ci.ID, ci.ExternalID, ci.Name)
	} else {
		fmt.Println(ci)
	}
	return nil
}
