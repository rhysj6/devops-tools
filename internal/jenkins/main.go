package jenkins

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var ErrUnauthorized = errors.New("jenkins authentication failed")

type JenkinsClient struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (j JenkinsClient) IsJobURL(s string) bool {
	return strings.HasPrefix(s, j.URL+"/job/")
}

func (j JenkinsClient) GetJobNameAndNumberFromURL(u string) (name string, buildNumber int, err error) {

	if !j.IsJobURL(u) {
		return "", 0, fmt.Errorf("%v is not a Jenkins job url", u)
	}

	s := strings.TrimPrefix(u, j.URL+"/job/")

	urlParts := strings.Split(s, "/")
	if len(urlParts) < 2 || urlParts[0] == "" || urlParts[1] == "" {
		return "", 0, fmt.Errorf("%v is not a valid Jenkins job build url", u)
	}

	name, err = url.QueryUnescape(urlParts[0])

	if err != nil {
		return "", 0, err
	}

	buildNumber, err = strconv.Atoi(urlParts[1])
	if err != nil {
		return "", 0, err
	}

	return name, buildNumber, nil
}

func (j JenkinsClient) GetBuildLogs(jobName string, buildNumber int) (io.ReadCloser, error) {
	return j.GetBuildLogsWithContext(context.Background(), jobName, buildNumber)
}

func (j JenkinsClient) GetBuildLogsWithContext(ctx context.Context, jobName string, buildNumber int) (io.ReadCloser, error) {
	if strings.TrimSpace(jobName) == "" {
		return nil, fmt.Errorf("job name is required")
	}
	if buildNumber <= 0 {
		return nil, fmt.Errorf("build number must be greater than zero")
	}

	encodedJobName := url.PathEscape(jobName)
	logURL := fmt.Sprintf("%s/job/%s/%d/consoleText", strings.TrimSuffix(j.URL, "/"), encodedJobName, buildNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, logURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(j.Username, j.Password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to fetch build logs: status %d", resp.StatusCode)
	}

	return resp.Body, nil
}
