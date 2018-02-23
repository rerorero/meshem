# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|

  hosts = [
    {name: "controller",    ip: "192.168.34.61"},
    {name: "frontend",      ip: "192.168.34.62"},
    {name: "products01",    ip: "192.168.34.63"},
    {name: "products02",    ip: "192.168.34.64"},
    {name: "review01",      ip: "192.168.34.65"},
    {name: "review02",      ip: "192.168.34.66"}
  ]

  def configure(c, hostname, addr)
    c.vm.box = "bento/centos-7.3"
    c.vm.network :private_network, ip: addr
    c.vm.provision "shell", inline: "sudo systemctl restart network"
    c.vm.box_check_update = false
    c.vm.provider :virtualbox do |vb|
      vb.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
    end
  end

  hosts.each {|host|
    config.vm.define host[:name] do |c|
      configure(c, host[:name], host[:ip])
    end
  }
end
