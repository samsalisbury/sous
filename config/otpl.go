package config

// OTPLFlags set options for sniffing otpl-deploy configuration during manifest
// initialisation.
type OTPLFlags struct {
	UseOTPLDeploy    bool `flag:"use-otpl-deploy"`
	IgnoreOTPLDeploy bool `flag:"ignore-otpl-deploy"`
}
