{ pkgs ? import <nixpkgs> {} }:
let
  inherit (pkgs) lib stdenv ruby rake bundler bundlerEnv;

  rubyEnv = bundlerEnv {
    name = "sous-danger";

    gemdir = ./.;
  };
in
  rubyEnv.env
