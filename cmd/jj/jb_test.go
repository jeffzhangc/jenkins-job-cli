package jj

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getEnv(name string) Env {
	for _, e := range GetEnvs() {
		if e.Name == EName(name) {
			return e
		}
	}
	panic("no any envs")
}

func TestEnv(t *testing.T) {
	varEnv := Env{
		Name: "uat",
		Url:  "http://jenkins.uat.example.com",
		Type: "basic",
	}
	t.Errorf("env: %v", varEnv)
}

func TestInit(t *testing.T) {
	e := getEnv("aicc")
	time.Sleep(time.Second * 3)
	initBundle(e)
	er, ff := GetJobInfo(e, "cc-cas")
	// fmt.Println("jd ", e)
	t.Errorf("env: %v,%v", er, ff)
}

func TestGetLastSuccessfulBuildDuration(t *testing.T) {
	rsp, err := GetLastSuccessfulBuildInfo(getEnv("pi"), "config-deploy-manual")
	assert.NoError(t, err)
	fmt.Println("jd ", rsp)
}

func TestCancelJob(t *testing.T) {
	status, err := CancelJob(getEnv("uat"), "web-rpm-build-manual", 40)
	assert.NoError(t, err)
	fmt.Println(status)
}

func TestCancelQueue(t *testing.T) {
	CancelQueue(getEnv("uat"), 657)

}
