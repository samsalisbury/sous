{ pkgs ? import ~/dev/nixpkgs {} }:
#{ pkgs ? import <nixpkgs> {} }:
let
  inherit (pkgs) lib stdenv ruby rake bundler bundlerEnv postgresql100 liquibase;

  rubyEnv = bundlerEnv {
    name = "sous-danger";

    gemdir = ./.;
  };
in
  stdenv.mkDerivation {
    name = "sous-env";
    src = ./.;

    buildInputs = [
      rubyEnv
      postgresql100
      liquibase
    ];
  }
