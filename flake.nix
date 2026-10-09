{
  description = "Keeps a directory of Obsidian notes in sync with your saved Linkwarden links";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = if (self ? shortRev) then self.shortRev else "dev";

        # nixpkgs-unstable's default `go` (1.26 as of writing) is older
        # than the go 1.27.2 directive in go.mod, and GOTOOLCHAIN can't
        # reach the network to fetch a newer one inside Nix's sandboxed
        # build — go_1_27 is nixpkgs' own matching package instead.
        # Passing `go` as a plain buildGoModule argument is silently
        # ignored; it has to go through .override, confirmed live after
        # the plain-argument form kept building with 1.26.7 anyway.
        #
        # nixos-unstable still ships go 1.27.1, which can't build a module
        # that requires 1.27.2, so bump the patch release here. Drop this
        # override once the channel's go_1_27 reaches 1.27.2.
        go = pkgs.go_1_27.overrideAttrs (old: rec {
          version = "1.27.2";
          src = pkgs.fetchurl {
            url = "https://go.dev/dl/go${version}.src.tar.gz";
            hash = "sha256-A0ldorpkiU1A9cSZLklFT6eLUGkGBP+Stq//UIG3bmI=";
          };
        });
        buildGoModule = pkgs.buildGoModule.override { inherit go; };
      in
      {
        packages.default = buildGoModule {
          pname = "linkwarden-obsidian-sync";
          inherit version;
          src = ./.;
          # Regenerate whenever go.mod/go.sum change: nix build with this
          # set to pkgs.lib.fakeHash, then substitute the hash Nix reports
          # as "got". No command computes it standalone — it's derived
          # from a real build attempt, same as goreleaser and the AUR
          # PKGBUILD both need a real build to prove their own inputs.
          vendorHash = "sha256-JlIn6CBnyEZC3FCbXBpit8SNXYIpNfelJtr4GMho46M=";
          subPackages = [ "cmd/linkwarden-obsidian-sync" ];
          # Matches goreleaser's own ldflags in .goreleaser.yaml, other
          # than the version source: goreleaser reads it off the tag,
          # this reads it off the flake's own revision.
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          meta = {
            description = "Keeps a directory of Obsidian notes in sync with your saved Linkwarden links";
            homepage = "https://github.com/alrayyes/linkwarden-obsidian-sync";
            license = pkgs.lib.licenses.gpl3Only;
            mainProgram = "linkwarden-obsidian-sync";
          };
        };

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.go_1_27 ];
        };
      }
    );
}
