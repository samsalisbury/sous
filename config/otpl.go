package config

// OTPLFlags set options for sniffing otpl-deploy configuration during manifest
// initialisation.
type OTPLFlags struct {
	UseOTPLDeploy    bool `flag:"use-otpl-deploy"`
	IgnoreOTPLDeploy bool `flag:"ignore-otpl-deploy"`
}

const otplFlagsHelp = `
	-use-otpl-deploy
		use existing otpl config in ./config/<cluster>/...
	
	-ignore-otpl-deploy
		ignore existing otpl config in ./config/<cluster>/...

`
