package jenkinssource

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/rhysj6/devops-tools/pkg/logparser"
)

var _ logparser.RecursiveLogSource = (*JenkinsLogSource)(nil)

type JenkinsLogSource struct {
	client                    Client
	jobName                   string
	buildNumber               int
	downstreamFailedBuildRule *logparser.Rule
}

func NewJenkinsLogSource(client Client, cmdArgs []string) (*JenkinsLogSource, error) {

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
		return nil, fmt.Errorf("expected either 1 or 2 arguments, got %v", len(cmdArgs))
	}

	return j, nil
}

func (j *JenkinsLogSource) GetLogs() (io.ReadCloser, error) {
	return j.client.GetBuildLogs(j.jobName, j.buildNumber)
}

func (j *JenkinsLogSource) GetDownstreamErrorRule() *logparser.Rule {
	if j.downstreamFailedBuildRule == nil {
		j.downstreamFailedBuildRule = &logparser.Rule{
			Name: "Downstream Failed Jenkins Build",
			Checks: []logparser.LineMatcher{
				{Contains: "completed: FAILURE", Regex: regexp.MustCompile(`(?m)^Build\s+(?P<job>.+?)\s+#(?P<number>\d+)(?::\s*(?P<suffix>.*?))?\s+completed:\s+FAILURE\s*$`)},
			},
			Solution: "If there are no other matches, then look at the logs of the downstream failed build for more information on why the build failed.",
		}
	}
	return j.downstreamFailedBuildRule
}

func (j *JenkinsLogSource) getJobNameAndBuildNumberFromMatch(match *logparser.ParseMatch) (string, int, error) {
	regex := match.Rule.Checks[0].Regex
	if regex == nil {
		return "", 0, fmt.Errorf("regex is nil for downstream failed build rule")
	}
	matchGroups := regex.FindStringSubmatch(match.MatchedLines[0].Content)
	if matchGroups == nil {
		return "", 0, fmt.Errorf("regex did not match log line for downstream failed build rule")
	}
	jobNameIndex := regex.SubexpIndex("job")
	buildNumberIndex := regex.SubexpIndex("number")
	if jobNameIndex == -1 || buildNumberIndex == -1 {
		return "", 0, fmt.Errorf("regex does not contain 'job' or 'number' named groups for downstream failed build rule")
	}
	jobName := matchGroups[jobNameIndex]
	buildNumberStr := matchGroups[buildNumberIndex]
	buildNumber, err := strconv.Atoi(buildNumberStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid build number: %v", err)
	}
	return jobName, buildNumber, nil
}

func (j *JenkinsLogSource) GetDownstreamErrorLogs(match *logparser.ParseMatch) (io.ReadCloser, error) {
	if match.Rule != j.GetDownstreamErrorRule() {
		return nil, fmt.Errorf("match rule does not match downstream failed build rule")
	}
	if len(match.MatchedLines) == 0 {
		return nil, fmt.Errorf("no matched lines in match for downstream failed build rule")
	}
	jobName, buildNumber, err := j.getJobNameAndBuildNumberFromMatch(match)
	if err != nil {
		return nil, err
	}
	return j.client.GetBuildLogs(jobName, buildNumber)
}

func (j *JenkinsLogSource) GetMaxRecursionDepth() int {
	return 3
}
