NAME = $(notdir $(PWD))

VERSION = $(shell printf "%s.%s" \
	$$(git rev-list --count HEAD) \
	$$(git rev-parse --short HEAD) \
)

GOFLAGS = GO111MODULE=off CGO_ENABLED=0

ORGALORG = orgalorg -u root -o $(HOST) -y

version:
	@echo $(VERSION)

test:
	$(GOFLAGS) go test -failfast -v ./cmd/...

get:
	$(GOFLAGS) go get -v -d -v ./cmd/...

build@%:
	$(GOFLAGS) go build \
		 -ldflags="-s -w -X main.version=$(VERSION)" \
		 -gcflags="-trimpath=$(GOPATH)" ./cmd/$*

dist@aurorad: build@aurorad
	mkdir -p dist/usr/lib/systemd/system/
	mkdir -p dist/usr/bin/
	cp aurorad dist/usr/bin/
	cp systemd/* dist/usr/lib/systemd/system

release@aurorad: clean dist@aurorad
	$(if $(HOST),,$(error HOST is not set))
	@echo :: releasing version $(VERSION)
	@echo :: stopping aurora on server
	@$(ORGALORG) -C systemctl stop aurora aurora-web
	@echo :: uploading dist
	@cd dist && $(ORGALORG) --root / -e -U .
	@echo :: reloading daemon
	@$(ORGALORG) -C 'systemctl daemon-reload'
	@echo :: starting services
	@$(ORGALORG) -C 'systemctl start aurora aurora-web'
	@$(ORGALORG) -C 'systemctl status aurora aurora-web'

clean:
	rm -rf dist
	rm -rf aurorad aurora
