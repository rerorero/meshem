meshem
=======
[![Build Status](https://travis-ci.org/rerorero/meshem.svg?branch=master)](https://travis-ci.org/rerorero/meshem)

meshem is a simple service mesh control plane application which depends on [Envoy](https://www.envoyproxy.io/). This project consists of the followings.
- [meshem server (xds and API)](https://github.com/rerorero/meshem/releases)
- [meshem CLI (meshemctl)](https://github.com/rerorero/meshem/releases)
- [Ansible playbooks that deploys control plane components.](/ansible)
- [Example ansible playbooks of data planes.](/exampls/ansible)
- [Docker Example.](/exampls/docker)

You can get meshem server and CLI binary from [Github release page](https://github.com/rerorero/meshem/releases). There is also a docker image that contains both of them. Try `docker pull rerorero/meshem`.

This implementation is not production ready as the purpose of meshem is mainly to learn envoy and service mesh.

Vagrant + Ansible example: Oneshot
=======
```
go get github.com/rerorero/meshem
cd $GOPATH/src/github.com/rerorero/meshem
./run.sh
```

Vagrant + Ansible example: Step by step
=======

### Requirments
- Vagrant + Virtual Box
- python

Create all VMs.
```
vagrant up
```

### Launch example services without proxy
Provision example services without envoy proxy. The example consists of `app` service and `front` service, both of them provide HTTP API.
```
cd ./examples/ansible
pip install -r requirements.txt
ansible-playbook -i vagrant site.yml
# default timezone is JST. You can change this by passing 'common_timezone' argument.
# ansible-playbook -i vagrant site.yml -e "common_timezone=UTC"
```
Check the response from front application.
```
curl 192.168.34.70:8080
```
At this point the service dependencies are as follows (`myapp2` is not used yet).
<img src="https://raw.githubusercontent.com/rerorero/meshem/master/images/data-plane1.png" height="70%" width="70%">

### Let's mesh'em
Provision the meshem control plane.
```
cd ./ansible
ansible-playbook -i inventories/vagrant site.yml
```
Launch envoy proxies for each services.
```
cd ./examples/ansible
ansible-playbook -i vagrant data-plane.yml
```

#### Manage meshem
Download meshem CLI binary(meshemctl) from [Github release page](https://github.com/rerorero/meshem/releases) and put it somewhere in your `$PATH`. meshemctl also can be built from source by running `go install github.com/rerorero/meshem/src/meshemctl` command.

Then let us regsiter the app service and front service. 
```
cd ./examples/ansible
export MESHEM_CTLAPI_ENDPOINT="http://192.168.34.61:8091"
meshemctl svc apply app -f ./meshem-conf/app.yaml
meshemctl svc apply front -f ./meshem-conf/front.yaml
```

#### Deploy services as a service mesh
Deploy the front application sot that it uses envoy as egress proxy.
```
ansible-playbook -i vagrant site.yml -e "front_app_endpoint=http://127.0.0.1:9001/"
```
Currently the service dependencies are as follows.

<img src="https://raw.githubusercontent.com/rerorero/meshem/master/images/data-plane2.png" height="70%" width="70%">

The following figure shows the relationship between the control plane and the data plane.

<img src="https://raw.githubusercontent.com/rerorero/meshem/master/images/control-plane.png" height="70%" width="70%">

#### Send a request
Try to send HTTP requests to front proxy several times. From the response you can confirm that the requests are round robin balanced.
```
curl 192.168.34.70:80
```
Envoy metrics and trace results can be displayed.
- Zipkin is running on http://192.168.34.62:9411/
- Grafana is running on http://192.168.34.62:3000/ 
    - Prometheus is running on http://192.168.34.62:9090/ 
    - Dashoboard uses [transferwise/prometheus-envoy-dashboards](https://github.com/transferwise/prometheus-envoy-dashboards). Thanks!

Docker example
=======
Docker exapmle starts containers and build meshem binary from local source code.
```
go get github.com/rerorero/meshem
go get github.com/golang/dep/cmd/dep
cd $GOPATH/src/github.com/rerorero/meshem
dep ensure
cd ./examples/docker
# run.sh starts all of the relevant containers and register example services.
./run.sh
```
