.PHONY: $(MAKECMDGOALS)

# `make setup` will be used after cloning or downloading to fulfill
# dependencies, and setup the the project in an initial state.
# This is where you might download rubygems, node_modules, packages,
# compile code, build container images, initialize a database,
# anything else that needs to happen before your server is started
# for the first time
setup:
  sudo apt-get update
  sudo apt-get install build-essential
  sudo apt-get update -y && sudo apt-get upgrade -y && sudo apt-get install build-essential mysql-server libmysqlclient-dev npm -y  
  sudo mysql -e "SET PASSWORD FOR root@localhost = PASSWORD('pass');FLUSH PRIVILEGES;"  
  sudo mysql -e "DELETE FROM mysql.user WHERE User='';"  
  sudo mysql -e "DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');"  
  sudo mysql -e "DROP DATABASE test;DELETE FROM mysql.db WHERE Db='test' OR Db='test_%';"  
  sudo mysql -u root -ppass -e "CREATE USER 'ubuntu'@'localhost' IDENTIFIED BY 'pass';GRANT ALL PRIVILEGES ON *.* TO 'ubuntu'@'localhost';FLUSH PRIVILEGES;"
  sudo mysql -u root -ppass -e "create database blogs; use blogs; create table ShortLink(Shortened varchar(8) PRIMARY KEY,Original varchar(255), Expiry int, Created  datetime, Hits int);"
  sudo apt install golang-go -y
  mkdir -p ~/go_projects/{bin,src,pkg}
  cd ~/go_projects
  ls
  export  PATH=$PATH:/usr/local/go/bin 
  export GOPATH="$HOME/go_projects"
  export GOBIN="$GOPATH/bin"
  source ~/.bash_profile
# `make server` will be used after `make setup` in order to start
# an http server process that listens on any unreserved port
#	of your choice (e.g. 8080). 
server:
  include config.json
  go build -o bin/main main.go
  ./main

# `make test` will be used after `make setup` in order to run
# your test suite.
test:
  cd shortener
  go test -cover
