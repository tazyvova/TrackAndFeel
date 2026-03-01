FROM mcr.microsoft.com/devcontainers/base:ubuntu

# Go 1.23
RUN wget -qO- https://go.dev/dl/go1.23.6.linux-amd64.tar.gz | tar -C /usr/local -xzf -
ENV PATH="/usr/local/go/bin:/home/vscode/go/bin:${PATH}"

# Node 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y nodejs

# act (local CI runner)
RUN curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | bash && \
    mv bin/act /usr/local/bin/ && rm -rf bin

# gh CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
      | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
    chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
      | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
    apt-get update && apt-get install -y gh
