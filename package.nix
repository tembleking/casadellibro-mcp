{ buildGoModule }:
buildGoModule {
  pname = "app";
  version = "0.1.0";
  src = ./.;
  vendorHash = "sha256-KkzsEsq9EtRY7Rk/bE2BHfFIBtvx7yNVe3XTEBjjD4w=";

  subPackages = [ "cmd/app" ];

  ldflags = [
    "-w"
    "-s"
  ];

  env.CGO_ENABLED = 0;

  meta.mainProgram = "app";
}
