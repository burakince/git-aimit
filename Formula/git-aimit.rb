class GitAimit < Formula
  desc "AI-powered Git commit message generator using local Ollama models"
  homepage "https://github.com/burakince/git-aimit"
  url "https://github.com/burakince/git-aimit/archive/refs/tags/v0.0.1.tar.gz"
  sha256 "b813a57487c19c20517450d17efecc2ef0776337c7ac363142a7767c8fd3f84f"
  license "MIT"
  head "https://github.com/burakince/git-aimit.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "."
  end

  test do
    assert_match "AI-powered Git commit message generator", shell_output("#{bin}/git-aimit --help")
  end
end
