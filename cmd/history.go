package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/gocruncher/jenkins-job-cli/cmd/jj"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
	"gopkg.in/yaml.v2"
)

/*
*
history run alais1
按 alias 找到 命令 执行，并修改 最新执行时间和 执行次数

history list env  查看环境下的 历史记录
history run alias1 alias2  运行 alias1 alias2 对应的命令
history view alias1 alias2  查看 alias1 alias2 对应的命令
history delete alias1 alias2  删除 alias1 alias2 对应的命令
history clear  清空历史记录
*/
var (
	// 全局标志
	forceFlag        bool
	limitFlag        int
	allFlag          bool
	formatFlag       string // 输出格式：table, json, yaml
	exportFormatFlag string // export 命令的格式：json, yaml, csv
)

func init() {
	// 创建 history 主命令
	var historyCmd = &cobra.Command{
		Use:     "history",
		Aliases: []string{"hist", "h"},
		Short:   "Manage saved quick commands history",
		Long:    "Manage and run saved quick commands from history with aliases",
		Run: func(cmd *cobra.Command, args []string) {
			// 默认显示列表
			cmd.Help()
		},
	}

	// 全局标志
	historyCmd.PersistentFlags().StringVar(&formatFlag, "format", "table", "Output format (table, json, yaml)")
	historyCmd.PersistentFlags().IntVarP(&limitFlag, "limit", "l", 0, "Limit number of results")

	// 为 --format 添加自动补全
	formatFlagLookup := historyCmd.PersistentFlags().Lookup("format")
	if formatFlagLookup != nil {
		formatFlagLookup.Annotations = map[string][]string{
			cobra.BashCompCustom: {"__jj_format_completion"},
		}
	}

	// ========== history list 命令 ==========
	var historyListCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "l"},
		Short:   "List saved quick commands",
		Long: `List all saved quick commands.
You can limit the number of results.`,
		Example: `  jj history list                      # List all commands
  jj history list -l 10                 # List last 10 commands
  jj history list --format=json               # Output in JSON format`,
		ValidArgsFunction: completeAliases,
		Run: func(cmd *cobra.Command, args []string) {
			listQuickCmds()
		},
	}
	historyListCmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Show all details including full command")
	historyCmd.AddCommand(historyListCmd)

	// ========== history run 命令 ==========
	var historyRunCmd = &cobra.Command{
		Use:               "run",
		Aliases:           []string{"r", "exec"},
		ValidArgsFunction: completeAliases,
		Short:             "Run saved quick command by alias",
		Long: `Run saved quick command by alias.
You can run multiple commands by providing multiple aliases.`,
		Example: `  jj history run myjob_quick            # Run single command
  jj history run alias1 alias2 alias3  # Run multiple commands`,
		Run: func(cmd *cobra.Command, args []string) {

			// 运行指定的命令
			runQuickCmds(args)
		},
	}
	historyRunCmd.Flags().BoolVarP(&forceFlag, "force", "", false, "Force run without confirmation")
	historyCmd.AddCommand(historyRunCmd)

	// ========== history view 命令 ==========
	var historyViewCmd = &cobra.Command{
		Use:               "view",
		Aliases:           []string{"v", "show", "info"},
		Short:             "View details of saved quick commands",
		Long:              "View detailed information of saved quick commands.",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeAliases,
		Example: `  jj history view alias1              # View single command
  jj history view alias1 alias2       # View multiple commands
  jj history view alias1      # Output in JSON format`,
		Run: func(cmd *cobra.Command, args []string) {
			for _, alias := range args {
				viewQuickCmd(alias)
				if len(args) > 1 {
					fmt.Println(strings.Repeat("-", 80))
				}
			}
		},
	}
	historyCmd.AddCommand(historyViewCmd)

	// ========== history delete 命令 ==========
	var historyDeleteCmd = &cobra.Command{
		Use:     "delete",
		Aliases: []string{"del", "rm", "remove"},
		Short:   "Delete saved quick commands",
		Long:    "Delete saved quick commands by alias.",
		Args:    cobra.MinimumNArgs(1),
		Example: `  jj history delete alias1            # Delete single command
  jj history delete alias1 alias2     # Delete multiple commands
  jj history delete -f alias1         # Force delete without confirmation`,
		Run: func(cmd *cobra.Command, args []string) {
			for _, alias := range args {
				deleteQuickCmd(alias)
			}
		},
		ValidArgsFunction: completeAliases,
	}
	historyDeleteCmd.Flags().BoolVarP(&forceFlag, "force", "", false, "Force delete without confirmation")
	historyCmd.AddCommand(historyDeleteCmd)

	// ========== history clear 命令 ==========
	var historyClearCmd = &cobra.Command{
		Use:     "clear",
		Aliases: []string{"clean", "clr"},
		Short:   "Clear all history records",
		Long:    "Clear all saved quick commands history.",
		Example: `  jj history clear                   # Clear all history with confirmation
  jj history clear -f               # Force clear without confirmation`,
		Run: func(cmd *cobra.Command, args []string) {
			clearAllHistory()
		},
	}
	historyClearCmd.Flags().BoolVarP(&forceFlag, "force", "", false, "Force clear without confirmation")
	historyCmd.AddCommand(historyClearCmd)

	// ========== history search 命令 ==========
	var historySearchCmd = &cobra.Command{
		Use:     "search",
		Aliases: []string{"s", "find", "grep"},
		Short:   "Search saved quick commands",
		Long:    "Search saved quick commands by keyword in alias, job name, or description.",
		Args:    cobra.ExactArgs(1),
		Example: `  jj history search "prod"            # Search for "prod" in all fields
  jj history search --format=json "job"      # Output in JSON format`,
		Run: func(cmd *cobra.Command, args []string) {
			searchQuickCmds(args[0])
		},
		// 添加补全
		ValidArgsFunction: completeAliases,
	}
	historyCmd.AddCommand(historySearchCmd)

	// ========== history export 命令 ==========
	var historyExportCmd = &cobra.Command{
		Use:     "export",
		Aliases: []string{"exp", "save"},
		Short:   "Export history to file",
		Long:    "Export history records to a file in various formats.",
		Example: `  jj history export history.json      # Export to JSON file
  jj history export history.yaml      # Export to YAML file`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				exportHistory("")
			} else {
				exportHistory(args[0])
			}
		},
	}
	historyExportCmd.Flags().StringVarP(&exportFormatFlag, "format", "", "yaml", "Export format (json, yaml, csv)")
	// 为 export 的 --format 添加自动补全
	exportFormatFlagLookup := historyExportCmd.Flags().Lookup("format")
	if exportFormatFlagLookup != nil {
		exportFormatFlagLookup.Annotations = map[string][]string{
			cobra.BashCompCustom: {"__jj_export_format_completion"},
		}
	}
	historyCmd.AddCommand(historyExportCmd)

	// ========== history import 命令 ==========
	var historyImportCmd = &cobra.Command{
		Use:     "import",
		Aliases: []string{"imp", "load"},
		Short:   "Import history from file",
		Long:    "Import history records from a file.",
		Args:    cobra.ExactArgs(1),
		Example: `  jj history import backup.yaml      # Import from YAML file
  jj history import history.json      # Import from JSON file`,
		Run: func(cmd *cobra.Command, args []string) {
			importHistory(args[0])
		},
	}
	historyImportCmd.Flags().BoolVarP(&forceFlag, "merge", "m", false, "Merge instead of replace")
	historyCmd.AddCommand(historyImportCmd)

	// 添加到根命令
	rootCmd.AddCommand(historyCmd)
}

// 补全函数：为所有需要别名的命令提供补全
func completeAliases(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// 获取所有可能的别名
	allAliases := getAllAliases(toComplete)

	// 创建已使用别名的集合
	usedAliases := make(map[string]bool)
	for _, arg := range args {
		usedAliases[arg] = true
	}

	// 过滤掉已使用的别名
	var availableAliases []string
	for _, alias := range allAliases {
		// 跳过已经在 args 中的别名
		if !usedAliases[alias] {
			// 同时确保别名以 toComplete 开头（已经在 getAllAliases 中处理）
			availableAliases = append(availableAliases, alias)
		}
	}

	return availableAliases, cobra.ShellCompDirectiveNoFileComp
}

// 获取所有别名
func getAllAliases(prefix string) []string {
	history := jj.GetQuickRunCmdList()

	var aliases []string
	for _, cmd := range history {
		// 先检查前缀匹配
		if prefix == "" || strings.HasPrefix(cmd.Alias, prefix) {
			aliases = append(aliases, cmd.Alias)
		}
	}

	return aliases
}

// ========== 实现函数 ==========

// 运行多个命令
func runQuickCmds(aliases []string) {
	history := jj.GetQuickRunCmdList()

	// 验证别名是否存在
	var validCmdIndices []int // 保存 history 中的索引
	var notFoundAliases []string

	for _, alias := range aliases {
		found := false
		for i := range history {
			if history[i].Alias == alias {
				validCmdIndices = append(validCmdIndices, i)
				found = true
				break
			}
		}

		if !found {
			notFoundAliases = append(notFoundAliases, alias)
		}
	}

	// 显示未找到的别名
	if len(notFoundAliases) > 0 {
		fmt.Print(chalk.Red.Color("Error: The following aliases were not found:\n"))
		for _, alias := range notFoundAliases {
			fmt.Printf("  - %s\n", alias)
		}

		if len(validCmdIndices) == 0 {
			return
		}

		// 询问是否继续执行找到的命令
		if !forceFlag {
			fmt.Print("\nContinue with found commands? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				return
			}
		}
	}

	// 确认执行
	if !forceFlag && len(validCmdIndices) > 1 {
		fmt.Printf("\n%d commands will be executed:\n", len(validCmdIndices))
		for _, idx := range validCmdIndices {
			fmt.Printf("  • %s: %s\n", history[idx].Alias, history[idx].JobName)
		}

		fmt.Print("\nContinue? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			return
		}
	}

	// 执行命令
	for _, idx := range validCmdIndices {
		cmd := &history[idx]
		fmt.Printf("\n%s Running: %s %s\n",
			chalk.Green.Color("→"),
			chalk.Bold.TextStyle(cmd.Alias),
			chalk.Dim.TextStyle("("+cmd.JobName+")"))

		// 执行命令（这里需要根据你的实际情况实现）
		var jobUrl string
		var err error
		if jobUrl, err = executeQuickCommand(cmd); err != nil {
			fmt.Printf(chalk.Red.Color("Error executing %s: %v\n"), cmd.Alias, err)
		} else {
			fmt.Printf(chalk.Green.Color("✓ Command %s executed successfully\n"), cmd.Alias)
			fmt.Printf(chalk.Green.Color("✓ res job url %s\n"), jobUrl)
			// 更新执行统计和 LastJobUrl
			history[idx].Executimes++
			history[idx].LastExecTime = time.Now().Unix()
			if jobUrl != "" {
				history[idx].LastJobUrl = jobUrl
			}
		}
	}

	jj.SaveAllHistory(history)
}

// 执行单个命令（需要根据你的实际执行逻辑实现）
// 返回 jobUrl 和 error
func executeQuickCommand(cmd *jj.QuickRunCmdDefinition) (string, error) {
	// 这里需要解析 cmd.Cmd 并执行
	// 例如：解析 "jj run jobname -a param=value" 这样的命令

	// 临时示例实现：
	fmt.Printf("Executing: %s\n", cmd.Cmd)
	// TODO: 实际执行命令的逻辑
	// shell 中执行 cmd.Cmd
	ENV = cmd.Env

	inputArgs = arguments{args: make([]string, 0, 20)}

	// 解析命令
	cmdArgs := strings.Split(cmd.Cmd, "-a")

	// 解析参数
	for _, arg := range cmdArgs[1:] {
		inputArgs.args = append(inputArgs.args, strings.Trim(arg, " "))
	}

	// 保存当前的 curSt，以便在执行后获取 jobNum

	jobNum := runJob(cmd.JobName)

	// 从 curSt 获取 jobNum，构建 jobUrl
	var jobUrl string
	if jobNum > 0 {
		env := jj.Init(cmd.Env)
		jobUrl = jj.GetConsoleUrl(env, cmd.JobName, jobNum)
	} else {
		return "", fmt.Errorf("job '%s' run failed", cmd.JobName)
	}

	return jobUrl, nil
}

// 格式化相对时间
func formatRelativeTime(timestamp int64) string {
	if timestamp <= 0 {
		return "Never"
	}

	now := time.Now()
	t := time.Unix(timestamp, 0)
	diff := now.Sub(t)

	if diff < time.Minute {
		return "Just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / (7 * 24))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else {
		// 超过30天显示具体日期
		return t.Format("2006-01-02 15:04:05")
	}
}

// 格式化时间显示（详细格式）
func formatTimeDetailed(timestamp int64) string {
	if timestamp <= 0 {
		return "Never"
	}
	t := time.Unix(timestamp, 0)
	now := time.Now()
	diff := now.Sub(t)

	// 如果是今天，只显示时间
	if t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day() {
		return fmt.Sprintf("Today %s", t.Format("15:04:05"))
	}

	// 如果是昨天
	if diff < 48*time.Hour && diff >= 24*time.Hour {
		yesterday := now.AddDate(0, 0, -1)
		if t.Year() == yesterday.Year() && t.Month() == yesterday.Month() && t.Day() == yesterday.Day() {
			return fmt.Sprintf("Yesterday %s", t.Format("15:04:05"))
		}
	}

	// 如果是今年，显示月-日 时:分:秒
	if t.Year() == now.Year() {
		return t.Format("01-02 15:04:05")
	}

	// 其他情况显示完整日期
	return t.Format("2006-01-02 15:04:05")
}

// 查看命令详情
func viewQuickCmd(alias string) {
	history := jj.GetQuickRunCmdList()

	var findCmd *jj.QuickRunCmdDefinition
	for _, cmd := range history {
		if strings.EqualFold(cmd.Alias, alias) {
			fmt.Printf(chalk.Green.Color("Quick command '%s' found:\n"), alias)
			findCmd = &cmd
			break
		}
	}

	if findCmd == nil {
		fmt.Printf(chalk.Red.Color("Error: Quick command '%s' not found\n"), alias)
		return
	}

	// 根据格式输出
	switch strings.ToLower(formatFlag) {
	case "json":
		formatted := formatQuickCmd(*findCmd)
		data, _ := json.MarshalIndent(formatted, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		formatted := formatQuickCmd(*findCmd)
		data, _ := yaml.Marshal(formatted)
		fmt.Println(string(data))
	default:
		// 表格格式
		fmt.Println(chalk.Cyan.Color("╭─────────────────────────────────────────────────────────────────────────────╮"))
		fmt.Println(chalk.Cyan.Color("│ ") + chalk.Bold.TextStyle("Quick Command Details") + chalk.Cyan.Color("                                                       │"))
		fmt.Println(chalk.Cyan.Color("╰─────────────────────────────────────────────────────────────────────────────╯"))

		fmt.Printf("%s %s\n", chalk.Bold.TextStyle("Alias:"), chalk.Green.Color(findCmd.Alias))
		fmt.Printf("%s %s\n", chalk.Bold.TextStyle("Command:"), findCmd.Cmd)
		fmt.Printf("%s %s\n", chalk.Bold.TextStyle("Job Name:"), chalk.Cyan.Color(findCmd.JobName))
		if findCmd.LastJobUrl != "" {
			fmt.Printf("%s %s\n", chalk.Bold.TextStyle("Last Job URL:"), chalk.Underline.TextStyle(findCmd.LastJobUrl))
		} else {
			fmt.Printf("%s %s\n", chalk.Bold.TextStyle("Last Job URL:"), chalk.Dim.TextStyle("Not available"))
		}
		fmt.Printf("%s %s\n", chalk.Bold.TextStyle("Environment:"), chalk.Magenta.Color(findCmd.Env))
		fmt.Printf("%s %d\n", chalk.Bold.TextStyle("Executed times:"), findCmd.Executimes)

		if findCmd.LastExecTime > 0 {
			fmt.Printf("%s %s (%s)\n", chalk.Bold.TextStyle("Last executed:"),
				formatTimeDetailed(findCmd.LastExecTime),
				chalk.Dim.TextStyle(formatRelativeTime(findCmd.LastExecTime)))
		} else {
			fmt.Printf("%s %s\n", chalk.Bold.TextStyle("Last executed:"), "Never")
		}

		fmt.Println()
		fmt.Println(chalk.Dim.TextStyle("To run this command: jj history run " + alias))
	}
}

// 删除命令
func deleteQuickCmd(alias string) {

	history := jj.GetQuickRunCmdList()

	var findCmd *jj.QuickRunCmdDefinition
	for _, cmd := range history {
		if strings.EqualFold(cmd.Alias, alias) {
			findCmd = &cmd
			break
		}
	}

	if findCmd == nil {
		fmt.Printf(chalk.Red.Color("Error: Quick command '%s' not found\n"), alias)
		return
	}

	// 确认删除
	if !forceFlag {
		fmt.Printf("Delete quick command '%s' (%s)? [y/N]: ",
			chalk.Green.Color(alias),
			chalk.Cyan.Color(findCmd.JobName))

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer != "y" && answer != "yes" {
			fmt.Println(chalk.Yellow.Color("Cancelled"))
			return
		}
	}

	// 执行删除
	newHistory := []jj.QuickRunCmdDefinition{}
	for _, cmd := range history {
		if !strings.EqualFold(cmd.Alias, alias) {
			newHistory = append(newHistory, cmd)
		}
	}

	jj.SaveAllHistory(newHistory)

	fmt.Println(chalk.Green.Color("✓ Quick command deleted successfully!"))
}

// 清空所有历史
func clearAllHistory() {
	if !forceFlag {
		fmt.Print(chalk.Red.Color("⚠️  This will delete ALL saved quick commands. Are you sure? [y/N]: "))
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer != "y" && answer != "yes" {
			fmt.Println(chalk.Yellow.Color("Cancelled"))
			return
		}
	}

	// 创建空的history对象并保存
	history := []jj.QuickRunCmdDefinition{}
	jj.SaveAllHistory(history)

	fmt.Println(chalk.Green.Color("✓ All history cleared successfully!"))
}

// 搜索命令
func searchQuickCmds(keyword string) {
	history := jj.GetQuickRunCmdList()
	if history == nil {
		fmt.Println(chalk.Yellow.Color("No saved commands found"))
		return
	}

	var results []jj.QuickRunCmdDefinition
	keyword = strings.ToLower(keyword)

	for _, cmd := range history {
		// 搜索别名、作业名、环境、描述
		if strings.Contains(strings.ToLower(cmd.Alias), keyword) ||
			strings.Contains(strings.ToLower(cmd.JobName), keyword) ||
			strings.Contains(strings.ToLower(cmd.Env), keyword) {
			results = append(results, cmd)
		}
	}

	if len(results) == 0 {
		fmt.Println(chalk.Yellow.Color("No commands found matching: ") + keyword)
		return
	}

	// 显示结果
	fmt.Printf(chalk.Cyan.Color("Found %d commands matching '%s':\n\n"), len(results), keyword)

	for i, cmd := range results {
		fmt.Printf("%2d. %-25s %-20s %-10s %s\n",
			i+1,
			chalk.Green.Color(cmd.Alias),
			chalk.Cyan.Color(cmd.JobName),
			chalk.Magenta.Color("["+cmd.Env+"]"),
			chalk.Dim.TextStyle(cmd.Cmd))
	}
}

// 导出历史
func exportHistory(filename string) {
	// 实现导出逻辑
	fmt.Println("Export function - TODO")
}

// 导入历史
func importHistory(filename string) {
	// 实现导入逻辑
	fmt.Println("Import function - TODO")
}

// 格式化输出的结构体（用于 JSON/YAML 输出）
type formattedQuickCmd struct {
	Alias        string `json:"alias" yaml:"alias"`
	JobName      string `json:"jobName" yaml:"jobName"`
	Cmd          string `json:"cmd" yaml:"cmd"`
	Executimes   int    `json:"executimes" yaml:"executimes"`
	LastExecTime string `json:"lastExecTime" yaml:"lastExecTime"` // 格式化为字符串
	LastExecUnix int64  `json:"lastExecUnix" yaml:"lastExecUnix"` // 保留原始时间戳
	Env          string `json:"env" yaml:"env"`
	LastJobUrl   string `json:"lastJobUrl" yaml:"lastJobUrl"`
}

// 将 QuickRunCmdDefinition 转换为格式化结构体
func formatQuickCmd(cmd jj.QuickRunCmdDefinition) formattedQuickCmd {
	var lastExecTimeStr string
	if cmd.LastExecTime > 0 {
		lastExecTimeStr = time.Unix(cmd.LastExecTime, 0).Format("2006-01-02 15:04:05")
	} else {
		lastExecTimeStr = "Never"
	}

	return formattedQuickCmd{
		Alias:        cmd.Alias,
		JobName:      cmd.JobName,
		Cmd:          cmd.Cmd,
		Executimes:   cmd.Executimes,
		LastExecTime: lastExecTimeStr,
		LastExecUnix: cmd.LastExecTime,
		Env:          cmd.Env,
		LastJobUrl:   cmd.LastJobUrl,
	}
}

// 列表命令
func listQuickCmds() {

	history := jj.GetQuickRunCmdList()

	if len(history) == 0 {
		fmt.Println(chalk.Yellow.Color("No saved commands found"))
		return
	}

	// 根据格式输出
	switch strings.ToLower(formatFlag) {
	case "json":
		filtered := history

		if limitFlag > 0 && limitFlag < len(filtered) {
			filtered = filtered[:limitFlag]
		}

		// 转换为格式化结构体
		formatted := make([]formattedQuickCmd, len(filtered))
		for i, cmd := range filtered {
			formatted[i] = formatQuickCmd(cmd)
		}

		data, _ := json.MarshalIndent(formatted, "", "  ")
		fmt.Println(string(data))
		return

	case "yaml":
		filtered := history

		if limitFlag > 0 && limitFlag < len(filtered) {
			filtered = filtered[:limitFlag]
		}

		// 转换为格式化结构体
		formatted := make([]formattedQuickCmd, len(filtered))
		for i, cmd := range filtered {
			formatted[i] = formatQuickCmd(cmd)
		}

		data, _ := yaml.Marshal(formatted)
		fmt.Println(string(data))
		return
	}

	// 默认表格格式
	fmt.Println(chalk.Cyan.Color("╭─────────────────────────────────────────────────────────────────────────────────────────────────────╮"))
	title := "Saved Quick Commands"
	fmt.Println(chalk.Cyan.Color("│ ") + chalk.Bold.TextStyle(title) + chalk.Cyan.Color(strings.Repeat(" ", 88-len(title))) + " │")
	fmt.Println(chalk.Cyan.Color("╰─────────────────────────────────────────────────────────────────────────────────────────────────────╯"))

	// 过滤结果
	filtered := history

	if limitFlag > 0 && limitFlag < len(filtered) {
		filtered = filtered[:limitFlag]
	}

	// 打印表格头
	fmt.Printf(chalk.Bold.TextStyle("%-25s %-20s %-15s %-8s %-20s %s\n"),
		"Alias", "Job", "Env", "Exec#", "Last Executed", "Cmd")
	fmt.Println(chalk.Dim.TextStyle(strings.Repeat("─", 120)))

	for _, cmd := range filtered {
		// 格式化时间 - 使用相对时间
		var lastExecTime string
		if cmd.LastExecTime > 0 {
			lastExecTime = formatRelativeTime(cmd.LastExecTime)
		} else {
			lastExecTime = "Never"
		}

		// 截断过长的描述
		desc := cmd.Cmd
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}

		// 高亮最近使用的命令
		aliasColor := chalk.Green
		if cmd.Executimes > 0 {
			aliasColor = chalk.Yellow
		}

		fmt.Printf("%-25s %-20s %-15s %-8d %-20s %s\n",
			aliasColor.Color(cmd.Alias),
			chalk.Cyan.Color(cmd.JobName),
			chalk.Magenta.Color(cmd.Env),
			cmd.Executimes,
			chalk.Dim.TextStyle(lastExecTime),
			desc)
	}

	fmt.Printf("\n%s Total: %d commands\n", chalk.Dim.TextStyle("↳"), len(filtered))
	if limitFlag > 0 && limitFlag < len(history) {
		fmt.Printf(chalk.Dim.TextStyle("Showing %d of %d total commands\n"), len(filtered), len(history))
	}
}

// 保存快速运行命令到历史记录，并提示用户输入别名
func askToSaveQuickCmd(cmd, jobName string, jobNum int, env jj.Env) {
	fmt.Println("\n" + chalk.Cyan.Color("╭─────────────────────────────────────────────╮"))
	fmt.Println(chalk.Cyan.Color("│ ") + chalk.Bold.TextStyle("Save quick command to history?") + chalk.Cyan.Color("           │"))
	fmt.Println(chalk.Cyan.Color("│ ") + chalk.Yellow.Color("(y)") + " Yes, save with alias    " + chalk.Yellow.Color("(n)") + " No, skip " + chalk.Cyan.Color(" │"))
	fmt.Println(chalk.Cyan.Color("╰─────────────────────────────────────────────╯"))
	fmt.Printf("%s", chalk.Dim.TextStyle("Auto skip in 20 seconds... \n"))

	// 创建带超时的读取器
	answerChan := make(chan string, 1)

	stdinListener.NewListener()
	readline.Stdin = stdinListener

	go func() {
		rl, err := readline.New(fmt.Sprintf("There is active build: %s. Do you want to cancel it [Y/n]:", curSt.name))
		if err != nil {
			answerChan <- "n"
			return
		}
		defer rl.Close()

		line, err := rl.Readline()
		if err != nil {
			answerChan <- "n"
			return
		}

		answerChan <- strings.TrimSpace(strings.ToLower(line))
	}()

	// 等待用户输入或超时
	var answer string
	select {
	case ans := <-answerChan:
		answer = ans
	case <-time.After(20 * time.Second):
		answer = "n"
		fmt.Println(chalk.Dim.TextStyle("Timeout, skipping save...\n"))
	}

	// 处理用户选择
	switch answer {
	case "y", "yes":
		saveQuickCmdWithAlias(cmd, jobName, jobNum, env)
	case "n", "no", "":
		fmt.Println(chalk.Dim.TextStyle("Quick command not saved.\n"))
	default:
		fmt.Println(chalk.Red.Color("Invalid choice. Quick command not saved.\n"))
	}
}

func saveQuickCmdWithAlias(cmd, jobName string, jobNum int, env jj.Env) {
	// 加载现有历史
	history := jj.GetQuickRunCmdList()

	// 获取用户输入的别名
	alias := getAliasFromUser(history)
	if alias == "" {
		fmt.Println(chalk.Dim.TextStyle("No alias entered, skipping save...\n"))
		return
	}

	// 构建 LastJobUrl
	var lastJobUrl string
	if jobNum > 0 {
		lastJobUrl = jj.GetConsoleUrl(env, jobName, jobNum)
	}

	// 创建命令定义
	cmdDef := jj.QuickRunCmdDefinition{
		Alias:        alias,
		Cmd:          cmd,
		JobName:      jobName,
		Executimes:   1,
		LastExecTime: time.Now().Unix(),
		Env:          string(env.Name),
		LastJobUrl:   lastJobUrl,
	}

	// 保存到历史
	jj.AddQuickRunCmd(cmdDef)
}

func getAliasFromUser(history []jj.QuickRunCmdDefinition) string {
	historyNames := []string{}
	for _, item := range history {
		historyNames = append(historyNames, item.Alias)
	}

	stdinListener.NewListener()
	readline.Stdin = stdinListener

	for {
		rl, err := readline.New(fmt.Sprintf("Enter alias name (press Enter): %s", curSt.name))
		if err != nil {
			fmt.Println(chalk.Red.Color("Error creating input reader"))
			return ""
		}
		defer rl.Close()

		alias, err := rl.Readline()
		if err != nil {
			fmt.Println(chalk.Dim.TextStyle("Alias input cancelled"))
			return ""
		}

		alias = strings.TrimSpace(alias)

		if alias == "" {
			// 自动生成别名
			fmt.Println(chalk.Dim.TextStyle("No alias entered, please try again."))
			continue
		}

		// 检查别名是否已存在
		if slices.Contains(historyNames, alias) {
			fmt.Println(chalk.Red.Color("Error: Alias already exists. Please choose a different one."))
			continue
		}

		return alias
	}
}
