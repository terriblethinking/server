{
  pkgs ? import <nixpkgs> { },
}:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go_latest
    gopls
    gomodifytags
    impl
  ];

  shellHook = ''
    export OLLAMA_API_KEY=""
  '';
}
