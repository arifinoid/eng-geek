{
  description = "eng-geek — personalized English learning tracker";

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
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            bun
            sqlc
            goose
            postgresql_17
            gopls
            gotools
          ];

          shellHook = ''
            echo "eng-geek dev shell"
            echo "  go     $(go version | awk '{print $3}')"
            echo "  bun    $(bun --version)"
            echo "  sqlc   $(sqlc version)"
            echo "  goose  $(goose -version 2>/dev/null || echo n/a)"
            echo "  pg     $(postgres --version)"
            echo
            echo "Postgres: docker compose up -d db"
            echo "Migrate:  goose -dir apps/api/db/migrations postgres \"$DATABASE_URL\" up"
          '';
        };
      }
    );
}
