# -*- mode: ruby -*-
# vi: set ft=ruby :
#

LINUX_BASE_BOX = "bento/ubuntu-16.04"
FREEBSD_BASE_BOX = "jen20/FreeBSD-11.1-RELEASE"

SOURCE="/opt/gopath/src/github.com/opentable/sous"
HOST_VAGRANT_DIR="./dev_support/vagrant"

Vagrant.configure(2) do |config|
	config.vm.provision :docker
	config.vm.provision :docker_compose

	config.vm.define "linux", autostart: true, primary: true do |vmCfg|
		vmCfg.vm.box = LINUX_BASE_BOX
		vmCfg.vm.hostname = "linux"
		vmCfg = configureProviders vmCfg,
			cpus: suggestedCPUCores()

		vmCfg = configureLinuxProvisioners(vmCfg)

		vmCfg.vm.provision "shell", privileged: true, inline:
			"cd " + SOURCE + " && go install"

		vmCfg.vm.provision "shell", privileged: true, inline:
			"cd " + SOURCE + "/dev_support/sous_qa_setup && go install"

		vmCfg.vm.provision "shell", inline: "sous_qa_setup --compose-dir=$GOPATH/src/github.com/opentable/sous/integration/test-registry/ > /home/vagrant/qa_desc.json"

		vmCfg.vm.provision "shell", inline: "cd " + SOURCE + "/integration/test-registry && docker-compose down"

		vmCfg.vm.provision :docker_compose, yml: SOURCE + "/integration/test-registry/docker-compose.yml", run: "always"

		# Copy server config to a nonstandard location (need to set
		# SOUS_CONFIG_DIR=/home/vagrant/.config/sous-server when starting server)

		vmCfg.vm.synced_folder '.', SOURCE

		vmCfg.vm.provision "shell", path: "scripts/sous-server-cluster.sh"

		# Expose sous server to the host.
        vmCfg.vm.network "forwarded_port", guest: 5550, host: 5550, auto_correct: true
        vmCfg.vm.network "forwarded_port", guest: 5551, host: 5551, auto_correct: true
        vmCfg.vm.network "forwarded_port", guest: 5552, host: 5552, auto_correct: true

	end
end

def configureLinuxProvisioners(vmCfg)
	vmCfg.vm.provision "shell",
		privileged: true,
		inline: 'rm -f /home/vagrant/linux.iso'

	vmCfg.vm.provision "shell",
		privileged: true,
		path: './dev_support/vagrant/install-go.sh'

	vmCfg.vm.provision "shell",
		privileged: true,
		inline: 'echo "deb http://apt.postgresql.org/pub/repos/apt/ xenial-pgdg main" >> /etc/apt/sources.list.d/pgdg.list && wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add'

	vmCfg.vm.provision "shell",
		privileged: true,
		inline: 'apt-get update && apt-get install -y jq postgresql-10'

	return vmCfg
end

def configureProviders(vmCfg, cpus: "2", memory: "2048")
	vmCfg.vm.provider "virtualbox" do |v|
		v.memory = memory
		v.cpus = cpus
	end

	["vmware_fusion", "vmware_workstation"].each do |p|
		vmCfg.vm.provider p do |v|
			v.enable_vmrun_ip_lookup = false
			v.vmx["memsize"] = memory
			v.vmx["numvcpus"] = cpus
		end
	end

	vmCfg.vm.provider "virtualbox" do |v|
		v.memory = memory
		v.cpus = cpus
	end

	return vmCfg
end

def suggestedCPUCores()
	case RbConfig::CONFIG['host_os']
	when /darwin/
		Integer(`sysctl -n hw.ncpu`) / 2
	when /linux/
		Integer(`cat /proc/cpuinfo | grep processor | wc -l`) / 2
	else
		2
	end
end
