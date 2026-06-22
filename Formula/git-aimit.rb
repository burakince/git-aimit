class GitAimit < Formula
  desc "AI-powered Git commit message generator using local Ollama models"
  homepage "https://github.com/burakince/git-aimit"
  url "https://github.com/burakince/git-aimit/archive/refs/tags/v0.0.2.tar.gz"
  sha256 "8e30ec9d12b0640bcf30f4c0f18c61d37f844d31a3a3f818ec4bb95f7e44456e"
  license "MIT"
  head "https://github.com/burakince/git-aimit.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", "-trimpath", "-ldflags", "-s -w", "-o", bin/"git-aimit", "."
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
