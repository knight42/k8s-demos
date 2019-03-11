.PHONY: all clean

all: kubectl-podstatus kubectl-rm

clean:
	@rm -f bin/*

kubectl-podstatus:
	CGO_ENABLED=0 go build -o bin/$@ ./cmd/$@

kubectl-rm:
	CGO_ENABLED=0 go build -o bin/$@ ./cmd/$@
