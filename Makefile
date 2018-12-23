.DEFAULT_GOAL=build-server

PACKAGES_WITH_TESTS:=$(shell go list -f="{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}" ./... | grep -v '/vendor/' | grep -v '/todo/')
TEST_TARGETS:=$(foreach p,${PACKAGES_WITH_TESTS},test-$(p))
TMP_DIR:=tmp
TEST_OUT_DIR:=$(TMP_DIR)/testout

APP:=image-resizer
TAG:=1.0.0
PROJECT:=github.com/khevse/$(APP)
COMMIT:=$(shell git rev-parse --short HEAD)
BUILD_TIME:=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION:=$(shell $(GO) version| sed -e 's/ /_/g' )

BUILD_GOOS?=linux
BUILD_GOARCH?=amd64
BUILD_CGO_ENABLED?=1

.PHONY:clear
clear:
	rm -f cmd/server/${APP}

.PHONY:govendor
govendor: clear
	dep ensure -v

.PHONY:testall
testall: govendor
	mkdir -p -m 755 $(TEST_OUT_DIR)
	$(MAKE) -j 1 $(TEST_TARGETS)
	@echo "=== tests: ok ==="

.PHONY: $(TEST_TARGETS)
$(TEST_TARGETS):
	$(eval $@_package := $(subst test-,,$@))
	$(eval $@_filename := $(subst /,_,$($@_package)))

	@echo "== test directory $($@_package) =="
	@go test $($@_package) -v -race -coverprofile $(TEST_OUT_DIR)/$($@_filename)_cover.out \
	 >> $(TEST_OUT_DIR)/$($@_filename).out \
	 || ( echo 'fail $($@_package)' &&  cat $(TEST_OUT_DIR)/$($@_filename).out; exit 1);


.PHONY: build-server
build-server: testall
	(cd cmd/server; CGO_ENABLED=$(BUILD_CGO_ENABLED) GOOS=$(BUILD_GOOS) GOARCH=$(BUILD_GOARCH) go build \
        -ldflags "-s -w \
        -X ${PROJECT}/description.AppName=${APP} \
        -X ${PROJECT}/description.Commit=${COMMIT} \
        -X ${PROJECT}/description.GoVersion=${GO_VERSION} \
        -X ${PROJECT}/description.BuildDate=${BUILD_TIME} \
        -X ${PROJECT}/description.Version=${RELEASE}" \
        -o ${APP})