{ buildGoModule }:
buildGoModule {
  pname = "app";
  version = "0.1.0";
  src = ./.;
  vendorHash = "sha256-yu/1zJyb0sFuk6dofBXTcf6/5StVWLdLGGQ4xV5klkM=";

  subPackages = [ "cmd/app" ];

  ldflags = [
    "-w"
    "-s"
  ];

  env.CGO_ENABLED = 0;

  meta.mainProgram = "app";
}
