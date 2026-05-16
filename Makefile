.PHONY: test conformance cross-platform fuzz oracle ci clean

test:
	go test ./ccnf/...

conformance:
	go test -run TestGoldenVectors ./ccnf/...

cross-platform:
	@echo "--- cross-compile: linux/amd64 ---"
	GOOS=linux   GOARCH=amd64 go build -o /dev/null ./conformance
	@echo "--- cross-compile: darwin/arm64 ---"
	GOOS=darwin  GOARCH=arm64 go build -o /dev/null ./conformance
	@echo "--- cross-compile: windows/amd64 ---"
	GOOS=windows GOARCH=amd64 go build -o /dev/null ./conformance
	@echo "--- cross-compile OK ---"

fuzz:
	go test -run TestFuzz -count=1 -timeout 60s ./ccnf/...

oracle:
	go build -o ./bin/ccnf-conformance ./conformance

ci: test conformance cross-platform fuzz
	@echo "--- CI gate: all OK ---"

clean:
	rm -rf ./bin

.DEFAULT_GOAL := test
