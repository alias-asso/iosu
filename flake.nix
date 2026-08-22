{
  description = "iosu — the ALIAS programming contest platform";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      # x86_64-darwin is no longer supported by nixos-unstable and throws on
      # evaluation, so it is deliberately absent.
      systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" ];

      forAllSystems = f:
        nixpkgs.lib.genAttrs systems (system: f {
          inherit system;
          pkgs = nixpkgs.legacyPackages.${system};
        });
    in
    {
      packages = forAllSystems ({ pkgs, ... }: rec {
        iosu = pkgs.buildGoModule {
          pname = "iosu";
          version = "0.1.0";
          src = ./.;

          vendorHash = "sha256-HC5muHihjxKQmjsPUvPMO/slCTiVnsh74OXforfK28k=";

          subPackages = [ "cmd/iosud" "cmd/iosu" ];

          # The SQLite driver is pure Go, so the binaries link statically.
          env.CGO_ENABLED = 0;
          ldflags = [ "-s" "-w" ];

          # subPackages would narrow the check phase to cmd/, which has no
          # tests. Run the whole suite instead. -race needs cgo, so it stays a
          # dev-shell and CI concern.
          checkPhase = ''
            runHook preCheck
            go test ./...
            runHook postCheck
          '';

          meta = {
            description = "Programming contest platform with per-contestant inputs";
            homepage = "https://github.com/alias-asso/iosu";
            license = pkgs.lib.licenses.bsd3;
            mainProgram = "iosud";
          };
        };
        default = iosu;
      });

      devShells = forAllSystems ({ pkgs, ... }: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            go-tools # staticcheck
            govulncheck

            # Regenerates internal/store/sqlc from the migrations and queries.
            sqlc

            # For poking at the database by hand.
            sqlite
          ];

          shellHook = ''
            echo "iosu dev shell — go $(go version | cut -d' ' -f3 | sed 's/^go//'), sqlc ${pkgs.sqlc.version}"
            echo
            echo "  go test -race ./...                  test"
            echo "  go vet ./... && staticcheck ./...    lint"
            echo "  sqlc generate                        regenerate internal/store/sqlc"
            echo "  nix build                            build both binaries into ./result"
            echo
          '';
        };
      });

      # nix flake check builds the package, which runs the tests too.
      checks = forAllSystems ({ system, ... }: {
        build = self.packages.${system}.iosu;
      });

      apps = forAllSystems ({ system, ... }: rec {
        iosud = {
          type = "app";
          program = "${self.packages.${system}.iosu}/bin/iosud";
        };
        iosu = {
          type = "app";
          program = "${self.packages.${system}.iosu}/bin/iosu";
        };
        default = iosud;
      });
    };
}
