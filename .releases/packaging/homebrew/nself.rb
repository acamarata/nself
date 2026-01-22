class Nself < Formula
  desc "Self-hosted infrastructure manager for developers"
  homepage "https://github.com/acamarata/nself"
  url "https://github.com/acamarata/nself/archive/refs/tags/v0.4.1.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"
  head "https://github.com/acamarata/nself.git", branch: "main"

  depends_on "bash" => :build
  depends_on "docker"
  depends_on "docker-compose"

  def install
    # Install all source files
    libexec.install Dir["*"]

    # Create wrapper script
    (bin/"nself").write <<~EOS
      #!/bin/bash
      export NSELF_HOME="#{libexec}"
      exec "#{libexec}/bin/nself" "$@"
    EOS

    # Make executable
    chmod 0755, bin/"nself"
  end

  test do
    assert_match "0.4.1", shell_output("#{bin}/nself version")
  end
end
