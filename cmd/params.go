package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gocruncher/jenkins-job-cli/cmd/jj"
	"github.com/spf13/cobra"
)

func init() {
	var paramsCmd = &cobra.Command{
		Use:   "params JOB",
		Short: "Display build parameters for a Jenkins job",
		RunE: func(cmd *cobra.Command, args []string) error {
			env := jj.Init(ENV)
			jobName := args[0]

			err, jobInfo := jj.GetJobInfo(env, jobName)
			if err == jj.ErrNoJob {
				return fmt.Errorf("job '%s' does not exist", jobName)
			}
			if err != nil {
				return err
			}

			params := jobInfo.GetParameterDefinitions(env, jobName)
			waitForGitParameterChoices(params)
			if len(params) == 0 {
				fmt.Printf("job '%s' has no build parameters\n", jobName)
				return nil
			}

			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 0, 8, 0, '\t', 0)
			fmt.Fprintf(w, "%s\t%s\t%s\n", "Name", "Type", "Choices")
			for _, pd := range params {
				fmt.Fprintf(w, "%s\t%s\t%s\n", pd.Name, pd.Type, formatParameterChoices(pd.Choices))
			}
			fmt.Fprintln(w)
			w.Flush()
			return nil
		},
		Args:    cobra.ExactArgs(1),
		PreRunE: preRunE,
	}

	paramsCmd.Flags().StringVarP(&ENV, "name", "n", "", "current Jenkins env name")
	rootCmd.AddCommand(paramsCmd)
}

func waitForGitParameterChoices(params []*jj.ParameterDefinitions) {
	for i := 0; i < 5; i++ {
		pending := false
		for _, pd := range params {
			if pd.Type == "GitParameterDefinition" && len(pd.Choices) == 0 {
				pending = true
				break
			}
		}
		if !pending {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func formatParameterChoices(choices []string) string {
	if len(choices) == 0 {
		return "-"
	}
	if len(choices) <= 5 {
		return strings.Join(choices, ", ")
	}
	return fmt.Sprintf("%s ...(+%d)", strings.Join(choices[:5], ", "), len(choices)-5)
}
