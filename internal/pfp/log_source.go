package pfp

import "io"

type LogSource interface {
	// Get the logs to be parsed. This should return an io.ReadCloser that can be used to read the logs line by line.
	GetLogs() (io.ReadCloser, error)
}

type RecursiveLogSource interface {
	LogSource
	// GetDownstreamErrorRule returns the rule that should match a log line to be an indication of a downstream error. E.g. a failed build in a downstream job in Jenkins.
	GetDownstreamErrorRule() *Rule
	// GetDownstreamErrorLogs returns the logs of a downstream error given a ParseMatch that contains the rule that indicates a downstream error. E.g. the logs of the failed downstream build in Jenkins given a ParseMatch that matched the downstream failed build rule.
	GetDownstreamErrorLogs(*ParseMatch) (io.ReadCloser, error)
	// GetMaxRecursionDepth returns the maximum recursion depth for parsing downstream logs. E.g. if a downstream log indicates another downstream error, should we parse the logs of that downstream error as well, and so on, up to the maximum recursion depth.
	GetMaxRecursionDepth() int
}
