package jj

import (
	"crypto/tls"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

var homeDir string
var Debug = false

const cacheFile = "cache"
const configFile = "config.yaml"
const historyFile = "history.yaml"
const defaultHistoryLimit = 10

var config Config
var bundles []*Bundle
var mutex sync.Mutex
var ErrNoEnv = errors.New("no env")
var ErrNoJob = errors.New("no job")

func init() {
	home, _ := os.UserHomeDir()
	newHomeDir := home + "/.config/jj/"
	oldHomeDir := home + "/.jj/"
	migrateLegacyConfig(oldHomeDir, newHomeDir)
	homeDir = newHomeDir
	initConfig()
}

// migrateLegacyConfig relocates a legacy config directory into newDir when
// the legacy location still holds a config file but the new location does
// not. Rename is attempted first (atomic on the same filesystem) and a
// copy + RemoveAll is used as a cross-device fallback. The function never
// blocks startup: failures are reported to stderr and the caller continues
// with whatever already exists in newDir.
func migrateLegacyConfig(oldDir, newDir string) {
	if _, err := os.Stat(oldDir); err != nil {
		return
	}
	if _, err := os.Stat(oldDir + configFile); err != nil {
		return
	}
	if _, err := os.Stat(newDir + configFile); err == nil {
		fmt.Fprintf(os.Stderr, "warning: legacy config at %s found but %s already exists; leaving both untouched\n", oldDir, newDir+configFile)
		return
	}
	if err := os.MkdirAll(newDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to prepare %s: %v\n", newDir, err)
		return
	}
	if err := os.Rename(oldDir, newDir); err == nil {
		fmt.Fprintf(os.Stderr, "notice: migrated legacy config from %s to %s\n", oldDir, newDir)
		return
	}
	if err := os.CopyFS(newDir, os.DirFS(oldDir)); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to copy legacy config from %s to %s: %v\n", oldDir, newDir, err)
		return
	}
	if err := os.RemoveAll(oldDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to remove legacy config dir %s: %v\n", oldDir, err)
		return
	}
	fmt.Fprintf(os.Stderr, "notice: migrated legacy config from %s to %s\n", oldDir, newDir)
}

func Init(envName string) Env {
	err, env := GetEnv(envName)
	if err != nil {
		panic(ErrNoEnv)
	}
	initBundle(env)
	return env

}

type EName string
type EType string

func (t EType) String() string {
	if t == EType("n") {
		return "none"
	}
	// return "undefined"
	return string(t)
}

type Config struct {
	Use  EName `yaml:"use"`
	Envs []Env `yaml:"envs"`
	// HistoryLimit caps how many non-pinned quick commands are retained.
	// 0 or negative means defaultHistoryLimit (10).
	HistoryLimit int `yaml:"history_limit"`
	// HistoryAliasTemplate controls how auto-generated quick command
	// aliases are produced. Supports {job}, {env}, {hash}, {seq} and
	// {param.<name>} placeholders. Empty means "<job>_<hash>".
	HistoryAliasTemplate string `yaml:"history_alias_template"`
}

type Bundle struct {
	Name     EName     `json:"name"`
	Views    []View    `json:"views"`
	JobsInfo []JobInfo `json:"JobsInfo"`
}

type Job struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type View struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Desc string `json:"desc"`
	Jobs []Job  `json:"jobs"`
}

type BuildInfo struct {
	Id      string `json:"id"`
	Actions []struct {
		Class      string `json:"_class"`
		Parameters []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"parameters,omitempty"`
		Causes []struct {
			ShortDescription string `json:"shortDescription"`
			UpstreamBuild    int    `json:"upstreamBuild"`
			UpstreamProject  string `json:"upstreamProject"`
			UpstreamURL      string `json:"upstreamUrl"`
		} `json:"causes,omitempty"`
	} `json:"actions"`
	Duration int    `json:"duration"`
	Building bool   `json:"building"`
	Result   string `json:"result"`
	QueueId  int    `json:"queueId"`
}

type ParameterDefinitions struct {
	DefaultParameterValue struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"defaultParameterValue"`
	Description string   `json:"description"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Choices     []string `json:"choices,omitempty"`
}

type Env struct {
	Url    string `yaml:"url"`
	Name   EName  `yaml:"name"`
	Type   EType  `yaml:"type"`
	Login  string `yaml:"login"`
	Secret string `yaml:"secret"`
}

type JobInfo struct {
	Name               string `json:"name"`
	NextBuildNumber    int    `json:"nextBuildNumber"`
	DownstreamProjects []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"downstreamProjects"`
	LastBuild struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	} `json:"lastBuild"`
	LastCompletedBuild struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	}
	InQueue  bool `json:"inQueue"`
	Property []struct {
		ParameterDefinitions []ParameterDefinitions `json:"parameterDefinitions,omitempty"`
	} `json:"property"`
}

type QueueInfo struct {
	Actions []struct {
		Parameters []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"parameters,omitempty"`
		Causes []struct {
			ShortDescription string      `json:"shortDescription"`
			UserID           interface{} `json:"userId"`
			UserName         string      `json:"userName"`
		} `json:"causes,omitempty"`
	} `json:"actions"`
	Blocked      bool   `json:"blocked"`
	Buildable    bool   `json:"buildable"`
	ID           int    `json:"id"`
	InQueueSince int64  `json:"inQueueSince"`
	Params       string `json:"params"`
	Stuck        bool   `json:"stuck"`
	Task         struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Color string `json:"color"`
	} `json:"task"`
	URL        string      `json:"url"`
	Why        interface{} `json:"why"`
	Cancelled  bool        `json:"cancelled"`
	Executable struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	} `json:"executable"`
}

type Queues struct {
	DiscoverableItems []interface{} `json:"discoverableItems"`
	Items             []struct {
		Actions []struct {
			Parameters []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"parameters,omitempty"`
			Causes []struct {
				ShortDescription string `json:"shortDescription"`
				UpstreamBuild    int    `json:"upstreamBuild"`
				UpstreamProject  string `json:"upstreamProject"`
				UpstreamURL      string `json:"upstreamUrl"`
			} `json:"causes,omitempty"`
		} `json:"actions"`
		Blocked                    bool   `json:"blocked"`
		Buildable                  bool   `json:"buildable"`
		BuildableStartMilliseconds int64  `json:"buildableStartMilliseconds"`
		ID                         int    `json:"id"`
		InQueueSince               int64  `json:"inQueueSince"`
		Params                     string `json:"params"`
		Stuck                      bool   `json:"stuck"`
		Task                       struct {
			Color string `json:"color"`
			Name  string `json:"name"`
			URL   string `json:"url"`
		} `json:"task"`
		URL string `json:"url"`
		Why string `json:"why"`
	} `json:"items"`
}

type GitParameterDefinition struct {
	Values []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"values"`
	Class string `json:"_class"`
}

type QuickRunCmdDefinition struct {
	Alias        string `json:"alias" yaml:"alias"`
	JobName      string `json:"jobName" yaml:"jobName"`
	Cmd          string `json:"cmd" yaml:"cmd"`
	// ParamsHash is a stable digest of the parameter set, used to dedupe
	// history entries written by automatic command save.
	ParamsHash string `json:"paramsHash,omitempty" yaml:"paramsHash,omitempty"`
	Executimes   int    `json:"executimes" yaml:"executimes"`
	LastExecTime int64  `json:"lastExecTime" yaml:"lastExecTime"`
	Env          string `json:"env" yaml:"env"`
	LastJobUrl   string `json:"lastJobUrl" yaml:"lastJobUrl"`
	// Pinned entries are excluded from automatic eviction.
	Pinned bool `json:"pinned,omitempty" yaml:"pinned,omitempty"`
}

func initConfig() {
	data, err := os.ReadFile(homeDir + configFile)
	if err != nil {
		config.HistoryLimit = defaultHistoryLimit
		return
	}
	err = yaml.Unmarshal(data, &config)
	if config.HistoryLimit <= 0 {
		config.HistoryLimit = defaultHistoryLimit
	}
	changed := false
	for {
	Repeat:
		for i, env1 := range config.Envs {
			for j, env2 := range config.Envs {
				if env1.Name == env2.Name && i != j {
					config.Envs[len(config.Envs)-1], config.Envs[j] = config.Envs[j], config.Envs[len(config.Envs)-1]
					config.Envs = config.Envs[:len(config.Envs)-1]
					changed = true
					break Repeat
				}
			}
		}
		if changed {
			SetConf()
			changed = false
		} else {
			break
		}
	}
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

func setViews(env Env, views []View) {
	mutex.Lock()
	defer mutex.Unlock()
	bundle := GetBundle(env)
	if bundle == nil {
		bundle = &Bundle{Name: env.Name, Views: views, JobsInfo: []JobInfo{}}
		bundles = append(bundles, bundle)
	} else {
		bundle.Views = views
	}
	updateCache(env, bundle)
}

func initBundle(env Env) {
	var bundle Bundle
	cachebin, err := os.ReadFile(homeDir + cacheFile + "." + string(env.Name))
	if err != nil {
		fetchBundle(env, false)
	} else {
		err = json.Unmarshal(cachebin, &bundle)
		if err != nil {
			fmt.Println("failed to Unmarshal cache info, ", err.Error())
		}
		mutex.Lock()
		defer mutex.Unlock()
		bundles = append(bundles, &bundle)
		go fetchBundle(env, true)
	}
}

func fetchBundle(env Env, async bool) {
	code, rspbin, _, err := req(env, "POST", "api/json", []byte{})
	if err != nil {
		if !async {
			fmt.Printf("error: %s\n", err)
		}
		return
	}
	if code != 200 {
		if !async {
			fmt.Println("failed to get job details:code " + strconv.Itoa(code) + " " + env.Login)
		}

		return
	}
	var rsp struct {
		Views []View `json:"views"`
	}
	err = json.Unmarshal(rspbin, &rsp)
	if err != nil {
		panic(err)
	}
	for i, view := range rsp.Views {
		code, vrspbin, _, err := req(env, "POST", "view/"+view.Name+"/api/json", []byte{})
		if err != nil {
			panic(err)
		}
		if code != 200 {
			panic("failed to get view details,code" + strconv.Itoa(code) + ", " + string(rspbin))
		}
		err = json.Unmarshal(vrspbin, &rsp.Views[i])
		if err != nil {
			panic(err)
		}
	}
	setViews(env, rsp.Views)

}

func updateCache(env Env, bundle *Bundle) {
	cachebin, err := json.Marshal(bundle)
	if err == nil {
		path := homeDir + cacheFile + "." + string(env.Name)
		os.WriteFile(path, cachebin, 0644)
	} else {
		fmt.Println("failed to Marshal cache info, ", err.Error())
	}
}

//	func reqPOST(env Env, method, path string, body []byte) (int, []byte, map[string][]string, error) {
//		return req(env, method, path, body)
//	}
func req(env Env, method, path string, body []byte) (int, []byte, map[string][]string, error) {
	base_url := env.Url
	if base_url[len(base_url)-1:] != "/" {
		base_url = base_url + "/"
	}

	url := base_url + path
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: time.Second * 30}
	request, err := http.NewRequest(method, url, strings.NewReader(string(body)))
	if err != nil {
		return 0, nil, nil, err
	}
	request.Header.Add("Accept-Language", "en-us")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if env.Type == "a" {
		request.SetBasicAuth(env.Login, env.Secret)
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, nil, nil, err
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, nil, nil, err
	}

	if Debug {
		fmt.Println("req: ", url, "\t data: ", string(body), "type:", response.Header.Get("Content-Type"), "\n rsp: ", string(contents))
	}
	return response.StatusCode, contents, response.Header, nil
}

func GetQuickRunCmdList() []QuickRunCmdDefinition {
	fileName := path.Join(homeDir, historyFile)
	// Read the file，yaml 转换为 实体列表
	quickRunCmdList := []QuickRunCmdDefinition{}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return quickRunCmdList
	}
	err = yaml.Unmarshal(data, &quickRunCmdList)
	if err != nil {
		return quickRunCmdList
	}

	return quickRunCmdList
}

func AddQuickRunCmd(quickRunCmd QuickRunCmdDefinition) {
	quickRunCmdList := GetQuickRunCmdList()
	// alias 重复时覆盖已有条目（保留 pinned 标记与 paramsHash）。
	for _, cmd := range quickRunCmdList {
		if cmd.Alias == quickRunCmd.Alias {
			quickRunCmd.Pinned = cmd.Pinned
			if quickRunCmd.ParamsHash == "" {
				quickRunCmd.ParamsHash = cmd.ParamsHash
			}
			quickRunCmdList[getIndex(quickRunCmdList, cmd.Alias)] = quickRunCmd
			SaveAllHistory(applyHistoryLimit(quickRunCmdList))
			return
		}
	}
	if quickRunCmd.ParamsHash == "" {
		quickRunCmd.ParamsHash = ComputeParamsHash(quickRunCmd.Cmd)
	}
	if quickRunCmd.Alias == "" {
		quickRunCmd.Alias = generateAlias(quickRunCmd, quickRunCmdList)
	}
	quickRunCmdList = append(quickRunCmdList, quickRunCmd)
	SaveAllHistory(applyHistoryLimit(quickRunCmdList))
}

// UpsertQuickRunCmd inserts or updates a quick command by (Env, JobName,
// ParamsHash) — the canonical identity of a saved invocation. When the
// identity matches an existing entry, Executimes/LastExecTime/LastJobUrl
// are refreshed and the alias/pinned flag preserved. When the identity is
// new, an alias is generated automatically and the entry is appended. The
// history limit is enforced after every write so callers do not have to.
//
// When the caller's quickRunCmd.Alias is non-empty, that alias is treated as
// authoritative: the lookup switches to alias-based matching, and the alias
// is preserved verbatim on the stored entry.
func UpsertQuickRunCmd(quickRunCmd QuickRunCmdDefinition) {
	if quickRunCmd.ParamsHash == "" {
		quickRunCmd.ParamsHash = ComputeParamsHash(quickRunCmd.Cmd)
	}
	list := GetQuickRunCmdList()

	// User-provided alias: take precedence over identity-based dedupe.
	if quickRunCmd.Alias != "" {
		for i := range list {
			if list[i].Alias == quickRunCmd.Alias {
				quickRunCmd.Pinned = list[i].Pinned
				quickRunCmd.Executimes = list[i].Executimes + 1
				list[i] = quickRunCmd
				SaveAllHistory(applyHistoryLimit(list))
				return
			}
		}
		list = append(list, quickRunCmd)
		SaveAllHistory(applyHistoryLimit(list))
		return
	}

	for i := range list {
		if list[i].Env == quickRunCmd.Env &&
			list[i].JobName == quickRunCmd.JobName &&
			list[i].ParamsHash == quickRunCmd.ParamsHash {
			quickRunCmd.Alias = list[i].Alias
			quickRunCmd.Pinned = list[i].Pinned
			quickRunCmd.Executimes = list[i].Executimes + 1
			list[i] = quickRunCmd
			SaveAllHistory(applyHistoryLimit(list))
			return
		}
	}
	if quickRunCmd.Alias == "" {
		quickRunCmd.Alias = generateAlias(quickRunCmd, list)
	}
	list = append(list, quickRunCmd)
	SaveAllHistory(applyHistoryLimit(list))
}

// PinQuickRunCmd marks a quick command so it is excluded from automatic
// eviction. Pinning an already-pinned entry is a no-op.
func PinQuickRunCmd(alias string) error {
	list := GetQuickRunCmdList()
	for i := range list {
		if list[i].Alias == alias {
			if list[i].Pinned {
				return nil
			}
			list[i].Pinned = true
			SaveAllHistory(applyHistoryLimit(list))
			return nil
		}
	}
	return fmt.Errorf("alias %q not found", alias)
}

// UnpinQuickRunCmd clears the pinned flag. Unpinning a non-pinned entry
// is a no-op.
func UnpinQuickRunCmd(alias string) error {
	list := GetQuickRunCmdList()
	for i := range list {
		if list[i].Alias == alias {
			if !list[i].Pinned {
				return nil
			}
			list[i].Pinned = false
			SaveAllHistory(applyHistoryLimit(list))
			return nil
		}
	}
	return fmt.Errorf("alias %q not found", alias)
}

// HistoryLimit returns the configured maximum number of non-pinned entries.
func HistoryLimit() int {
	if config.HistoryLimit <= 0 {
		return defaultHistoryLimit
	}
	return config.HistoryLimit
}

// ComputeParamsHash returns a stable digest of the parameter portion of a
// "jj run" command string. Extracted key=value pairs are sorted so that
// argument order does not change the hash; jobs and environments are not
// part of the hash because they are encoded separately in the dedupe key.
func ComputeParamsHash(cmd string) string {
	parts := strings.Fields(cmd)
	pairs := make([]string, 0, len(parts)/2)
	for i := 0; i < len(parts); i++ {
		if parts[i] == "-a" && i+1 < len(parts) {
			pairs = append(pairs, parts[i+1])
			i++
		}
	}
	sort.Strings(pairs)
	h := sha256.Sum256([]byte(strings.Join(pairs, "\x00")))
	return hex.EncodeToString(h[:])[:8]
}

// generateAlias builds a unique "<jobName>_<hash>" alias that does not
// collide with any alias already present in history.
func generateAlias(cmd QuickRunCmdDefinition, history []QuickRunCmdDefinition) string {
	if cmd.ParamsHash == "" {
		cmd.ParamsHash = ComputeParamsHash(cmd.Cmd)
	}
	used := make(map[string]bool, len(history))
	for _, c := range history {
		used[c.Alias] = true
	}
	candidate := renderAliasTemplate(cmd)
	if candidate == "" {
		candidate = cmd.JobName + "_" + cmd.ParamsHash
	}
	if !used[candidate] {
		return candidate
	}
	for i := 2; ; i++ {
		alt := fmt.Sprintf("%s_%d", candidate, i)
		if !used[alt] {
			return alt
		}
	}
}

// RenameQuickRunCmd changes the alias of an existing quick command.
// It validates the new alias, refuses to clobber an existing entry,
// and rewrites history.yaml on success.
func RenameQuickRunCmd(oldAlias, newAlias string) error {
	if err := ValidateAlias(newAlias); err != nil {
		return err
	}
	list := GetQuickRunCmdList()
	idxOld := -1
	for i, c := range list {
		if c.Alias == oldAlias {
			idxOld = i
			continue
		}
		if c.Alias == newAlias {
			return fmt.Errorf("alias %q already exists", newAlias)
		}
	}
	if idxOld < 0 {
		return fmt.Errorf("alias %q not found", oldAlias)
	}
	list[idxOld].Alias = newAlias
	SaveAllHistory(applyHistoryLimit(list))
	return nil
}

// ValidateAlias checks that a user-provided alias is syntactically valid:
// non-empty, ≤ 64 bytes, no leading dash, and limited to alphanumerics,
// underscores, dots and hyphens.
func ValidateAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("alias is empty")
	}
	if len(alias) > 64 {
		return fmt.Errorf("alias too long (max 64 characters)")
	}
	if alias[0] == '-' {
		return fmt.Errorf("alias cannot start with '-'")
	}
	for _, r := range alias {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '.' || r == '-':
		default:
			return fmt.Errorf("alias contains invalid character %q", r)
		}
	}
	return nil
}

// SanitizeAlias renders arbitrary text into a CLI-friendly alias by
// replacing whitespace and shell-special characters with underscores and
// stripping leading dashes.
func SanitizeAlias(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimLeft(b.String(), "-")
}

// extractParams parses "-a key=value" pairs from a command string and
// returns them keyed by parameter name.
func extractParams(cmd string) map[string]string {
	out := map[string]string{}
	parts := strings.Fields(cmd)
	for i := 0; i < len(parts); i++ {
		if parts[i] == "-a" && i+1 < len(parts) {
			kv := strings.SplitN(parts[i+1], "=", 2)
			if len(kv) == 2 {
				out[kv[0]] = kv[1]
			}
			i++
		}
	}
	return out
}

var aliasTemplateRE = regexp.MustCompile(`\{([a-zA-Z][a-zA-Z0-9_.]*)\}`)

// renderAliasTemplate expands the configured history_alias_template for
// the given quick command. Supported placeholders: {job}, {env}, {hash},
// {seq} (always rendered as empty here; the caller adds a numeric suffix
// on collision) and {param.<name>}. An empty or unset template yields an
// empty string so the caller can fall back to "<job>_<hash>".
func renderAliasTemplate(cmd QuickRunCmdDefinition) string {
	tmpl := config.HistoryAliasTemplate
	if tmpl == "" {
		return ""
	}
	params := extractParams(cmd.Cmd)
	return aliasTemplateRE.ReplaceAllStringFunc(tmpl, func(token string) string {
		key := token[1 : len(token)-1]
		switch {
		case key == "job":
			return SanitizeAlias(cmd.JobName)
		case key == "env":
			return SanitizeAlias(cmd.Env)
		case key == "hash":
			return cmd.ParamsHash
		case strings.HasPrefix(key, "param."):
			return SanitizeAlias(params[key[len("param."):]])
		case key == "seq":
			return ""
		default:
			return ""
		}
	})
}

// getIndex returns the index of the entry with the given alias, or -1.
func getIndex(list []QuickRunCmdDefinition, alias string) int {
	for i := range list {
		if list[i].Alias == alias {
			return i
		}
	}
	return -1
}

// applyHistoryLimit enforces the configured history limit while never
// dropping pinned entries. Pinned entries are placed first; non-pinned
// entries are sorted by LastExecTime descending (LRU) and truncated to
// the remaining capacity.
func applyHistoryLimit(list []QuickRunCmdDefinition) []QuickRunCmdDefinition {
	limit := config.HistoryLimit
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	if len(list) <= limit {
		return list
	}
	pinned := make([]QuickRunCmdDefinition, 0)
	nonPinned := make([]QuickRunCmdDefinition, 0, len(list))
	for _, c := range list {
		if c.Pinned {
			pinned = append(pinned, c)
		} else {
			nonPinned = append(nonPinned, c)
		}
	}
	if len(pinned) >= limit {
		return pinned[:limit]
	}
	sort.SliceStable(nonPinned, func(i, j int) bool {
		return nonPinned[i].LastExecTime > nonPinned[j].LastExecTime
	})
	nonPinned = nonPinned[:limit-len(pinned)]
	out := make([]QuickRunCmdDefinition, 0, limit)
	out = append(out, pinned...)
	out = append(out, nonPinned...)
	return out
}

func SaveAllHistory(history []QuickRunCmdDefinition) {
	data, err := yaml.Marshal(&history)
	if err != nil {
		fmt.Println("failed to Marshal quickRunCmdList, ", err.Error())
		return
	}
	os.WriteFile(path.Join(homeDir, historyFile), data, 0644)
}
