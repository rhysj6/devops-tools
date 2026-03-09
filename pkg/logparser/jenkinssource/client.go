package jenkinssource

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

type Client interface {
	IsJobURL(string) bool
	GetJobNameAndNumberFromURL(string) (string, int, error)
	GetBuildLogs(jobName string, buildNumber int) (io.ReadCloser, error)
	GetBuildLogsWithContext(ctx context.Context, jobName string, buildNumber int) (io.ReadCloser, error)
}

var _ Client = (*JenkinsClient)(nil)

type JenkinsClient struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (j JenkinsClient) IsJobURL(s string) bool {
	jenkinsURL := strings.TrimSuffix(j.URL, "/")
	return strings.HasPrefix(s, jenkinsURL+"/job/")
}

func (j JenkinsClient) GetJobNameAndNumberFromURL(u string) (name string, buildNumber int, err error) {

	if !j.IsJobURL(u) {
		return "", 0, fmt.Errorf("%v is not a Jenkins job url", u)
	}
	jenkinsURL := strings.TrimSuffix(j.URL, "/")

	s := strings.TrimPrefix(u, jenkinsURL+"/job/")

	folderPathParts := strings.Split(s, "/job/")

	if len(folderPathParts) > 1 {
		for i, folder := range folderPathParts {
			if i+1 == len(folderPathParts) {
				continue
			}
			decodedFolder, err := url.QueryUnescape(folder)
			if err != nil {
				return "", 0, err
			}
			name += decodedFolder + "/"
		}
	}

	urlParts := strings.Split(folderPathParts[len(folderPathParts)-1], "/")
	if len(urlParts) < 2 || urlParts[0] == "" || urlParts[1] == "" {
		return "", 0, fmt.Errorf("%v is not a valid Jenkins job build url", u)
	}

	jobName, err := url.QueryUnescape(urlParts[0])
	if err != nil {
		return "", 0, err
	}
	name += jobName

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

	var baseURL strings.Builder
	baseURL.WriteString(strings.TrimSuffix(j.URL, "/"))

	for pathPart := range strings.SplitSeq(jobName, "/") {
		baseURL.WriteString("/job/" + url.PathEscape(pathPart))
	}

	logURL := fmt.Sprintf("%s/%d/consoleText", baseURL.String(), buildNumber)
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
		_ = resp.Body.Close()
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("failed to fetch build logs: status %d", resp.StatusCode)
	}

	return resp.Body, nil
}
