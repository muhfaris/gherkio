.PHONY: build install run test clean dev lint bump-patch bump-minor

# ─── Metadata ────────────────────────────────────────────────
APP      := gherkio
PKG      := github.com/muhfaris/gherkio/cmd
VERSION  := $(shell grep 'Version\s*=' cmd/root.go | head -1 | sed "s/.*= \"//;s/\"//")

# ─── Build flags ──────────────────────────────────────────────
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w \
            -X $(PKG).Version=$(VERSION) \
            -X $(PKG).Commit=$(COMMIT) \
            -X $(PKG).BuildDate=$(DATE)

# ─── Development — run from source (fast, no build) ──────────
dev:
	go run . $(ARGS)

run: dev

# ─── Build ────────────────────────────────────────────────────
build:
	go build -ldflags="$(LDFLAGS)" -o $(APP) .

# ─── Install to PATH ──────────────────────────────────────────
install: build
	@echo "Installing $(APP) $(VERSION) ($(COMMIT))…"
	sudo cp $(APP) /usr/local/bin/$(APP)
	@echo "✅ $(APP) installed: $$(which $(APP))"
	@which $(APP) && $(APP) --version

# ─── Test ─────────────────────────────────────────────────────
test:
	go test ./...

test-verbose:
	go test -v ./...

# ─── Clean ────────────────────────────────────────────────────
clean:
	rm -f $(APP)

# ─── Lint ─────────────────────────────────────────────────────
lint:
	go vet ./...

# ─── Version bumping ──────────────────────────────────────────
# Updates the hardcoded default in cmd/root.go.
# (The goreleaser ldflags still override on tagged releases.)
bump-patch:
	@echo "Bumping patch: $(VERSION) → $$(echo $(VERSION) | awk -F. '{print $$1"."$$2"."$$3+1}')"
	@sed -i "s/Version\s*= \"$(VERSION)\"/Version = \"$$(echo $(VERSION) | awk -F. '{print $$1"."$$2"."$$3+1}')\"/" cmd/root.go
	$(eval VERSION := $(shell grep 'Version\s*=' cmd/root.go | head -1 | sed "s/.*= \"//;s/\"//"))
	@echo "✅ Now v$(VERSION)"

bump-minor:
	@echo "Bumping minor: $(VERSION) → $$(echo $(VERSION) | awk -F. '{print $$1"."$$2+1".0"}')"
	@sed -i "s/Version\s*= \"$(VERSION)\"/Version = \"$$(echo $(VERSION) | awk -F. '{print $$1"."$$2+1".0"}')\"/" cmd/root.go
	$(eval VERSION := $(shell grep 'Version\s*=' cmd/root.go | head -1 | sed "s/.*= \"//;s/\"//"))
	@echo "✅ Now v$(VERSION)"

bump-major:
	@echo "Bumping major: $(VERSION) → $$(echo $(VERSION) | awk -F. '{print $$1+1".0.0"}')"
	@sed -i "s/Version\s*= \"$(VERSION)\"/Version = \"$$(echo $(VERSION) | awk -F. '{print $$1+1".0.0"}')\"/" cmd/root.go
	$(eval VERSION := $(shell grep 'Version\s*=' cmd/root.go | head -1 | sed "s/.*= \"//;s/\"//"))
	@echo "✅ Now v$(VERSION)"
