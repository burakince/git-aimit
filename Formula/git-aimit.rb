class GitAimit < Formula
  desc "AI-powered Git commit message generator using local Ollama models"
  homepage "https://github.com/burakince/git-aimit"
  version "0.0.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.1/git-aimit-darwin-arm64"
      sha256 "2f4368b21cecd42d69839f4a2e510cde8e8f7abe6bbcd51293075dcff2bc233f"
    end
    on_intel do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.1/git-aimit-darwin-amd64"
      sha256 "3434a104de13c7280d155a6ffb7985359cee562ee5ef5e256e584703e5d453a3"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.1/git-aimit-linux-arm64"
      sha256 "9b38b6de9017e4e7a1a3495ea48092f3f0b5e77a59ac9aeb79877ee169bee17f"
    end
    on_intel do
      url "https://github.com/burakince/git-aimit/releases/download/v0.0.1/git-aimit-linux-amd64"
      sha256 "e1cbec354db184cef06a13ded5fb27a7fe4b5a1a941878b76854f14653ef787a"
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
