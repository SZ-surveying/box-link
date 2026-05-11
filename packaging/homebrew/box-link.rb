class BoxLink < Formula
  desc "Direct-link box networking tool for macOS"
  homepage "https://github.com/rainy/box-link"
  version "dev"

  on_arm do
    url "https://github.com/rainy/box-link/releases/download/v#{version}/box-link-v#{version}-darwin-arm64.tar.gz"
    sha256 "ff22b49254d87be046e2191bedfb38e9212f117da777197962408989c04b36ba"
  end

  on_intel do
    url "https://github.com/rainy/box-link/releases/download/v#{version}/box-link-v#{version}-darwin-amd64.tar.gz"
    sha256 "a2d60c93ee3d31b6e2d56c94e3e353885aa16cf8254a73e8073589d1e72fb9b6"
  end

  def install
    bin.install "box-link"
    prefix.install "README.md"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/box-link version")
  end
end
