{
  description = "MailerLite CLI - command-line interface for the MailerLite API";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      version = "1.0.1";

      # Map nix system to goreleaser naming
      systemMap = {
        "x86_64-linux" = { os = "linux"; arch = "amd64"; };
        "aarch64-linux" = { os = "linux"; arch = "arm64"; };
        "x86_64-darwin" = { os = "darwin"; arch = "amd64"; };
        "aarch64-darwin" = { os = "darwin"; arch = "arm64"; };
      };

      # SHA256 hashes for each platform (updated by CI on release)
      hashes = {
        "x86_64-linux" = "sha256-emBunsgBGtsQTRrnrDLUSWIglcD6Uy1SZ+6DfGb0xiQ=";
        "aarch64-linux" = "sha256-rxekCZfTfZDxh80h9GoFelkp8ceoqfP8OyCL33oQrPI=";
        "x86_64-darwin" = "sha256-5yWNYFbTNOjWQJgZ5D8nr9Qwnqq9ZXG8j4yYbTLBu+Q=";
        "aarch64-darwin" = "sha256-oSLYrARP3W6Ar+x9U4WkvdfqYuMP+ewhdPE2OKT7Sho=";
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
