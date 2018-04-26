#{ pkgs ? import ~/dev/nixpkgs {} }:
{ pkgs ? import <nixpkgs> {} }:

let
  inherit (pkgs) lib stdenv bundlerEnv;

  rubyEnv = bundlerEnv {
    name = "sous-danger";

    gemdir = ./.;
  };
in
  stdenv.mkDerivation {
    name = "sous-env";

    buildInputs = with pkgs; [ rubyEnv proselint postgresql100 liquibase go ];
  }
