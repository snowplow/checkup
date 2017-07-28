.PHONY: all format lint test release release-dry clean

# -----------------------------------------------------------------------------
#  CONSTANTS
# -----------------------------------------------------------------------------

build_dir = build

depend_log    = $(build_dir)/.depend

coverage_dir  = $(build_dir)/coverage
coverage_out  = $(coverage_dir)/coverage.out
coverage_html = $(coverage_dir)/coverage.html

output_dir    = $(build_dir)/output
linux_dir     = $(output_dir)/linux
darwin_dir    = $(output_dir)/darwin
windows_dir   = $(output_dir)/windows

bin_name      = checkup
bin_linux     = $(linux_dir)/$(bin_name)
bin_darwin    = $(darwin_dir)/$(bin_name)
bin_windows   = $(windows_dir)/$(bin_name)

# -----------------------------------------------------------------------------
#  BUILDING
# -----------------------------------------------------------------------------

all:
	gox -osarch=linux/amd64 -output=$(bin_linux) ./cmd/checkup
	gox -osarch=darwin/amd64 -output=$(bin_darwin) ./cmd/checkup
	gox -osarch=windows/amd64 -output=$(bin_windows) ./cmd/checkup

# -----------------------------------------------------------------------------
#  FORMATTING
# -----------------------------------------------------------------------------

format:
	go fmt ./
	gofmt -s -w ./

lint:
	golint ./

# -----------------------------------------------------------------------------
#  TESTING
# -----------------------------------------------------------------------------

test:
	mkdir -p $(coverage_dir)
	go test -tags test -v -covermode=count -coverprofile=$(coverage_out)
	go tool cover -html=$(coverage_out) -o $(coverage_html)

# -----------------------------------------------------------------------------
#  RELEASE
# -----------------------------------------------------------------------------

release: all
	release-manager --config .release.yml --check-version --make-artifact --make-version --upload-artifact

release-dry: all
	release-manager --config .release.yml --check-version --make-artifact

# -----------------------------------------------------------------------------
#  CLEANUP
# -----------------------------------------------------------------------------

clean: $(depend_log)
	rm -rf $(build_dir)

# -----------------------------------------------------------------------------
#  DEPENDENCIES
# -----------------------------------------------------------------------------

depend: $(depend_log)

$(depend_log):
	mkdir -p $(build_dir)

	# Build dependencies
	go get -u -t ./cmd/checkup

	# Test dependencies
	go get -u github.com/stretchr/testify/assert
	go get -u golang.org/x/tools/cmd/cover/...

	# Formatting dependencies
	go get -u github.com/golang/lint/golint

	# Build dependencies
	go get -u github.com/mitchellh/gox/...
	go get -u github.com/inconshreveable/mousetrap

	@echo Dependencies fetched at: `/bin/date "+%Y-%m-%d---%H-%M-%S"` >> $(depend_log);

include $(depend_log)
