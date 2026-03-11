package jenkinssource

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Client defines the Jenkins operations required by this package. This is public to allow us to mock Jenkins interactions tests.
type Client interface {
	IsJobURL(string) bool
	GetJobNameAndNumberFromURL(string) (string, int, error)
	GetBuildLogs(ctx context.Context, jobName string, buildNumber int) (io.ReadCloser, error)
}

var _ Client = (*JenkinsClient)(nil)

// JenkinsClient is an HTTP client for retrieving Jenkins build logs.
type JenkinsClient struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// IsJobURL reports whether a URL points to a job under this Jenkins instance.
func (j JenkinsClient) IsJobURL(s string) bool {
	jenkinsURL := strings.TrimSuffix(j.URL, "/")
	return strings.HasPrefix(s, jenkinsURL+"/job/")
}

// GetJobNameAndNumberFromURL parses a Jenkins job URL into name and build number.
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

// GetBuildLogs fetches console text for a Jenkins job build.
func (j JenkinsClient) GetBuildLogs(ctx context.Context, jobName string, buildNumber int) (io.ReadCloser, error) {
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

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("failed to fetch build logs: status %d", resp.StatusCode)
	}

	return resp.Body, nil
}
