sh -c "curl https://get.docker.io/gpg | apt-key add -"
sh -c "echo deb https://get.docker.io/ubuntu docker main > /etc/apt/sources.list.d/docker.list"
apt-get update
apt-get install lxc-docker -y
sed -i "s/docker -d$/docker -d -H=tcp:\\/\\/0.0.0.0/g" /etc/init/docker.conf
stop docker
initctl reload docker
start docker
