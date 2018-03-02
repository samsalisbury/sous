{ pkgs ? import ~/dev/nixpkgs {} }:
#{ pkgs ? import <nixpkgs> {} }:
let
  inherit (pkgs) lib stdenv bundlerEnv;

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
      pkgs.proselint
      pkgs.postgresql100
      pkgs.liquibase
      pkgs.go
    ];
  }
