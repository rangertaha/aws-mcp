# Install

Prebuilt binaries are published on the [latest release](https://github.com/rangertaha/aws-mcp/releases/latest). Download the archive for your platform, extract the `aws` binary, and put it on your `PATH`.

| Platform | Architecture          | Download (latest)                                                                                                            |
| -------- | --------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| macOS    | Apple Silicon (arm64) | [`aws-mcp_darwin_arm64.tar.gz`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_darwin_arm64.tar.gz) |
| macOS    | Intel (amd64)         | [`aws-mcp_darwin_amd64.tar.gz`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_darwin_amd64.tar.gz) |
| Linux    | amd64                 | [`aws-mcp_linux_amd64.tar.gz`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_linux_amd64.tar.gz)   |
| Linux    | arm64                 | [`aws-mcp_linux_arm64.tar.gz`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_linux_arm64.tar.gz)   |
| Windows  | amd64                 | [`aws-mcp_windows_amd64.zip`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_windows_amd64.zip)     |
| Windows  | arm64                 | [`aws-mcp_windows_arm64.zip`](https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_windows_arm64.zip)     |

Each link always resolves to the newest release. A [`checksums.txt`](https://github.com/rangertaha/aws-mcp/releases/latest/download/checksums.txt) is published alongside the archives.

??? example "macOS / Linux"
    Pick your `OS`/`ARCH`:

    ```sh
    OS=darwin ARCH=arm64   # OS: darwin|linux   ARCH: amd64|arm64
    curl -sSL "https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_${OS}_${ARCH}.tar.gz" | tar -xz aws
    sudo mv aws /usr/local/bin/
    aws --version
    ```

??? example "Windows (PowerShell)"
    Pick your `$Arch`:

    ```powershell
    $Arch = "amd64"   # ARCH: amd64|arm64
    Invoke-WebRequest "https://github.com/rangertaha/aws-mcp/releases/latest/download/aws-mcp_windows_${Arch}.zip" -OutFile aws.zip
    Expand-Archive aws.zip -DestinationPath .
    .\aws.exe --version
    ```

## Alternative: install with Go

```sh
go install github.com/rangertaha/aws-mcp/cmd/aws@latest
```

## Alternative: build from source

```sh
git clone https://github.com/rangertaha/aws-mcp
cd aws-mcp
make build        # produces ./bin/aws
```

See [Development](development.md) for the full build/test/lint workflow if you're contributing.

## Next: configure it

Once `aws` is on your `PATH`, head to [Configuration](configuration.md) to set up credentials and your MCP client.
