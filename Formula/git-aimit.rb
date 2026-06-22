class GitAimit < Formula
  desc "AI-powered Git commit message generator using local Ollama models"
  homepage "https://github.com/burakince/git-aimit"
  version "0.0.2"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.2/git-aimit-darwin-arm64"
      sha256 "8becb817a3e4473243f16ffefc8cdc4b01943c68269adf017098867f0ee52434"
    end
    on_intel do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.2/git-aimit-darwin-amd64"
      sha256 "b274d3100b80b0fdb58a299e98f2f8a0926369b099fef335b61fb4fda46a9256"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.2/git-aimit-linux-arm64"
      sha256 "eb33c6c396972020333af701b1d574d15f75d5d79725792f061c1fb88edb4fce"
    end
    on_intel do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.2/git-aimit-linux-amd64"
      sha256 "ca1a61c8425f31b8c9c6982692a2681e55ec39653ef8cc347cb37a55297530f9"
    end
  end

  def install
    os   = OS.mac? ? "darwin" : "linux"
    arch = Hardware::CPU.arm? ? "arm64" : "amd64"
    bin.install "git-aimit-#{os}-#{arch}" => "git-aimit"
  end

  def caveats
    <<~EOS
      Before first use, run the interactive setup:
        git aimit init
    EOS
  end

  test do
    assert_match "AI-powered Git commit message generator", shell_output("#{bin}/git-aimit --help")
  end
end
