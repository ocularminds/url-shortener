.PHONY: $(MAKECMDGOALS)

# `make setup` will be used after cloning or downloading to fulfill
# dependencies, and setup the the project in an initial state.
# This is where you might download rubygems, node_modules, packages,
# compile code, build container images, initialize a database,
# anything else that needs to happen before your server is started
# for the first time
#include config.json
PROJECTNAME=url-shortener

setup: go-get
# `make server` will be used after `make setup` in order to start
# an http server process that listens on any unreserved port
#	of your choice (e.g. 8080). 
server:
	go run main.go

# `make test` will be used after `make setup` in order to run
# your test suite.
test:
	cd shortner && go test -race -coverprofile=coverage.txt -covermode=atomic && cd ..
go-compile: go-get go-build

go-get:
	echo "  >  Checking if there is any missing dependencies..."
	go get $(get)

go-build:
	@echo "  >  Building binary..."
	go build -o $(PROJECTNAME)

go-generate:
	@echo	"  >  Generating dependency files..."
	go generate $(generate)

install-db: 
	db-check db-get db-update db-pass db-user db-drop db-create db-install
db-check: 
	apt-get update
db-get: 
	apt-get install build-essential
db-update: 
	apt-get update -y && sudo apt-get upgrade -y && sudo apt-get install build-essential mysql-server libmysqlclient-dev npm -y  
db-pass: 
	mysql -e "SET PASSWORD FOR root@localhost = PASSWORD('pass');FLUSH PRIVILEGES;"  
db-user: 
	mysql -e "DELETE FROM mysql.user WHERE User='';"  
db-drop:
	mysql -e "DROP DATABASE test;DELETE FROM mysql.db WHERE Db='test' OR Db='test_%';"  
db-create: 
	mysql -u root -ppass -e "create database blogs; use blogs; create table ShortLink(Shortened varchar(8) PRIMARY KEY,Original varchar(255), Expiry int, Created  datetime, Hits int);"
go-install:
	apt install golang-go -y
