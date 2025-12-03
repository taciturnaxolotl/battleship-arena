{
  description = "Battleship Arena - SSH battleship tournament service";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      allSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems =
        f:
        nixpkgs.lib.genAttrs allSystems (
          system:
          f {
            pkgs = import nixpkgs { inherit system; };
          }
        );
    in
    {
      packages = forAllSystems (
        { pkgs }:
        {
          default = pkgs.buildGoModule {
            pname = "battleship-arena";
            version = "0.1.0";
            subPackages = [ "cmd/battleship-arena" ];
            src = self;

            vendorHash = null;

            nativeBuildInputs = [ pkgs.makeWrapper ];
            buildInputs = [ pkgs.gcc ];

            env.CGO_ENABLED = "1";

            ldflags = [
              "-s"
              "-w"
            ];

            meta = with pkgs.lib; {
              description = "SSH-based battleship tournament service";
              homepage = "https://github.com/taciturnaxolotl/battleship-arena";
              license = licenses.mit;
              mainProgram = "battleship-arena";
            };
          };
        }
      );
    };
}
