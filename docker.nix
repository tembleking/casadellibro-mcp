{
  dockerTools,
  cacert,
  app,
}:
# OCI image built straight from the `app` package. The tag tracks
# package.nix's version so bumping it there produces a new image tag.
dockerTools.buildLayeredImage {
  name = "casadellibro-mcp";
  tag = app.version;

  # CA certificates are required for the outbound HTTPS calls to the
  # empathy.co and casadellibro APIs (the binary is otherwise static).
  contents = [ cacert ];

  config = {
    Entrypoint = [ "${app}/bin/app" ];
    Cmd = [
      "serve"
      "--transport"
      "http"
    ];
    ExposedPorts."8080/tcp" = { };
    Env = [
      "PORT=8080"
      "SSL_CERT_FILE=/etc/ssl/certs/ca-bundle.crt"
    ];
  };
}
