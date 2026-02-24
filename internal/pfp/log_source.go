package pfp

import "io"

type LogSource interface {
	// Get the logs to be parsed. This should return an io.ReadCloser that can be used to read the logs line by line.
	GetLogs() (io.ReadCloser, error)
}

type RecursiveLogSource interface {
	LogSource
	// GetDownstreamFailedBuildRule returns the rule that should match a log line to be considered the results of a failed downstream build.
	GetDownstreamFailedBuildRule() *Rule
	// GetDownstreamFailedBuildLogs returns the logs of a downstream failed build given a ParseMatch that contains the rule that matched the failed downstream build and the log lines that matched that rule.
	GetDownstreamFailedBuildLogs(*ParseMatch) (io.ReadCloser, error)
	// GetMaxRecursionDepth returns the maximum recursion depth for parsing downstream failed builds. This is to prevent infinite recursion in case of circular dependencies between builds. If not implemented, it defaults to 3, meaning that only one level of downstream failed builds will be parsed.
	GetMaxRecursionDepth() int
}
