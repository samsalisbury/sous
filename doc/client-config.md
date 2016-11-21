# sous client configuration

The sous client is configured either by configuration file or environment variable.
The local configuration file is stored as ~/.config/sous/config.yaml. If the file 
does not exist, a single run of
	$ sous config
is enough to create it.

The configuration options are defined (relative to this file) in ../config/config.go.

To understand the configuration format, use godoc. The element names of the 
struct are the names of the options used by the local configuration file 
~/.config/sous/config.yaml. The "env" tags denote the names of the configuration 
options when configured by environment variable.
	$ go doc github.com/opentable/sous/config Config

One of the elements in the Config struct is the docker.Config struct. 
It is definied in its own package,which can be read with
	$ go doc github.com/opentable/sous/ext/docker Config


