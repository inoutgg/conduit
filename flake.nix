{
  description = "conduit is an SQL migrator that is easy to embed";

  inputs = {
    devenv.url = "github:cachix/devenv";
    nixpkgs.url = "nixpkgs/nixos-unstable";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    flake-parts.url = "github:hercules-ci/flake-parts";
    flake-root.url = "github:srid/flake-root";
  };

  outputs =
    {
      flake-parts,
      ...
    }@inputs:
    let
      isCI = builtins.getEnv "GITHUB_ACTIONS" != "";
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      flake = { };

      systems = [
        "x86_64-linux"
        "x86_64-darwin"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      imports = [
        inputs.flake-root.flakeModule
        inputs.treefmt-nix.flakeModule
        inputs.devenv.flakeModule
      ];

      perSystem =
        {
          pkgs,
          lib,
          config,
          ...
        }:
        {
          formatter = config.treefmt.build.wrapper;
          treefmt.config = {
            inherit (config.flake-root) projectRootFile;
            package = pkgs.treefmt;

            programs = {
              nixfmt.enable = true;
              gofumpt.enable = true;
              yamlfmt.enable = true;
            };
          };

          devenv.shells.default =
            let
              default = {
                containers = lib.mkForce { };

                cachix.enable = true;
                cachix.pull = [
                  "devenv"
                ];

                scripts = {
                  # golangci-lint is running by gh action.
                  lint-ci = {
                    exec = ''
                      modernize ./...
                        govulncheck ./...
                    '';
                  };
                  lint-all = {
                    exec = ''
                      lint-ci
                      golangci-lint run ./...
                    '';
                  };
                  lint-fix = {
                    exec = ''
                      modernize --fix ./...
                      golangci-lint run --fix ./...
                    '';
                  };
                  go-test =
                    let
                      parallel = pkgs.stdenv.hostPlatform.parsed.cpu.cores or 4;
                    in
                    {
                      exec = ''
                        go test -parallel ${toString parallel} ./...
                      '';
                    };
                };

                packages = with pkgs; [
                  sqlc
                  typos

                  gcc
                  gotools
                  govulncheck
                  golangci-lint
                ];

                languages.go = {
                  enable = true;
                  package = pkgs.go_1_25;
                };

                env.GOTOOLCHAIN = lib.mkForce "local";
                env.GOFUMPT_SPLIT_LONG_LINES = lib.mkForce "on";

                env.TEST_DATABASE_URL = lib.mkForce "postgres://test:test@localhost:5432/test";
              };

              services = {
                services.postgres = {
                  enable = true;
                  package = pkgs.postgresql_17;

                  initialScript = ''
                    CREATE USER test SUPERUSER PASSWORD 'test';
                    CREATE DATABASE test OWNER test;
                  '';
                  listen_addresses = "localhost";
                  port = 5432;
                };
              };

              hooks = {
                git-hooks = {
                  hooks = {
                    nix-check = {
                      enable = true;
                      name = "nix-check";
                      description = "Nix Check";
                      entry = ''
                        nix flake check --no-pure-eval
                      '';
                      pass_filenames = false;
                    };
                    lint = {
                      enable = true;
                      name = "lint";
                      description = "Lint";
                      entry = ''
                        lint-all
                      '';
                      pass_filenames = false;
                    };
                  };
                };
              };
            in

            default // (if isCI then { } else services // hooks);
        };
    };
}
