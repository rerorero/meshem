# -*- mode: ruby -*-
# vi: set ft=ruby :
#
Vagrant::DEFAULT_SERVER_URL.replace('https://vagrantcloud.com')

Vagrant.configure("2") do |config|

  hosts = [
    {name: "meshem-meshem",     ip: "192.168.34.61"},
    {name: "meshem-monitor",    ip: "192.168.34.62"},
    {name: "myfront",           ip: "192.168.34.70"},
    {name: "myapp1",            ip: "192.168.34.71"},
    {name: "myapp2",            ip: "192.168.34.72"},
  ]


  def configure(c, hostname, addr)
    c.vm.box = "centos/7"
    c.vm.hostname = hostname
    c.vm.network :private_network, ip: addr
    c.vm.provision "shell", inline: "sudo systemctl restart network"
    c.vm.box_check_update = false
    c.vm.provider :virtualbox do |vb|
      vb.name = hostname
      vb.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
    end
    provision_script=<<EOS
set -eux
sed -i -e 's/^PasswordAuthentication no/PasswordAuthentication yes/' /etc/ssh/sshd_config
sudo systemctl restart sshd.service
sed -i -e 's/^SELINUX=enforcing/SELINUX=disabled/' /etc/selinux/config
EOS
    c.vm.provision "shell", privileged: true, inline: provision_script
  end

  hosts.each {|host|
    config.vm.define host[:name] do |c|
      configure(c, host[:name], host[:ip])
    end
  }

end
