class Nself < Formula
  desc "Self-hosted infrastructure manager for developers"
  homepage "https://github.com/acamarata/nself"
  url "https://github.com/acamarata/nself/archive/refs/tags/v0.3.8.tar.gz"
  sha256 "bf6ee149699bddda77f72ceed9c0491eef8ba04431239eadb8b6b77616d260de"
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
    assert_match "v0.3.8", shell_output("#{bin}/nself version")
  end
end