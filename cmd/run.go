package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"github.com/gocruncher/bar"
	"github.com/gocruncher/jenkins-job-cli/cmd/jj"
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
)

var usageTamplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

var listenerStatus bool

type st struct {
	name  string
	id    int
	queue int
}

var curSt st
var barMutex sync.Mutex
var closeCh chan struct{}
var stdinListener *jjStdin
var aliasFlag string
var outputFile string

func init() {
	var runCmd = &cobra.Command{
		Use:     "run JOB",
		Aliases: []string{"r"},
		Short:   "Run the specified jenkins job",
		Run: func(cmd *cobra.Command, args []string) {
			if outputFile != "" {
				// --output forces non-interactive mode. The job (plus any
				// downstream projects) is run and the file receives the
				// job URL, complete invocation and the full console log.
				number := runJobNoOutput(args[0], true)
				saveOutputRunHistory(jj.Init(ENV), args[0], number)
				return
			}
			runJob(args[0])
		},

		Args:         cobra.ExactArgs(1),
		PreRunE:      runPreRunE,
		SilenceUsage: false,
	}
	inputArgs = arguments{args: make([]string, 0, 20)}
	runCmd.Flags().StringArrayVarP(&inputArgs.args, "arg", "a", []string{}, "input arguments of a job. Usage: -a key=val")
	runCmd.Flags().StringVarP(&ENV, "name", "n", "", "current Jenkins env name")
	runCmd.Flags().StringVar(&aliasFlag, "alias", "", "explicit alias to use when saving the command to history (overrides template/auto)")
	runCmd.Flags().StringVarP(&outputFile, "output", "o", "", "write the build console log to this file; when set the run is non-interactive (missing -a params fall back to the job defaults, no prompts)")
	runCmd.SetUsageTemplate(usageTamplate)
	rootCmd.AddCommand(runCmd)
}

func runPreRunE(cmd *cobra.Command, args []string) error {
	err := inputArgs.validate()
	if err != nil {
		return err
	}
	return preRunE(cmd, args)
}

func askParams(params []*jj.ParameterDefinitions) map[string]string {
	data := map[string]string{}
	for _, pd := range params {
		cline := ""
		defVal := pd.DefaultParameterValue.Value
		curChoices := pd.Choices
		for {
			rl, err := NewReadLine(chalk.Underline.TextStyle(pd.Name)+": ", pd.Choices)
			defer rl.Close()
			if err != nil {
				os.Exit(1)
			}
			if pd.Type == "ChoiceParameterDefinition" {
				defVal = ""
			}
			line, err := rl.ReadlineWithDefault(defVal)
			line = strings.TrimSpace(line)
			if err != nil { // io.EOF
				os.Exit(1)
			}
			// log.Println(line, pd.Choices, pd.Name, pd.Type, "test info.......")
			if pd.Type == "ChoiceParameterDefinition" || pd.Type == "GitParameterDefinition" {
				for _, val := range pd.Choices {
					if line == val {
						cline = val
						break
					}
				}
				if cline == "" {
					curChoices = findBestChoices(line, pd.Choices)
					if len(curChoices) == 0 {
						curChoices = pd.Choices
					} else if len(curChoices) == 1 {
						defVal = curChoices[0]
					} else {
						defVal = line
					}
					for _, val := range curChoices {
						fmt.Printf("%s\t", val)
					}
					if len(curChoices) > 0 {
						fmt.Println()
					}

					continue
				}
				// } else if pd.Type == "GitParameterDefinition" {
				// 	// git parameter
				// 	cline = line
			} else {
				cline = line
			}
			break
		}

		data[pd.Name] = cline

	}
	return data
}

func runJob(name string) int {
	env := jj.Init(ENV)
	time.Sleep(time.Millisecond * 200)
	fmt.Printf("Job will be started in the %s environment\n", chalk.Underline.TextStyle(string(env.Name)))
	time.Sleep(time.Millisecond * 200)
	if env.Url[len(env.Url)-1:] != "/" {
		env.Url = env.Url + "/"
	}
	fmt.Println("Link: ", env.Url+"job/"+name)
	time.Sleep(time.Millisecond * 200)

	bar.InitTerminal()
	data := map[string]string{}
	err, jobInfo := jj.GetJobInfo(env, name)
	if err == jj.ErrNoJob {
		err = fmt.Errorf("job '%s' does not exist", name)
	}
	check(err)
	params := jobInfo.GetParameterDefinitions(env, name)
	if len(params) == 0 {
		rl, err := readline.New("Press any key to continue: ")
		defer rl.Close()
		_, err = rl.Readline()
		if err != nil {
			os.Exit(1)
		}
	}
	if len(inputArgs.args) > 0 {
		for _, pd := range params {
			val, err := inputArgs.get(pd.Name)
			if err != nil {
				data[pd.Name] = pd.DefaultParameterValue.Value
			} else {
				data[pd.Name] = val
			}
		}
	} else {
		data = askParams(params)
	}

	urlquery := url.Values{}
	for key, val := range data {
		urlquery.Add(key, val)
	}
	err, queueId := jj.Build(env, name, urlquery.Encode())
	check(err)

	keyCh := make(chan string)
	stdinListener = NewStdin()
	go listenKeys(keyCh)
	go listenInterrupt(env)
	queueId1, _ := strconv.Atoi(queueId)
	curSt.queue = queueId1
	curSt.name = name
	number := waitForExecutor(env, queueId1)
	curSt.id = number
	err = watchTheJob(env, name, number, keyCh)
	if err != nil {
		return -1
	}
	curSt = st{}
	for _, jChild := range jobInfo.DownstreamProjects {
		err = watchNext(env, name, jChild.Name, number, keyCh)
		if err != nil {
			return -1
		}
		curSt = st{}
	}
	fmt.Println(chalk.Green.Color("Done"))
	stdinListener.Close()
	printQuickRunJobByID(env, name, number, aliasFlag)
	return number
}

// runJobNoOutput runs a job non-interactively. Nothing is prompted from
// stdin: parameters missing from -a fall back to the job's default values
// (including empty defaults). The output file receives:
//
//	<job console URL>
//	jj run <job> -n <env> -a <k=v> ...   (complete invocation)
//	<blank line>
//	<entire console log, fetched after the build finishes>
//
// Returns the build number, or -1 when the build fails or the file cannot
// be written. With truncate=true the file is started fresh (jj run -o);
// with truncate=false the report is appended (history run -o, multiple
// aliases share one file).
func runJobNoOutput(name string, truncate bool) int {
	env := jj.Init(ENV)
	err, jobInfo := jj.GetJobInfo(env, name)
	if err == jj.ErrNoJob {
		err = fmt.Errorf("job '%s' does not exist", name)
	}
	check(err)
	params := jobInfo.GetParameterDefinitions(env, name)

	query := url.Values{}
	argPairs := make([]string, 0, len(params))
	for _, pd := range params {
		val, err := inputArgs.get(pd.Name)
		if err != nil {
			val = pd.DefaultParameterValue.Value
		}
		query.Add(pd.Name, val)
		argPairs = append(argPairs, pd.Name+"="+val)
	}

	fmt.Printf("Build %s started in the %s environment\n", name, env.Name)
	err, queueId := jj.Build(env, name, query.Encode())
	check(err)
	queueId1, _ := strconv.Atoi(queueId)
	number := waitForExecutor(env, queueId1)
	jobUrl := jj.GetConsoleUrl(env, name, number)

	// Complete invocation line, e.g.:
	//   jj run <job> -n <env> -a <k=v> -a <k=v>
	invocation := []string{"jj run " + name, "-n", string(env.Name)}
	for _, kv := range argPairs {
		invocation = append(invocation, "-a "+kv)
	}
	invocationLine := strings.Join(invocation, " ")

	// Console header: job URL, execution time and complete invocation, then
	// the full console log. Nothing is prompted; the same block is echoed to
	// stdout.
	execTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("%s\n%s\n%s\n", jobUrl, invocationLine, execTime)
	fmt.Println()

	flags := os.O_CREATE | os.O_APPEND | os.O_WRONLY
	if truncate {
		flags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	}
	f, err := os.OpenFile(outputFile, flags, 0644)
	if err != nil {
		fmt.Printf("error: failed to open output file %s: %s\n", outputFile, err)
		os.Exit(1)
	}
	defer f.Close()

	// Header: job address, execution time and the complete invocation, then
	// a blank line before the full console log. Multiple builds (history run
	// with several aliases) are separated in the file by their header.
	if truncate {
		fmt.Fprintf(f, "%s\n%s\n%s\n\n", jobUrl, execTime, invocationLine)
	} else {
		fmt.Fprintf(f, "\n========== %s #%d ==========\n%s\n%s\n%s\n\n", name, number, jobUrl, execTime, invocationLine)
	}

	// Wait for the build to finish, then fetch the complete console log at
	// once (no console streaming).
	for {
		curBuild, err := jj.GetBuildInfo(env, name, number)
		if err == nil && !curBuild.Building {
			var result string
			if curBuild.Result == "SUCCESS" {
				result = "SUCCESS"
			} else {
				result = curBuild.Result
			}
			fmt.Printf("\nResult: %s\n", result)

			log, err := jj.ConsoleText(env, name, number)
			if err != nil {
				fmt.Printf("error: failed to read console log: %s\n", err)
				if curBuild.Result != "SUCCESS" {
					return -1
				}
				return number
			}
			if _, err := f.WriteString(log + "\n"); err != nil {
				fmt.Printf("error: failed to write output file %s: %s\n", outputFile, err)
				return -1
			}
			if curBuild.Result != "SUCCESS" {
				return -1
			}
			return number
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// saveOutputRunHistory records the -o run as a history entry so the command
// can later be re-run with "jj history run". number is the build number (or
// -1 when the run failed); failures are only reported but history is still
// persisted unless the build itself never produced a number.
func saveOutputRunHistory(env jj.Env, jobName string, number int) {
	if number <= 0 {
		return
	}
	cmdLine := []string{"jj run " + jobName, "-n", string(env.Name)}
	for _, arg := range inputArgs.args {
		cmdLine = append(cmdLine, "-a "+arg)
	}
	cmdStr := strings.Join(cmdLine, " ")

	cmdDef := jj.QuickRunCmdDefinition{
		Cmd:          cmdStr,
		JobName:      jobName,
		Env:          string(env.Name),
		LastJobUrl:   jj.GetConsoleUrl(env, jobName, number),
		Executimes:   1,
		LastExecTime: time.Now().Unix(),
	}
	if aliasFlag != "" {
		cmdDef.Alias = aliasFlag
	}
	jj.UpsertQuickRunCmd(cmdDef)
	fmt.Println(chalk.Dim.TextStyle(fmt.Sprintf("↳ History updated: %s", cmdDef.Alias)))
}

func waitForExecutor(env jj.Env, queueId int) int {
	informed := false
	for {
		err, queueInfo := jj.GetQueueInfo(env, queueId)
		check(err)
		if !queueInfo.Blocked && queueInfo.Executable.URL != "" {
			return queueInfo.Executable.Number
		} else {
			if !informed {
				//clearer := strings.Repeat(" ", int(110)-1)
				fmt.Println("waiting for next available executor..  ")
				informed = true
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func barHandler(jobUrl string, keyCh chan string, chMsg chan string, finishCh chan struct {
	err    error
	result string
}, wg *sync.WaitGroup) {
	defer wg.Done()
	barMutex.Lock()
	fmt.Print("\033[F")
	br := bar.NewWithOpts(
		bar.WithDimensions(100, 20),
		bar.WithLines(15),
		bar.WithFormat(
			fmt.Sprintf(
				"%srunning...%s :percent :bar %s:eta%s",
				chalk.White,
				chalk.Reset,
				chalk.Green,
				chalk.Reset)))
	br.Tick()
	barMutex.Unlock()
	for {
		select {
		case stdin, _ := <-keyCh:
			if []byte(stdin)[0] == 10 {
				barMutex.Lock()
				br.SetLines(br.GetLines() + 1)
				barMutex.Unlock()
			}
		case msg := <-chMsg:

			if msg != "" {
				barMutex.Lock()
				br.Interrupt(msg)
				barMutex.Unlock()
			} else {
				barMutex.Lock()
				br.Tick()
				barMutex.Unlock()
			}

		case info := <-finishCh:
			if info.err != nil && br.GetLines() < 5 {
				for br.GetLines() < 10 {
					barMutex.Lock()
					br.SetLines(br.GetLines() + 1)
					barMutex.Unlock()
				}

			}
			if info.err == nil {
				fmt.Printf("\r%s", strings.Repeat(" ", int(50)-1))
				fmt.Print("\033[F")
			}

			barMutex.Lock()
			br.SetFormat(jobUrl + ": " + info.result)
			br.Done()
			barMutex.Unlock()
			if info.err != nil {
				fmt.Println(chalk.Red.Color("failed"))
			}
			return
		case <-closeCh:
			return
		}

	}
}

func watchTheJob(env jj.Env, name string, number int, keyCh chan string) error {
	// jobUrl := env.Url + "/job/" + name + "/" + strconv.Itoa(number) + "/console"
	jobUrl := jj.GetConsoleUrl(env, name, number)
	fmt.Println(chalk.Green.Color("Job link:"), chalk.Underline.TextStyle(jobUrl))
	os.Stdout.Sync()
	//clearer := strings.Repeat(" ", int(110)-1)
	fmt.Println("")
	fmt.Println("")
	fmt.Println("")
	time.Sleep(time.Millisecond * 200)
	lastBuild, _ := jj.GetLastSuccessfulBuildInfo(env, name)
	listenerStatus = true
	defer func() {
		listenerStatus = false
	}()
	ticks := 1
	cursor := "0"
	stime := getTime()
	chMsg := make(chan string)
	closeCh = make(chan struct{})
	finishCh := make(chan struct {
		err    error
		result string
	})
	var wg sync.WaitGroup
	wg.Add(1)
	go barHandler(jobUrl, keyCh, chMsg, finishCh, &wg)
	defer close(closeCh)
	defer wg.Wait()
	go func() {
		for {
			select {
			case <-time.After(time.Millisecond * 100):
				ctime := getTime()
				dtime := ctime - stime
				newTicks := int(float64(dtime) / float64(lastBuild.Duration) * 100)
				for ticks < newTicks && ticks < 99 {
					chMsg <- ""
					ticks++
				}
			case <-closeCh:
				return
			}
		}
	}()

	handle := func(cursor string, sleepTime int) string {
		output, nextCursor, err := jj.Console(env, name, number, cursor)
		if err != nil || cursor == nextCursor {
			return cursor
		}
		lines := strings.Split(output, "\n")
		count := len(lines)
		if count > 50 {
			count = 50
		}
		for i := count; i >= 1; i-- {
			rline := []rune(string(lines[len(lines)-i]))
			if err != nil { // io.EOF
				break
			}
			j := 0
			size := 100
			for {
				var fline string
				s := j * size
				e := (j + 1) * size
				if len(rline) > e {
					fline = string(rline[s:e])
				} else {
					fline = string(rline[s:len(rline)])
				}
				if len(strings.TrimSpace(fline)) > 0 {
					chMsg <- fline
					time.Sleep(time.Duration(sleepTime) * time.Millisecond)
					//dotick()

				}
				j++
				if len(rline) <= e || len(rline) > 10*size {
					break
				}
			}
		}
		return nextCursor
	}

	for {
		curBuild, err := jj.GetBuildInfo(env, name, number)
		if err != nil {
			if getTime()-stime > int64(30*time.Millisecond) {
				err := errors.New("failed")
				finishCh <- struct {
					err    error
					result string
				}{err, err.Error()}
				return err
			}
		} else {
			if !curBuild.Building {
				if curBuild.Result == "SUCCESS" {
					k := 0
					for {
						k++
						nc := handle(cursor, 1)
						if k > 5 {
							fmt.Println()
							fmt.Println("nc", nc, cursor)
							fmt.Println()
						}
						if nc == cursor {
							break
						}
						cursor = nc
					}
					finishCh <- struct {
						err    error
						result string
					}{nil, curBuild.Result}
					return nil
				} else {
					err := errors.New("failed")
					finishCh <- struct {
						err    error
						result string
					}{err, curBuild.Result}
					return err
				}
			}
		}
		ncursor := handle(cursor, 100)
		if ncursor != cursor {
			cursor = ncursor
			//dotick()
			ctime := getTime()
			dtime := ctime - stime
			if dtime < 10 {
				time.Sleep(time.Duration(10-dtime) * time.Millisecond)
			}
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func watchNext(env jj.Env, parentName string, childName string, parentJobID int, keyCh chan string) error {
	for i := 0; ; i++ {
		bi, err := findDownstreamInBuilds(env, parentName, childName, parentJobID)
		if err != nil {
			queueId, err := findDownstreamInQueue(env, parentName, childName, parentJobID)
			curSt.queue = queueId
			curSt.name = childName
			if err != nil {
				time.Sleep(250 * time.Millisecond)
				continue
			}
			number := waitForExecutor(env, queueId)
			curSt.id = number
			return watchTheJob(env, childName, number, keyCh)
		} else {
			id, _ := strconv.Atoi(bi.Id)
			curSt.name = childName
			curSt.id = id
			return watchTheJob(env, childName, id, keyCh)
		}
	}
}

func findDownstreamInBuilds(env jj.Env, parentName string, childName string, parent int) (*jj.BuildInfo, error) {
	err, jobInfo := jj.GetJobInfo(env, childName)
	check(err)
	number := jobInfo.LastBuild.Number
	for i := 5; i >= 0; i-- {
		bi, err := jj.GetBuildInfo(env, childName, number-i)
		if err != nil {
			continue
		}
		for _, a := range bi.Actions {
			for _, c := range a.Causes {
				if c.UpstreamBuild == parent && c.UpstreamProject == parentName {
					return bi, nil
				}
			}
		}
	}
	return &jj.BuildInfo{}, errors.New("not found")
}

func findDownstreamInQueue(env jj.Env, parentName string, childName string, parentJobID int) (int, error) {
	queues := jj.GetQueues(env)
	for _, queue := range queues.Items {
		if queue.Task.Name == childName {
			for _, action := range queue.Actions {
				for _, cause := range action.Causes {
					if cause.UpstreamBuild == parentJobID && cause.UpstreamProject == parentName {
						return queue.ID, nil
					}
				}
			}
		}
	}
	return 0, errors.New("not found")
}

func listenKeys(out chan string) {
	stdinListener.NewListener()
	bt := make([]byte, 1)
	for {
		n, err := stdinListener.Read(bt)
		if err != nil || n == 0 {
			return
		}
		barMutex.Lock()
		if listenerStatus {
			out <- string(bt)
		}
		barMutex.Unlock()
	}

}

func listenInterrupt(env jj.Env) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			if curSt.name == "" {
				continue
			}
			barMutex.Lock()
			stdinListener.NewListener()
			readline.Stdin = stdinListener
			rl, err := readline.New(fmt.Sprintf("There is active build: %s. Do you want to cancel it [Y/n]:", curSt.name))
			if err != nil {
				barMutex.Unlock()
				os.Exit(1)
			}
			line, err := rl.Readline()
			rl.Close()
			barMutex.Unlock()
			if err != nil { // io.EOF
				os.Exit(1)
			}
			if line == "Y" || line == "y" {

				if curSt.queue != 0 {
					fmt.Println("canceling queue...")
					jj.CancelQueue(env, curSt.queue)
				}
				if curSt.id != 0 {
					fmt.Println("canceling job...")
					status, err := jj.CancelJob(env, curSt.name, curSt.id)
					if err != nil {
						fmt.Printf("failed to cancel job, error %s", err)
						os.Exit(0)
					}
					if status != "ABORTED" {
						fmt.Printf("Job already has been executed, status: %s", status)
						os.Exit(0)
					}
					fmt.Println("Canceled")
					os.Exit(0)
				}
				if curSt.queue != 0 && curSt.id == 0 {
					err, jobInfo := jj.GetJobInfo(env, curSt.name)
					check(err)
					number := jobInfo.LastBuild.Number
					for i := 0; i < 3; i++ {
						bi, err := jj.GetBuildInfo(env, curSt.name, number-i)
						if err != nil {
							continue
						}
						if bi.QueueId == curSt.queue {
							if bi.Result != "ABORTED" {
								fmt.Printf("Job already has been executed, status: %s", bi.Result)
								os.Exit(0)
							} else {
								fmt.Println("Canceled!")
								os.Exit(0)
							}
						}
					}
					fmt.Println("Canceled!!!")
				}
			}
			os.Exit(0)
		}
	}()
}

func check(err error) {
	if err != nil {
		fmt.Printf("\nError: %s\n", err.Error())
		os.Exit(1)
	}
}

func printQuickRunJobByID(env jj.Env, jobName string, jobNum int, userAlias string) {
	if jobNum <= 0 {
		err, jobInfo := jj.GetJobInfo(env, jobName)
		if err != nil {
			err = fmt.Errorf("job '%s' does not exist", jobName)
			fmt.Printf("error: %s\n", err)
			os.Exit(1)
		}
		jobNum = jobInfo.LastBuild.Number
		if jobNum == 0 {
			fmt.Printf("job '%s' has no build yet\n", jobName)
			os.Exit(1)
		}
	}

	buildInfo, err := jj.GetBuildInfo(env, jobName, jobNum)
	if err != nil {
		msg := fmt.Errorf("job '%s - %d' does not exist", jobName, jobNum)
		fmt.Printf("error: %s\n", msg)
		os.Exit(1)
	}

	quickCmd := []string{
		"jj run " + jobName,
		"-n",
		string(env.Name),
	}

	if len(buildInfo.Actions) > 0 {
		for _, action := range buildInfo.Actions {
			if action.Class == "hudson.model.ParametersAction" {
				for _, param := range action.Parameters {
					if param.Value == "" {
						continue
					}
					// quickCmd[0] += " -a " + param.Name + "=" + param.Value
					quickCmd = append(quickCmd, "-a "+param.Name+"="+param.Value)
				}
			}
		}
	}

	quickCmdStr := strings.Join(quickCmd, " ")
	fmt.Printf("\n\nquick commands jobName:%s, historyJobNum:%d \n%s\n", jobName, jobNum, quickCmdStr)
	if len(inputArgs.args) == 0 {
		askToSaveQuickCmd(quickCmdStr, jobName, jobNum, env, userAlias)
	}
}
