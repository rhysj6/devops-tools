package pfp

import (
	"fmt"
	"io"
	"strconv"

	"github.com/rhysj6/devops-tools/internal/jenkins"
)

var _ RecursiveLogSource = (*JenkinsLogSource)(nil)

type JenkinsLogSource struct {
	client      jenkins.Client
	jobName     string
	buildNumber int
}

func NewJenkinsLogSource(client jenkins.Client, cmdArgs []string) (*JenkinsLogSource, error) {

	j := &JenkinsLogSource{
		client: client,
	}

	if len(cmdArgs) == 1 {
		jobName, buildNumber, err := client.GetJobNameAndNumberFromURL(cmdArgs[0])
		if err != nil {
			return nil, err
		}
		j.jobName = jobName
		j.buildNumber = buildNumber
	} else if len(cmdArgs) == 2 {
		j.jobName = cmdArgs[0]
		buildNumber, err := strconv.Atoi(cmdArgs[1])
		if err != nil {
			return nil, err
		}
		j.buildNumber = buildNumber
	} else {
		return nil, fmt.Errorf("Expected either 1 or 2 arguments, got %v", len(cmdArgs))
	}

	return j, nil
}

func (j *JenkinsLogSource) GetLogs() (io.ReadCloser, error) {
	return j.client.GetBuildLogs(j.jobName, j.buildNumber)
}

func (j *JenkinsLogSource) GetDownstreamFailedBuildRule() *Rule {
	return nil
}

func (j *JenkinsLogSource) GetDownstreamFailedBuildLogs(*ParseMatch) (io.ReadCloser, error) {
	return nil, nil
}

func (j *JenkinsLogSource) GetMaxRecursionDepth() int {
	return 3
}
