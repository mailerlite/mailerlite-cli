{
  description = "MailerLite CLI - command-line interface for the MailerLite API";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      version = "0.0.2";

      # Map nix system to goreleaser naming
      systemMap = {
        "x86_64-linux" = { os = "linux"; arch = "amd64"; };
        "aarch64-linux" = { os = "linux"; arch = "arm64"; };
        "x86_64-darwin" = { os = "darwin"; arch = "amd64"; };
        "aarch64-darwin" = { os = "darwin"; arch = "arm64"; };
      };

      # SHA256 hashes for each platform (updated by CI on release)
      hashes = {
        "x86_64-linux" = "sha256-E8s7hQh3U1gqK8VUWNjCcLlChH/ikOd+M/pDMdW0Thk=";
        "aarch64-linux" = "sha256-m3MHZE7PetO+a97XS6tm0v4+9MxXGi9bKGv2bV3U3LU=";
        "x86_64-darwin" = "sha256-3kwSJ/M5lqK3yaPQ7654y9vgMPSLOgDRYqm9xL3a8NU=";
        "aarch64-darwin" = "sha256-LRHfBvGdCzCM9+ee1ilysniAYgnVtSYdagwYepljW3A=";
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
