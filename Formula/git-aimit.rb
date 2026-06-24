class GitAimit < Formula
  desc "AI-powered Git commit message generator using local Ollama models"
  homepage "https://github.com/burakince/git-aimit"
  version "0.0.4"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.4/git-aimit-darwin-arm64"
      sha256 "6ddab81ad8dc1f40d2ec70f819f5f0844ce57d2573604f33e9264f52aa68c286"
    end
    on_intel do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.4/git-aimit-darwin-amd64"
      sha256 "5743de3e6036976295d78db46f67c82bad4c9174c6352c1df8122e599d2c4190"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.4/git-aimit-linux-arm64"
      sha256 "083f5c15a4c78dd465c8568cba6c1eb48e9aaf626890b4393ccb1769857fda61"
    end
    on_intel do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.4/git-aimit-linux-amd64"
      sha256 "67e710a2815d73a58c943cb31ec3abcf03f8d82c76c11a1e1e9d932cf4c5aaf5"
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
