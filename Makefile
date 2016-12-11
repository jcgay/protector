OS = darwin freebsd linux openbsd windows
ARCHS = 386 arm amd64 arm64

all: build release

build: deps
	go build

release: clean deps
	@for arch in $(ARCHS);\
	do \
		for os in $(OS);\
		do \
			echo "Building $$os-$$arch"; \
			mkdir -p build/protector-$$os-$$arch/; \
			GOOS=$$os GOARCH=$$arch go build -o build/protector-$$os-$$arch/protector; \
			tar cz -C build -f build/protector-$$os-$$arch.tar.gz protector-$$os-$$arch; \
		done \
	done

test: deps
	go test ./...

deps:
	go get -d -v -t ./...

clean:
	rm -rf build
	rm -f protector
