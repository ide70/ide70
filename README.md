# ide70

Minimal web ide builder (early phase)

Own IDE and API

Sample application


## Install and run

### Linux

Install go 1.13. first. Then run the following commands:

```
git clone https://github.com/ide70/ide70.git
cd ide70
export GOPATH=`pwd`
(cd src/github.com/ide70/ide70 && go get ./...)
bin/ide70
```

### Windows

```
git clone https://github.com/ide70/ide70.git
cd ide70
set GOPATH=%cd%
pushd src\github.com\ide70\ide70 && go get ./... && popd
bin\ide70.exe
```

Open [IDE](http://localhost:7080/app/ide/login) or [Sample application](http://localhost:7080/app/airplane/login)
in your browser and enjoy.
