package jenkins

type JenkinsClient struct {
	Url      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}
