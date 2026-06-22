class GitAimit < Formula
  desc "AI-powered Git commit message generator using local Ollama models"
  homepage "https://github.com/burakince/git-aimit"
  version "0.0.3"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.3/git-aimit-darwin-arm64"
      sha256 "7dcd79e95ad3adcee6a39718d92c3eeaaf4107b8a90bcbadf2bef9c0a919aae0"
    end
    on_intel do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.3/git-aimit-darwin-amd64"
      sha256 "911854b6cf7bd5e7e0ffaadb5101ac8dd7cb13694124ab8867734e96970fb5b9"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.3/git-aimit-linux-arm64"
      sha256 "b28629cb2ae0091ba8711ca8a92899e22e4206ce90b31cd59b1ab374b34b4f7a"
    end
    on_intel do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.3/git-aimit-linux-amd64"
      sha256 "262cfe4638560286f3aa83043d3acef284a3744c7b88d2ddaae9fa6bbcd46685"
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
