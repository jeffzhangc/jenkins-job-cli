package jj

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// External API

func GetBundle(env Env) *Bundle {
	for _, b := range bundles {
		if b.Name == env.Name {
			return b
		}
	}
	return nil
}

func DelEnv(name EName) error {
	for i, e := range config.Envs {
		if e.Name == name {
			config.Envs = append(config.Envs[:i], config.Envs[i+1:]...)
			SetConf()
			return nil
		}
	}
	return fmt.Errorf("'%s' name is not found", name)
}

func Check(env Env) error {
	code, rspbin, _, err := req(env, "GET", "api/json", []byte{})
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("http code: %d\nresponse: %s", code, rspbin)
	}
	return nil
}

func GetEnvs() []Env {
	return config.Envs
}

func GetEnv(name string) (error, Env) {
	var env Env
	if name == "" {
		name = string(GetDefEnv())
	}
	for _, e := range GetEnvs() {
		if e.Name == EName(name) {
			env = e
			break
		}
	}
	if env == (Env{}) {
		return ErrNoEnv, env
	}
	return nil, env
}

func (ji *JobInfo) GetParameterDefinitions(env Env, jobName string) []*ParameterDefinitions {
	res := []*ParameterDefinitions{}
	for _, j := range ji.Property {
		if len(j.ParameterDefinitions) > 0 {
			// res :=
			// return j.ParameterDefinitions
			for _, pd := range j.ParameterDefinitions {
				newPd := pd
				res = append(res, &newPd)
			}
			go ji.initGitParameterDefinitions(res, env, jobName)
		}
	}
	return res
}

func (ji *JobInfo) initGitParameterDefinitions(res []*ParameterDefinitions, env Env, jobName string) {
	for _, pd := range res {
		if pd.Type == "GitParameterDefinition" {
			err, item := GetGitParameterDefinitionItems(env, jobName, pd.Name)
			if err != nil {
				fmt.Printf("error: %s\n", err)
			} else {
				// pd.Items = items
				choices := []string{}
				for _, item := range item.Values {
					choices = append(choices, item.Name)
				}
				pd.Choices = choices
			}
		}
	}
}

func GetDefEnv() EName {
	if config.Use == "" {
		return GetEnvs()[0].Name
	}
	return config.Use
}
func SetDef(eName string) {
	var env Env
	for _, e := range GetEnvs() {
		if e.Name == EName(eName) {
			env = e
			break
		}
	}
	if env == (Env{}) {
		panic("Environment " + eName + " is not found or could not be initialised")
	}
	config.Use = env.Name
	SetConf()
}

func SetConf() {
	out, _ := yaml.Marshal(config)
	if _, err := os.Stat(homeDir); os.IsNotExist(err) {
		err := os.MkdirAll(homeDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	err := ioutil.WriteFile(homeDir+configFile, out, 0644)
	if err != nil {
		panic(err)
	}
}

func SetEnv(env Env) {
	added := false
	for i, e := range config.Envs {
		if e.Name == env.Name {
			config.Envs[i] = env
			added = true
			break
		}
	}
	if !added {
		config.Envs = append(config.Envs, env)
	}
	SetConf()

}

//	func GetJobInfo(env Env, jobName string) JobInfo{
//		var jobinfo JobInfo
//		code, rsp, _, err := req(env,"job/"+jobName+"/api/json", []byte{})
//		if err != nil {
//			panic(err)
//		}
//		if code != 200 {
//			panic("failed to get job details,code" + strconv.Itoa(code) + ", " + string(rsp))
//		}
//		err = json.Unmarshal(rsp, &jobinfo)
//		if err!=nil{
//			panic("failed to get Job information")
//		}
//		return jobinfo
//	}
func GetJobInfo(env Env, jobName string) (error, *JobInfo) {
	bundle := GetBundle(env)
	var jobInfo *JobInfo
	for _, ji := range bundle.JobsInfo {
		if ji.Name == jobName {
			jobInfo = &ji
			break
		}
	}
	var fetchJobInfo = func() (error, JobInfo) {
		var ji JobInfo
		code, rsp, _, err := req(env, "POST", "job/"+jobName+"/api/json", []byte{})
		if err != nil {
			return err, ji
		}
		if code != 200 {
			return ErrNoJob, ji
		}
		err = json.Unmarshal(rsp, &ji)
		if err != nil {
			panic("failed to get Job information")
		}
		mutex.Lock()
		defer mutex.Unlock()
		// the cache will be grown unlimitedly, need to be optimized
		contained := false
		for i, j := range bundle.JobsInfo {
			if j.Name == jobName {
				bundle.JobsInfo[i] = ji
				contained = true
				break
			}
		}
		if !contained {
			bundle.JobsInfo = append(bundle.JobsInfo, ji)
		}
		// bundle.JobsInfo = append(bundle.JobsInfo, ji)
		updateCache(env, bundle)
		return nil, ji
	}
	if jobInfo != nil {
		//fmt.Println("async")
		go fetchJobInfo()
	} else {
		//fmt.Println("sync")
		err, ji := fetchJobInfo()
		if err != nil {
			return err, jobInfo
		}
		jobInfo = &ji
	}
	return nil, jobInfo

}

//
//func GetJobParameterDefinitions(env Env, jobName string) []ParameterDefinitions {
//	bundle:=GetBundle(env)
//	params:= []ParameterDefinitions{}
//	for _,jp:= range bundle.JobsParameters{
//		if jp.Name == jobName{
//			params = jp.Parameters
//		}
//	}
//
//	var fetchParameters = func() []ParameterDefinitions{
//		jobinfo:=GetJobInfo(env,jobName)
//		parameters:=jobinfo.GetParameterDefinitions()
//		mutex.Lock()
//		defer mutex.Unlock()
//		bundle.JobsParameters = append(bundle.JobsParameters,JobsParameters{Name: jobName, Parameters: parameters})
//		updateCache(env, bundle)
//		return parameters
//	}
//
//	if len(params)>0{
//		fmt.Println("async")
//		go fetchParameters()
//	}else{
//		fmt.Println("sync")
//		return fetchParameters()
//	}
//	return params
//
//
//
//}
//

func GetBuildInfo(env Env, job string, id int) (*BuildInfo, error) {
	code, rsp, _, err := req(env, "POST", "job/"+job+"/"+strconv.Itoa(id)+"/api/json", []byte{})
	if err != nil {
		panic(err)
	}
	if code != 200 {
		return nil, errors.New("failed to get job details,code" + strconv.Itoa(code) + ", " + string(rsp))
	}
	var bi BuildInfo
	err = json.Unmarshal(rsp, &bi)
	if err != nil {
		return nil, err
	}
	return &bi, nil
}

func GetLastSuccessfulBuildInfo(env Env, job string) (*BuildInfo, error) {
	code, rsp, _, err := req(env, "POST", "job/"+job+"/lastSuccessfulBuild/api/json", []byte{})
	if err != nil {
		panic(err)
	}
	if code != 200 {
		return nil, errors.New("failed to get job details,code" + strconv.Itoa(code) + ", " + string(rsp))
	}
	var bi BuildInfo
	err = json.Unmarshal(rsp, &bi)
	if err != nil {
		return nil, err
	}
	return &bi, nil
}

func Build(env Env, job string, query string) (error, string) {
	target := "/build"
	if len(query) > 0 {
		target = "/buildWithParameters?" + query
	}
	code, rsp, headers, err := req(env, "POST", "job/"+job+target, []byte{})
	if err != nil {
		return err, ""
	}
	if code != 201 {
		return errors.New("failed to start job details,code" + strconv.Itoa(code) + ", " + string(rsp)), ""
	}
	location := headers["Location"][0]
	splitedUrl := strings.Split(location, "/")
	return nil, splitedUrl[len(splitedUrl)-2]

}

func CancelQueue(env Env, id int) {
	req(env, "POST", "queue/cancelItem?id="+strconv.Itoa(id), []byte{})
}
func CancelJob(env Env, job string, id int) (string, error) {
	code, _, _, err := req(env, "POST", "job/"+job+"/"+strconv.Itoa(id)+"/stop", []byte{})
	if err != nil {
		panic(err)
	}
	if code != 200 {
		return "", errors.New("failed to cancel the job,code" + strconv.Itoa(code))
	}
	bi, err := GetBuildInfo(env, job, id)
	if err != nil {
		return "", err
	}
	return bi.Result, nil
}

func GetConsoleUrl(env Env, job string, id int) string {
	url := env.Url
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return url + "job/" + job + "/" + strconv.Itoa(id) + "/console"
}

func Console(env Env, job string, id int, start string) (string, string, error) {
	//web-rpm-build-manual/149/logText/progressiveHtml
	code, rsp, h, err := req(env, "POST", "job/"+job+"/"+strconv.Itoa(id)+"/logText/progressiveHtml", []byte("start="+start))
	if err != nil {
		return "", "", err
	}
	if code != 200 {
		return "", "", errors.New("code = " + strconv.Itoa(code))
	}
	//fmt.Println(h)
	return string(rsp), h["X-Text-Size"][0], nil
}

func GetQueueInfo(env Env, id int) (error, QueueInfo) {
	var queueInfo QueueInfo
	code, rsp, _, err := req(env, "POST", "/queue/item/"+strconv.Itoa(id)+"/api/json", []byte{})
	if err != nil {
		panic(err)
	}
	if code != 200 {
		return errors.New("failed to get queue details,code" + strconv.Itoa(code) + ", " + string(rsp)), QueueInfo{}
	}
	err = json.Unmarshal(rsp, &queueInfo)
	if err != nil {
		return errors.New("failed to get Job information"), QueueInfo{}
	}
	return nil, queueInfo
}

func GetQueues(env Env) Queues {
	var queues Queues
	code, rsp, _, err := req(env, "POST", "/queue/api/json", []byte{})
	if err != nil {
		panic(err)
	}
	if code != 200 {
		panic("failed to get queue list,code" + strconv.Itoa(code) + ", " + string(rsp))
	}
	err = json.Unmarshal(rsp, &queues)
	if err != nil {
		panic("failed to get Queue list")
	}
	return queues
}

// 获取所有正在运行的 build（即 building=true 的 build）
type RunningBuild struct {
	JobName  string
	BuildNum int
	Result   string
	URL      string
	Duration int64 `json:"duration"`
}

// 使用计算机API直接获取正在执行的构建（推荐）
func GetRunningBuildsByComputer(env Env) ([]RunningBuild, error) {
	apiPath := "/computer/api/json?tree=computer[displayName,executors[currentExecutable[url,number,timestamp,duration,fullDisplayName]]]"
	code, rsp, _, err := req(env, "GET", apiPath, []byte{})
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("failed to get computer info, code %d", code)
	}

	var data struct {
		Computer []struct {
			DisplayName string `json:"displayName"`
			Executors   []struct {
				CurrentExecutable *struct {
					URL             string `json:"url"`
					Number          int    `json:"number"`
					FullDisplayName string `json:"fullDisplayName"`
					Duration        int64  `json:"duration"`
				} `json:"currentExecutable"`
			} `json:"executors"`
		} `json:"computer"`
	}

	err = json.Unmarshal(rsp, &data)
	if err != nil {
		return nil, err
	}

	var running []RunningBuild
	for _, computer := range data.Computer {
		for _, executor := range computer.Executors {
			if executor.CurrentExecutable != nil {
				// 从 fullDisplayName 解析 job name，格式如: "job-name #123"
				fullName := executor.CurrentExecutable.FullDisplayName
				jobName := ""
				if idx := strings.LastIndex(fullName, " #"); idx != -1 {
					jobName = fullName[:idx]
				} else {
					jobName = fullName
				}

				running = append(running, RunningBuild{
					JobName:  jobName,
					BuildNum: executor.CurrentExecutable.Number,
					Result:   "BUILDING", // 正在执行中
					URL:      executor.CurrentExecutable.URL,
					Duration: executor.CurrentExecutable.Duration,
				})
			}
		}
	}
	return running, nil
}

//	curl 'http://10.1.106.141:8080/job/report-web/descriptorByName/net.uaznia.lukanus.hudson.plugins.gitparameter.GitParameterDefinition/fillValueItems?param=adminBranch' \
//	  -X 'POST' \
//	  -H 'Accept: */*' \
//	  -H 'Accept-Language: zh-CN,zh;q=0.9,en;q=0.8' \
//	  -H 'Connection: keep-alive' \
//	  -H 'Content-Length: 0' \
//	  -H 'Content-Type: application/x-www-form-urlencoded' \
//	  -b 'jenkins-timestamper-offset=-28800000; remember-me=YWRtaW46MTc1OTg5NTA4MDU5OTo0MWI4NDZjYzFkNjg0YTFlMmExOTEzYTkxMDczOWRkODdmODE0MjgxY2I1MmNiYjk0NWNjZDcwYzZlNjAyODUx; JSESSIONID.d81c5b23=node017iwlciq0ccko15fez9e7es4vx1821.node0' \
//	  -H 'Jenkins-Crumb: e0c529a2b0dd44e5df35eef7837952a1437d92e707937dc5e967f3623095892c' \
//	  -H 'Origin: http://10.1.106.141:8080' \
//	  -H 'Referer: http://10.1.106.141:8080/job/report-web/build?delay=0sec' \
//	  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36' \
func GetGitParameterDefinitionItems(env Env, jobName string, paramName string) (error, *GitParameterDefinition) {
	code, rsp, _, err := req(env, "POST", "job/"+jobName+"/descriptorByName/net.uaznia.lukanus.hudson.plugins.gitparameter.GitParameterDefinition/fillValueItems?param="+paramName, []byte{})
	if err != nil {
		return err, nil
	}
	if code != 200 {
		return errors.New("failed to get job details,code" + strconv.Itoa(code) + ", " + string(rsp)), nil
	}
	var items GitParameterDefinition
	err = json.Unmarshal(rsp, &items)
	if err != nil {
		return errors.New("failed to get Job information"), nil
	}
	return nil, &items

}
