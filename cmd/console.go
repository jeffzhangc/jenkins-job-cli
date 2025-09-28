/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocruncher/jenkins-job-cli/cmd/jj"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
)

// consoleCmd represents the console command
// 1. list all running jobs
// 2. display latest console output of a job

func init() {
	var (
		noheader bool
		ENV      string
	)
	var consoleCmd = &cobra.Command{
		Use:   "console [job]",
		Short: "Display running jobs or latest job console output",
		RunE: func(cmd *cobra.Command, args []string) error {
			env := jj.Init(ENV)
			// 列出正在运行的 jobs
			items, err := jj.GetRunningBuildsByComputer(env)
			if len(args) == 0 {
				if err != nil {
					return fmt.Errorf("get running builds failed: %v", err)
				}
				PrintRunningBuildsSorted(items)
				return nil
			}
			// 显示某个 job 最新的 console 日志
			jobName := args[0]

			err, jobInfo := jj.GetJobInfo(env, jobName)
			if err != nil {
				return fmt.Errorf("get job info failed: %v", err)
			}

			runningFlag := false
			var runningBuild *jj.RunningBuild = nil
			for _, item := range items {
				if item.JobName == jobName {
					runningFlag = true
					runningBuild = &item
					break
				}
			}

			buildNum := jobInfo.LastBuild.Number
			if runningFlag {
				buildNum = runningBuild.BuildNum
			}
			if buildNum == 0 {
				return fmt.Errorf("job '%s' has no build yet", jobName)
			}
			output, _, err2 := jj.Console(env, jobName, buildNum, "0")
			if err2 != nil {
				return fmt.Errorf("get console output failed: %v", err2)
			}
			fmt.Println(strings.TrimRight(output, "\n"))
			jobUrl := jj.GetConsoleUrl(env, jobName, buildNum)
			fmt.Println("")

			msg := fmt.Sprintf("Job Name: %s  Build #: %d  Running Status: %s",
				jobName, buildNum, chalk.Red.Color(fmt.Sprintf("%v", runningFlag)))
			if runningFlag && runningBuild != nil {
				msg += fmt.Sprintf("  Duration: %s", formatDuration(runningBuild.Timestamp))
			}
			fmt.Println(msg)
			fmt.Printf("Console URL: %s\n", chalk.Underline.TextStyle(jobUrl))

			if !runningFlag {
				printQuickRunJobByID(env, jobName, buildNum)
			}
			return nil
		},
		PreRunE: preRunE,
	}

	consoleCmd.Flags().BoolVar(&noheader, "no-headers", false, "no-headers")
	consoleCmd.Flags().StringVarP(&ENV, "name", "n", "", "current Jenkins name")

	rootCmd.AddCommand(consoleCmd)
}

// 优化打印函数，缓存英文输出，所有列宽对齐
var runningBuildsCache string

func PrintRunningBuilds(items []jj.RunningBuild) {
	if len(items) == 0 {
		runningBuildsCache = "No running builds\n"
		fmt.Print(runningBuildsCache)
		return
	}

	// 计算合适的列宽
	maxJobName := len("Job Name")
	maxResult := len("Status")
	maxURL := len("URL")
	for _, item := range items {
		if len(item.JobName) > maxJobName {
			maxJobName = len(item.JobName)
		}
		if len(item.Result) > maxResult {
			maxResult = len(item.Result)
		}
		if len(item.URL) > maxURL {
			maxURL = len(item.URL)
		}
	}
	if maxJobName > 40 {
		maxJobName = 40
	}
	if maxResult > 20 {
		maxResult = 20
	}
	if maxURL > 60 {
		maxURL = 60
	}

	// 构造格式化字符串
	format := fmt.Sprintf("%%-%ds  #%%-6s  %%-%ds  %%-%ds  %%-%ds\n",
		maxJobName, maxResult, 12, maxURL)
	sep := strings.Repeat("-", maxJobName+8+maxResult+14+maxURL+10) + "\n"

	var sb strings.Builder
	sb.WriteString(sep)
	sb.WriteString(fmt.Sprintf(format, "Job Name", "Build", "Status", "Duration", "URL"))
	sb.WriteString(sep)

	for _, item := range items {
		jobName := item.JobName
		if len(jobName) > maxJobName {
			jobName = jobName[:maxJobName-3] + "..."
		}
		result := item.Result
		if result == "" {
			result = "RUNNING"
		}
		if len(result) > maxResult {
			result = result[:maxResult-3] + "..."
		}
		durationStr := formatDuration(item.Timestamp)
		urlDisplay := item.URL
		if len(urlDisplay) > maxURL {
			urlDisplay = urlDisplay[:maxURL-3] + "..."
		}
		sb.WriteString(fmt.Sprintf(format, jobName, fmt.Sprintf("%d", item.BuildNum), result, durationStr, urlDisplay+"console"))
	}
	sb.WriteString(sep)
	sb.WriteString(fmt.Sprintf("Total: %d running builds\n", len(items)))

	runningBuildsCache = sb.String()
	fmt.Print(runningBuildsCache)
}

// 格式化持续时间
func formatDuration(timestamp int64) string {
	if timestamp == 0 {
		return "0s"
	}

	milliseconds := getTime() - timestamp
	duration := time.Duration(milliseconds) * time.Millisecond
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %02ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

// PrintRunningBuildsSorted prints running builds sorted by duration (descending)
func PrintRunningBuildsSorted(items []jj.RunningBuild) {
	if len(items) == 0 {
		fmt.Println("No running builds")
		return
	}

	// Sort by duration (longest first)
	sortedItems := make([]jj.RunningBuild, len(items))
	copy(sortedItems, items)
	for i := 0; i < len(sortedItems)-1; i++ {
		for j := i + 1; j < len(sortedItems); j++ {
			if sortedItems[i].Timestamp < sortedItems[j].Timestamp {
				sortedItems[i], sortedItems[j] = sortedItems[j], sortedItems[i]
			}
		}
	}

	PrintRunningBuilds(sortedItems)
}
