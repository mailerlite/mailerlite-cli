{
  description = "MailerLite CLI - command-line interface for the MailerLite API";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      version = "2.0.0";

      # Map nix system to Rust target triples (cargo-dist artifact naming)
      systemMap = {
        "x86_64-linux" = "x86_64-unknown-linux-gnu";
        "aarch64-linux" = "aarch64-unknown-linux-gnu";
        "x86_64-darwin" = "x86_64-apple-darwin";
        "aarch64-darwin" = "aarch64-apple-darwin";
      };

      # SHA256 hashes for each platform (updated by CI on release)
      hashes = {
        "x86_64-linux" = "sha256-Izz73HSWs/L6kpDSGFU6KQ76mUvsFuRzorCWNNarGxY=";
        "aarch64-linux" = "sha256-292Pv9Li3I5tx8xYalMEGasP/6/cOldnPsVlyysLsps=";
        "x86_64-darwin" = "sha256-DZByEits2ctnhzD7L0rCvG/y0Fai+1G8jeBP0Kxqos8=";
        "aarch64-darwin" = "sha256-3mmlu4leb0QruLKFUJvJOFW154t5MQiLvXwApUdd1NE=";
      };
    in
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        target = systemMap.${system} or (throw "Unsupported system: ${system}");

        mailerlite = pkgs.stdenv.mkDerivation {
          pname = "mailerlite";
          inherit version;

          src = pkgs.fetchurl {
            url = "https://github.com/mailerlite/mailerlite-cli/releases/download/v${version}/mailerlite-${target}.tar.gz";
            sha256 = hashes.${system};
          };

          # cargo-dist tarballs unpack to a directory named after the archive
          sourceRoot = "mailerlite-${target}";

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
            cargo
            rustc
            rustfmt
            clippy
            lefthook
          ];
        };
      }
    );
}
