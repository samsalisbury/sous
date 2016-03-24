package dockermachine

import "github.com/opentable/sous/util/shell"

type (
	Client struct {
		Sh *shell.Sh
	}
	Machine struct {
		Driver Driver
	}
	Driver struct {
		IPAddress string
	}
)

func NewClient(sh *shell.Sh) *Client {
	return &Client{sh}
}

func (c *Client) Installed() (bool, error) {
	code, err := c.Sh.ExitCode("docker-machine")
	if err != nil {
		return false, err
	}
	return code == 0, nil
}

func (c *Client) VMs() ([]string, error) {
	return c.Sh.Lines("docker-machine", "ls", "-q")
}

func (c *Client) RunningVMs() ([]string, error) {
	list := []string{}
	vms, err := c.VMs()
	if err != nil {
		return nil, err
	}
	for _, v := range vms {
		status, err := c.Sh.Stdout("docker-machine", "status", v)
		if err != nil {
			return nil, err
		}
		if status == "Running" {
			list = append(list, v)
		}
	}
	return list, nil
}

func (c *Client) HostIP(vm string) (string, error) {
	var dmi *Machine
	if err := c.Sh.JSON(&dmi, "docker-machine", "inspect", vm); err != nil {
		return "", err
	}
	return dmi.Driver.IPAddress, nil
}
