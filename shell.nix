#{ pkgs ? import ~/dev/nixpkgs {} }:
{ pkgs ? import <nixpkgs> {} }:
let
  inherit (pkgs) lib stdenv ruby rake bundler bundlerEnv postgresql100 liquibase proselint docker_compose;

  rubyEnv = bundlerEnv {
    name = "sous-danger";

    gemdir = ./.;
  };
in
  stdenv.mkDerivation {
    name = "sous-env";
    src = ./.;

    buildInputs = [
      proselint
      rubyEnv
      postgresql100
      liquibase
      docker_compose
    ];
  }
