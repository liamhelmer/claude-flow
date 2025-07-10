class KubectlSwarm < Formula
  desc "kubectl plugin for managing AI agent swarms in Kubernetes"
  homepage "https://github.com/claude-flow/kubectl-swarm"
  version "0.1.0"
  
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/claude-flow/kubectl-swarm/releases/download/v#{version}/kubectl-swarm-darwin-arm64.tar.gz"
      sha256 "TO_BE_GENERATED"
    else
      url "https://github.com/claude-flow/kubectl-swarm/releases/download/v#{version}/kubectl-swarm-darwin-amd64.tar.gz"
      sha256 "TO_BE_GENERATED"
    end
  end
  
  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/claude-flow/kubectl-swarm/releases/download/v#{version}/kubectl-swarm-linux-arm64.tar.gz"
      sha256 "TO_BE_GENERATED"
    else
      url "https://github.com/claude-flow/kubectl-swarm/releases/download/v#{version}/kubectl-swarm-linux-amd64.tar.gz"
      sha256 "TO_BE_GENERATED"
    end
  end

  depends_on "kubectl"

  def install
    bin.install "kubectl-swarm"
    
    # Install shell completions
    output = Utils.safe_popen_read(bin/"kubectl-swarm", "completion", "bash")
    (bash_completion/"kubectl-swarm").write output
    
    output = Utils.safe_popen_read(bin/"kubectl-swarm", "completion", "zsh")
    (zsh_completion/"_kubectl-swarm").write output
    
    output = Utils.safe_popen_read(bin/"kubectl-swarm", "completion", "fish")
    (fish_completion/"kubectl-swarm.fish").write output
  end

  test do
    run_output = shell_output("#{bin}/kubectl-swarm --help")
    assert_match "Manage AI agent swarms in Kubernetes", run_output
  end
end