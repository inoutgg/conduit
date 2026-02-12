{
  pkgs,
  lib,
  ...
}:

let
  isCI = builtins.getEnv "GITHUB_ACTIONS" != "";

  default = {
    containers = lib.mkForce { };

    cachix.enable = false;

    scripts = {
      lint-ci = {
        exec = ''
          modernize ./...
          # govulncheck ./...
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
    };

    enterTest = ''
      go test -race ./...
    '';

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
      package = pkgs.go;
    };

    env.GOTOOLCHAIN = lib.mkForce "local";
    env.GOFUMPT_SPLIT_LONG_LINES = lib.mkForce "on";

    env.TEST_DATABASE_URL = lib.mkForce "postgres://test:test@localhost:5432/test?sslmode=disable";
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

default // services // (if isCI then { } else hooks)
