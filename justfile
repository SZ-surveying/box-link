set dotenv-load

root := justfile_directory()
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`

_default:
    @just --list

fmt:
    gofmt -w ./cmd ./internal

build:
    mkdir -p {{ root }}/bin
    go build -o {{ root }}/bin/box-link ./cmd/box-link

test:
    go test ./...

run *args:
    go run ./cmd/box-link {{ args }}

clean:
    rm -rf {{ root }}/bin {{ root }}/dist

package:
    ./packaging/package.sh --version {{ version }}

package-release:
    ./packaging/package.sh --release --version {{ version }}

package-pkg identifier="com.example.box-link":
    ./packaging/build-pkg.sh --version {{ version }} --identifier {{ identifier }}

homebrew-formula repo="your-org/box-link":
    ./packaging/generate-homebrew-formula.sh --version {{ version }} --repo {{ repo }}

homebrew-tap repo="your-org/box-link" tap_dir="../homebrew-tools":
    ./packaging/generate-homebrew-formula.sh --version {{ version }} --repo {{ repo }} --tap-dir {{ tap_dir }}

changelog:
    @if command -v devbox >/dev/null 2>&1; then \
        devbox run -- git-cliff -o CHANGELOG.md; \
    else \
        git-cliff -o CHANGELOG.md; \
    fi

release version:
    just changelog
    git add CHANGELOG.md
    git commit -m "chore(release): version {{ version }}" --no-verify
    git push -u origin main
    git tag -a "v{{ version }}" -m "Release version {{ version }}"
    git push origin "v{{ version }}"
