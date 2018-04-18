package singularity

// DeployerOption is an option for configuring singularity deployers.
type DeployerOption func(*deployer)

// OptMaxHTTPReqsPerServer overrides the DefaultMaxHTTPConcurrencyPerServer
// for this server.
func OptMaxHTTPReqsPerServer(n int) DeployerOption {
	return func(d *deployer) { d.ReqsPerServer = n }
}
