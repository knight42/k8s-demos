.PHONY: all clean artifacts

ARCH := $(shell go env GOARCH)
OS := $(shell go env GOOS)

all: kubectl-podstatus kubectl-rm kubectl-nodestat kubectl-scaleig kubectl-roles

clean:
	@rm -f bin/*

kubectl-podstatus:
	CGO_ENABLED=0 go build -o bin/$@ ./cmd/$@

kubectl-rm:
	CGO_ENABLED=0 go build -o bin/$@ ./cmd/$@

kubectl-nodestat:
	CGO_ENABLED=0 go build -o bin/$@ ./cmd/$@

kubectl-scaleig:
	CGO_ENABLED=0 go build -o bin/$@ ./cmd/$@

kubectl-roles:
	CGO_ENABLED=0 go build -o bin/$@ ./cmd/$@

artifacts:
	CGO_ENABLED=0 go build -o bin/kubectl-nodestat_$(OS)_$(ARCH) ./cmd/kubectl-nodestat
	CGO_ENABLED=0 go build -o bin/kubectl-podstatus_$(OS)_$(ARCH) ./cmd/kubectl-podstatus
	CGO_ENABLED=0 go build -o bin/kubectl-rm_$(OS)_$(ARCH) ./cmd/kubectl-rm
	CGO_ENABLED=0 go build -o bin/kubectl-scaleig_$(OS)_$(ARCH) ./cmd/kubectl-scaleig
	CGO_ENABLED=0 go build -o bin/kubectl-roles_$(OS)_$(ARCH) ./cmd/kubectl-roles
