{
  description = "MailerLite CLI - command-line interface for the MailerLite API";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      version = "1.0.2";

      # Map nix system to goreleaser naming
      systemMap = {
        "x86_64-linux" = { os = "linux"; arch = "amd64"; };
        "aarch64-linux" = { os = "linux"; arch = "arm64"; };
        "x86_64-darwin" = { os = "darwin"; arch = "amd64"; };
        "aarch64-darwin" = { os = "darwin"; arch = "arm64"; };
      };

      # SHA256 hashes for each platform (updated by CI on release)
      hashes = {
        "x86_64-linux" = "sha256-ZIMckB4Yu0A0Xrfe7MDgSToztoXN/ePP4MeSBzJWud8=";
        "aarch64-linux" = "sha256-Wm5XFVh2l9voM8d8mSfYxTq4yBm00ihorsVkbio3njA=";
        "x86_64-darwin" = "sha256-oMrmRzlJwUOMH0Dx3l44Vudr8gLr4EaMa5dpRlM82ao=";
        "aarch64-darwin" = "sha256-elIKPI9os7J+33dXqBjhyJ3tRcKTXxCedZucz7wbuGc=";
      };
    in
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        platformInfo = systemMap.${system} or (throw "Unsupported system: ${system}");

        mailerlite = pkgs.stdenv.mkDerivation {
          pname = "mailerlite";
          inherit version;

          src = pkgs.fetchurl {
            url = "https://github.com/mailerlite/mailerlite-cli/releases/download/v${version}/mailerlite-cli_${version}_${platformInfo.os}_${platformInfo.arch}.tar.gz";
            sha256 = hashes.${system};
          };

          sourceRoot = ".";

          installPhase = ''
            install -Dm755 mailerlite $out/bin/mailerlite
          '';

          meta = with pkgs.lib; {
            description = "Command-line interface for the MailerLite API";
            homepage = "https://github.com/mailerlite/mailerlite-cli";
            license = licenses.mit;
            mainProgram = "mailerlite";
            platforms = builtins.attrNames systemMap;
          };
        };
      in
      {
        packages = {
          inherit mailerlite;
          default = mailerlite;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            golangci-lint
            lefthook
          ];
        };
      }
    );
}
